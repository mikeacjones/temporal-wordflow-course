# 10 · Continue-As-New (Go)

## 1. Carry state in the input

In `app/shared/shared.go`, add `import "wordflow/internal/game"` back, then:

```go
type LeaderboardInput struct {
	State *game.Leaderboard `json:"state,omitempty"`
}

type PlayerInput struct {
	Name string `json:"name"`
	// State is set when the Workflow continues as new.
	State *game.Player `json:"state,omitempty"`
}
```

The game types are plain structs with exported, JSON-tagged fields, so they serialize as Workflow input.

## 2. The Player

In `app/workflows/player.go`, add near the top:

```go
// historyLimit is low so you can watch Continue-As-New happen. Real code can
// rely on GetContinueAsNewSuggested alone.
const historyLimit = 100
```

Start from the input state if there is one:

```go
	player := input.State
	if player == nil {
		player = game.NewPlayer(input.Name, workflow.Now(ctx))
	}
```

Replace the start of the main loop, up to `var result game.Result`:

```go
	for {
		err := workflow.Await(ctx, func() bool { return activeGame != nil || shouldContinueAsNew(ctx) })
		if err != nil {
			return err
		}
		if activeGame == nil {
			// Let running handlers finish. One of them may start a game.
			if err := workflow.Await(ctx, func() bool { return workflow.AllHandlersFinished(ctx) }); err != nil {
				return err
			}
			if activeGame == nil {
				return workflow.NewContinueAsNewError(ctx, PlayerWorkflow, shared.PlayerInput{Name: player.Name, State: player})
			}
		}

		var result game.Result
		// ... unchanged
```

Add the check:

```go
func shouldContinueAsNew(ctx workflow.Context) bool {
	info := workflow.GetInfo(ctx)
	return info.GetContinueAsNewSuggested() || info.GetCurrentHistoryLength() >= historyLimit
}
```

- Continue-As-New is an error you return. `NewContinueAsNewError` takes the Workflow function and its new input.
- `GetCurrentHistoryLength` is the event count, and it's deterministic.
- After waiting for handlers, check `activeGame` again: a `startGame` might have just finished.

## 3. The leaderboard

In `app/workflows/leaderboard.go`, start from the input state:

```go
	board := input.State
	if board == nil {
		board = &game.Leaderboard{}
	}
```

At the end of the loop, after `board.Upsert(entry)`:

```go
		if shouldContinueAsNew(ctx) {
			// Apply Signals that already arrived. They do not carry over to the new run.
			for {
				var pending game.LeaderboardEntry
				if !scores.ReceiveAsync(&pending) {
					break
				}
				board.Upsert(pending)
			}
			return workflow.NewContinueAsNewError(ctx, LeaderboardWorkflow, shared.LeaderboardInput{State: board})
		}
```

`ReceiveAsync` returns false right away when the channel is empty.

## 4. Run it

```sh
make clean-workflows
```

Restart the Worker. The API didn't change. Compare with `solutions/10`.
