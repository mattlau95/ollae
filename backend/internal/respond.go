package internal

import (
	"encoding/json"
	"net/http"
)

// JSONError writes {"error": msg} so every API failure has the same shape.
func JSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// JSONErrorCode adds a machine-readable code, for failures the frontend
// handles differently from a generic error (e.g. "name_taken").
func JSONErrorCode(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg, "code": code})
}
