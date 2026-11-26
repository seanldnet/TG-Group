package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrCurrentPasswordInvalid = errors.New("current password is invalid")

// ChangePassword verifies the current credential, changes the password and
// revokes every prior session. It then issues a fresh session for this device.
func (s *Service) ChangePassword(ctx context.Context, accessToken, currentPassword, newPassword, userAgent, remoteAddr string) (Session, error) {
	if accessToken == "" || len(currentPassword) < 1 || len(currentPassword) > 72 ||
		len(newPassword) < 10 || len(newPassword) > 72 || currentPassword == newPassword {
		return Session{}, ErrInvalidInput
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return Session{}, fmt.Errorf("hash new password: %w", err)
	}
	accessTokenHash := sha256.Sum256([]byte(accessToken))
	newAccessToken, newAccessHash, err := newToken()
	if err != nil {
		return Session{}, err
	}
	newRefreshToken, newRefreshHash, err := newToken()
	if err != nil {
		return Session{}, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, fmt.Errorf("begin password change: %w", err)
	}
	defer tx.Rollback(ctx)

	var user User
	var passwordHash, userStatus, tenantStatus string
	err = tx.QueryRow(ctx, `SELECT u.id::text,u.tenant_id::text,u.email,u.display_name,u.role,u.created_at,
		u.password_hash,u.status,t.status
		FROM user_sessions s
		JOIN users u ON u.id=s.user_id
		JOIN tenants t ON t.id=u.tenant_id
		WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now()
		FOR UPDATE OF s,u,t`, accessTokenHash[:]).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.DisplayName, &user.Role, &user.CreatedAt,
		&passwordHash, &userStatus, &tenantStatus,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrUnauthorized
	}
	if err != nil {
		return Session{}, fmt.Errorf("load account for password change: %w", err)
	}
	if userStatus != "active" || tenantStatus != "active" {
		return Session{}, ErrAccountInactive
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword)) != nil {
		return Session{}, ErrCurrentPasswordInvalid
	}

	if _, err := tx.Exec(ctx, "UPDATE users SET password_hash=$2,updated_at=now() WHERE id=$1", user.ID, string(newHash)); err != nil {
		return Session{}, fmt.Errorf("update account password: %w", err)
	}
	if _, err := tx.Exec(ctx, "UPDATE user_sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE user_id=$1", user.ID); err != nil {
		return Session{}, fmt.Errorf("revoke previous access sessions: %w", err)
	}
	if _, err := tx.Exec(ctx, "UPDATE user_refresh_tokens SET revoked_at=COALESCE(revoked_at,now()) WHERE user_id=$1", user.ID); err != nil {
		return Session{}, fmt.Errorf("revoke previous refresh tokens: %w", err)
	}

	now := time.Now().UTC()
	accessExpires := now.Add(accessSessionLifetime)
	refreshExpires := now.Add(refreshTokenLifetime)
	var familyID string
	if err := tx.QueryRow(ctx, `INSERT INTO user_refresh_tokens(user_id,token_hash,expires_at,user_agent,ip_address)
		VALUES($1,$2,$3,$4,NULLIF($5,'')::inet) RETURNING family_id::text`,
		user.ID, newRefreshHash, refreshExpires, userAgent, clientIP(remoteAddr)).Scan(&familyID); err != nil {
		return Session{}, fmt.Errorf("create post-change refresh token: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_sessions(user_id,family_id,token_hash,expires_at,user_agent,ip_address)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::inet)`,
		user.ID, familyID, newAccessHash, accessExpires, userAgent, clientIP(remoteAddr)); err != nil {
		return Session{}, fmt.Errorf("create post-change access session: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id,ip_address)
		VALUES($1,$2,'auth.password_change','user',$3,NULLIF($4,'')::inet)`,
		user.TenantID, user.ID, user.ID, clientIP(remoteAddr)); err != nil {
		return Session{}, fmt.Errorf("audit password change: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit password change: %w", err)
	}

	return Session{
		Token: newAccessToken, ExpiresAt: accessExpires,
		RefreshToken: newRefreshToken, RefreshExpiresAt: refreshExpires,
		User: user,
	}, nil
}
