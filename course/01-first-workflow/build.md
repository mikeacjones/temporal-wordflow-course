# 01 · Your first Workflow (Go)

## 1. Shared names

The API, the Worker, and the Workflows all need the Task Queue name and the Workflow's input type. Create `app/shared/shared.go`:

```go
// Package shared holds names and types used by Workflows, Activities, and the API.
package shared

import "wordflow/internal/game"

// TaskQueue is where the API sends work and where the Worker picks it up.
const TaskQueue = "wordflow"

type GameInput struct {
	Puzzle game.Puzzle `json:"puzzle"`
}

// GuestGameID derives a Workflow ID from the HTTP request ID, so a retried
// request maps to the same Workflow.
func GuestGameID(requestID string) string {
	return "game-" + requestID
}
```

Workflow inputs and results are serialized to JSON and stored in the Event History, so use plain structs with exported fields.

## 2. The Workflow

Create `app/workflows/game.go`:

```go
package workflows

import (
	"go.temporal.io/sdk/workflow"

	"wordflow/app/shared"
	"wordflow/internal/game"
)

// GameWorkflow builds a new game and returns its first view.
func GameWorkflow(ctx workflow.Context, input shared.GameInput) (game.View, error) {
	gameID := workflow.GetInfo(ctx).WorkflowExecution.ID
	state := game.NewState(gameID, "", input.Puzzle, workflow.Now(ctx))
	return state.View(), nil
}
```

- A Workflow is a function that takes a `workflow.Context` first and returns a result and an error.
- `workflow.GetInfo(ctx)` gives you the Workflow ID, Run ID, and more.
- `workflow.Now(ctx)` replaces `time.Now()`. It returns the same value when the code is re-run.

## 3. The Worker

Replace `app/cmd/worker/main.go`:

```go
// Command worker runs your Workflows and Activities.
package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"wordflow/app/shared"
	"wordflow/app/workflows"
)

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("unable to connect to Temporal:", err)
	}
	defer c.Close()

	w := worker.New(c, shared.TaskQueue, worker.Options{})
	w.RegisterWorkflow(workflows.GameWorkflow)

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalln("worker stopped:", err)
	}
}
```

`client.Options{}` connects to `localhost:7233` and the `default` namespace. The Workflow type name is the function name, `GameWorkflow`.

## 4. Give the API a Client

In `app/cmd/server/main.go`, connect and pass the Client to `api.New`:

```go
func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("unable to connect to Temporal:", err)
	}
	defer c.Close()

	backend := api.New(c, puzzles.NewStore())
	log.Fatal(httpapi.Serve(":8080", backend))
}
```

Add `"go.temporal.io/sdk/client"` to the imports. Create one Client per process and share it; it is safe for concurrent use.

## 5. Start the Workflow from the API

In `app/api/api.go`, replace `StartGame`:

```go
func (a *API) StartGame(ctx context.Context, req httpapi.StartGameRequest) (httpapi.StartGameResponse, error) {
	puzzle, err := a.Puzzles.Get(req.PuzzleID)
	if err != nil {
		return httpapi.StartGameResponse{}, err
	}

	options := client.StartWorkflowOptions{
		ID:        shared.GuestGameID(req.RequestID),
		TaskQueue: shared.TaskQueue,
	}
	run, err := a.Temporal.ExecuteWorkflow(ctx, options, workflows.GameWorkflow, shared.GameInput{Puzzle: puzzle})
	if err != nil {
		return httpapi.StartGameResponse{}, err
	}

	var view game.View
	if err := run.Get(ctx, &view); err != nil {
		return httpapi.StartGameResponse{}, err
	}
	return httpapi.StartGameResponse{GameID: run.GetID(), Game: &view}, nil
}
```

Add `"wordflow/app/shared"` and `"wordflow/app/workflows"` to the imports.

- `ExecuteWorkflow` returns as soon as the Temporal Service has recorded the start. It doesn't wait for the Workflow to run.
- `run.Get` blocks until the Workflow completes, then decodes its result.
- Passing the function `workflows.GameWorkflow` gives you type checking. You can also pass the type name as a string.
- `ctx` carries the HTTP request's 15-second timeout. When it expires, `Get` stops waiting, but the Workflow keeps going.

## 6. Run it

```sh
make worker   # terminal 3
make server   # restart terminal 2
```

Compare with `solutions/01`.
