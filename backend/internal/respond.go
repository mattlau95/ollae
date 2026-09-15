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
