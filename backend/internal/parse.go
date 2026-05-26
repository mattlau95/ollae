package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type ParsedEvent struct {
	Title    string  `json:"title"`
	Date     *string `json:"date"`
	Time     *string `json:"time"`
	Location *string `json:"location"`
}

var anthropicClient = &http.Client{Timeout: 15 * time.Second}

func (h *EventHandlers) ParseEvent(w http.ResponseWriter, r *http.Request) {
	if h.AnthropicKey == "" {
		http.Error(w, "AI parsing not configured", http.StatusServiceUnavailable)
		return
	}

	var body struct {
		Input string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Input == "" {
		http.Error(w, "input is required", http.StatusBadRequest)
		return
	}

	today := time.Now().Format("2006-01-02")
	systemPrompt := fmt.Sprintf(`You are a parser that extracts event details from natural language. Today's date is %s.

Return ONLY a JSON object with these exact keys:
- "title": string (event name, required, never null)
- "date": string or null (ISO date YYYY-MM-DD, resolve relative dates like "Saturday" using today's date)
- "time": string or null (24-hour HH:MM format, e.g. "14:00" for 2pm)
- "location": string or null

No explanation. No markdown. JSON only.`, today)

	payload, _ := json.Marshal(map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 256,
		"system":     systemPrompt,
		"messages":   []map[string]string{{"role": "user", "content": body.Input}},
	})

	req, err := http.NewRequestWithContext(r.Context(), "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
	if err != nil {
		http.Error(w, "failed to build request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", h.AnthropicKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := anthropicClient.Do(req)
	if err != nil {
		http.Error(w, "failed to reach Claude API", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("Anthropic API error %d: %s", resp.StatusCode, raw)
		http.Error(w, "Claude API error", http.StatusBadGateway)
		return
	}

	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &apiResp); err != nil || len(apiResp.Content) == 0 {
		log.Printf("Unexpected Anthropic response: %s", raw)
		http.Error(w, "unexpected Claude API response", http.StatusInternalServerError)
		return
	}

	text := strings.TrimSpace(apiResp.Content[0].Text)
	// Strip markdown code fences if model wrapped the JSON
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var parsed ParsedEvent
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		log.Printf("Failed to parse Claude JSON: %s", text)
		http.Error(w, "failed to parse Claude response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(parsed)
}
