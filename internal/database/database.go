package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type DB struct { Pool *pgxpool.Pool }

func Open(ctx context.Context, databaseURL string) (*DB, error) {
	if databaseURL == "" { return nil, fmt.Errorf("DATABASE_URL is required") }
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil { return nil, fmt.Errorf("parse DATABASE_URL: %w", err) }
	cfg.MaxConns, cfg.MinConns = 20, 2
	cfg.MaxConnLifetime = time.Hour
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil { return nil, fmt.Errorf("create PostgreSQL pool: %w", err) }
	if err := pool.Ping(ctx); err != nil { pool.Close(); return nil, fmt.Errorf("connect PostgreSQL: %w", err) }
	return &DB{Pool: pool}, nil
}

func (db *DB) Close() { if db != nil && db.Pool != nil { db.Pool.Close() } }

func (db *DB) Migrate(ctx context.Context) error {
	conn, err := db.Pool.Acquire(ctx)
	if err != nil { return fmt.Errorf("acquire migration lock connection: %w", err) }
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(8675309)"); err != nil { return fmt.Errorf("lock migrations: %w", err) }
	defer func() { _, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock(8675309)") }()
	if _, err := db.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil { return fmt.Errorf("create schema_migrations: %w", err) }
	entries, err := fs.Glob(migrationFiles, "migrations/*.up.sql")
	if err != nil { return err }
	sort.Strings(entries)
	for _, name := range entries {
		version := name[len("migrations/"):]
		var exists bool
		if err := db.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", version).Scan(&exists); err != nil { return err }
		if exists { continue }
		sql, err := migrationFiles.ReadFile(name)
		if err != nil { return err }
		tx, err := db.Pool.Begin(ctx)
		if err != nil { return err }
		if _, err = tx.Exec(ctx, string(sql)); err == nil {
			_, err = tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES($1)", version)
		}
		if err != nil { _ = tx.Rollback(ctx); return fmt.Errorf("apply %s: %w", name, err) }
		if err = tx.Commit(ctx); err != nil { return fmt.Errorf("commit %s: %w", name, err) }
	}
	return nil
}
