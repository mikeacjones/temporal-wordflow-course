// Package shared holds names and types used by Workflows, Activities, and the API.
package shared

// TaskQueue is where the API sends work and where the Worker picks it up.
const TaskQueue = "wordflow"

// Message names for GameWorkflow.
const (
	QueryState       = "state"
	UpdateReady      = "ready"
	UpdateGuess      = "guess"
	UpdateExtendTime = "extendTime"
)

// Message names for PlayerWorkflow.
const (
	QueryPlayer     = "player"
	UpdateJoin      = "join"
	UpdateStartGame = "startGame"
)

// PlayerWorkflowID is the one Workflow ID for a player.
func PlayerWorkflowID(name string) string {
	return "player-" + name
}

type PlayerInput struct {
	Name string `json:"name"`
}

type StartGameInput struct {
	PuzzleID string `json:"puzzleId"`
}

type StartGameResult struct {
	GameID string `json:"gameId"`
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
