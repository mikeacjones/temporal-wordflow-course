# 05 · Durable Timers (Go)

## 1. Name

In `app/shared/shared.go`, add `UpdateExtendTime = "extendTime"` to the Game message names.

## 2. The extendTime Update

In `app/workflows/game.go`, register it next to the other handlers:

```go
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
```

`state.Extend` re-checks the deadline using the current time. The validator can pass just before the deadline, and the handler can run just after it.

## 3. Set the deadline and run the clock

Replace the lines after `loadPuzzle`, up to the `AllHandlersFinished` wait:

```go
	now := workflow.Now(ctx)
	state = game.NewState(gameID, "", puzzle, now)
	state.SetDeadline(now.Add(puzzle.TimeLimit()))

	if err := runClock(ctx, state); err != nil {
		return game.Result{}, err
	}
```

Add `runClock`:

```go
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
```

- `NewTimer` returns a Future that becomes ready when the Timer fires. `Summary` labels it in the Temporal UI.
- Cancelling `timerCtx` cancels the Timer. Calling `cancelTimer` after the Timer fired does nothing.
- `workflow.AwaitWithTimeout` looks like a shortcut here, but in this SDK version it doesn't cancel its Timer when the condition wins. Stale Timers would keep firing.
- The loop handles all three exits: the game ended (loop condition), the deadline moved (go round with the new deadline), or the Timer fired (expire).

## 4. The API

Implement `ExtendTime` in `app/api/api.go`:

```go
func (a *API) ExtendTime(ctx context.Context, req httpapi.GameRequest) (game.View, error) {
	handle, err := a.Temporal.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   req.GameID,
		UpdateID:     req.RequestID,
		UpdateName:   shared.UpdateExtendTime,
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return game.View{}, err
	}
	var view game.View
	err = handle.Get(ctx, &view)
	return view, err
}
```

## 5. Run it

```sh
make clean-workflows
```

Restart the Worker and the server. Compare with `solutions/05`.
