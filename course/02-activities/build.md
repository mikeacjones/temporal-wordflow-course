# 02 · Activities (Go)

## 1. The Activity

Create `app/activities/puzzles.go`:

```go
// Package activities holds code that talks to the outside world.
package activities

import (
	"context"
	"errors"

	"go.temporal.io/sdk/temporal"

	"wordflow/internal/game"
	"wordflow/internal/puzzles"
)

// PuzzleActivities gets its dependencies when the Worker starts.
type PuzzleActivities struct {
	Store *puzzles.Store
}

func (a *PuzzleActivities) LoadPuzzle(ctx context.Context, puzzleID string) (game.Puzzle, error) {
	puzzle, err := a.Store.Get(puzzleID)
	if errors.Is(err, puzzles.ErrNotFound) {
		return game.Puzzle{}, temporal.NewNonRetryableApplicationError("no puzzle named "+puzzleID, "puzzle_not_found", err)
	}
	return puzzle, err
}
```

- Activities are plain Go. They take a normal `context.Context`.
- Methods on a struct are a simple way to give Activities their dependencies. The Activity type name is the method name, `LoadPuzzle`.
- Any other error is retryable. An application error has a message and a type, and both show up in the Temporal UI.

## 2. Pass an ID, not a puzzle

In `app/shared/shared.go`, change the input. You can also remove the `internal/game` import, since nothing else uses it yet:

```go
type GameInput struct {
	PuzzleID string `json:"puzzleId"`
}
```

## 3. Call the Activity from the Workflow

Replace `app/workflows/game.go`:

```go
package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"wordflow/app/activities"
	"wordflow/app/shared"
	"wordflow/internal/game"
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
```

- `WithActivityOptions` returns a context that carries the timeout and retry policy.
- `var a *activities.PuzzleActivities` is a nil pointer. It is only used so that `a.LoadPuzzle` gives the Activity's name and type checking. The real struct lives in the Worker.
- `ExecuteActivity` returns a Future. `Get` waits for the result.
- Returning the error fails the Workflow. Because the Activity error is non-retryable, that happens on the first attempt.

## 4. Register the Activity

In `app/cmd/worker/main.go`, after `RegisterWorkflow`:

```go
	w.RegisterActivity(&activities.PuzzleActivities{Store: puzzles.NewStore()})
```

Add imports for `"wordflow/app/activities"` and `"wordflow/internal/puzzles"`. Registering a struct registers each of its exported methods as an Activity.

## 5. Update the API

In `StartGame`, delete the `a.Puzzles.Get` call and pass the ID:

```go
	run, err := a.Temporal.ExecuteWorkflow(ctx, options, workflows.GameWorkflow, shared.GameInput{PuzzleID: req.PuzzleID})
```

## 6. Run it

Restart the Worker and the server. Compare with `solutions/02`.
