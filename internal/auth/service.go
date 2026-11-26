package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrEmailTaken = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized = errors.New("unauthorized")
	ErrAccountInactive = errors.New("account is not active")
	ErrLoginRateLimited = errors.New("login temporarily rate limited")
)


type Service struct { pool *pgxpool.Pool }

type User struct {
	ID string `json:"id"`
	TenantID string `json:"tenant_id"`
	Email string `json:"email"`
	DisplayName string `json:"display_name"`
	Role string `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	Token string `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	RefreshToken string `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	User User `json:"user"`
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) Register(ctx context.Context, email, password, displayName, userAgent, remoteAddr string) (Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)
	if len(email) > 254 || !validEmail(email) || len(password) < 10 || len(password) > 72 || len(displayName) > 120 {
		return Session{}, ErrInvalidInput
	}
	if displayName == "" { displayName = strings.Split(email, "@")[0] }
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil { return Session{}, fmt.Errorf("hash password: %w", err) }
	token, tokenHash, err := newToken()
	if err != nil { return Session{}, err }
	expires := time.Now().UTC().Add(accessSessionLifetime)
	refreshToken, refreshHash, err := newToken()
	if err != nil { return Session{}, err }
	refreshExpires := time.Now().UTC().Add(refreshTokenLifetime)
	ip := clientIP(remoteAddr)

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil { return Session{}, fmt.Errorf("begin registration: %w", err) }
	defer tx.Rollback(ctx)

	var tenantID, userID string
	var createdAt time.Time
	if err := tx.QueryRow(ctx, "INSERT INTO tenants(name) VALUES($1) RETURNING id::text", displayName).Scan(&tenantID); err != nil {
		return Session{}, fmt.Errorf("create tenant: %w", err)
	}
	err = tx.QueryRow(ctx, `INSERT INTO users(tenant_id,email,password_hash,display_name)
		VALUES($1,$2,$3,$4) RETURNING id::text,created_at`, tenantID, email, string(hash), displayName).Scan(&userID, &createdAt)
	if err != nil {
		if isUniqueViolation(err) { return Session{}, ErrEmailTaken }
		return Session{}, fmt.Errorf("create user: %w", err)
	}
	var familyID string
	if err := tx.QueryRow(ctx, `INSERT INTO user_refresh_tokens(user_id,token_hash,expires_at,user_agent,ip_address)
		VALUES($1,$2,$3,$4,NULLIF($5,'')::inet) RETURNING family_id::text`, userID, refreshHash, refreshExpires, userAgent, ip).Scan(&familyID); err != nil {
		return Session{}, fmt.Errorf("create refresh token: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_sessions(user_id,family_id,token_hash,expires_at,user_agent,ip_address)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::inet)`, userID, familyID, tokenHash, expires, userAgent, ip); err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id,ip_address)
		VALUES($1,$2,'auth.register','user',$3,NULLIF($4,'')::inet)`, tenantID, userID, userID, ip); err != nil {
		return Session{}, fmt.Errorf("write registration audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil { return Session{}, fmt.Errorf("commit registration: %w", err) }
	return Session{Token: token, ExpiresAt: expires, RefreshToken: refreshToken, RefreshExpiresAt: refreshExpires, User: User{ID:userID,TenantID:tenantID,Email:email,DisplayName:displayName,Role:"user",CreatedAt:createdAt}}, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, remoteAddr string) (Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	ip := clientIP(remoteAddr)
	identifiers := loginIdentifiers(email,ip)

	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{})
	if err!=nil { return Session{},fmt.Errorf("begin login transaction: %w",err) }
	defer tx.Rollback(ctx)
	if err:=lockLoginIdentifiers(ctx,tx,identifiers);err!=nil { return Session{},err }
	limited,err:=loginRateLimited(ctx,tx,identifiers)
	if err!=nil { return Session{},err }
	if limited {
		if err:=tx.Commit(ctx);err!=nil { return Session{},fmt.Errorf("commit rate-limited login: %w",err) }
		return Session{},ErrLoginRateLimited
	}

	var user User
	var passwordHash,status,tenantStatus string
	err=tx.QueryRow(ctx,`SELECT u.id::text,u.tenant_id::text,u.email,u.display_name,u.role,u.password_hash,u.status,u.created_at,t.status
		FROM users u JOIN tenants t ON t.id=u.tenant_id WHERE u.email=$1`,email).
		Scan(&user.ID,&user.TenantID,&user.Email,&user.DisplayName,&user.Role,&passwordHash,&status,&user.CreatedAt,&tenantStatus)
	if err!=nil && !errors.Is(err,pgx.ErrNoRows) {
		return Session{},fmt.Errorf("load account for login: %w",err)
	}
	if errors.Is(err,pgx.ErrNoRows) || bcrypt.CompareHashAndPassword([]byte(passwordHash),[]byte(password))!=nil {
		if err:=recordLoginFailure(ctx,tx,identifiers);err!=nil { return Session{},err }
		if err:=tx.Commit(ctx);err!=nil { return Session{},fmt.Errorf("commit failed login attempt: %w",err) }
		return Session{},ErrInvalidCredentials
	}
	if status!="active" || tenantStatus!="active" { return Session{},ErrAccountInactive }

	token,tokenHash,err:=newToken()
	if err!=nil { return Session{},err }
	refreshToken,refreshHash,err:=newToken()
	if err!=nil { return Session{},err }
	now:=time.Now().UTC()
	expires:=now.Add(accessSessionLifetime)
	refreshExpires:=now.Add(refreshTokenLifetime)
	var familyID string
	if err:=tx.QueryRow(ctx,`INSERT INTO user_refresh_tokens(user_id,token_hash,expires_at,user_agent,ip_address)
		VALUES($1,$2,$3,$4,NULLIF($5,'')::inet) RETURNING family_id::text`,
		user.ID,refreshHash,refreshExpires,userAgent,ip).Scan(&familyID);err!=nil {
		return Session{},fmt.Errorf("create login refresh token: %w",err)
	}
	if _,err:=tx.Exec(ctx,`INSERT INTO user_sessions(user_id,family_id,token_hash,expires_at,user_agent,ip_address)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::inet)`,user.ID,familyID,tokenHash,expires,userAgent,ip);err!=nil {
		return Session{},fmt.Errorf("create login session: %w",err)
	}
	if _,err:=tx.Exec(ctx,`INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id,ip_address)
		VALUES($1,$2,'auth.login','user',$3,NULLIF($4,'')::inet)`,user.TenantID,user.ID,user.ID,ip);err!=nil {
		return Session{},fmt.Errorf("audit login: %w",err)
	}
	if err:=tx.Commit(ctx);err!=nil { return Session{},fmt.Errorf("commit login: %w",err) }
	return Session{Token:token,ExpiresAt:expires,RefreshToken:refreshToken,RefreshExpiresAt:refreshExpires,User:user},nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	if token == "" { return User{}, ErrUnauthorized }
	hash := sha256.Sum256([]byte(token))
	var user User
	err := s.pool.QueryRow(ctx, `SELECT u.id::text,u.tenant_id::text,u.email,u.display_name,u.role,u.created_at
		FROM user_sessions s JOIN users u ON u.id=s.user_id JOIN tenants t ON t.id=u.tenant_id
		WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now()
		AND u.status='active' AND t.status='active'`, hash[:]).Scan(&user.ID,&user.TenantID,&user.Email,&user.DisplayName,&user.Role,&user.CreatedAt)
	if err != nil { return User{}, ErrUnauthorized }
	return user,nil
}

func (s *Service) Logout(ctx context.Context, token string, remoteAddrs ...string) error {
	if token == "" { return nil }
	hash := sha256.Sum256([]byte(token))
	ip := ""
	if len(remoteAddrs)>0 { ip=clientIP(remoteAddrs[0]) }
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil { return fmt.Errorf("begin logout: %w", err) }
	defer tx.Rollback(ctx)
	var familyID *string
	var userID string
	err = tx.QueryRow(ctx, `UPDATE user_sessions SET revoked_at=now()
		WHERE token_hash=$1 AND revoked_at IS NULL RETURNING family_id::text,user_id::text`, hash[:]).Scan(&familyID,&userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) { return fmt.Errorf("revoke access session: %w", err) }
	if err == nil {
		if familyID != nil {
			if _, err := tx.Exec(ctx, "UPDATE user_refresh_tokens SET revoked_at=COALESCE(revoked_at,now()) WHERE family_id=$1", *familyID); err != nil {
				return fmt.Errorf("revoke refresh token family: %w", err)
			}
		}
		var tenantID string
		if err:=tx.QueryRow(ctx,"SELECT tenant_id::text FROM users WHERE id=$1",userID).Scan(&tenantID);err!=nil {
			return fmt.Errorf("load tenant for logout audit: %w",err)
		}
		if _,err:=tx.Exec(ctx,`INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id,ip_address)
			VALUES($1,$2,'auth.logout','session',$3,NULLIF($4,'')::inet)`,tenantID,userID,userID,ip);err!=nil {
			return fmt.Errorf("audit logout: %w",err)
		}
	}
	if err := tx.Commit(ctx); err != nil { return fmt.Errorf("commit logout: %w", err) }
	return nil
}

func validEmail(value string) bool {
	addr, err := mail.ParseAddress(value)
	return err == nil && addr.Address == value && strings.Contains(value, ".")
}

func newToken() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil { return "",nil,fmt.Errorf("generate session token: %w",err) }
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	return token,hash[:],nil
}

func clientIP(remote string) string {
	host,_,err := net.SplitHostPort(remote)
	if err == nil { return host }
	if net.ParseIP(remote) != nil { return remote }
	return ""
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err,&pgErr) && pgErr.SQLState()=="23505"
}
