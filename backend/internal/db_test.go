package internal

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"
)

// testDB returns a connection to a fresh, empty schema loaded from schema.sql.
// Tests that need Postgres are skipped unless TEST_DATABASE_URL is set, e.g.
// postgres://user@localhost:5432/ollae_test?sslmode=disable
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	admin, err := sql.Open("postgres", base)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)
		admin.Close()
	})

	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	ddl, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(ddl)); err != nil {
		t.Fatalf("load schema.sql: %v", err)
	}
	return db
}

// insertEvent creates an event starting at eventDate and returns its id.
func insertEvent(t *testing.T, db *sql.DB, slug string, eventDate time.Time) string {
	t.Helper()
	var id string
	if err := db.QueryRow(`
		INSERT INTO events (slug, title, event_date) VALUES ($1, $1, $2) RETURNING id
	`, slug, eventDate).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func insertRemindMe(t *testing.T, db *sql.DB, eventID, name string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO responses (event_id, name, status, notify_via) VALUES ($1, $2, 'remind_me', $3)
	`, eventID, name, name+"@example.com"); err != nil {
		t.Fatal(err)
	}
}
