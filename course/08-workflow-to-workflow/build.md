# 08 · Workflow to Workflow (Go)

## 1. Names and types

In `app/shared/shared.go`, add `UpdateHint = "hint"` to the Game names and `UpdateSpendPoints = "spendPoints"` to the Player names. Then add:

```go
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
```

## 2. The Player accepts payments

In `app/workflows/player.go`, after the `startGame` handler:

```go
	err = workflow.SetUpdateHandlerWithOptions(ctx, shared.UpdateSpendPoints,
		func(ctx workflow.Context, input shared.SpendPointsInput) (shared.SpendPointsResult, error) {
			player.Spend(input.Amount)
			return shared.SpendPointsResult{Remaining: player.Points}, nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context, input shared.SpendPointsInput) error {
				return player.CheckSpend(input.GameID, input.Amount)
			},
		},
	)
	if err != nil {
		return err
	}
```

## 3. An Activity with a Client

Create `app/activities/players.go`:

```go
package activities

import (
	"context"
	"errors"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"wordflow/app/shared"
)

// PlayerActivities send messages to Player Workflows using a Temporal Client.
type PlayerActivities struct {
	Client client.Client
}

// SpendPoints asks a Player Workflow to spend points.
func (a *PlayerActivities) SpendPoints(ctx context.Context, input shared.SpendPointsActivityInput) (shared.SpendPointsResult, error) {
	handle, err := a.Client.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   shared.PlayerWorkflowID(input.PlayerID),
		UpdateID:     input.UpdateID,
		UpdateName:   shared.UpdateSpendPoints,
		Args:         []any{shared.SpendPointsInput{GameID: input.GameID, Amount: input.Amount}},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	var result shared.SpendPointsResult
	if err == nil {
		err = handle.Get(ctx, &result)
	}

	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		// The Player said no. Asking again will get the same answer.
		return result, temporal.NewNonRetryableApplicationError(appErr.Message(), appErr.Type(), err)
	}
	return result, err
}
```

The Activity uses the same API as your HTTP code. Network errors stay retryable.

Register it in `app/cmd/worker/main.go`, passing the Worker's Client:

```go
	w.RegisterActivity(&activities.PlayerActivities{Client: c})
```

## 4. The hint Update

In `app/workflows/game.go`, create a mutex after `var state *game.State`:

```go
	// Handlers that block (on an Activity) take this lock so they run one at a time.
	lock := workflow.NewMutex(ctx)
```

At the top of the `guess` handler, take it:

```go
			if err := lock.Lock(ctx); err != nil {
				return game.GuessResult{}, err
			}
			defer lock.Unlock()
```

Register `hint` with the other handlers:

```go
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
```

Add the helper:

```go
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
```

- `workflow.Mutex` is safe in Workflow code. `sync.Mutex` is not.
- `GetCurrentUpdateInfo(ctx).ID` is the ID of the `hint` Update being handled: the API's request ID.
- The guest check is in `CheckHint`, so guests never reach `spendPoints`.

## 5. The API

```go
func (a *API) UseHint(ctx context.Context, req httpapi.GameRequest) (game.HintResult, error) {
	handle, err := a.Temporal.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   req.GameID,
		UpdateID:     req.RequestID,
		UpdateName:   shared.UpdateHint,
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return game.HintResult{}, err
	}
	var result game.HintResult
	err = handle.Get(ctx, &result)
	return result, err
}
```

## 6. Run it

```sh
make clean-workflows
```

Restart the Worker and the server. Compare with `solutions/08`.
