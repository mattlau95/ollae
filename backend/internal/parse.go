package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type ParsedEvent struct {
	Title    string  `json:"title"`
	Date     *string `json:"date"`
	Time     *string `json:"time"`
	Location *string `json:"location"`
	Emoji    string  `json:"emoji"`
}

var (
	anthropicClient = &http.Client{Timeout: 15 * time.Second}
	anthropicURL    = "https://api.anthropic.com/v1/messages"
)

const (
	anthropicAttempts = 3
	parseDeadline     = 25 * time.Second
	maxRetryAfter     = 5 * time.Second
)

// callAnthropic posts to the Messages API, retrying transport errors, 429s
// and 5xxs with exponential backoff (Retry-After is honored when sent).
// Any other status is returned to the caller as-is.
func callAnthropic(ctx context.Context, apiKey string, payload []byte) (int, []byte, error) {
	var (
		lastStatus int
		lastBody   []byte
		lastErr    error
		retryAfter time.Duration
	)
	for attempt := 0; attempt < anthropicAttempts; attempt++ {
		if attempt > 0 {
			delay := time.Duration(500<<(attempt-1)) * time.Millisecond
			if retryAfter > delay {
				delay = retryAfter
			}
			select {
			case <-ctx.Done():
				return lastStatus, lastBody, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicURL, bytes.NewReader(payload))
		if err != nil {
			return 0, nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := anthropicClient.Do(req)
		if err != nil {
			lastStatus, lastBody, lastErr = 0, nil, err
			log.Printf("Anthropic API unreachable (attempt %d/%d): %v", attempt+1, anthropicAttempts, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastStatus, lastBody, lastErr = resp.StatusCode, body, nil
			retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
			log.Printf("Anthropic API %d (attempt %d/%d): %s", resp.StatusCode, attempt+1, anthropicAttempts, body)
			continue
		}
		return resp.StatusCode, body, nil
	}
	return lastStatus, lastBody, lastErr
}

func parseRetryAfter(header string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(header))
	if err != nil || secs <= 0 {
		return 0
	}
	d := time.Duration(secs) * time.Second
	if d > maxRetryAfter {
		return maxRetryAfter
	}
	return d
}

func (h *EventHandlers) ParseEvent(w http.ResponseWriter, r *http.Request) {
	if h.AnthropicKey == "" {
		JSONError(w, http.StatusServiceUnavailable, "AI parsing not configured")
		return
	}

	var body struct {
		Input string `json:"input"`
		Today string `json:"today"` // the visitor's local date, YYYY-MM-DD
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Input) == "" {
		JSONError(w, http.StatusBadRequest, "input is required")
		return
	}
	if utf8.RuneCountInString(body.Input) > maxParseInput {
		JSONErrorCode(w, http.StatusBadRequest, "invalid",
			fmt.Sprintf("Descriptions can be up to %d characters.", maxParseInput))
		return
	}

	if err := takeClaudeCall(r.Context(), h.DB, h.ClaudeDailyCap); err != nil {
		if err == errClaudeCapReached {
			JSONErrorCode(w, http.StatusServiceUnavailable, "daily_cap",
				"Our AI helper is done for today. Fill in the details yourself below.")
		} else {
			JSONError(w, http.StatusServiceUnavailable, "AI parsing is busy, try again in a moment")
		}
		return
	}

	today := localToday(body.Today, time.Now())
	systemPrompt := fmt.Sprintf(`You are a parser that extracts event details from natural language. Today's date is %s.

Return ONLY a JSON object with these exact keys:
- "title": string (event name, required, never null)
- "date": string or null (ISO date YYYY-MM-DD, resolve relative dates like "Saturday" using today's date)
- "time": string or null (24-hour HH:MM format, e.g. "14:00" for 2pm)
- "location": string or null
- "emoji": string (single emoji that best represents the event, e.g. "🏐" for volleyball, "🎉" for a party, "🍕" for a dinner — never null, never more than one emoji)

No explanation. No markdown. JSON only.`, today)

	payload, _ := json.Marshal(map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 256,
		"system":     systemPrompt,
		"messages":   []map[string]string{{"role": "user", "content": body.Input}},
	})

	ctx, cancel := context.WithTimeout(r.Context(), parseDeadline)
	defer cancel()

	status, raw, err := callAnthropic(ctx, h.AnthropicKey, payload)
	if err != nil {
		JSONError(w, http.StatusBadGateway, "failed to reach Claude API")
		return
	}
	if status == http.StatusTooManyRequests {
		JSONError(w, http.StatusServiceUnavailable, "AI parsing is busy, try again in a moment")
		return
	}
	if status != http.StatusOK {
		log.Printf("Anthropic API error %d: %s", status, raw)
		JSONError(w, http.StatusBadGateway, "Claude API error")
		return
	}

	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &apiResp); err != nil || len(apiResp.Content) == 0 {
		log.Printf("Unexpected Anthropic response: %s", raw)
		JSONError(w, http.StatusInternalServerError, "unexpected Claude API response")
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
		JSONError(w, http.StatusInternalServerError, "failed to parse Claude response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitizeParsed(parsed))
}

// localToday returns the visitor's date when it's a real date within a day
// of the server's UTC date (every timezone is), otherwise the server's.
func localToday(visitor string, now time.Time) string {
	server := now.UTC()
	if d, err := time.Parse("2006-01-02", visitor); err == nil {
		if diff := d.Sub(server.Truncate(24 * time.Hour)); diff >= -24*time.Hour && diff <= 24*time.Hour {
			return visitor
		}
	}
	return server.Format("2006-01-02")
}

// sanitizeParsed caps what Claude returned to what the form accepts. The
// visitor reviews every field before creating, and create validates again.
func sanitizeParsed(p ParsedEvent) ParsedEvent {
	p.Title = truncateRunes(p.Title, maxTitleLen)
	if p.Location != nil {
		loc := truncateRunes(*p.Location, maxLocationLen)
		p.Location = &loc
		if loc == "" {
			p.Location = nil
		}
	}
	if p.Date != nil && !validDate(*p.Date) {
		p.Date = nil
	}
	if p.Time != nil && !validClock(*p.Time) {
		p.Time = nil
	}
	if !validEmoji(p.Emoji) {
		p.Emoji = "🎉"
	}
	return p
}
