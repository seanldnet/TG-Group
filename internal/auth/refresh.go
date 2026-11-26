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
	ErrRefreshUnauthorized = errors.New("refresh token is invalid or expired")
	ErrRefreshReuse = errors.New("refresh token reuse detected")
)

const (
	accessSessionLifetime = 15 * time.Minute
	refreshTokenLifetime = 30 * 24 * time.Hour
)

// Refresh rotates both credentials. A previously rotated token invalidates its entire
// family, preventing a stolen refresh token from silently creating parallel sessions.
func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, remoteAddr string) (Session, error) {
	if refreshToken == "" {
		return Session{}, ErrRefreshUnauthorized
	}
	oldHash := sha256.Sum256([]byte(refreshToken))
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, fmt.Errorf("begin refresh transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var userID, familyID string
	var expiresAt time.Time
	var revokedAt *time.Time
	var replacedByHash []byte
	err = tx.QueryRow(ctx, `SELECT user_id::text,family_id::text,expires_at,revoked_at,replaced_by_hash
		FROM user_refresh_tokens WHERE token_hash=$1 FOR UPDATE`, oldHash[:]).Scan(&userID, &familyID, &expiresAt, &revokedAt, &replacedByHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrRefreshUnauthorized
	}
	if err != nil {
		return Session{}, fmt.Errorf("load refresh token: %w", err)
	}
	if revokedAt != nil && len(replacedByHash) > 0 {
		if _, err := tx.Exec(ctx, `UPDATE user_refresh_tokens SET revoked_at=COALESCE(revoked_at,now())
			WHERE family_id=$1`, familyID); err != nil {
			return Session{}, fmt.Errorf("revoke reused refresh family: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE user_sessions SET revoked_at=COALESCE(revoked_at,now())
			WHERE family_id=$1`, familyID); err != nil {
			return Session{}, fmt.Errorf("revoke sessions after refresh reuse: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return Session{}, fmt.Errorf("commit refresh reuse revocation: %w", err)
		}
		return Session{}, ErrRefreshReuse
	}
	if revokedAt != nil {
		return Session{}, ErrRefreshUnauthorized
	}
	if !expiresAt.After(time.Now().UTC()) {
		_, _ = tx.Exec(ctx, "UPDATE user_refresh_tokens SET revoked_at=now() WHERE token_hash=$1", oldHash[:])
		if err := tx.Commit(ctx); err != nil {
			return Session{}, fmt.Errorf("commit expired refresh token: %w", err)
		}
		return Session{}, ErrRefreshUnauthorized
	}

	var user User
	var userStatus, tenantStatus string
	err = tx.QueryRow(ctx, `SELECT u.id::text,u.tenant_id::text,u.email,u.display_name,u.role,u.created_at,u.status,t.status
		FROM users u JOIN tenants t ON t.id=u.tenant_id WHERE u.id=$1 FOR UPDATE OF u,t`,
		userID).Scan(&user.ID,&user.TenantID,&user.Email,&user.DisplayName,&user.Role,&user.CreatedAt,&userStatus,&tenantStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrRefreshUnauthorized
		}
		return Session{}, fmt.Errorf("load user for refresh: %w", err)
	}
	if userStatus != "active" || tenantStatus != "active" {
		_, _ = tx.Exec(ctx, "UPDATE user_refresh_tokens SET revoked_at=now() WHERE family_id=$1", familyID)
		_, _ = tx.Exec(ctx, "UPDATE user_sessions SET revoked_at=now() WHERE family_id=$1", familyID)
		if err := tx.Commit(ctx); err != nil {
			return Session{}, fmt.Errorf("commit inactive account refresh revocation: %w", err)
		}
		return Session{}, ErrRefreshUnauthorized
	}

	accessToken, accessHash, err := newToken()
	if err != nil {
		return Session{}, err
	}
	nextRefreshToken, nextRefreshHash, err := newToken()
	if err != nil {
		return Session{}, err
	}
	now := time.Now().UTC()
	accessExpires := now.Add(accessSessionLifetime)
	refreshExpires := now.Add(refreshTokenLifetime)
	if _, err := tx.Exec(ctx, `UPDATE user_refresh_tokens SET revoked_at=now(),replaced_by_hash=$2
		WHERE token_hash=$1 AND revoked_at IS NULL`, oldHash[:], nextRefreshHash); err != nil {
		return Session{}, fmt.Errorf("rotate previous refresh token: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_refresh_tokens(user_id,family_id,token_hash,expires_at,user_agent,ip_address)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::inet)`, userID,familyID,nextRefreshHash,refreshExpires,userAgent,clientIP(remoteAddr)); err != nil {
		return Session{}, fmt.Errorf("create rotated refresh token: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_sessions(user_id,family_id,token_hash,expires_at,user_agent,ip_address)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::inet)`, userID,familyID,accessHash,accessExpires,userAgent,clientIP(remoteAddr)); err != nil {
		return Session{}, fmt.Errorf("create refreshed access session: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id)
		VALUES($1,$2,'auth.refresh','session',$3)`, user.TenantID,user.ID,user.ID); err != nil {
		return Session{}, fmt.Errorf("audit session refresh: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit refresh rotation: %w", err)
	}
	return Session{Token:accessToken,ExpiresAt:accessExpires,RefreshToken:nextRefreshToken,RefreshExpiresAt:refreshExpires,User:user},nil
}
