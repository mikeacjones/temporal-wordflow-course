# Extra · Approval

**Goal:** players submit new puzzles. Every word is checked against the dictionary at once, then a person approves or rejects the puzzle before it's published.

## Concepts

**Parallel Execution.** Start all the Activities first, then wait for each result. Eight lookups take about as long as one.

**Approval.** A Workflow waits for a person's decision, as a Signal, with a deadline. Waiting a day costs nothing: no Worker is busy, and the Timer lives in the Temporal Service.

**Status from a Query.** The submitter or reviewer can check progress at any time without changing anything.

## Patterns

- [Parallel Execution](https://docs.temporal.io/design-patterns/parallel-execution)
- [Approval](https://docs.temporal.io/design-patterns/approval)

## What you'll build

- `PuzzleSubmissionWorkflow`, with Workflow ID `submission-<puzzleId>`:
  1. Check the puzzle's shape with the provided rules.
  2. Look up every word in parallel, and note any the dictionary doesn't know.
  3. Wait for a `review` Signal (`{approved, reviewer, note}`) or 24 hours.
  4. If approved, save the puzzle with a `SavePuzzle` Activity and announce it.
- A `submission` Query that returns the status, the unknown words, and the review.
- A command that submits a puzzle file.

## Build it

Follow [`build.md`](build.md).

## Check it

- [ ] Submit `submissions/bread.json`. The `submission` Query returns `awaiting_review`.
- [ ] In the history, eight `ActivityTaskScheduled` events come before any `ActivityTaskCompleted`, then a `TimerStarted` labelled **review deadline**.
- [ ] Send the `review` Signal with `{"approved": true, "reviewer": "you"}`. The Timer is cancelled, the puzzle is saved, the feed shows **New puzzle: Bread, approved by you**, and **Bread** appears in the puzzle list.
- [ ] Copy the file, change its `id` to `bread-2`, and add the word `DRAEB`. Submit it. The Query shows `DRAEB` under `unknownWords`. Reject it, and the Workflow completes with status `rejected`.

Delete `puzzles/bread.json` afterwards if you want the original puzzle list back.
