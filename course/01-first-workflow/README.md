# 01 · Your first Workflow

**Goal:** clicking a puzzle starts a Workflow that creates a new game and returns the board.

## Concepts

**Workflow.** A function whose progress Temporal records. Each step it takes is saved as an event in its **Event History**. If the Worker running it crashes, another Worker rebuilds its state by re-running the code against that history. This is what makes a Workflow durable.

**Determinism.** Because Workflow code is re-run, it must make the same decisions every time. It can't read the system clock, generate random numbers, or call the network directly. Use the SDK's clock (`workflow.Now` and its equivalents), which returns the time recorded in history. That's why the game rules take `now` as an argument.

**Worker.** Your process. It connects to the Temporal Service, polls a **Task Queue**, and runs the Workflows registered with it. Ours polls `wordflow`.

**Temporal Client.** How the API talks to Temporal. *Execute Workflow* starts a Workflow and returns a handle. Calling *get result* on the handle waits for the Workflow to finish.

**Workflow ID.** Your name for a Workflow. Only one Workflow with a given ID can run at a time. The web app sends an `Idempotency-Key` with each request, and the API uses it as the Workflow ID (`game-<key>`). If a request is retried, it can't start a second game.

## What you'll build

- Shared names: the Task Queue and the Workflow's input type.
- `GameWorkflow`: takes a puzzle, creates the game state, returns its view.
- The Worker: connect, register `GameWorkflow`, run.
- `StartGame` in the API: read the puzzle file, start `GameWorkflow` with it, wait for the result.

For now the API reads the puzzle and passes it in. Lesson 2 moves that into the Workflow.

## Build it

Follow [`build.md`](build.md).

## Check it

Run `make worker` and `make server`.

- [ ] Click a puzzle. The board appears: letters, and blank slots for each word.
- [ ] Try a guess. You see **This arrives in Lesson 3**.
- [ ] Click **Workflow ↗**. The Temporal UI shows a `GameWorkflow` with status **Completed**.
- [ ] Its Event History has five events: `WorkflowExecutionStarted`, then a Workflow Task (scheduled, started, completed), then `WorkflowExecutionCompleted`.
- [ ] The **Input** contains the whole puzzle, answers included. Anyone who can read the history can see them. Lesson 2 fixes that.
- [ ] Stop the Worker and click a puzzle. After 15 seconds the app shows a timeout. In the Temporal UI the new Workflow is **Running** and waiting for a Worker. Start the Worker again and it completes right away.

The last check is the core idea: Temporal holds the work until a Worker is available to do it.
