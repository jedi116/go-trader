package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"
)

func main() {
	dir := flag.String("dir", "scripts/migrations", "migrations directory")
	dsn := flag.String("dsn", os.Getenv("DATABASE_URL"), "Postgres DSN")
	flag.Parse()

	if *dsn == "" {
		log.Fatal("DATABASE_URL or --dsn is required")
	}

	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (filename TEXT PRIMARY KEY, applied_at TIMESTAMPTZ DEFAULT NOW())`); err != nil {
		log.Fatal(err)
	}

	entries, err := os.ReadDir(*dir)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		var exists bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename=$1)`, e.Name()).Scan(&exists); err != nil {
			log.Fatal(err)
		}
		if exists {
			continue
		}
		path := filepath.Join(*dir, e.Name())
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			log.Fatalf("failed applying %s: %v", e.Name(), err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO schema_migrations(filename) VALUES ($1)`, e.Name()); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("applied %s\n", e.Name())
	}
}
