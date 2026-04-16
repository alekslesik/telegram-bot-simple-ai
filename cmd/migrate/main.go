package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alekslesik/telegram-bot-simple/internal/storage/postgres"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s [up|down]", os.Args[0])
	}
	cmd := strings.TrimSpace(os.Args[1])

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		log.Fatal("env var DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := postgres.Open(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	switch cmd {
	case "up":
		if err := migrateUp(ctx, db); err != nil {
			log.Fatalf("migrate up failed: %v", err)
		}
	case "down":
		if err := migrateDown(ctx, db); err != nil {
			log.Fatalf("migrate down failed: %v", err)
		}
	default:
		log.Fatalf("unknown command %q (expected up|down)", cmd)
	}
}

func migrateUp(ctx context.Context, db *sql.DB) error {
	// A minimal migration runner: executes SQL files once, in filename sort order.
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	migrationsDir := filepath.Join("internal", "storage", "postgres", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".sql") {
			files = append(files, name)
		}
	}
	sort.Strings(files)

	for _, filename := range files {
		applied, err := isApplied(ctx, db, filename)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", filename, err)
		}
		if applied {
			continue
		}

		fullPath := filepath.Join(migrationsDir, filename)
		sqlBytes, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filename, err)
		}

		// Important: the SQL migration files may include their own BEGIN/COMMIT.
		// We intentionally avoid wrapping Exec in an outer transaction to prevent
		// transaction boundary conflicts.
		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("exec migration %s: %w", filename, err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO schema_migrations(filename) VALUES ($1)`, filename); err != nil {
			return fmt.Errorf("record migration %s: %w", filename, err)
		}
	}

	return nil
}

func migrateDown(ctx context.Context, db *sql.DB) error {
	// Minimal down: drop known MVP tables and clear migration records.
	// This keeps `make migrate-down` usable without requiring reverse migrations scripts.
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	_, err := db.ExecContext(ctx, `
		DROP TABLE IF EXISTS raw_images CASCADE;
		DROP TABLE IF EXISTS raw_chunks CASCADE;
		DROP TABLE IF EXISTS ingest_jobs CASCADE;
		DROP TABLE IF EXISTS user_sessions CASCADE;
		DROP TABLE IF EXISTS user_attempts CASCADE;
		DROP TABLE IF EXISTS user_progress CASCADE;
		DROP TABLE IF EXISTS block_relations CASCADE;
		DROP TABLE IF EXISTS block_content CASCADE;
		DROP TABLE IF EXISTS learning_blocks CASCADE;
		DROP TABLE IF EXISTS chapters CASCADE;
		DROP TABLE IF EXISTS sections CASCADE;
		DROP TABLE IF EXISTS users CASCADE;
	`)
	if err != nil {
		return fmt.Errorf("drop tables: %w", err)
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM schema_migrations;`); err != nil {
		return fmt.Errorf("clear schema_migrations: %w", err)
	}
	return nil
}

func isApplied(ctx context.Context, db *sql.DB, filename string) (bool, error) {
	var applied bool
	err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename=$1)`, filename).Scan(&applied)
	return applied, err
}

