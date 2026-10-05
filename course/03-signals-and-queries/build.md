# 03 · Signals and Queries (Go)

## 1. Message names

In `app/shared/shared.go`, add:

```go
// Message names for GameWorkflow.
const (
	QueryState  = "state"
	SignalGuess = "guess"
)

type GuessInput struct {
	Word string `json:"word"`
}
```

Signals and Queries are identified by name, so the API and the Workflow must agree on the strings.

## 2. A Workflow that waits

Replace `GameWorkflow` in `app/workflows/game.go` (keep `loadPuzzle`):

```go
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
```

- The return type is now `game.Result`. The Workflow returns it when the game ends.
- The Query handler is a closure over `state`, so it always sees the latest value.
- `GetSignalChannel` returns a channel of incoming Signals. `Receive` blocks until one arrives. Signals that arrive while the Workflow is busy are buffered.
- Use `workflow.Channel` and `Receive`, never Go channels or `select`, in Workflow code. The SDK controls how Workflow code is scheduled so that it re-runs the same way every time.
- `state.Guess` ignores invalid words. Nobody is waiting for its answer.

## 3. The API

In `app/api/api.go`, `StartGame` no longer waits for a result. Replace the end of the method, after the `ExecuteWorkflow` error check:

```go
	return httpapi.StartGameResponse{GameID: run.GetID()}, nil
```

Implement `GetGame`:

```go
func (a *API) GetGame(ctx context.Context, gameID string) (game.View, error) {
	response, err := a.Temporal.QueryWorkflow(ctx, gameID, "", shared.QueryState)
	if err != nil {
		return game.View{}, err
	}
	var view game.View
	err = response.Get(&view)
	return view, err
}
```

Implement `SubmitGuess`:

```go
func (a *API) SubmitGuess(ctx context.Context, req httpapi.GuessRequest) (*game.GuessResult, error) {
	err := a.Temporal.SignalWorkflow(ctx, req.GameID, "", shared.SignalGuess, shared.GuessInput{Word: req.Word})
	return nil, err
}
```

- The empty string is the Run ID. Leaving it empty targets the current run of that Workflow ID.
- `SignalWorkflow` returns once the Signal is recorded. A `nil` result tells the HTTP layer to answer `202`.

## 4. Run it

```sh
make clean-workflows
```

Restart the Worker and the server. Compare with `solutions/03`.
