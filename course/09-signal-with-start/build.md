# 09 · Signal-with-Start (Go)

## 1. Names

In `app/shared/shared.go`:

```go
// The leaderboard is a single Workflow with a fixed ID.
const (
	LeaderboardWorkflowID = "leaderboard"
	// Activities cannot import the workflows package, so they start it by name.
	LeaderboardWorkflowType = "LeaderboardWorkflow"
	QueryLeaderboard        = "leaderboard"
	SignalScore             = "score"
)

type LeaderboardInput struct{}
```

`LeaderboardInput` is empty for now. Lesson 10 adds a field. Using a struct from the start lets you add fields later without changing the Workflow's signature.

## 2. The leaderboard

Create `app/workflows/leaderboard.go`:

```go
package workflows

import (
	"go.temporal.io/sdk/workflow"

	"wordflow/app/shared"
	"wordflow/internal/game"
)

// LeaderboardWorkflow is the single, global leaderboard.
func LeaderboardWorkflow(ctx workflow.Context, input shared.LeaderboardInput) error {
	board := &game.Leaderboard{}

	err := workflow.SetQueryHandler(ctx, shared.QueryLeaderboard, func() (game.LeaderboardView, error) {
		return board.View(), nil
	})
	if err != nil {
		return err
	}

	scores := workflow.GetSignalChannel(ctx, shared.SignalScore)
	for {
		var entry game.LeaderboardEntry
		scores.Receive(ctx, &entry)
		board.Upsert(entry)
	}
}
```

Register it in the Worker:

```go
	w.RegisterWorkflow(workflows.LeaderboardWorkflow)
```

## 3. Publish with Signal-with-Start

In `app/activities/players.go`, add (and import `"wordflow/internal/game"`):

```go
// PublishScore sends a player's score to the leaderboard, starting the
// leaderboard Workflow if it is not running yet.
func (a *PlayerActivities) PublishScore(ctx context.Context, entry game.LeaderboardEntry) error {
	options := client.StartWorkflowOptions{
		ID:        shared.LeaderboardWorkflowID,
		TaskQueue: shared.TaskQueue,
	}
	_, err := a.Client.SignalWithStartWorkflow(ctx, shared.LeaderboardWorkflowID, shared.SignalScore, entry,
		options, shared.LeaderboardWorkflowType, shared.LeaderboardInput{})
	return err
}
```

The arguments are: Workflow ID, Signal name, Signal argument, start options, Workflow type, and Workflow input. The input is only used if the Workflow has to be started.

## 4. Call it from the Player

In `app/workflows/player.go`, replace the `else` branch in the main loop:

```go
		} else if player.FinishGame(result) {
			publishScore(ctx, player)
		}
```

`FinishGame` returns false if it has already seen this result. Add the helper, and import `"time"` and `"wordflow/app/activities"`:

```go
func publishScore(ctx workflow.Context, player *game.Player) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})
	var a *activities.PlayerActivities
	err := workflow.ExecuteActivity(ctx, a.PublishScore, player.LeaderboardEntry()).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("publish score failed", "error", err)
	}
}
```

## 5. The API

In `app/api/api.go`, import `"errors"` and `"go.temporal.io/api/serviceerror"`:

```go
func (a *API) GetLeaderboard(ctx context.Context) (game.LeaderboardView, error) {
	response, err := a.Temporal.QueryWorkflow(ctx, shared.LeaderboardWorkflowID, "", shared.QueryLeaderboard)
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		// Nobody has finished a game yet, so the leaderboard has not started.
		return game.LeaderboardView{Entries: []game.LeaderboardRank{}}, nil
	}
	if err != nil {
		return game.LeaderboardView{}, err
	}
	var view game.LeaderboardView
	err = response.Get(&view)
	return view, err
}
```

## 6. Run it

Restart the Worker and the server **without** running `make clean-workflows`, and follow the checklist. Compare with `solutions/09`.
