// Smoke test for the deployed artifact: boots server.ts in MOCK mode (no
// GATEWAY_URL) and asserts the public API. This is exactly what runs on the
// single-service online deploy, so CI tests what it ships.
//
// Run: node tests/smoke/server.smoke.mjs
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const PORT = process.env.SMOKE_PORT || "3100";
const base = `http://localhost:${PORT}`;

let passed = 0;
let failed = 0;
function check(name, cond) {
  if (cond) { passed++; console.log(`  PASS  ${name}`); }
  else { failed++; console.error(`  FAIL  ${name}`); }
}

async function waitFor(url, timeoutMs = 30000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const r = await fetch(url);
      if (r.ok) return true;
    } catch {}
    await new Promise((r) => setTimeout(r, 500));
  }
  return false;
}

const env = { ...process.env, PORT, NODE_ENV: "development" };
delete env.GATEWAY_URL; // force mock mode

const server = spawn(process.execPath, ["--import", "tsx", "server.ts"], {
  cwd: repoRoot,
  env,
  stdio: ["ignore", "inherit", "inherit"],
});

let exitCode = 1;
try {
  const up = await waitFor(`${base}/api/v1/health`);
  if (!up) throw new Error("server did not become healthy in time");

  const health = await (await fetch(`${base}/api/v1/health`)).json();
  check("health: status healthy", health.status === "healthy");
  check("health: has instances", !!health.instances);
  check("health: has metrics", typeof health.metrics?.throughput_rps === "number");

  const login = await (await fetch(`${base}/api/v1/login`, { method: "POST" })).json();
  const token = login?.data?.token;
  check("login: returns token", typeof token === "string" && token.length > 0);

  const recs = await (await fetch(`${base}/api/v1/recommendations?user_id=u1`)).json();
  check("recommendations: returns videos", Array.isArray(recs?.data?.videos) && recs.data.videos.length > 0);

  const cfg = await (await fetch(`${base}/api/v1/configs`)).json();
  check("configs GET: has strategy_name", typeof cfg?.data?.strategy_name === "string");

  const noAuth = await fetch(`${base}/api/v1/configs`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ strategy_name: "chronological", weight: 0.5 }),
  });
  check("configs PUT without token -> 401", noAuth.status === 401);

  const withAuth = await fetch(`${base}/api/v1/configs`, {
    method: "PUT",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({ strategy_name: "chronological", weight: 0.5 }),
  });
  check("configs PUT with token -> 200", withAuth.status === 200);

  const history = await (await fetch(`${base}/api/v1/configs/history`)).json();
  check("configs/history: records the change", Array.isArray(history?.data) && history.data.length >= 1);

  console.log(`\nSmoke: ${passed} passed, ${failed} failed`);
  exitCode = failed === 0 ? 0 : 1;
} catch (err) {
  console.error("Smoke test error:", err.message);
  exitCode = 1;
} finally {
  server.kill("SIGTERM");
  setTimeout(() => process.exit(exitCode), 300);
}
