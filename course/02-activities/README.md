# 02 · Activities

**Goal:** the Workflow loads its own puzzle, so the API only passes a puzzle ID.

## Concepts

**Activity.** A normal function for work that touches the outside world: files, databases, HTTP calls. Workflows can't do this work directly because it isn't deterministic. A Workflow asks for an Activity to run. A Worker runs it, and the result is recorded in the Event History. When the Workflow is re-run, it reads the recorded result instead of calling the Activity again.

**Timeouts.** Every Activity needs a **Start-To-Close** timeout: the longest one attempt may take. If an attempt runs longer, it fails and Temporal retries it.

**Retries.** Failed Activities are retried automatically. The default policy retries forever, waiting 1 s, then 2 s, 4 s, and so on, up to 100 s between attempts. Retries don't add events to the history. You see one `ActivityTaskStarted` event that records the final attempt number.

**Non-retryable errors.** Some failures will never succeed on retry. Asking for a puzzle that doesn't exist is one. Mark these errors **non-retryable** and the Activity fails at once.

**Dependencies.** Activities get their dependencies, such as the puzzle store, when the Worker starts. Workflows never touch them.

## Patterns

- [Non-Retryable Errors](https://docs.temporal.io/design-patterns/non-retryable-errors)
- [Activity Dependency Injection](https://docs.temporal.io/design-patterns/activity-dependency-injection) (first look; Lesson 8 goes further)

## What you'll build

- A `LoadPuzzle` Activity that reads from the puzzle store. A missing puzzle is a non-retryable error.
- `GameWorkflow` takes `{puzzleId}` and calls `LoadPuzzle` with a 5-second Start-To-Close timeout.
- The Worker registers the Activity with a puzzle store.
- `StartGame` passes the puzzle ID instead of the puzzle.

## Build it

Follow [`build.md`](build.md).

## Check it

- [ ] Starting a game works as before.
- [ ] In the Temporal UI, the Workflow's input is `{"puzzleId": "..."}`, with no answers.
- [ ] The history has `ActivityTaskScheduled`, `ActivityTaskStarted`, and `ActivityTaskCompleted` for `LoadPuzzle`. The puzzle is in the completed event's result.
- [ ] Restart the Worker with `PUZZLES_FAIL_FIRST=2 make worker`. The puzzle store now fails its first two reads. Start a game. It takes about 3 seconds: a 1 s wait, then a 2 s wait, then success.
- [ ] Open that Workflow. `ActivityTaskStarted` shows `attempt: 3`, and there is still only one of each Activity event. Click the Activity to see the last failure.
- [ ] Start a Workflow for a puzzle that doesn't exist (the API checks for this, so use the CLI):

  ```sh
  temporal workflow start --type GameWorkflow --task-queue wordflow \
    --workflow-id bad-puzzle --input '{"puzzleId":"nope"}'
  ```

  It fails at once with `no puzzle named nope`, after one attempt.

Restart the Worker with plain `make worker` when you're done.
