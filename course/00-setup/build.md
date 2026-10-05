# 00 · Setup (Go)

## 1. Start the Temporal Service

The Codespace installs the Temporal CLI. In a terminal:

```sh
make temporal
```

Leave it running. Open the **Ports** tab and open port **8233** to see the Temporal UI.

## 2. Start the web app

In a second terminal:

```sh
make server
```

Port **8080** opens in your browser.

## 3. Look around

```
app/
  api/api.go          # one method per endpoint; all return ErrNotImplemented
  cmd/server/main.go  # starts the web app; you add a Temporal Client in Lesson 1
  cmd/worker/main.go  # you build the Worker in Lesson 1
internal/
  game/               # game rules: State, Player, Leaderboard
  puzzles/            # reads puzzles/*.json
  httpapi/            # routes, JSON, error mapping
  services/           # fake dictionary and feed
```

Open `app/api/api.go`. Each method is one endpoint and names the lesson that builds it. The HTTP layer in `internal/httpapi` decodes the request, calls your method, and turns its result or error into a response.

Skim `internal/game/game.go`. Its functions take the current time as an argument instead of calling `time.Now()`. You'll see why in Lesson 1.

## 4. Useful commands

| Command | What |
|---|---|
| `make worker` / `make server` | Run your code in `app/`. |
| `make worker S=05` | Run Lesson 5's solution. |
| `make build` | Compile and vet everything. |
| `make clean-workflows` | Terminate every running Workflow. |
| `make reset-temporal` | Delete all Temporal data (stop `make temporal` first). |
| `temporal workflow list` | List Workflows from the terminal. |
