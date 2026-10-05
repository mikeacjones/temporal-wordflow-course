package shared

import "wordflow/internal/game"

// Extra: Saga. PlayerWorkflow Updates used to move points between players.
const (
	UpdateWithdraw = "withdraw"
	UpdateDeposit  = "deposit"
)

type PointsInput struct {
	Amount int `json:"amount"`
}

type PointsActivityInput struct {
	PlayerID string `json:"playerId"`
	Amount   int    `json:"amount"`
	UpdateID string `json:"updateId"`
}

type TransferInput struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int    `json:"amount"`
}

// Extra: Pick First.
type LookupInput struct {
	Word   string `json:"word"`
	Region string `json:"region"`
}

// Extra: Approval.
const (
	SignalReview    = "review"
	QuerySubmission = "submission"
)

func SubmissionWorkflowID(puzzleID string) string {
	return "submission-" + puzzleID
}

type SubmissionInput struct {
	Puzzle game.Puzzle `json:"puzzle"`
}

type ReviewInput struct {
	Approved bool   `json:"approved"`
	Reviewer string `json:"reviewer"`
	Note     string `json:"note,omitempty"`
}

type SubmissionView struct {
	Status       string       `json:"status"`
	UnknownWords []string     `json:"unknownWords,omitempty"`
	Review       *ReviewInput `json:"review,omitempty"`
}

// Extra: Schedules.
const DailyScheduleID = "daily-puzzle"
