package internal

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// ClientIP keys rate limits on Fly-Client-IP, which Fly's proxy sets from
// the TCP connection and overwrites if a client sends its own. The browser
// calls ollae-backend.fly.dev directly, so this is the visitor's address.
// X-Forwarded-For is ignored because a client can set it. IPv6 addresses
// are grouped by /64, the block one connection usually controls.
func ClientIP(r *http.Request) (string, error) {
	raw := r.Header.Get("Fly-Client-IP")
	if raw == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		raw = host
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return raw, nil
	}
	if ip.To4() == nil {
		return ip.Mask(net.CIDRMask(64, 128)).String() + "/64", nil
	}
	return ip.String(), nil
}

// PerIP limits each client to n requests per window. msg is shown to the
// visitor when they hit it.
func PerIP(n int, window time.Duration, msg string) func(http.Handler) http.Handler {
	return httprate.Limit(n, window,
		httprate.WithKeyFuncs(ClientIP),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			JSONErrorCode(w, http.StatusTooManyRequests, "rate_limited", msg)
		}))
}

// DefaultClaudeDailyCap applies when CLAUDE_DAILY_CAP isn't set.
const DefaultClaudeDailyCap = 300

var errClaudeCapReached = errors.New("daily Claude call cap reached")

// takeClaudeCall counts one Claude call against today's cap (UTC) in
// Postgres, so the count holds across restarts and machines. It returns
// errClaudeCapReached once cap calls have been made today.
func takeClaudeCall(ctx context.Context, db *sql.DB, cap int) error {
	if cap <= 0 {
		return errClaudeCapReached
	}
	var calls int
	err := db.QueryRowContext(ctx, `
		INSERT INTO claude_usage (day, calls) VALUES ((now() AT TIME ZONE 'UTC')::date, 1)
		ON CONFLICT (day) DO UPDATE SET calls = claude_usage.calls + 1
		WHERE claude_usage.calls < $1
		RETURNING calls
	`, cap).Scan(&calls)
	if err == sql.ErrNoRows {
		return errClaudeCapReached
	}
	if err != nil {
		log.Printf("claude cap: %v", err)
		return err
	}
	return nil
}
