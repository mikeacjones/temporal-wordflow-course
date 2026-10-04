// Temporal Wordflow web UI. Provided by the course; you do not need to edit it.
// It calls the HTTP API and copes with endpoints that are not built yet (501).

const $ = (selector) => document.querySelector(selector);

const ui = {
  player: localStorage.getItem("player") || "",
  gameId: localStorage.getItem("gameId") || "",
  game: null,
  letterOrder: null,
  canQueryGame: true,
  playerTimer: null,
};

const OUTCOMES = {
  found: ["good", (w) => `✓ ${w}`],
  already_found: ["", (w) => `${w} is already found`],
  not_in_puzzle: ["bad", (w) => `${w} is not in this puzzle`],
  bonus: ["good", (w) => `★ ${w} is a bonus word (+2)`],
  invalid: ["bad", (w) => `${w} can't be made from these letters`],
  timed_out: ["bad", () => "Time is up"],
  game_over: ["", () => "The game is over"],
};

// ---------- HTTP ----------

function newKey() {
  return Array.from(crypto.getRandomValues(new Uint8Array(6)), (b) => b.toString(16).padStart(2, "0")).join("");
}

async function api(method, path, body) {
  const headers = { "Content-Type": "application/json" };
  if (method !== "GET") headers["Idempotency-Key"] = newKey();
  const res = await fetch(path, { method, headers, body: body ? JSON.stringify(body) : undefined });
  let data = null;
  try { data = await res.json(); } catch { /* empty body */ }
  if (!res.ok) {
    const err = new Error((data && data.error) || res.statusText);
    Object.assign(err, { status: res.status, lesson: data && data.lesson, code: data && data.code });
    throw err;
  }
  return { status: res.status, data };
}

function showProblem(panel, err) {
  const note = panel.querySelector(".lesson-note");
  const error = panel.querySelector(".error");
  if (err && err.status === 501) {
    note.textContent = `Not built yet. This arrives in Lesson ${err.lesson}.`;
    note.hidden = false;
    error.hidden = true;
  } else if (err) {
    error.textContent = err.message;
    error.hidden = false;
    note.hidden = true;
  } else {
    note.hidden = true;
    error.hidden = true;
  }
}

// ---------- Temporal UI links ----------

function temporalUIBase() {
  const { protocol, hostname } = location;
  const codespace = hostname.match(/^(.*)-8080(\..*)$/);
  if (codespace) return `${protocol}//${codespace[1]}-8233${codespace[2]}`;
  return `${protocol}//${hostname}:8233`;
}

function workflowURL(id) {
  return `${temporalUIBase()}/namespaces/default/workflows/${encodeURIComponent(id)}`;
}

// ---------- Puzzles ----------

async function loadPuzzles() {
  const panel = $("#puzzle-panel");
  try {
    const { data } = await api("GET", "/api/puzzles");
    $("#puzzles").replaceChildren(...data.map((p) => {
      const row = document.createElement("div");
      row.className = "puzzle";
      row.innerHTML = `<span><strong></strong> <span class="muted"></span></span><button>Play</button>`;
      row.querySelector("strong").textContent = p.title;
      row.querySelector(".muted").textContent = `${p.words} words · ${p.timeLimitSeconds}s`;
      row.querySelector("button").onclick = () => startGame(p.id);
      return row;
    }));
  } catch (err) {
    showProblem(panel, err);
  }
}

async function startGame(puzzleId) {
  const panel = $("#puzzle-panel");
  showProblem(panel, null);
  try {
    const { data } = await api("POST", "/api/games", { puzzleId, player: ui.player });
    setGame(data.gameId);
    if (data.game) renderGame(data.game);
    else renderGame({ gameId: data.gameId, status: "loading", words: [], rejected: [], bonusWords: [] });
    setMessage("", "");
    refreshGame();
    refreshPlayer();
  } catch (err) {
    showProblem(panel, err);
  }
}

// ---------- Game ----------

function setGame(gameId) {
  ui.gameId = gameId;
  ui.letterOrder = null;
  ui.canQueryGame = true;
  localStorage.setItem("gameId", gameId);
}

async function refreshGame() {
  if (!ui.gameId || !ui.canQueryGame) return;
  try {
    const { data } = await api("GET", `/api/games/${encodeURIComponent(ui.gameId)}`);
    renderGame(data);
  } catch (err) {
    if (err.status === 501) {
      ui.canQueryGame = false;
    } else if (err.status === 404 && !ui.game) {
      localStorage.removeItem("gameId");
      ui.gameId = "";
    } else {
      showProblem($("#game-panel"), err);
    }
  }
}

function renderGame(game) {
  const previous = ui.game;
  ui.game = game;
  const panel = $("#game-panel");
  panel.hidden = false;
  showProblem(panel, null);

  $("#game-title").textContent = game.title || "Loading puzzle…";
  const status = $("#game-status");
  status.textContent = game.status.replace("_", " ");
  status.className = `badge ${game.status}`;
  const link = $("#game-link");
  link.href = workflowURL(game.gameId);
  link.textContent = `${game.gameId} ↗`;

  $("#words").replaceChildren(...game.words.map((w) => {
    const row = document.createElement("div");
    row.className = `word${w.found ? " found" : ""}`;
    for (const ch of w.pattern) {
      const cell = document.createElement("div");
      cell.className = "cell";
      cell.textContent = ch === "_" ? "" : ch;
      row.append(cell);
    }
    return row;
  }));

  if (!ui.letterOrder || ui.letterOrder.length !== (game.letters || "").length) {
    ui.letterOrder = (game.letters || "").split("");
  }
  $("#letters").replaceChildren(...ui.letterOrder.map((ch) => {
    const b = document.createElement("button");
    b.type = "button";
    b.className = "letter secondary";
    b.textContent = ch;
    b.onclick = () => { $("#guess").value += ch; $("#guess").focus(); };
    return b;
  }));

  chips("#rejected", game.rejected);
  chips("#bonus", game.bonusWords);

  const playing = game.status === "playing";
  for (const id of ["#guess", "#guess-form button[type=submit]", "#hint", "#extend", "#shuffle"]) {
    $(id).disabled = !playing;
  }
  $("#hint").textContent = game.freeHints > 0 ? `Hint (${game.freeHints} free)` : `Hint (${game.hintCost} pts)`;
  $("#extend").textContent = `+30s (${game.extensionsLeft ?? 0} left)`;
  if (playing && (!game.expiresAt || game.extensionsLeft === 0)) $("#extend").disabled = true;

  const summary = [`Found ${game.foundCount ?? 0}/${game.totalWords ?? 0}`, `Guesses ${game.guesses ?? 0}`];
  if (game.status === "won" || game.status === "timed_out") summary.push(`Score ${game.score}`);
  $("#game-summary").textContent = summary.join(" · ");

  if (previous && previous.gameId === game.gameId && previous.status === "playing" && game.status !== "playing") {
    setMessage(game.status === "won" ? "good" : "bad", game.status === "won" ? `Solved! ${game.score} points` : "Time is up");
    setTimeout(refreshPlayer, 500);
  }
}

function chips(selector, words) {
  $(selector).replaceChildren(...(words || []).map((w) => {
    const span = document.createElement("span");
    span.className = "chip";
    span.textContent = w;
    return span;
  }));
}

function setMessage(kind, text) {
  const el = $("#message");
  el.className = kind;
  el.textContent = text;
}

function tickTimer() {
  const el = $("#game-timer");
  const game = ui.game;
  if (!game || !game.expiresAt || game.status !== "playing") {
    el.textContent = "";
    return;
  }
  const seconds = Math.max(0, Math.ceil((new Date(game.expiresAt) - Date.now()) / 1000));
  el.textContent = `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
  el.className = `timer${seconds <= 15 ? " low" : ""}`;
}

async function gameAction(fn) {
  const panel = $("#game-panel");
  showProblem(panel, null);
  try {
    await fn();
  } catch (err) {
    if (err.status === 501) showProblem(panel, err);
    else setMessage("bad", err.message);
  }
}

$("#guess-form").onsubmit = (event) => {
  event.preventDefault();
  const word = $("#guess").value.trim();
  if (!word || !ui.gameId) return;
  $("#guess").value = "";
  gameAction(async () => {
    const { status, data } = await api("POST", `/api/games/${encodeURIComponent(ui.gameId)}/guesses`, { word });
    if (status === 202) {
      setMessage("", `Sent ${word.toUpperCase()}. The Workflow will apply it shortly.`);
      setTimeout(refreshGame, 300);
      return;
    }
    renderGame(data.game);
    const [kind, text] = OUTCOMES[data.outcome] || ["", data.outcome];
    setMessage(kind, text(data.word));
  });
};

$("#hint").onclick = () => gameAction(async () => {
  const { data } = await api("POST", `/api/games/${encodeURIComponent(ui.gameId)}/hints`);
  renderGame(data.game);
  setMessage("good", data.pointsSpent ? `Revealed a letter (−${data.pointsSpent} points)` : "Revealed a letter");
  if (data.pointsSpent) refreshPlayer();
});

$("#extend").onclick = () => gameAction(async () => {
  const { data } = await api("POST", `/api/games/${encodeURIComponent(ui.gameId)}/extend`);
  renderGame(data);
  setMessage("good", "+30 seconds");
});

$("#shuffle").onclick = () => {
  if (!ui.letterOrder) return;
  for (let i = ui.letterOrder.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [ui.letterOrder[i], ui.letterOrder[j]] = [ui.letterOrder[j], ui.letterOrder[i]];
  }
  if (ui.game) renderGame(ui.game);
};

// ---------- Player ----------

function showPlayer(view) {
  $("#join-form").hidden = !!view;
  $("#player-info").hidden = !view;
  if (!view) return;
  $("#player-name").textContent = view.name;
  $("#player-stats").textContent =
    `${view.points} pts · ${view.gamesWon}/${view.gamesPlayed} won · ${view.totalEarned} earned`;
  const link = $("#player-link");
  link.href = workflowURL(`player-${view.name}`);
  if (view.activeGameId && view.activeGameId !== ui.gameId) {
    setGame(view.activeGameId);
    refreshGame();
  }
}

async function refreshPlayer() {
  const panel = $("#player-panel");
  if (!ui.player) {
    showPlayer(null);
    return;
  }
  try {
    const { data } = await api("GET", `/api/players/${encodeURIComponent(ui.player)}`);
    showProblem(panel, null);
    showPlayer(data);
  } catch (err) {
    if (err.status === 404) {
      ui.player = "";
      localStorage.removeItem("player");
      showPlayer(null);
    } else {
      showProblem(panel, err);
    }
  }
}

$("#join-form").onsubmit = async (event) => {
  event.preventDefault();
  const panel = $("#player-panel");
  showProblem(panel, null);
  try {
    const { data } = await api("POST", "/api/players", { name: $("#join-name").value });
    ui.player = data.name;
    localStorage.setItem("player", data.name);
    showPlayer(data);
  } catch (err) {
    showProblem(panel, err);
  }
};

$("#leave").onclick = () => {
  ui.player = "";
  localStorage.removeItem("player");
  showPlayer(null);
};

// ---------- Leaderboard ----------

let leaderboardBuilt = true;

async function refreshLeaderboard() {
  if (!leaderboardBuilt) return;
  const panel = $("#leaderboard-panel");
  try {
    const { data } = await api("GET", "/api/leaderboard");
    showProblem(panel, null);
    $("#leaderboard").replaceChildren(...data.entries.map((e) => {
      const li = document.createElement("li");
      li.textContent = `${e.name} · ${e.totalEarned} pts · ${e.gamesWon} won`;
      return li;
    }));
    if (data.entries.length === 0) $("#leaderboard").innerHTML = `<span class="muted">No scores yet</span>`;
  } catch (err) {
    if (err.status === 501) leaderboardBuilt = false;
    showProblem(panel, err);
  }
}

// ---------- External services ----------

async function loadServices() {
  try {
    const { data } = await api("GET", "/services/dictionary/mode");
    const select = $("#dictionary-mode");
    select.replaceChildren(...data.modes.map((m) => new Option(m, m, false, m === data.mode)));
    select.onchange = () => fetch("/services/dictionary/mode", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ mode: select.value }),
    });
  } catch { /* services are optional */ }
}

async function refreshFeed() {
  try {
    const { data } = await api("GET", "/services/feed");
    $("#feed").replaceChildren(...data.events.map((e) => {
      const li = document.createElement("li");
      li.textContent = `${new Date(e.at).toLocaleTimeString()} ${e.message}`;
      return li;
    }));
    if (data.events.length === 0) $("#feed").innerHTML = "<li>Nothing yet</li>";
  } catch { /* services are optional */ }
}

// ---------- Start ----------

$("#temporal-ui").href = temporalUIBase();
loadPuzzles();
loadServices();
refreshPlayer();
refreshLeaderboard();
refreshFeed();
if (ui.gameId) refreshGame();

setInterval(tickTimer, 250);
setInterval(() => {
  if (ui.game && (ui.game.status === "playing" || ui.game.status === "loading")) refreshGame();
}, 1500);
setInterval(() => { if (ui.player) refreshPlayer(); }, 4000);
setInterval(refreshLeaderboard, 5000);
setInterval(refreshFeed, 3000);
