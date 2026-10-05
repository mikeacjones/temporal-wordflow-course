// Package services fakes two external services the Worker calls over HTTP:
// a dictionary with switchable failure modes, and an activity feed.
// It is provided by the course and served by the web server under /services/.
package services

import (
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

var Modes = []string{"ok", "flaky", "slow", "rate_limited", "down", "unauthorized"}

type Services struct {
	words map[string]bool

	mu       sync.Mutex
	mode     string
	requests int
	feed     []FeedEvent
}

type FeedEvent struct {
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

func New(words map[string]bool) *Services {
	return &Services{words: words, mode: "ok", feed: []FeedEvent{}}
}

func (s *Services) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /services/dictionary/words/{word}", s.lookup)
	mux.HandleFunc("GET /services/dictionary/mode", s.getMode)
	mux.HandleFunc("PUT /services/dictionary/mode", s.setMode)
	mux.HandleFunc("GET /services/feed", s.getFeed)
	mux.HandleFunc("POST /services/feed", s.postFeed)
}

func (s *Services) lookup(w http.ResponseWriter, r *http.Request) {
	word := strings.ToUpper(r.PathValue("word"))
	label := word
	// ?region=NAME adds 0-1.5s of random latency, like a distant replica.
	if region := r.URL.Query().Get("region"); region != "" {
		label = word + " via " + region
		delay := time.Duration(rand.IntN(1500)) * time.Millisecond
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			log.Printf("dictionary %s -> caller gave up after %v", label, delay.Round(time.Millisecond))
			return
		}
		label += " in " + delay.Round(time.Millisecond).String()
	}
	s.mu.Lock()
	s.requests++
	mode, n := s.mode, s.requests
	s.mu.Unlock()

	status := http.StatusOK
	switch mode {
	case "flaky":
		if n%3 != 0 {
			status = http.StatusInternalServerError
		}
	case "slow":
		time.Sleep(3 * time.Second)
	case "rate_limited":
		if n%2 == 1 {
			status = http.StatusTooManyRequests
			w.Header().Set("Retry-After", "3")
		}
	case "down":
		status = http.StatusServiceUnavailable
	case "unauthorized":
		status = http.StatusUnauthorized
	}
	if status == http.StatusOK && !s.words[word] {
		status = http.StatusNotFound
	}
	log.Printf("dictionary [%s] %s -> %d", mode, label, status)
	writeJSON(w, status, map[string]any{"word": word, "valid": status == http.StatusOK})
}

func (s *Services) getMode(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"mode": s.mode, "modes": Modes})
}

func (s *Services) setMode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !slices.Contains(Modes, body.Mode) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown mode"})
		return
	}
	s.mu.Lock()
	s.mode, s.requests = body.Mode, 0
	s.mu.Unlock()
	log.Printf("dictionary mode set to %s", body.Mode)
	writeJSON(w, http.StatusOK, map[string]any{"mode": body.Mode, "modes": Modes})
}

func (s *Services) getFeed(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"events": s.feed})
}

func (s *Services) postFeed(w http.ResponseWriter, r *http.Request) {
	var event FeedEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil || event.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "message is required"})
		return
	}
	event.At = time.Now()
	s.mu.Lock()
	s.feed = append([]FeedEvent{event}, s.feed...)
	if len(s.feed) > 20 {
		s.feed = s.feed[:20]
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, event)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
