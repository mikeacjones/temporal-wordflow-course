# 04 · Updates (Go)

## 1. Names

In `app/shared/shared.go`, replace the message names:

```go
// Message names for GameWorkflow.
const (
	QueryState  = "state"
	UpdateReady = "ready"
	UpdateGuess = "guess"
)
```

## 2. Update handlers

In `app/workflows/game.go`, register two Update handlers after the Query handler and before `loadPuzzle`:

```go
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
```

- `workflow.Await` blocks until the condition is true. The SDK re-checks it whenever the Workflow's state may have changed.
- The handler and validator take a `workflow.Context` and then the Update's arguments.
- `game.Error` creates the application errors that `CheckGuess` returns, and the API turns them into `400` responses.

Add the helper at the bottom of the file:

```go
func checkLoaded(state *game.State) error {
	if state == nil {
		return game.Error("loading", "the game is still loading")
	}
	return nil
}
```

## 3. The main loop

Replace the Signal loop with:

```go
	if err := workflow.Await(ctx, state.Over); err != nil {
		return game.Result{}, err
	}
	// Let any Update that is still running reply before the Workflow completes.
	if err := workflow.Await(ctx, func() bool { return workflow.AllHandlersFinished(ctx) }); err != nil {
		return game.Result{}, err
	}
	return state.Result(), nil
```

The handlers now change `state`. The main loop only waits.

## 4. Start with an Update

In `app/api/api.go`, replace `StartGame`:

```go
func (a *API) StartGame(ctx context.Context, req httpapi.StartGameRequest) (httpapi.StartGameResponse, error) {
	startOp := a.Temporal.NewWithStartWorkflowOperation(client.StartWorkflowOptions{
		ID:                       shared.GuestGameID(req.RequestID),
		TaskQueue:                shared.TaskQueue,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, workflows.GameWorkflow, shared.GameInput{PuzzleID: req.PuzzleID})

	handle, err := a.Temporal.UpdateWithStartWorkflow(ctx, client.UpdateWithStartWorkflowOptions{
		StartWorkflowOperation: startOp,
		UpdateOptions: client.UpdateWorkflowOptions{
			UpdateName:   shared.UpdateReady,
			WaitForStage: client.WorkflowUpdateStageCompleted,
		},
	})
	if err != nil {
		return httpapi.StartGameResponse{}, err
	}

	var view game.View
	if err := handle.Get(ctx, &view); err != nil {
		return httpapi.StartGameResponse{}, err
	}
	return httpapi.StartGameResponse{GameID: handle.WorkflowID(), Game: &view}, nil
}
```

Import `enumspb "go.temporal.io/api/enums/v1"`.

- Update-with-Start requires a conflict policy. *Use existing* means "if it's already running, send the Update to it".
- `WaitForStage: Completed` waits for the handler's result. `Accepted` would return once the validator passes.

## 5. Send guesses as Updates

Replace `SubmitGuess`:

```go
func (a *API) SubmitGuess(ctx context.Context, req httpapi.GuessRequest) (*game.GuessResult, error) {
	handle, err := a.Temporal.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   req.GameID,
		UpdateID:     req.RequestID,
		UpdateName:   shared.UpdateGuess,
		Args:         []any{shared.GuessInput{Word: req.Word}},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return nil, err
	}
	var result game.GuessResult
	if err := handle.Get(ctx, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
```

A validator rejection comes back as an error from `handle.Get` or `UpdateWorkflow`, already in a form the HTTP layer maps to `400`.

## 6. Run it

```sh
make clean-workflows
```

Restart the Worker and the server. Compare with `solutions/04`.
