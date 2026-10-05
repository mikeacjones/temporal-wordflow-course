# Wordflow: learn Temporal by building a word game

You'll build the backend of Wordflow, a timed word game, one Temporal concept at a time. The web app and HTTP routing are already written. You write the Workflows, Activities, and Worker, plus the API code that uses the Temporal Client to start, signal, query, and update Workflows.

By the end you'll have used the [Temporal design patterns](https://docs.temporal.io/design-patterns) that come up most in real systems.

## How each lesson works

- **Concepts** and **What you'll build** explain the idea and the feature it adds to the game.
- **Build it** walks you through the code. You write all of it in `app/`.
- **Check it** lists what you should see in the game and in the Temporal UI. Tick each item as you see it. Your ticks are saved in this browser, and a finished lesson gets a ✓ in the sidebar.
- Each lesson's finished code is in `solutions/NN/`. Run it with `make worker S=NN` and `make server S=NN` to compare.

Use the **Web app** and **Temporal UI** buttons in the sidebar to open them once they're running.

## Start

Begin with [00 · Setup](/course/00-setup/README.md).
