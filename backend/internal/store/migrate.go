package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// Serialize migration runners across replicas using a session-level lock.
	lockConn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration lock connection: %w", err)
	}
	defer lockConn.Release()
	const migrationLockID int64 = 721983401
	if _, err := lockConn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("acquire migration advisory lock: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = lockConn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", migrationLockID)
	}()
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil { return fmt.Errorf("create migration ledger: %w", err) }
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil { return fmt.Errorf("read embedded migrations: %w", err) }
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") { names = append(names, entry.Name()) }
	}
	sort.Strings(names)
	for _, name := range names {
		var exists bool
		if err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", name).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if exists { continue }
		sqlBytes, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil { return fmt.Errorf("read migration %s: %w", name, err) }
		tx, err := pool.Begin(ctx)
		if err != nil { return fmt.Errorf("begin migration %s: %w", name, err) }
		if _, err = tx.Exec(ctx, string(sqlBytes)); err == nil {
			_, err = tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES ($1)", name)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil { return fmt.Errorf("commit migration %s: %w", name, err) }
	}
	return nil
}
