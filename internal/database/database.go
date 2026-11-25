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
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil { return nil, fmt.Errorf("create PostgreSQL pool: %w", err) }
	if err := pool.Ping(ctx); err != nil { pool.Close(); return nil, fmt.Errorf("connect PostgreSQL: %w", err) }
	return &DB{Pool: pool}, nil
}

func (db *DB) Close() { if db != nil && db.Pool != nil { db.Pool.Close() } }

func (db *DB) Migrate(ctx context.Context) error {
	entries, err := fs.Glob(migrationFiles, "migrations/*.up.sql")
	if err != nil { return err }
	sort.Strings(entries)
	for _, name := range entries {
		sql, err := migrationFiles.ReadFile(name)
		if err != nil { return err }
		tx, err := db.Pool.Begin(ctx)
		if err != nil { return err }
		if _, err = tx.Exec(ctx, string(sql)); err != nil { _ = tx.Rollback(ctx); return fmt.Errorf("apply %s: %w", name, err) }
		if err = tx.Commit(ctx); err != nil { return fmt.Errorf("commit %s: %w", name, err) }
	}
	return nil
}
