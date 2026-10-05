package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"wordflow/internal/game"
	"wordflow/solutions/03/activities"
	"wordflow/solutions/03/shared"
)

// GameWorkflow runs one game until every word is found.
func GameWorkflow(ctx workflow.Context, input shared.GameInput) (game.Result, error) {
	gameID := workflow.GetInfo(ctx).WorkflowExecution.ID
	var state *game.State

	err := workflow.SetQueryHandler(ctx, shared.QueryState, func() (game.View, error) {
		if state == nil {
			return game.LoadingView(gameID), nil
		}
		return state.View(), nil
	})
	if err != nil {
		return game.Result{}, err
	}

	puzzle, err := loadPuzzle(ctx, input.PuzzleID)
	if err != nil {
		return game.Result{}, err
	}
	state = game.NewState(gameID, "", puzzle, workflow.Now(ctx))

	guesses := workflow.GetSignalChannel(ctx, shared.SignalGuess)
	for !state.Over() {
		var guess shared.GuessInput
		guesses.Receive(ctx, &guess)
		state.Guess(guess.Word, workflow.Now(ctx))
	}
	return state.Result(), nil
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
