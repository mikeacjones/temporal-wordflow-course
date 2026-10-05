// Captures the screenshots used in course/*/README.md.
//
// Run from a language branch, with nothing else on ports 7233, 8233, or 8080:
//   cd tools/screenshots && npm install && npx playwright install chromium
//   node capture.mjs            every scenario
//   node capture.mjs 05 extras  only these
//
// Each scenario starts the lesson's solution with `make worker S=NN` and
// `make server S=NN` against a fresh Temporal dev server.
import { readFileSync, rmSync } from "node:fs";
import { hostname } from "node:os";
import { join as pathJoin } from "node:path";
import { chromium } from "playwright";
import * as L from "./lib.mjs";

const puzzle = (id) => JSON.parse(readFileSync(pathJoin(L.ROOT, "puzzles", `${id}.json`), "utf8"));

let browser;

async function appPage() {
  const context = await browser.newContext({ viewport: { width: 1100, height: 900 }, deviceScaleFactor: 1.5 });
  const page = await context.newPage();
  await page.goto(L.APP);
  await page.waitForSelector("#puzzles .puzzle");
  return page;
}

async function uiPage() {
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, deviceScaleFactor: 1.5 });
  return context.newPage();
}

async function play(page, title) {
  await page.locator(".puzzle", { hasText: title }).locator("button").click();
  await page.waitForSelector("#game-panel:not([hidden])");
  await page.waitForFunction(() => document.querySelector("#letters").children.length > 0);
  await L.sleep(500);
}

async function guess(page, word) {
  await page.fill("#guess", word);
  await page.press("#guess", "Enter");
  await L.sleep(800);
}

async function join(page, name) {
  await page.fill("#join-name", name);
  await page.press("#join-name", "Enter");
  await page.waitForFunction((n) => document.querySelector("#player-panel").textContent.includes(n), name);
  await L.sleep(500);
}

// Runs TransferPointsWorkflow to completion. A failed transfer is expected.
function transfer(workflowID, input) {
  try {
    L.temporal("workflow", "execute", "--type", "TransferPointsWorkflow", "--task-queue", "wordflow",
      "--workflow-id", workflowID, "--input", JSON.stringify(input));
  } catch {}
}

const gameIdOf = (page) => page.evaluate(() => localStorage.getItem("gameId"));

// Plays a whole game through the API: start it, then find every word.
async function winGame(player, puzzleId) {
  const { gameId } = await L.api("POST", "/api/games", { puzzleId, player });
  for (const word of puzzle(puzzleId).words) {
    await L.api("POST", `/api/games/${gameId}/guesses`, { word });
  }
  return gameId;
}

// Opens a Workflow in the Temporal UI, oldest event first.
async function openWorkflow(page, workflowID, { runID, tab, view = "All" } = {}) {
  await page.goto(L.workflowURL(workflowID, runID || L.latestRunID(workflowID)));
  await page.getByText("Event History", { exact: false }).first().waitFor();
  const order = page.getByRole("button", { name: /Descending/ });
  if (await order.count()) await order.first().click();
  if (view) await page.getByRole("button", { name: view, exact: true }).first().click();
  if (tab) await page.getByText(tab, { exact: false }).first().click();
  await L.sleep(1500);
}

async function maskHost(page) {
  const host = hostname();
  await page.evaluate((host) => {
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    while (walker.nextNode()) {
      const node = walker.currentNode;
      if (node.nodeValue.includes(host)) node.nodeValue = node.nodeValue.replaceAll(host, "codespace");
    }
  }, host);
}

async function uiShot(page, dir, name, scrollTo) {
  if (scrollTo) await page.getByText(scrollTo, { exact: false }).last().scrollIntoViewIfNeeded();
  await maskHost(page);
  await page.mouse.move(0, 0);
  await L.sleep(300);
  await L.shotPage(page, dir, name);
}

// ---- Scenarios ------------------------------------------------------------

const scenarios = {
  async "00"() {
    const dir = "00-setup";
    const lesson = await L.startLesson("00", {}, { worker: false });
    try {
      const page = await appPage();
      await page.locator(".puzzle", { hasText: "Durable" }).locator("button").click();
      await page.waitForSelector("#puzzle-panel .lesson-note:not([hidden])");
      await L.shotPage(page, dir, "app-starter");
      const ui = await uiPage();
      await ui.goto(`${L.UI}/namespaces/default/workflows`);
      await L.sleep(2500);
      await uiShot(ui, dir, "temporal-ui-empty");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "01"() {
    const dir = "01-first-workflow";
    const lesson = await L.startLesson("01");
    try {
      const page = await appPage();
      await play(page, "Signal");
      await L.shotElement(page, "#game-panel", dir, "board");
      const ui = await uiPage();
      await openWorkflow(ui, await gameIdOf(page));
      await uiShot(ui, dir, "history");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "02"() {
    const dir = "02-activities";
    const lesson = await L.startLesson("02", { PUZZLES_FAIL_FIRST: "2" });
    try {
      const page = await appPage();
      await play(page, "Signal");
      const ui = await uiPage();
      await openWorkflow(ui, await gameIdOf(page));
      await ui.getByText("Activity Task Started").first().click();
      await L.sleep(800);
      await uiShot(ui, dir, "activity-retried");
      L.temporal("workflow", "start", "--type", "GameWorkflow", "--task-queue", "wordflow",
        "--workflow-id", "bad-puzzle", "--input", '{"puzzleId":"nope"}');
      await L.sleep(2000);
      await openWorkflow(ui, "bad-puzzle");
      await uiShot(ui, dir, "non-retryable");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "03"() {
    const dir = "03-signals-and-queries";
    const lesson = await L.startLesson("03");
    try {
      const page = await appPage();
      await play(page, "Timers");
      await guess(page, "merit");
      await guess(page, "term");
      await guess(page, "zz");
      await L.shotElement(page, "#game-panel", dir, "signals-board");
      const ui = await uiPage();
      await openWorkflow(ui, await gameIdOf(page));
      await uiShot(ui, dir, "signals-history", "Workflow Execution Signaled");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "04"() {
    const dir = "04-updates";
    const lesson = await L.startLesson("04");
    try {
      const page = await appPage();
      await play(page, "Timers");
      await guess(page, "merit");
      await L.shotElement(page, "#game-panel", dir, "guess-found");
      await guess(page, "zz");
      await L.shotElement(page, "#game-panel", dir, "guess-rejected");
      const ui = await uiPage();
      await openWorkflow(ui, await gameIdOf(page));
      await uiShot(ui, dir, "updates-history");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "05"() {
    const dir = "05-timers";
    const lesson = await L.startLesson("05");
    try {
      const page = await appPage();
      await play(page, "Timers");
      await page.click("#extend");
      await L.sleep(1000);
      await L.shotElement(page, "#game-panel", dir, "extended");
      const ui = await uiPage();
      await openWorkflow(ui, await gameIdOf(page));
      await uiShot(ui, dir, "timer-history", "Timer Canceled");
      await openWorkflow(ui, await gameIdOf(page), { tab: "Timeline" });
      await uiShot(ui, dir, "timer-timeline");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "06"() {
    const dir = "06-entity-workflows";
    const lesson = await L.startLesson("06");
    try {
      const page = await appPage();
      await join(page, "alice");
      await L.api("POST", "/api/players", { name: "alice" });
      await L.shotElement(page, "#player-panel", dir, "player");
      const ui = await uiPage();
      await openWorkflow(ui, "player-alice");
      await uiShot(ui, dir, "player-history");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "07"() {
    const dir = "07-child-workflows";
    const lesson = await L.startLesson("07");
    try {
      const page = await appPage();
      await join(page, "alice");
      await play(page, "Worker");
      await page.locator(".puzzle", { hasText: "Signal" }).locator("button").click();
      await page.waitForSelector("#puzzle-panel .error:not([hidden])");
      await L.shotElement(page, "#puzzle-panel", dir, "one-game-at-a-time");
      const ui = await uiPage();
      await openWorkflow(ui, "player-alice");
      await uiShot(ui, dir, "child-started");
      await openWorkflow(ui, "player-alice", { tab: "Relationships" });
      await uiShot(ui, dir, "relationships");
      for (const word of puzzle("worker").words) await guess(page, word);
      await L.sleep(4500);
      await L.shotElement(page, "#player-panel", dir, "player-after-game");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "08"() {
    const dir = "08-workflow-to-workflow";
    const lesson = await L.startLesson("08");
    try {
      const page = await appPage();
      await join(page, "alice");
      await play(page, "Replay");
      for (let i = 0; i < 3; i++) {
        await page.click("#hint");
        await L.sleep(1200);
      }
      await L.sleep(4500);
      const bottom = await page.evaluate(() => document.querySelector("#message").getBoundingClientRect().bottom + 16);
      await L.shotPage(page, dir, "paid-hint", { fullPage: true, clip: { x: 0, y: 0, width: 1100, height: bottom } });
      const ui = await uiPage();
      await openWorkflow(ui, await gameIdOf(page), { view: "Compact" });
      await uiShot(ui, dir, "spend-points-activity");
      await openWorkflow(ui, "player-alice");
      await uiShot(ui, dir, "spend-points-update", "Workflow Execution Update Completed");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "09"() {
    const dir = "09-signal-with-start";
    let lesson = await L.startLesson("08");
    try {
      await L.api("POST", "/api/players", { name: "alice" });
      await winGame("alice", "worker");
      await L.sleep(1500);
      await L.stopLesson(lesson);

      lesson = await L.startLesson("09");
      // Queries don't record failures. A new event makes the Worker replay
      // the player in a Workflow Task, which fails and shows in history.
      L.temporal("workflow", "signal", "--workflow-id", "player-alice", "--name", "nudge");
      await L.sleep(4000);
      const ui = await uiPage();
      await openWorkflow(ui, "player-alice");
      await uiShot(ui, dir, "nondeterminism", "Workflow Task Failed");
      L.cleanWorkflows();

      for (const name of ["alice", "bob"]) {
        await L.api("POST", "/api/players", { name });
        await winGame(name, name === "alice" ? "worker" : "signal");
      }
      await L.sleep(1500);
      const page = await appPage();
      await L.sleep(5500);
      await L.shotElement(page, "#leaderboard-panel", dir, "leaderboard");
      await openWorkflow(ui, "leaderboard");
      await uiShot(ui, dir, "leaderboard-history");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "10"() {
    const dir = "10-continue-as-new";
    const lesson = await L.startLesson("10");
    try {
      await L.api("POST", "/api/players", { name: "carol" });
      for (const id of ["worker", "signal", "durable", "timers", "replay", "worker"]) {
        await winGame("carol", id);
        await L.sleep(1500);
      }
      const ui = await uiPage();
      await ui.goto(L.workflowURL("player-carol"));
      await ui.getByText("Continued as New", { exact: false }).first().waitFor({ timeout: 15000 });
      await L.sleep(1000);
      await uiShot(ui, dir, "runs");
      await openWorkflow(ui, "player-carol");
      await uiShot(ui, dir, "new-run-input");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "11"() {
    const dir = "11-retries";
    const lesson = await L.startLesson("11");
    try {
      const page = await appPage();
      await play(page, "Timers");
      await guess(page, "timer");
      await guess(page, "remit");
      await L.shotElement(page, "#game-panel", dir, "bonus");
      await page.selectOption("#dictionary-mode", "down");
      await L.sleep(500);
      const gameId = await gameIdOf(page);
      const pending = L.api("POST", `/api/games/${gameId}/guesses`, { word: "miter" }).catch(() => {});
      await L.sleep(4000);
      const ui = await uiPage();
      await openWorkflow(ui, gameId, { tab: "Pending Activities" });
      await uiShot(ui, dir, "retrying");
      await pending;
      await L.setDictionaryMode("ok");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  // A game started on Lesson 13 code has a version marker that Lesson 12
  // code doesn't expect: a real nondeterminism error, in any language.
  async "12"() {
    const dir = "12-determinism-and-replay";
    let lesson = await L.startLesson("13");
    try {
      const { gameId } = await L.api("POST", "/api/games", { puzzleId: "timers" });
      await L.sleep(1000);
      await L.stopLesson(lesson);
      lesson = await L.startLesson("12");
      // Any new event makes the Lesson 12 Worker replay the game.
      L.temporal("workflow", "signal", "--workflow-id", gameId, "--name", "nudge");
      await L.sleep(4000);
      const ui = await uiPage();
      await openWorkflow(ui, gameId);
      await uiShot(ui, dir, "stuck-game", "Workflow Task Failed");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "13"() {
    const dir = "13-versioning";
    const lesson = await L.startLesson("13");
    try {
      const page = await appPage();
      await join(page, "bob");
      await play(page, "Timers");
      await L.sleep(3500);
      await L.shotElement(page, "#services-panel", dir, "feed");
      const ui = await uiPage();
      await openWorkflow(ui, await gameIdOf(page));
      await uiShot(ui, dir, "marker", "Marker Recorded");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async saga() {
    const lesson = await L.startLesson("extras");
    try {
      for (const name of ["alice", "bob"]) await L.api("POST", "/api/players", { name });
      transfer("gift-alice-bob", { from: "alice", to: "bob", amount: 5 });
      transfer("gift-alice-nobody", { from: "alice", to: "nobody", amount: 5 });
      const ui = await uiPage();
      await openWorkflow(ui, "gift-alice-nobody", { view: "Compact" });
      await uiShot(ui, "extras/saga", "compensation");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async "pick-first"() {
    const lesson = await L.startLesson("extras");
    try {
      const page = await appPage();
      await play(page, "Durable");
      for (const word of ["beard", "blare", "able"]) await guess(page, word);
      const ui = await uiPage();
      await openWorkflow(ui, await gameIdOf(page));
      await uiShot(ui, "extras/pick-first", "race", "Activity Task Cancel Requested");
    } finally {
      await L.stopLesson(lesson);
    }
  },

  async approval() {
    const lesson = await L.startLesson("extras");
    try {
      const submission = JSON.parse(readFileSync(pathJoin(L.ROOT, "submissions", "bread.json"), "utf8"));
      L.temporal("workflow", "start", "--type", "PuzzleSubmissionWorkflow", "--task-queue", "wordflow",
        "--workflow-id", "submission-bread", "--input", JSON.stringify({ puzzle: submission }));
      await L.sleep(2500);
      const ui = await uiPage();
      await openWorkflow(ui, "submission-bread");
      await ui.getByText("Event History", { exact: true }).last()
        .evaluate((n) => n.scrollIntoView({ block: "start" }));
      await uiShot(ui, "extras/approval", "parallel-lookups");
      L.temporal("workflow", "signal", "--workflow-id", "submission-bread", "--name", "review",
        "--input", '{"approved":true,"reviewer":"you"}');
      await L.sleep(2500);
      await openWorkflow(ui, "submission-bread", { view: "Compact" });
      await uiShot(ui, "extras/approval", "approved");
    } finally {
      rmSync(pathJoin(L.ROOT, "puzzles", "bread.json"), { force: true });
      await L.stopLesson(lesson);
    }
  },

  async schedules() {
    const lesson = await L.startLesson("extras");
    try {
      L.temporal("schedule", "create", "--schedule-id", "daily-puzzle", "--interval", "10s",
        "--workflow-id", "daily-puzzle-run", "--type", "DailyPuzzleWorkflow", "--task-queue", "wordflow");
      await L.sleep(32000);
      const ui = await uiPage();
      await ui.goto(`${L.UI}/namespaces/default/schedules/daily-puzzle`);
      await L.sleep(2500);
      await uiShot(ui, "extras/schedules", "schedule");
      const page = await appPage();
      await L.sleep(3500);
      await L.shotElement(page, "#services-panel", "extras/schedules", "feed");
    } finally {
      try {
        L.temporal("schedule", "delete", "--schedule-id", "daily-puzzle");
      } catch {}
      await L.stopLesson(lesson);
    }
  },
};

// ---- Main -------------------------------------------------------------------

for (const event of ["SIGINT", "SIGTERM", "uncaughtException", "unhandledRejection"]) {
  process.on(event, (err) => {
    if (err instanceof Error) console.error(err);
    L.killAll();
    process.exit(1);
  });
}

const wanted = process.argv.slice(2);
const names = wanted.length ? wanted : Object.keys(scenarios).sort();
const temporalServer = await L.startTemporal();
browser = await chromium.launch();
let failed = false;
try {
  for (const name of names) {
    console.log(`scenario ${name}`);
    try {
      await scenarios[name]();
    } catch (err) {
      failed = true;
      console.error(`  FAILED: ${err.message}`);
    }
    L.cleanWorkflows();
  }
} finally {
  await browser.close();
  await L.stopTemporal(temporalServer);
}
process.exit(failed ? 1 : 0);
