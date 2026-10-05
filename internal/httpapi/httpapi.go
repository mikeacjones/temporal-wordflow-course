// Package httpapi serves the web UI and routes HTTP requests to a Backend.
// It is provided by the course. You implement Backend in app/api.
package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/temporal"

	"wordflow/internal/game"
	"wordflow/internal/puzzles"
	"wordflow/internal/services"
)

// ErrNotImplemented makes an endpoint return 501 with the lesson that builds it.
var ErrNotImplemented = errors.New("not implemented yet")

// RequestTimeout bounds every call into the Backend.
const RequestTimeout = 15 * time.Second

type StartGameRequest struct {
	RequestID string `json:"-"`
	PuzzleID  string `json:"puzzleId"`
	Player    string `json:"player"`
}

type StartGameResponse struct {
	GameID string     `json:"gameId"`
	Game   *game.View `json:"game,omitempty"`
}

type GuessRequest struct {
	RequestID string `json:"-"`
	GameID    string `json:"-"`
	Word      string `json:"word"`
}

type GameRequest struct {
	RequestID string
	GameID    string
}

// Backend is every endpoint that talks to Temporal.
//
// RequestID comes from the Idempotency-Key header. The browser sends the same
// value when it retries a request.
type Backend interface {
	StartGame(ctx context.Context, req StartGameRequest) (StartGameResponse, error) // Lesson 1
	GetGame(ctx context.Context, gameID string) (game.View, error)                  // Lesson 3
	// SubmitGuess returns nil when the guess was accepted but has no result yet.
	SubmitGuess(ctx context.Context, req GuessRequest) (*game.GuessResult, error) // Lesson 3
	ExtendTime(ctx context.Context, req GameRequest) (game.View, error)           // Lesson 5
	JoinPlayer(ctx context.Context, name string) (game.PlayerView, error)         // Lesson 6
	GetPlayer(ctx context.Context, name string) (game.PlayerView, error)          // Lesson 6
	UseHint(ctx context.Context, req GameRequest) (game.HintResult, error)        // Lesson 8
	GetLeaderboard(ctx context.Context) (game.LeaderboardView, error)             // Lesson 9
}

// Serve starts the web server on addr.
func Serve(addr string, backend Backend) error {
	store := puzzles.NewStore()
	words, err := store.AllWords()
	if err != nil {
		return err
	}
	log.Printf("Wordflow is running at http://localhost%s", addr)
	return http.ListenAndServe(addr, Handler(backend, store, services.New(words)))
}

func Handler(b Backend, store *puzzles.Store, svc *services.Services) http.Handler {
	mux := http.NewServeMux()
	svc.Register(mux)

	mux.HandleFunc("GET /api/puzzles", func(w http.ResponseWriter, r *http.Request) {
		list, err := store.List()
		if err != nil {
			writeError(w, err, 0)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})

	handle(mux, "POST /api/games", 1, func(ctx context.Context, r *http.Request) (any, error) {
		var req StartGameRequest
		if err := decode(r, &req); err != nil {
			return nil, err
		}
		req.RequestID = requestID(r)
		if !store.Exists(req.PuzzleID) {
			return nil, &notFoundError{"no puzzle named " + req.PuzzleID}
		}
		if req.Player != "" {
			req.Player = game.NormalizeName(req.Player)
			if err := game.ValidateName(req.Player); err != nil {
				return nil, err
			}
		}
		return b.StartGame(ctx, req)
	})
	handle(mux, "GET /api/games/{id}", 3, func(ctx context.Context, r *http.Request) (any, error) {
		return b.GetGame(ctx, r.PathValue("id"))
	})
	handle(mux, "POST /api/games/{id}/guesses", 3, func(ctx context.Context, r *http.Request) (any, error) {
		var req GuessRequest
		if err := decode(r, &req); err != nil {
			return nil, err
		}
		req.RequestID, req.GameID = requestID(r), r.PathValue("id")
		result, err := b.SubmitGuess(ctx, req)
		if err == nil && result == nil {
			return accepted{}, nil
		}
		return result, err
	})
	handle(mux, "POST /api/games/{id}/extend", 5, func(ctx context.Context, r *http.Request) (any, error) {
		return b.ExtendTime(ctx, GameRequest{RequestID: requestID(r), GameID: r.PathValue("id")})
	})
	handle(mux, "POST /api/games/{id}/hints", 8, func(ctx context.Context, r *http.Request) (any, error) {
		return b.UseHint(ctx, GameRequest{RequestID: requestID(r), GameID: r.PathValue("id")})
	})
	handle(mux, "POST /api/players", 6, func(ctx context.Context, r *http.Request) (any, error) {
		var body struct {
			Name string `json:"name"`
		}
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		name := game.NormalizeName(body.Name)
		if err := game.ValidateName(name); err != nil {
			return nil, err
		}
		return b.JoinPlayer(ctx, name)
	})
	handle(mux, "GET /api/players/{name}", 6, func(ctx context.Context, r *http.Request) (any, error) {
		return b.GetPlayer(ctx, game.NormalizeName(r.PathValue("name")))
	})
	handle(mux, "GET /api/leaderboard", 9, func(ctx context.Context, r *http.Request) (any, error) {
		return b.GetLeaderboard(ctx)
	})

	webDir := os.Getenv("WEB_DIR")
	if webDir == "" {
		webDir = "web"
	}
	mux.Handle("GET /", http.FileServer(http.Dir(webDir)))
	return mux
}

type accepted struct{}

func handle(mux *http.ServeMux, pattern string, lesson int, fn func(context.Context, *http.Request) (any, error)) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
		defer cancel()
		result, err := fn(ctx, r)
		if err != nil {
			writeError(w, err, lesson)
			return
		}
		if _, ok := result.(accepted); ok {
			writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
}

type errorBody struct {
	Error  string `json:"error"`
	Code   string `json:"code,omitempty"`
	Lesson int    `json:"lesson,omitempty"`
}

func writeError(w http.ResponseWriter, err error, lesson int) {
	var (
		appErr        *temporal.ApplicationError
		notFound      *serviceerror.NotFound
		started       *serviceerror.WorkflowExecutionAlreadyStarted
		queryFailed   *serviceerror.QueryFailed
		unavailable   *serviceerror.Unavailable
		deadline      *serviceerror.DeadlineExceeded
		badRequest    *badRequestError
		missing       *notFoundError
		timeoutErr    *temporal.TimeoutError
		canceledError *temporal.CanceledError
	)
	switch {
	case errors.Is(err, ErrNotImplemented):
		writeJSON(w, http.StatusNotImplemented, errorBody{Error: "not implemented yet", Code: "not_implemented", Lesson: lesson})
	case errors.As(err, &badRequest):
		writeJSON(w, http.StatusBadRequest, errorBody{Error: badRequest.Error(), Code: "bad_request"})
	case errors.As(err, &missing):
		writeJSON(w, http.StatusNotFound, errorBody{Error: missing.Error(), Code: "not_found"})
	case errors.As(err, &appErr):
		writeJSON(w, http.StatusBadRequest, errorBody{Error: appErr.Message(), Code: appErr.Type()})
	case errors.As(err, &notFound):
		writeJSON(w, http.StatusNotFound, errorBody{Error: notFound.Message, Code: "not_found"})
	case errors.As(err, &started):
		writeJSON(w, http.StatusConflict, errorBody{Error: "workflow already started", Code: "already_started"})
	case errors.As(err, &queryFailed):
		writeJSON(w, http.StatusBadRequest, errorBody{Error: queryFailed.Message, Code: "query_failed"})
	case errors.As(err, &unavailable):
		writeJSON(w, http.StatusServiceUnavailable, errorBody{Error: "cannot reach Temporal; is `make temporal` running?", Code: "temporal_unavailable"})
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &deadline), errors.As(err, &timeoutErr), errors.As(err, &canceledError):
		writeJSON(w, http.StatusGatewayTimeout, errorBody{Error: "timed out waiting for the Workflow; is `make worker` running?", Code: "timeout"})
	default:
		log.Printf("error: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: err.Error(), Code: "internal"})
	}
}

type badRequestError struct{ msg string }

func (e *badRequestError) Error() string { return e.msg }

type notFoundError struct{ msg string }

func (e *notFoundError) Error() string { return e.msg }

func decode(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return &badRequestError{"invalid JSON body"}
	}
	return nil
}

func requestID(r *http.Request) string {
	if id := r.Header.Get("Idempotency-Key"); id != "" {
		return id
	}
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
