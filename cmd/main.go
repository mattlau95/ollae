package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mattlau95/showup-backend/internal"
)

func main() {
	db := internal.NewDB("postgres://postgres:Q94BjBR!s*lsCn@localhost:5432/showup?sslmode=disable")

	h := &internal.EventHandlers{DB: db}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Post("/events", h.CreateEvent)
	r.Get("/events/{slug}", h.GetEvent)

	r.Post("/events/{slug}/rsvp", h.SubmitRSVP)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
