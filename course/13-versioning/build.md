# 13 · Versioning (Go)

## 1. Save an old history

With your Lesson 12 code running, start a game as `alice` and leave it running:

```sh
make history ID=game-alice-3   # use your game's ID
```

## 2. Patch the Workflow

In `GameWorkflow`, after `NewState`:

```go
	state = game.NewState(gameID, input.PlayerID, puzzle, now)
	// Games started before this change have no "announce-game" marker in their
	// history, so GetVersion returns DefaultVersion and they skip the Activity.
	if workflow.GetVersion(ctx, "announce-game", workflow.DefaultVersion, 1) == 1 {
		announce(ctx, startedMessage(state))
	}
	state.SetDeadline(now.Add(puzzle.TimeLimit()))
```

`GetVersion(ctx, changeID, minSupported, maxSupported)`:

- On first execution, it records a marker with `maxSupported` (1) and returns 1.
- On replay with the marker, it returns the recorded version.
- On replay without the marker, it returns `DefaultVersion` (-1).
- If the history holds a version outside `[minSupported, maxSupported]`, the Workflow Task fails. That protects you from deploying code that has dropped support for a version still in use.

## 3. Check before deploying

```sh
make replay
```

Every history passes, including the one from step 1.

## 4. Deploy

Restart the Worker and follow the checklist in the README. To check rollback:

```sh
make history ID=game-bob-1
make replay S=12
```

`S=12` runs `solutions/12/cmd/replay`, the Lesson 12 code, against every saved history. Bob's game fails.

## 5. Later: retire the old path

When every game that started before the patch has finished:

```go
	workflow.GetVersion(ctx, "announce-game", 1, 1)
	announce(ctx, startedMessage(state))
```

Once every Workflow recorded with the marker has finished too, you can delete the `GetVersion` line.

Compare with `solutions/13`.
