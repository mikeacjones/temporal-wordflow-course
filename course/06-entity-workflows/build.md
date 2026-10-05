# 06 · Entity Workflows (Go)

## 1. Names

In `app/shared/shared.go`, add:

```go
// Message names for PlayerWorkflow.
const (
	QueryPlayer = "player"
	UpdateJoin  = "join"
)

// PlayerWorkflowID is the one Workflow ID for a player.
func PlayerWorkflowID(name string) string {
	return "player-" + name
}

type PlayerInput struct {
	Name string `json:"name"`
}
```

## 2. The Player Workflow

Create `app/workflows/player.go`:

```go
package workflows

import (
	"go.temporal.io/sdk/workflow"

	"wordflow/app/shared"
	"wordflow/internal/game"
)

// PlayerWorkflow is one player for as long as they play.
func PlayerWorkflow(ctx workflow.Context, input shared.PlayerInput) error {
	player := game.NewPlayer(input.Name, workflow.Now(ctx))

	err := workflow.SetQueryHandler(ctx, shared.QueryPlayer, func() (game.PlayerView, error) {
		return player.View(), nil
	})
	if err != nil {
		return err
	}

	err = workflow.SetUpdateHandler(ctx, shared.UpdateJoin, func(ctx workflow.Context) (game.PlayerView, error) {
		return player.View(), nil
	})
	if err != nil {
		return err
	}

	// The Workflow stays open, holding the player's state, until something
	// ends it. Lesson 10 adds Continue-As-New to keep its history small.
	return workflow.Await(ctx, func() bool { return false })
}
```

`join` doesn't change anything. Its job is to give Update-with-Start something to send, so the caller gets the player's view back.

Register it in `app/cmd/worker/main.go`:

```go
	w.RegisterWorkflow(workflows.PlayerWorkflow)
```

## 3. The API

The HTTP layer has already checked and normalized the name. In `app/api/api.go`:

```go
func (a *API) JoinPlayer(ctx context.Context, name string) (game.PlayerView, error) {
	startOp := a.Temporal.NewWithStartWorkflowOperation(client.StartWorkflowOptions{
		ID:                       shared.PlayerWorkflowID(name),
		TaskQueue:                shared.TaskQueue,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, workflows.PlayerWorkflow, shared.PlayerInput{Name: name})

	handle, err := a.Temporal.UpdateWithStartWorkflow(ctx, client.UpdateWithStartWorkflowOptions{
		StartWorkflowOperation: startOp,
		UpdateOptions: client.UpdateWorkflowOptions{
			UpdateName:   shared.UpdateJoin,
			WaitForStage: client.WorkflowUpdateStageCompleted,
		},
	})
	if err != nil {
		return game.PlayerView{}, err
	}
	var view game.PlayerView
	err = handle.Get(ctx, &view)
	return view, err
}

func (a *API) GetPlayer(ctx context.Context, name string) (game.PlayerView, error) {
	response, err := a.Temporal.QueryWorkflow(ctx, shared.PlayerWorkflowID(name), "", shared.QueryPlayer)
	if err != nil {
		return game.PlayerView{}, err
	}
	var view game.PlayerView
	err = response.Get(&view)
	return view, err
}
```

Querying a Workflow ID that doesn't exist returns a `*serviceerror.NotFound`, which the HTTP layer maps to `404`.

## 4. Run it

Restart the Worker and the server. Compare with `solutions/06`.
