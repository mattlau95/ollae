package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/mattlau95/ollae-backend/internal"
)

// headToGet converts HEAD requests to GET so chi routes them to the registered
// GET handler, then strips the response body. This satisfies crawlers (e.g.
// Facebook's scraper linter) that send HEAD to validate the page and image URLs.
func headToGet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			r.Method = http.MethodGet
			next.ServeHTTP(&noBodyWriter{ResponseWriter: w}, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// noBodyWriter wraps ResponseWriter and silently discards all body writes,
// used so HEAD responses include correct headers but no body bytes.
type noBodyWriter struct {
	http.ResponseWriter
}

func (noBodyWriter) Write([]byte) (int, error) { return 0, nil }

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:Q94BjBR!s*lsCn@localhost:5432/showup?sslmode=disable"
	}
	db := internal.NewDB(dbURL)

	resendKey := os.Getenv("RESEND_API_KEY")
	cronToken := os.Getenv("CRON_TOKEN")
	h := &internal.EventHandlers{DB: db, AnthropicKey: os.Getenv("ANTHROPIC_API_KEY")}

	// Best-effort background loop (only fires while the machine is alive).
	internal.StartReminderLoop(db, resendKey)

	allowedOrigins := []string{"http://localhost:5173", "http://localhost:5174"}
	if frontendURL := os.Getenv("FRONTEND_URL"); frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// Convert HEAD requests to GET so chi finds the registered handler,
	// then discard the response body — required for Facebook's scraper linter
	// which sends HEAD to validate the page URL and og:image URL.
	r.Use(headToGet)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Post("/parse-event", h.ParseEvent)
	r.Post("/events", h.CreateEvent)
	r.Get("/events/{slug}", h.GetEvent)
	r.Patch("/events/{slug}", h.UpdateEvent)

	r.Post("/events/{slug}/rsvp", h.SubmitRSVP)

	r.Get("/og/{slug}", h.OGImage)
	r.Get("/og-preview/{slug}", h.OGPreview)

	// Cron endpoint — called by an external cron service every 30 minutes.
	// The request itself wakes the Fly.io machine (via auto-start), then
	// SendReminders runs and the machine can go back to sleep.
	r.Get("/cron/remind", func(w http.ResponseWriter, r *http.Request) {
		if cronToken != "" && r.URL.Query().Get("token") != cronToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		internal.SendReminders(db, resendKey)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
