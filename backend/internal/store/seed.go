package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// SeedOwner creates the first platform owner only when no owner exists.
// It never updates an existing account or password on restart or upgrade.
func SeedOwner(ctx context.Context, pool *pgxpool.Pool, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil { return fmt.Errorf("hash initial administrator password: %w", err) }
	_, err = pool.Exec(ctx, `
		INSERT INTO platform_users (username, password_hash, role)
		SELECT $1, $2, 'owner'
		WHERE NOT EXISTS (SELECT 1 FROM platform_users WHERE role = 'owner')
		ON CONFLICT (username) DO NOTHING
	`, username, string(hash))
	if err != nil { return fmt.Errorf("seed initial platform owner: %w", err) }
	return nil
}
