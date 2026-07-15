// Integration test against the live microservice stack (gateway + user +
// content + recommendation + Postgres + Redis).
//
// Prereq: stack is up, e.g.
//   cd tiktok-glocal-ecommerce-recsys-mvp && docker compose up -d
// Run: BASE=http://localhost:8080 node tests/integration/gateway.integration.mjs
const BASE = process.env.BASE || "http://localhost:8080";

let passed = 0, failed = 0;
function check(name, cond) {
  if (cond) { passed++; console.log(`  PASS  ${name}`); }
  else { failed++; console.error(`  FAIL  ${name}`); }
}
const j = async (res) => res.json();

try {
  // --- Health aggregation reflects real downstream services ---
  const health = await j(await fetch(`${BASE}/api/v1/health`));
  check("health: user_service UP", health.instances?.user_service === "UP");
  check("health: content_service UP", health.instances?.content_service === "UP");
  check("health: rec_service UP", health.instances?.rec_service_go === "UP");
  check("health: postgres ACTIVE", health.instances?.postgres_primary === "ACTIVE");

  // --- Auth: writes require a JWT (check 401 before we have a token) ---
  const noAuth = await fetch(`${BASE}/api/v1/configs`, {
    method: "PUT", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ strategy_name: "chronological", weight: 0.5 }),
  });
  check("configs PUT without token -> 401", noAuth.status === 401);

  const token = (await j(await fetch(`${BASE}/api/v1/login`, { method: "POST" }))).data?.token;
  check("login: token issued", typeof token === "string" && token.length > 0);

  // Establish a known strategy first — the DB is persistent, so prior runs
  // could have left any strategy active. This keeps the test deterministic.
  await fetch(`${BASE}/api/v1/configs`, {
    method: "PUT", headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({ strategy_name: "engagement", weight: 0.85 }),
  });

  // --- Per-user personalization (DB-backed profiles drive ranking) ---
  const recs123 = await j(await fetch(`${BASE}/api/v1/recommendations?user_id=user_123`));
  const recsFashion = await j(await fetch(`${BASE}/api/v1/recommendations?user_id=user_fashion`));
  const top123 = recs123.data?.videos?.[0];
  const topFashion = recsFashion.data?.videos?.[0];
  check("recs: user_123 top is electronics/tech", ["electronics", "tech"].includes(top123?.category));
  check("recs: user_fashion top is fashion", topFashion?.category === "fashion");
  check("recs: different users get different top video", top123?.video_id !== topFashion?.video_id);
  check("recs: matched videos carry interest_match reason", String(top123?.reason || "").startsWith("interest_match"));

  const put = await fetch(`${BASE}/api/v1/configs`, {
    method: "PUT", headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({ strategy_name: "chronological", weight: 0.5 }),
  });
  check("configs PUT with token -> 200", put.status === 200);

  // --- Strategy change persists and changes ranking behavior ---
  const cfg = await j(await fetch(`${BASE}/api/v1/configs`));
  check("configs persisted as chronological", cfg.data?.strategy_name === "chronological");

  const recsChrono = await j(await fetch(`${BASE}/api/v1/recommendations?user_id=user_123`));
  check("recs: chronological reason", recsChrono.data?.videos?.[0]?.reason === "recency");

  const history = await j(await fetch(`${BASE}/api/v1/configs/history`));
  check("configs/history: persisted entries exist", Array.isArray(history.data) && history.data.length >= 1);

  // --- Interactions API through the gateway ---
  const like = await fetch(`${BASE}/api/v1/users/user_123/interactions`, {
    method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ event_type: "like", metadata: { category: "electronics" } }),
  });
  check("interactions POST via gateway -> 204", like.status === 204);

  // --- Request tracing: one X-Request-ID across gateway -> rec service ---
  const traced = await fetch(`${BASE}/api/v1/recommendations?user_id=user_123`, {
    headers: { "X-Request-ID": "it-trace-42" },
  });
  const tracedBody = await j(traced);
  check("trace: gateway echoes X-Request-ID", traced.headers.get("x-request-id") === "it-trace-42");
  check("trace: rec service trace_id matches", tracedBody?.trace_id === "it-trace-42");

  // --- Observability: Prometheus endpoint + aggregated metrics ---
  const promText = await (await fetch(`${BASE}/metrics`)).text();
  check("prometheus: gateway exposes http_requests_total", promText.includes("http_requests_total"));

  // Warm the profile cache (first hit misses, second hits Redis).
  await fetch(`${BASE}/internal/users/user_123/profile`);
  await fetch(`${BASE}/internal/users/user_123/profile`);

  const metrics = await j(await fetch(`${BASE}/api/v1/metrics`));
  check("metrics: gateway snapshot present", typeof metrics?.gateway?.requests_total === "number");
  check("metrics: downstream snapshots aggregated",
    !!metrics?.services?.user && !!metrics?.services?.content && !!metrics?.services?.recommendation);
  check("metrics: breakers closed under healthy stack",
    metrics?.gateway?.gauges?.breaker_user === "closed" &&
    metrics?.services?.recommendation?.gauges?.breaker_content === "closed");
  check("metrics: redis cache hits counted",
    (metrics?.services?.user?.counters?.cache_hits ?? 0) >= 1);

  // restore default
  await fetch(`${BASE}/api/v1/configs`, {
    method: "PUT", headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({ strategy_name: "engagement", weight: 0.85 }),
  });

  console.log(`\nIntegration: ${passed} passed, ${failed} failed`);
  process.exit(failed === 0 ? 0 : 1);
} catch (err) {
  console.error("Integration test error:", err.message);
  process.exit(1);
}
