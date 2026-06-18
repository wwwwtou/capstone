// Lightweight load/stress test (no external dependency — uses global fetch).
// Fires a fixed number of requests at a fixed concurrency and reports
// throughput + latency percentiles, then appends the run to RESULTS.md.
//
// Prereq: target is reachable (gateway or the :3000 proxy).
// Run: BASE=http://localhost:8080 TOTAL=2000 CONCURRENCY=50 node tests/stress/recommend.load.mjs
import { appendFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const BASE = process.env.BASE || "http://localhost:8080";
const PATH = process.env.TARGET_PATH || "/api/v1/recommendations?user_id=user_123";
const TOTAL = parseInt(process.env.TOTAL || "2000", 10);
const CONCURRENCY = parseInt(process.env.CONCURRENCY || "50", 10);
const url = BASE + PATH;

function percentile(sorted, p) {
  if (sorted.length === 0) return 0;
  const idx = Math.min(sorted.length - 1, Math.floor((p / 100) * sorted.length));
  return sorted[idx];
}

let issued = 0;
let ok = 0;
let errors = 0;
const latencies = [];

async function worker() {
  while (issued < TOTAL) {
    issued++;
    const t0 = performance.now();
    try {
      const res = await fetch(url);
      await res.text();
      const dt = performance.now() - t0;
      latencies.push(dt);
      if (res.ok) ok++; else errors++;
    } catch {
      errors++;
    }
  }
}

console.log(`Load test: ${TOTAL} requests @ concurrency ${CONCURRENCY} -> ${url}`);

// warmup
try { await fetch(url); } catch {}

const start = performance.now();
await Promise.all(Array.from({ length: CONCURRENCY }, () => worker()));
const wallMs = performance.now() - start;

latencies.sort((a, b) => a - b);
const sum = latencies.reduce((a, b) => a + b, 0);
const rps = (ok / (wallMs / 1000));

const report = {
  target: url,
  total: TOTAL,
  concurrency: CONCURRENCY,
  ok,
  errors,
  wall_seconds: +(wallMs / 1000).toFixed(2),
  throughput_rps: +rps.toFixed(1),
  latency_ms: {
    min: +(latencies[0] || 0).toFixed(1),
    avg: +(sum / (latencies.length || 1)).toFixed(1),
    p50: +percentile(latencies, 50).toFixed(1),
    p90: +percentile(latencies, 90).toFixed(1),
    p99: +percentile(latencies, 99).toFixed(1),
    max: +(latencies[latencies.length - 1] || 0).toFixed(1),
  },
};

console.log(JSON.stringify(report, null, 2));

const stamp = process.env.RUN_STAMP || new Date().toISOString();
const resultsPath = resolve(dirname(fileURLToPath(import.meta.url)), "RESULTS.md");
const row = `\n## Run ${stamp}\n\n` +
  `- Target: \`${report.target}\`\n` +
  `- Load: ${report.total} requests @ concurrency ${report.concurrency}\n` +
  `- OK / Errors: ${report.ok} / ${report.errors}\n` +
  `- Wall time: ${report.wall_seconds}s\n` +
  `- Throughput: **${report.throughput_rps} req/s**\n` +
  `- Latency ms — min ${report.latency_ms.min}, avg ${report.latency_ms.avg}, ` +
  `p50 ${report.latency_ms.p50}, p90 ${report.latency_ms.p90}, p99 ${report.latency_ms.p99}, max ${report.latency_ms.max}\n`;
appendFileSync(resultsPath, row);
console.log(`\nAppended results to ${resultsPath}`);

process.exit(errors > TOTAL * 0.05 ? 1 : 0); // fail if >5% errors
