# Wordflow: learn Temporal by building a word game

You'll build the backend of Wordflow, a timed word game, one Temporal concept at a time. The web app and HTTP routing are already written. You write the Workflows, Activities, and Worker, plus the API code that uses the Temporal Client to start, signal, query, and update Workflows.

Each lesson adds one feature to the game and teaches the concept behind it. By the end you'll have used the [Temporal design patterns](https://docs.temporal.io/design-patterns) that come up most in real systems.

**You need:** a GitHub account, and to be comfortable in the language you choose. No Temporal experience needed.

## Start

1. Pick a language. Each one lives on its own branch:

   | Language | Branch |
   |---|---|
   | Go | [`go`](../../tree/go) |
   | Python | `python` (coming soon) |

2. Switch to that branch on GitHub, then choose **Code → Codespaces → Create codespace on `<branch>`**. Setup takes a few minutes.
3. Open [`course/00-setup`](course/00-setup/README.md).

The `main` branch only has the shared material (lesson text, web app, puzzles), so it won't run on its own.

## How each lesson works

- `README.md` explains the concept and what you'll build. It is the same for every language.
- `build.md` walks you through the code in your language.
- **Check it** tells you what you should see in the game and in the Temporal UI. If you see it, move on.
- You write all your code in `app/`. Each lesson's finished code is in `solutions/NN/`. Run a solution with `make worker S=NN` and `make server S=NN` to compare.

## Lessons

| # | Lesson | What you add | Concepts and patterns |
|---|---|---|---|
| 00 | [Setup](course/00-setup/README.md) | Nothing yet | Temporal Service, Temporal UI, the project |
| 01 | [Your first Workflow](course/01-first-workflow/README.md) | Start a game | Workflow, Worker, Task Queue, Client, Workflow ID |
| 02 | [Activities](course/02-activities/README.md) | Load puzzles from storage | Activity, retries, timeouts, determinism, non-retryable errors |
| 03 | [Signals and Queries](course/03-signals-and-queries/README.md) | Guess words, see the board | Long-running Workflow, Signal, Query |
| 04 | [Updates](course/04-updates/README.md) | Instant guess results | Update, validator, Update-with-Start, Request-Response via Updates, Early Return |
| 05 | [Durable Timers](course/05-timers/README.md) | A game clock you can extend | Timer, cancellation, Updatable Timer |
| 06 | [Entity Workflows](course/06-entity-workflows/README.md) | Players with points | Entity Workflow, ID conflict policy |
| 07 | [Child Workflows](course/07-child-workflows/README.md) | Players start their own games | Child Workflow, parent close policy |
| 08 | [Workflow to Workflow](course/08-workflow-to-workflow/README.md) | Hints that cost points | Activity Dependency Injection, idempotent Updates, Workflow mutex |
| 09 | [Signal-with-Start](course/09-signal-with-start/README.md) | A global leaderboard | Signal-with-Start, singleton Workflow |
| 10 | [Continue-As-New](course/10-continue-as-new/README.md) | Players who play forever | Event History limits, Continue-As-New |
| 11 | [Retries in depth](course/11-retries/README.md) | Bonus words from a flaky dictionary | Retry policies, Fixed Count, Fixed Wall-Time, Delayed Retry, Non-Retryable Errors |
| 12 | [Determinism and replay](course/12-determinism-and-replay/README.md) | An activity feed, and a broken deploy | Replay, nondeterminism errors, replay testing |
| 13 | [Versioning](course/13-versioning/README.md) | Ship the change safely | Patching with `GetVersion`, Worker Versioning |

Optional exercises, after Lesson 13:

| Exercise | What you add | Pattern |
|---|---|---|
| [Saga](course/extras/saga/README.md) | Gift points to a friend | Saga, compensation |
| [Pick First](course/extras/pick-first/README.md) | Race two dictionary regions | Pick First, cancellation |
| [Approval](course/extras/approval/README.md) | Players submit puzzles for review | Parallel Execution, Approval |
| [Schedules](course/extras/schedules/README.md) | Puzzle of the day | Schedules, Delayed Start |

## Reference

- [API and Temporal names](api/contract.md)
- [Temporal documentation](https://docs.temporal.io)
- [Design patterns](https://docs.temporal.io/design-patterns)

## Adding a language

A language branch is created from `main`. It adds files and never edits the shared ones (`README.md`, `course/*/README.md`, `api/`, `web/`, `puzzles/`, `submissions/`). It provides:

- `.devcontainer/` with the language toolchain and the Temporal CLI.
- A `Makefile` with the targets `temporal`, `worker`, `server`, `replay`, `history ID=…`, and `clean-workflows`. `worker` and `server` accept `S=NN` to run a solution.
- The game rules, puzzle store, HTTP server, and fake services described in [the contract](api/contract.md).
- `app/` set to the Lesson 0 starter, and `solutions/00` through `solutions/13` plus `solutions/extras`.
- `course/*/build.md` for every lesson.

Use the Temporal names from the contract exactly, so every lesson's README stays true.
