package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrCurrentSessionRevoke = errors.New("cannot revoke current session")
)

type UserSession struct {
	ID string `json:"id"`
	UserAgent string `json:"user_agent"`
	IPAddress string `json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	Current bool `json:"current"`
}

func (s *Service) ListSessions(ctx context.Context, accessToken string, limit int) ([]UserSession,error) {
	if accessToken=="" { return nil,ErrUnauthorized }
	if limit<1 || limit>100 { limit=20 }
	hash:=sha256.Sum256([]byte(accessToken))
	var userID,currentSessionID string
	err:=s.pool.QueryRow(ctx,`SELECT s.user_id::text,s.id::text
		FROM user_sessions s JOIN users u ON u.id=s.user_id JOIN tenants t ON t.id=u.tenant_id
		WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now()
		AND u.status='active' AND t.status='active'`,hash[:]).Scan(&userID,&currentSessionID)
	if errors.Is(err,pgx.ErrNoRows) { return nil,ErrUnauthorized }
	if err!=nil { return nil,fmt.Errorf("identify current session: %w",err) }
	rows,err:=s.pool.Query(ctx,`SELECT id::text,user_agent,COALESCE(ip_address::text,''),created_at,expires_at,revoked_at
		FROM user_sessions WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`,userID,limit)
	if err!=nil { return nil,fmt.Errorf("list user sessions: %w",err) }
	defer rows.Close()
	sessions:=make([]UserSession,0)
	for rows.Next() {
		var item UserSession
		if err:=rows.Scan(&item.ID,&item.UserAgent,&item.IPAddress,&item.CreatedAt,&item.ExpiresAt,&item.RevokedAt);err!=nil {
			return nil,fmt.Errorf("scan user session: %w",err)
		}
		item.Current=item.ID==currentSessionID
		sessions=append(sessions,item)
	}
	if err:=rows.Err();err!=nil { return nil,fmt.Errorf("iterate user sessions: %w",err) }
	return sessions,nil
}

func (s *Service) RevokeSession(ctx context.Context, accessToken, sessionID string, remoteAddrs ...string) error {
	if accessToken=="" { return ErrUnauthorized }
	hash:=sha256.Sum256([]byte(accessToken))
	ip:=""
	if len(remoteAddrs)>0 { ip=clientIP(remoteAddrs[0]) }
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{})
	if err!=nil { return fmt.Errorf("begin session revocation: %w",err) }
	defer tx.Rollback(ctx)
	var userID,currentSessionID string
	err=tx.QueryRow(ctx,`SELECT s.user_id::text,s.id::text
		FROM user_sessions s JOIN users u ON u.id=s.user_id JOIN tenants t ON t.id=u.tenant_id
		WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now()
		AND u.status='active' AND t.status='active' FOR UPDATE OF s`,hash[:]).Scan(&userID,&currentSessionID)
	if errors.Is(err,pgx.ErrNoRows) { return ErrUnauthorized }
	if err!=nil { return fmt.Errorf("identify current session for revocation: %w",err) }
	if currentSessionID==sessionID { return ErrCurrentSessionRevoke }
	var familyID *string
	err=tx.QueryRow(ctx,`UPDATE user_sessions SET revoked_at=now()
		WHERE id::text=$1 AND user_id=$2 AND revoked_at IS NULL AND expires_at>now()
		RETURNING family_id::text`,sessionID,userID).Scan(&familyID)
	if errors.Is(err,pgx.ErrNoRows) { return ErrSessionNotFound }
	if err!=nil { return fmt.Errorf("revoke selected session: %w",err) }
	if familyID!=nil {
		if _,err:=tx.Exec(ctx,"UPDATE user_refresh_tokens SET revoked_at=COALESCE(revoked_at,now()) WHERE family_id=$1",*familyID);err!=nil {
			return fmt.Errorf("revoke selected session refresh family: %w",err)
		}
	}
	var tenantID string
	if err:=tx.QueryRow(ctx,"SELECT tenant_id::text FROM users WHERE id=$1",userID).Scan(&tenantID);err!=nil {
		return fmt.Errorf("load tenant for session revocation audit: %w",err)
	}
	if _,err:=tx.Exec(ctx,`INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id,ip_address)
		VALUES($1,$2,'auth.session_revoke','session',$3,NULLIF($4,'')::inet)`,tenantID,userID,sessionID,ip);err!=nil {
		return fmt.Errorf("audit session revocation: %w",err)
	}
	if err:=tx.Commit(ctx);err!=nil { return fmt.Errorf("commit session revocation: %w",err) }
	return nil
}
