# 12 · Determinism and replay

**Goal:** understand exactly why changing Workflow code can break running Workflows, and catch it before you deploy.

## Concepts

**Replay.** When a Worker needs a Workflow's state, it runs the Workflow code from the start and feeds it the recorded events. Activity results, Timer firings, and messages all come from history instead of happening again. A Worker that restarts, or one that loads a Workflow it hasn't cached, replays.

**Commands must match events.** As the code runs, it produces commands: "schedule this Activity", "start this Timer". During replay, each command must match the next recorded event. If the code now schedules an Activity where the history has a Timer, the Worker can't continue safely. That's a **nondeterminism error** (`TMPRL1100`).

**What happens then.** The Workflow doesn't fail. Its Workflow Task fails and is retried until a Worker with compatible code picks it up. Fix the code, or roll back the deploy, and it continues.

**What breaks replay:**

- Adding, removing, or reordering Activities, Timers, Child Workflows, or anything else that produces a command, in code that existing Workflows have already passed.
- Reading the clock, random numbers, environment variables, or global state directly.
- Ranging over maps, or using native threads, goroutines, or channels in Workflow code.

**What's safe:** changing Activity code, changing code that runs only after every existing Workflow has passed that point, and changing the logic inside a handler, as long as it produces the same commands.

**Replay testing.** Save real histories. Before deploying, replay them against the new code. A failure means the deploy would break running Workflows.

## What you'll build

- A `FeedActivities.Announce` Activity that posts to the feed service, and an `announce` helper that never fails the game. The helper isn't called yet.
- A replay command that loads saved histories and replays them against your Workflow code.
- Then you'll break it on purpose.

## Build it

Follow [`build.md`](build.md).

## Check it

- [ ] Play one game to the end and start another. Save their histories, and the player's, with `make history ID=...`. `make replay` prints `ok` for each.
- [ ] Add a call to `announce` right after the game state is created. `make replay` prints `FAIL` for every game, with a `TMPRL1100` error. The new code schedules an Activity where the history has a Timer.
- [ ] Start a game on the old code, then restart the Worker with the new code and make a guess. The guess times out. In the Temporal UI, the game shows a failed Workflow Task with the nondeterminism error, retrying.
- [ ] Remove the call and restart the Worker. The stuck game picks up where it left off.

Lesson 13 ships the same change safely.
