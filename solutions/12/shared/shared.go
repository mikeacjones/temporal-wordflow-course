// Package shared holds names and types used by Workflows, Activities, and the API.
package shared

import "wordflow/internal/game"

// TaskQueue is where the API sends work and where the Worker picks it up.
const TaskQueue = "wordflow"

// Message names for GameWorkflow.
const (
	QueryState       = "state"
	UpdateReady      = "ready"
	UpdateGuess      = "guess"
	UpdateExtendTime = "extendTime"
	UpdateHint       = "hint"
)

// Message names for PlayerWorkflow.
const (
	QueryPlayer       = "player"
	UpdateJoin        = "join"
	UpdateStartGame   = "startGame"
	UpdateSpendPoints = "spendPoints"
)

// The leaderboard is a single Workflow with a fixed ID.
const (
	LeaderboardWorkflowID = "leaderboard"
	// Activities cannot import the workflows package, so they start it by name.
	LeaderboardWorkflowType = "LeaderboardWorkflow"
	QueryLeaderboard        = "leaderboard"
	SignalScore             = "score"
)

type LeaderboardInput struct {
	State *game.Leaderboard `json:"state,omitempty"`
}

// PlayerWorkflowID is the one Workflow ID for a player.
func PlayerWorkflowID(name string) string {
	return "player-" + name
}

type PlayerInput struct {
	Name string `json:"name"`
	// State is set when the Workflow continues as new.
	State *game.Player `json:"state,omitempty"`
}

type StartGameInput struct {
	PuzzleID string `json:"puzzleId"`
}

type StartGameResult struct {
	GameID string `json:"gameId"`
}

type SpendPointsInput struct {
	GameID string `json:"gameId"`
	Amount int    `json:"amount"`
}

type SpendPointsResult struct {
	Remaining int `json:"remaining"`
}

type SpendPointsActivityInput struct {
	PlayerID string `json:"playerId"`
	GameID   string `json:"gameId"`
	Amount   int    `json:"amount"`
	UpdateID string `json:"updateId"`
}

type GameInput struct {
	PuzzleID string `json:"puzzleId"`
	PlayerID string `json:"playerId,omitempty"`
}

type GuessInput struct {
	Word string `json:"word"`
}

// GuestGameID derives a Workflow ID from the HTTP request ID, so a retried
// request maps to the same Workflow.
func GuestGameID(requestID string) string {
	return "game-" + requestID
}
