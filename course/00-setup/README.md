# 00 · Setup

**Goal:** get the Temporal Service, the web app, and the Temporal UI running, and find your way around the project.

## What's running

| Process | Command | Port | What it does |
|---|---|---|---|
| Temporal Service | `make temporal` | 7233, UI on 8233 | Stores Workflow state and hands out work. A dev server that keeps its data in `.temporal/`. |
| Web app and API | `make server` | 8080 | The game UI, the HTTP API, and fake external services. |
| Worker | `make worker` | | Runs your Workflow and Activity code. You build it in Lesson 1. |

The Temporal Service never runs your code. It records what happened and puts tasks on a Task Queue. Your Worker polls that queue, runs the code, and reports back.

## Project layout

| Path | Owner | What |
|---|---|---|
| `app/` | you | Everything you write: Workflows, Activities, the Worker, and the API functions that call Temporal. |
| `solutions/NN/` | course | The finished `app/` after each lesson. |
| `internal/` | course | Game rules, puzzle storage, HTTP routing, fake services. You call these; you don't change them. |
| `web/` | course | The browser app. |
| `puzzles/` | course | One JSON file per puzzle. |
| `course/` | course | These lessons. |

## Build it

Follow [`build.md`](build.md).

## Check it

- [ ] The Temporal UI opens on port 8233 and shows the `default` namespace with no Workflows.
- [ ] The web app opens on port 8080 and lists five puzzles.
- [ ] Clicking a puzzle shows **Not built yet. This arrives in Lesson 1.**
- [ ] The **External services** panel shows the dictionary mode and an empty feed.

## Working through the course

- Keep three terminals open: `make temporal`, `make server`, and `make worker`.
- Restart the Worker after changing Workflow or Activity code. Restart the server after changing API code.
- Games from earlier lessons keep running with the code they started on. If the Worker logs errors about old Workflows, end them with `make clean-workflows`. Lesson 12 explains why this happens.
- To compare with the finished code, run `make worker S=NN` and `make server S=NN`.
- `make reset-temporal` (with `make temporal` stopped) deletes every Workflow and starts fresh.
