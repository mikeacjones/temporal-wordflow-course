# 06 · Entity Workflows

**Goal:** players can join. Each player is a Workflow that holds their points and stats.

## Concepts

**Entity Workflow.** A Workflow that represents one thing, here a player, for as long as it exists. It keeps the entity's state in memory and changes it through messages. The Workflow ID is the entity's key: `player-alice`. There's no database: the Event History is the record.

**Only one at a time.** Temporal allows one running Workflow per ID. Two requests to create `player-alice` can't produce two players.

**ID conflict policy.** What happens when you start a Workflow whose ID is already running. The default *fail* returns an "already started" error. *Use existing* returns the running Workflow instead.

**ID reuse policy.** What happens when you start a Workflow whose ID belongs to a *closed* Workflow. The default allows it, which creates a new run with a fresh history.

**Join is idempotent.** Update-with-Start with *use existing*: the first join creates the player, and every later join just returns the player's view. The API doesn't need to check whether the player exists first.

## Patterns

- [Entity Workflow](https://docs.temporal.io/design-patterns/entity-workflow)

## What you'll build

- `PlayerWorkflow`: creates the player, registers a `player` Query and a `join` Update that returns the player's view, and then waits forever.
- `JoinPlayer`: Update-with-Start on `player-<name>`, with Update `join`.
- `GetPlayer`: the `player` Query.

Games are still guest games. Lesson 7 connects them to players.

## Build it

Follow [`build.md`](build.md).

## Check it

- [ ] Join as `alice`. The player panel shows 20 points. In the Temporal UI, `player-alice` is **Running**.
- [ ] Click **Switch player** and join as `alice` again. The history has a second `join` Update, and there's still one Workflow. Both Updates return the same `joinedAt`.
- [ ] Reload the page. You're still `alice`: the app queries the player.
- [ ] `curl -i localhost:8080/api/players/nobody` returns `404`.
- [ ] Play a game. Alice's points don't change yet.
