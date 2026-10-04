# 09 · Signal-with-Start

**Goal:** a global leaderboard that every player's finished games feed into.

## Concepts

**Singleton Workflow.** One Workflow with a fixed ID, `leaderboard`, for the whole app. It's an entity, like a player, with one instance.

**Signal-with-Start.** Sends a Signal to a Workflow and, if it isn't running, starts it first, in one atomic call. Nobody has to create the leaderboard ahead of time. The first finished game creates it.

**Why a Signal here.** The player doesn't need a reply, and the leaderboard should never slow a player down. Signals are recorded and processed in order, even if the leaderboard's Worker is busy.

**Safe to repeat.** Each entry carries a version (the player's games played). The leaderboard ignores anything older than what it has. A Signal that arrives twice or out of order can't make the board wrong.

**Activities can't import Workflows.** Workflows already import Activities, and Go doesn't allow import cycles. So the Activity starts the leaderboard by its type name, `LeaderboardWorkflow`.

## Patterns

- [Signal-with-Start](https://docs.temporal.io/design-patterns/signal-with-start)
- [Entity Workflow](https://docs.temporal.io/design-patterns/entity-workflow)

## What you'll build

- `LeaderboardWorkflow`: a `leaderboard` Query, and a loop that receives `score` Signals and updates the board.
- A `PublishScore` Activity that uses Signal-with-Start.
- The Player runs `PublishScore` after recording each finished game.
- `GetLeaderboard` in the API queries `leaderboard`. If it doesn't exist yet, it returns an empty board.

## Build it

Follow [`build.md`](build.md).

## Check it

**Don't** clean up first this time. Keep Lesson 8's `player-alice`, with at least one finished game, and restart the Worker and the server.

- [ ] Within a few seconds (the app queries the player, which makes the Worker load her history), the Worker logs an error with code `TMPRL1100`, a nondeterminism error. `player-alice` shows a failing Workflow Task in the Temporal UI. Her history says "game finished, then wait". The new code says "game finished, then run `PublishScore`". When the code and the history disagree, Temporal stops the Workflow instead of guessing. Lesson 12 is about this. For now, run `make clean-workflows`.
- [ ] Before any game finishes, the leaderboard is empty and there's no `leaderboard` Workflow.
- [ ] Join as `alice` and finish a game. The `leaderboard` Workflow appears. Its history starts with `WorkflowExecutionStarted` followed by `WorkflowExecutionSignaled`, from one call. Alice is on the board.
- [ ] Join as `bob` and finish a game. Both are listed, ranked by points earned. The leaderboard has one more Signal and is still the same Workflow.
