import { spawn, execFileSync } from "node:child_process";
import { randomBytes } from "node:crypto";
import { mkdirSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
export const APP = "http://localhost:8080";
export const UI = "http://localhost:8233";
const NS = "default";

export const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const children = new Set();

// Stops every process this tool started, for crashes and Ctrl+C.
export function killAll() {
  for (const child of children) {
    try {
      process.kill(-child.pid, "SIGKILL");
    } catch {}
  }
}

function start(cmd, args, { env = {}, label, quiet = true } = {}) {
  const child = spawn(cmd, args, {
    cwd: ROOT,
    env: { ...process.env, ...env },
    detached: true,
    stdio: ["ignore", "pipe", "pipe"],
  });
  child.output = "";
  const collect = (chunk) => {
    child.output += chunk;
    if (!quiet) process.stdout.write(`[${label}] ${chunk}`);
  };
  child.stdout.on("data", collect);
  child.stderr.on("data", collect);
  child.label = label;
  children.add(child);
  child.on("exit", () => children.delete(child));
  return child;
}

export async function stop(child) {
  if (!child || child.exitCode !== null) return;
  try {
    process.kill(-child.pid, "SIGINT");
  } catch {}
  for (let i = 0; i < 50 && child.exitCode === null; i++) await sleep(100);
  try {
    process.kill(-child.pid, "SIGKILL");
  } catch {}
}

async function waitFor(check, what, timeoutMs = 60000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      if (await check()) return;
    } catch (err) {
      if (err.fatal) throw err;
    }
    await sleep(250);
  }
  throw new Error(`timed out waiting for ${what}`);
}

export function temporal(...args) {
  return execFileSync("temporal", args, { cwd: ROOT, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
}

// A fresh dev server, so every run starts from an empty Temporal.
export async function startTemporal() {
  const dir = mkdtempSync(join(tmpdir(), "wordflow-shots-"));
  const server = start("temporal", ["server", "start-dev", "--db-filename", join(dir, "t.db"), "--log-level", "error"], {
    label: "temporal",
  });
  server.dir = dir;
  await waitFor(() => temporal("operator", "cluster", "health").includes("SERVING"), "Temporal");
  return server;
}

export async function stopTemporal(server) {
  await stop(server);
  rmSync(server.dir, { recursive: true, force: true });
}

// WORKER_READY is a pattern the Worker prints once it is polling. The Go
// SDK logs "Started Worker".
const workerReady = new RegExp(process.env.WORKER_READY || "Started Worker", "i");

// Runs `make worker S=..` and `make server S=..` from the language branch.
export async function startLesson(s, env = {}, { worker: withWorker = true } = {}) {
  const worker = withWorker ? start("make", ["worker", `S=${s}`], { env, label: `worker ${s}` }) : null;
  const server = start("make", ["server", `S=${s}`], { env, label: `server ${s}` });
  await waitFor(async () => (await fetch(`${APP}/api/puzzles`)).ok, `server ${s}`, 120000);
  if (worker) {
    await waitFor(() => {
      if (worker.exitCode !== null) {
        throw Object.assign(new Error(`worker ${s} exited:\n${worker.output}`), { fatal: true });
      }
      return workerReady.test(worker.output);
    }, `worker ${s}`, 120000);
  }
  return { worker, server, s };
}

export async function stopLesson(lesson) {
  if (!lesson) return;
  await stop(lesson.worker);
  await stop(lesson.server);
  await sleep(500);
}

export function cleanWorkflows() {
  try {
    temporal("workflow", "terminate", "--query", 'ExecutionStatus="Running"', "--reason", "clean slate", "--yes");
  } catch {}
}

// ---- HTTP helpers --------------------------------------------------------

// The same shape of key the web app sends.
const newKey = () => randomBytes(6).toString("hex");

export async function api(method, path, body) {
  const res = await fetch(APP + path, {
    method,
    headers: { "Content-Type": "application/json", "Idempotency-Key": newKey() },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok && res.status !== 202) throw new Error(`${method} ${path} -> ${res.status} ${text}`);
  return data;
}

export async function setDictionaryMode(mode) {
  await fetch(`${APP}/services/dictionary/mode`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ mode }),
  });
}

// ---- Screenshots ---------------------------------------------------------

export function shotPath(lessonDir, name) {
  const dir = join(ROOT, "course", lessonDir, "img");
  mkdirSync(dir, { recursive: true });
  return join(dir, `${name}.png`);
}

// Screenshots a panel, cropped to its content: grid rows stretch panels
// taller than what they show.
export async function shotElement(page, selector, lessonDir, name) {
  const el = page.locator(selector).first();
  await el.scrollIntoViewIfNeeded();
  const clip = await el.evaluate((node) => {
    const box = node.getBoundingClientRect();
    let bottom = box.top;
    for (const child of node.querySelectorAll("*")) {
      const r = child.getBoundingClientRect();
      if (r.height && getComputedStyle(child).visibility !== "hidden") bottom = Math.max(bottom, r.bottom);
    }
    return {
      x: box.left + scrollX,
      y: box.top + scrollY,
      width: box.width,
      height: Math.min(box.height, bottom - box.top + 18),
    };
  });
  await page.screenshot({ path: shotPath(lessonDir, name), animations: "disabled", fullPage: true, clip });
  console.log(`  ${lessonDir}/img/${name}.png`);
}

export async function shotPage(page, lessonDir, name, opts = {}) {
  await page.screenshot({ path: shotPath(lessonDir, name), animations: "disabled", ...opts });
  console.log(`  ${lessonDir}/img/${name}.png`);
}

export function workflowURL(workflowID, runID = "") {
  const id = encodeURIComponent(workflowID);
  return runID
    ? `${UI}/namespaces/${NS}/workflows/${id}/${runID}/history`
    : `${UI}/namespaces/${NS}/workflows?query=${encodeURIComponent(`WorkflowId="${workflowID}"`)}`;
}

export function latestRunID(workflowID) {
  const out = JSON.parse(temporal("workflow", "describe", "--workflow-id", workflowID, "-o", "json"));
  return out.workflowExecutionInfo.execution.runId;
}
