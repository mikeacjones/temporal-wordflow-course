package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"wordflow/internal/game"
	"wordflow/solutions/02/activities"
	"wordflow/solutions/02/shared"
)

// GameWorkflow loads a puzzle and returns the first view of a new game.
func GameWorkflow(ctx workflow.Context, input shared.GameInput) (game.View, error) {
	gameID := workflow.GetInfo(ctx).WorkflowExecution.ID

	puzzle, err := loadPuzzle(ctx, input.PuzzleID)
	if err != nil {
		return game.View{}, err
	}

	state := game.NewState(gameID, "", puzzle, workflow.Now(ctx))
	return state.View(), nil
}

func loadPuzzle(ctx workflow.Context, puzzleID string) (game.Puzzle, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})
	var a *activities.PuzzleActivities
	var puzzle game.Puzzle
	err := workflow.ExecuteActivity(ctx, a.LoadPuzzle, puzzleID).Get(ctx, &puzzle)
	return puzzle, err
}
