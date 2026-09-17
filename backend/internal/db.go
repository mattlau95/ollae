package internal

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func NewDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}
	log.Println("Database connected")
	return db, nil
}

func RunMigrations(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		INSERT INTO settings (key, value) VALUES ('retention_months', '2')
		ON CONFLICT (key) DO NOTHING;

		ALTER TABLE events ADD COLUMN IF NOT EXISTS is_demo BOOLEAN NOT NULL DEFAULT false;
		ALTER TABLE events ADD COLUMN IF NOT EXISTS append_only BOOLEAN NOT NULL DEFAULT false;
		ALTER TABLE events ADD COLUMN IF NOT EXISTS reminders_off BOOLEAN NOT NULL DEFAULT false;

		-- Blocked words and phrases, managed in /admin. Deliberately not
		-- seeded from any file: the terms stay out of this public repo.
		CREATE TABLE IF NOT EXISTS blocked_terms (
			term       TEXT PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		INSERT INTO settings (key, value) VALUES ('blocked_terms_version', '0')
		ON CONFLICT (key) DO NOTHING;

		CREATE TABLE IF NOT EXISTS claude_usage (
			day   DATE PRIMARY KEY,
			calls INT NOT NULL DEFAULT 0
		);
	`)
	if err != nil {
		log.Printf("migration warning: %v", err)
	}
}
