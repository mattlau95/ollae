package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/mattlau95/showup-backend/internal"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:Q94BjBR!s*lsCn@localhost:5432/showup?sslmode=disable"
	}
	db := internal.NewDB(dbURL)

	h := &internal.EventHandlers{DB: db}

	allowedOrigins := []string{"http://localhost:5173", "http://localhost:5174"}
	if frontendURL := os.Getenv("FRONTEND_URL"); frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Post("/events", h.CreateEvent)
	r.Get("/events/{slug}", h.GetEvent)
	r.Patch("/events/{slug}", h.UpdateEvent)

	r.Post("/events/{slug}/rsvp", h.SubmitRSVP)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
