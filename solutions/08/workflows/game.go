package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"wordflow/internal/game"
	"wordflow/solutions/08/activities"
	"wordflow/solutions/08/shared"
)

// GameWorkflow runs one game until every word is found.
func GameWorkflow(ctx workflow.Context, input shared.GameInput) (game.Result, error) {
	gameID := workflow.GetInfo(ctx).WorkflowExecution.ID
	var state *game.State
	// Handlers that block (on an Activity) take this lock so they run one at a time.
	lock := workflow.NewMutex(ctx)

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
			if err := lock.Lock(ctx); err != nil {
				return game.GuessResult{}, err
			}
			defer lock.Unlock()

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

	err = workflow.SetUpdateHandlerWithOptions(ctx, shared.UpdateExtendTime,
		func(ctx workflow.Context) (game.View, error) {
			if err := state.Extend(workflow.Now(ctx)); err != nil {
				return game.View{}, err
			}
			return state.View(), nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context) error {
				if err := checkLoaded(state); err != nil {
					return err
				}
				return state.CheckExtend()
			},
		},
	)
	if err != nil {
		return game.Result{}, err
	}

	err = workflow.SetUpdateHandlerWithOptions(ctx, shared.UpdateHint,
		func(ctx workflow.Context) (game.HintResult, error) {
			if err := lock.Lock(ctx); err != nil {
				return game.HintResult{}, err
			}
			defer lock.Unlock()

			// The game may have changed while this handler waited for the lock.
			if err := state.CheckHint(); err != nil {
				return game.HintResult{}, err
			}
			cost := state.NextHintCost()
			if cost > 0 {
				if err := spendPoints(ctx, state, cost); err != nil {
					return game.HintResult{}, err
				}
			}
			if err := state.RevealLetter(workflow.Now(ctx)); err != nil {
				return game.HintResult{}, err
			}
			return game.HintResult{PointsSpent: cost, Game: state.View()}, nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context) error {
				if err := checkLoaded(state); err != nil {
					return err
				}
				return state.CheckHint()
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
	now := workflow.Now(ctx)
	state = game.NewState(gameID, input.PlayerID, puzzle, now)
	state.SetDeadline(now.Add(puzzle.TimeLimit()))

	if err := runClock(ctx, state); err != nil {
		return game.Result{}, err
	}
	// Let any Update that is still running reply before the Workflow completes.
	if err := workflow.Await(ctx, func() bool { return workflow.AllHandlersFinished(ctx) }); err != nil {
		return game.Result{}, err
	}
	return state.Result(), nil
}

// runClock waits until the game is over or its deadline passes. Each time the
// deadline moves, it cancels the timer and starts a new one.
func runClock(ctx workflow.Context, state *game.State) error {
	for !state.Over() {
		deadline := state.Deadline
		remaining := deadline.Sub(workflow.Now(ctx))
		if remaining <= 0 {
			state.Expire(workflow.Now(ctx))
			continue
		}

		timerCtx, cancelTimer := workflow.WithCancel(ctx)
		timer := workflow.NewTimerWithOptions(timerCtx, remaining, workflow.TimerOptions{Summary: "game deadline"})
		err := workflow.Await(ctx, func() bool {
			return timer.IsReady() || state.Over() || !state.Deadline.Equal(deadline)
		})
		cancelTimer()
		if err != nil {
			return err
		}
		if timer.IsReady() {
			state.Expire(workflow.Now(ctx))
		}
	}
	return nil
}

// spendPoints asks the player's Workflow to pay for a hint.
func spendPoints(ctx workflow.Context, state *game.State, amount int) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})
	input := shared.SpendPointsActivityInput{
		PlayerID: state.PlayerID,
		GameID:   state.GameID,
		Amount:   amount,
		// Same hint request, same Update ID: a retried Activity cannot charge twice.
		UpdateID: state.GameID + "-hint-" + workflow.GetCurrentUpdateInfo(ctx).ID,
	}
	var a *activities.PlayerActivities
	return workflow.ExecuteActivity(ctx, a.SpendPoints, input).Get(ctx, nil)
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
