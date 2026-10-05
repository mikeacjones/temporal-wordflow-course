package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"wordflow/internal/game"
	"wordflow/solutions/04/activities"
	"wordflow/solutions/04/shared"
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

	// ready returns the first view as soon as the puzzle has loaded.
	err = workflow.SetUpdateHandler(ctx, shared.UpdateReady, func(ctx workflow.Context) (game.View, error) {
		if err := workflow.Await(ctx, func() bool { return state != nil }); err != nil {
			return game.View{}, err
		}
		return state.View(), nil
	})
	if err != nil {
		return game.Result{}, err
	}

	err = workflow.SetUpdateHandlerWithOptions(ctx, shared.UpdateGuess,
		func(ctx workflow.Context, guess shared.GuessInput) (game.GuessResult, error) {
			outcome := state.Guess(guess.Word, workflow.Now(ctx))
			return game.GuessResult{Word: game.Normalize(guess.Word), Outcome: outcome, Game: state.View()}, nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context, guess shared.GuessInput) error {
				if err := checkLoaded(state); err != nil {
					return err
				}
				return state.CheckGuess(guess.Word)
			},
		},
	)
	if err != nil {
		return game.Result{}, err
	}

	puzzle, err := loadPuzzle(ctx, input.PuzzleID)
	if err != nil {
		return game.Result{}, err
	}
	state = game.NewState(gameID, "", puzzle, workflow.Now(ctx))

	if err := workflow.Await(ctx, state.Over); err != nil {
		return game.Result{}, err
	}
	// Let any Update that is still running reply before the Workflow completes.
	if err := workflow.Await(ctx, func() bool { return workflow.AllHandlersFinished(ctx) }); err != nil {
		return game.Result{}, err
	}
	return state.Result(), nil
}

func checkLoaded(state *game.State) error {
	if state == nil {
		return game.Error("loading", "the game is still loading")
	}
	return nil
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
