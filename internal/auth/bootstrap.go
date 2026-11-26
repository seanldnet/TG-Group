package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// EnsureDemoAdmin creates the first-run demo administrator only when it does
// not already exist. It never resets an existing administrator's password.
func (s *Service) EnsureDemoAdmin(ctx context.Context) error {
	const email = "admin@tg-group.local"
	const password = "123456"
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil { return fmt.Errorf("begin demo admin bootstrap: %w", err) }
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", email).Scan(&exists); err != nil {
		return fmt.Errorf("check demo admin: %w", err)
	}
	if exists {
		return tx.Commit(ctx)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil { return fmt.Errorf("hash demo admin password: %w", err) }
	var tenantID, userID string
	if err := tx.QueryRow(ctx, "INSERT INTO tenants(name) VALUES('TG-Group Demo Admin') RETURNING id::text").Scan(&tenantID); err != nil {
		return fmt.Errorf("create demo admin tenant: %w", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO users(tenant_id,email,password_hash,display_name,role,status)
		VALUES($1,$2,$3,'Administrator','admin','active') RETURNING id::text`, tenantID,email,string(hash)).Scan(&userID); err != nil {
		return fmt.Errorf("create demo admin: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id)
		VALUES($1,$2,'auth.bootstrap_admin','user',$3)`, tenantID,userID,userID); err != nil {
		return fmt.Errorf("audit demo admin bootstrap: %w", err)
	}
	if err := tx.Commit(ctx); err != nil { return fmt.Errorf("commit demo admin bootstrap: %w", err) }
	return nil
}
