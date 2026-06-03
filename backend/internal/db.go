package internal

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func NewDB(connStr string) *sql.DB {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	fmt.Println("Database connected")
	return db
}

func RunMigrations(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		INSERT INTO settings (key, value) VALUES ('retention_months', '2')
		ON CONFLICT (key) DO NOTHING;
	`)
	if err != nil {
		log.Printf("migration warning: %v", err)
	}
}
