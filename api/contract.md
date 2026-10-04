# Wordflow contract

Every language branch implements the same HTTP API and uses the same Temporal names. The web app in `web/` only knows about the HTTP API.

## HTTP API

The routing, JSON encoding, and error mapping are provided. You implement the part of each endpoint that calls Temporal.

| Endpoint | Body | Returns | Lesson |
|---|---|---|---|
| `GET /api/puzzles` | | `[{id, title, words, timeLimitSeconds}]` | provided |
| `POST /api/games` | `{puzzleId, player?}` | `{gameId, game?}` | 1 (guest), 7 (player) |
| `GET /api/games/{id}` | | Game view | 3 |
| `POST /api/games/{id}/guesses` | `{word}` | `{word, outcome, game}`, or `202` with no body | 3 (202), 4 (200) |
| `POST /api/games/{id}/extend` | | Game view | 5 |
| `POST /api/players` | `{name}` | Player view | 6 |
| `GET /api/players/{name}` | | Player view | 6 |
| `POST /api/games/{id}/hints` | | `{pointsSpent, game}` | 8 |
| `GET /api/leaderboard` | | `{entries: [{rank, name, totalEarned, gamesWon}]}` | 9 |

Every `POST` from the web app sends an `Idempotency-Key` header. The API passes it to your code as the request ID.

An endpoint that is not built yet returns `501 {"error", "lesson"}`, and the web app shows "This arrives in Lesson N".

### Error mapping

| Temporal error | HTTP |
|---|---|
| Application error (a rejected Update, a failed Workflow) | `400 {"error", "code"}` |
| Workflow not found | `404` |
| Workflow already started | `409` |
| Temporal unavailable | `503` |
| Timed out waiting for a reply | `504` (usually: the Worker is not running) |

### Game view

```json
{
  "gameId": "game-alice-1", "playerId": "alice", "puzzleId": "signal", "title": "Signal",
  "letters": "LANGIS", "status": "playing",
  "words": [{"length": 4, "pattern": "S___", "found": false}],
  "foundCount": 0, "totalWords": 9,
  "rejected": ["GLIN"], "bonusWords": ["SAIL"],
  "guesses": 3, "incorrect": 1,
  "freeHints": 2, "hintCost": 0, "extensionsLeft": 2,
  "expiresAt": "2026-10-04T18:00:00Z", "score": 0
}
```

`status` is `loading`, `playing`, `won`, or `timed_out`. `score` is set when the game ends.

Guess `outcome` is `found`, `already_found`, `not_in_puzzle`, `bonus`, `invalid`, `timed_out`, or `game_over`.

### Player view

```json
{"name": "alice", "joinedAt": "2026-10-04T17:55:00Z", "points": 20, "totalEarned": 0,
 "gamesPlayed": 0, "gamesWon": 0, "activeGameId": "game-alice-1"}
```

## Temporal names

Use these exact names in every language, so the Temporal UI looks the same and the lessons read the same.

Task Queue: `wordflow`

| Workflow type | Workflow ID | Messages |
|---|---|---|
| `GameWorkflow` | `game-<requestId>` (guest), `game-<player>-<n>` (child of a player) | Query `state`, Updates `ready`, `guess` (a Signal in Lesson 3), `extendTime`, `hint` |
| `PlayerWorkflow` | `player-<name>` | Query `player`, Updates `join`, `startGame`, `spendPoints` |
| `LeaderboardWorkflow` | `leaderboard` | Query `leaderboard`, Signal `score` |

Optional exercises add `TransferPointsWorkflow` (Updates `withdraw`, `deposit` on `PlayerWorkflow`), `PuzzleSubmissionWorkflow` (ID `submission-<puzzleId>`, Signal `review`, Query `submission`), and `DailyPuzzleWorkflow` (Schedule ID `daily-puzzle`).

## External services

The server also runs fake external services. Use them from Activities, never from Workflows.

| Endpoint | Purpose |
|---|---|
| `GET /services/dictionary/words/{word}` | `200` real word, `404` not a word. Add `?region=NAME` for 0–1.5 s of random latency. |
| `GET` / `PUT /services/dictionary/mode` | `ok`, `flaky`, `slow`, `rate_limited`, `down`, `unauthorized` |
| `GET` / `POST /services/feed` | Activity feed. `POST {"message"}` |

## Game rules (provided)

Each language branch provides the game rules as plain, deterministic code: the game state, player state, leaderboard, and puzzle store. Lessons call that code; they never re-implement it.

- Words are 3+ letters, made from the puzzle's letters.
- A wrong guess costs 1 point. A real word that is not in the puzzle is a bonus worth 2.
- Score = letters in found words + bonuses − wrong guesses (+10 for finding every word), never below 0.
- 2 free hints, then 10 points each. Guests cannot buy hints.
- Up to 2 extensions of 30 seconds.
- New players start with 20 points. Finishing a game adds its score.
