// Package shared holds names and types used by Workflows, Activities, and the API.
package shared

// TaskQueue is where the API sends work and where the Worker picks it up.
const TaskQueue = "wordflow"

type GameInput struct {
	PuzzleID string `json:"puzzleId"`
}

// GuestGameID derives a Workflow ID from the HTTP request ID, so a retried
// request maps to the same Workflow.
func GuestGameID(requestID string) string {
	return "game-" + requestID
}
