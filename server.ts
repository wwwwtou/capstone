import express from "express";
import path from "path";
import { randomBytes } from "node:crypto";
import { createServer as createViteServer } from "vite";

// ============================================================================
// BFF / edge server. Two modes:
//   - PROXY (GATEWAY_URL set): forwards /api/v1/* to the Go gateway (full stack)
//   - MOCK  (GATEWAY_URL unset): a faithful in-memory replica of the microservice
//     behavior (catalog + interactions + profile-driven ranking), used for the
//     single-service online deploy and deterministic tests.
// In BOTH modes the BFF measures its own traffic (metrics registry below) and
// hosts a demo traffic generator (continuous load + one-click burst).
// ============================================================================

// ---------------------------------------------------------------------------
// Metrics registry — same shape as the Go services' /metricsz snapshot.
// ---------------------------------------------------------------------------
const LATENCY_BUCKETS_MS = [1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500];

class MetricsRegistry {
  service: string;
  start = Date.now();
  requestsTotal = 0;
  statusClass: Record<string, number> = {};
  routes: Record<string, number> = {};
  sumMs = 0;
  maxMs = 0;
  bucketCounts = new Array(LATENCY_BUCKETS_MS.length + 1).fill(0);
  counters: Record<string, number> = {};
  gauges: Record<string, () => string> = {};

  constructor(service: string) {
    this.service = service;
  }

  observe(route: string, status: number, durMs: number) {
    this.requestsTotal++;
    const cls = `${Math.floor(status / 100)}xx`;
    this.statusClass[cls] = (this.statusClass[cls] || 0) + 1;
    const key = Object.keys(this.routes).length < 50 || this.routes[route] !== undefined ? route : "other";
    this.routes[key] = (this.routes[key] || 0) + 1;
    this.sumMs += durMs;
    if (durMs > this.maxMs) this.maxMs = durMs;
    let idx = LATENCY_BUCKETS_MS.findIndex((ub) => durMs <= ub);
    if (idx === -1) idx = LATENCY_BUCKETS_MS.length;
    this.bucketCounts[idx]++;
  }

  inc(name: string, by = 1) {
    this.counters[name] = (this.counters[name] || 0) + by;
  }

  quantile(q: number): number {
    const total = this.bucketCounts.reduce((a: number, b: number) => a + b, 0);
    if (total === 0) return 0;
    const rank = q * total;
    let cum = 0;
    for (let i = 0; i < this.bucketCounts.length; i++) {
      const c = this.bucketCounts[i];
      cum += c;
      if (cum >= rank) {
        const lower = i > 0 ? LATENCY_BUCKETS_MS[i - 1] : 0;
        const upper = i < LATENCY_BUCKETS_MS.length ? LATENCY_BUCKETS_MS[i] : this.maxMs;
        if (c === 0) return upper;
        const within = rank - (cum - c);
        return lower + (upper - lower) * Math.min(1, within / c);
      }
    }
    return this.maxMs;
  }

  snapshot() {
    const r2 = (x: number) => Math.round(x * 100) / 100;
    const gauges: Record<string, string> = {};
    for (const [k, fn] of Object.entries(this.gauges)) gauges[k] = fn();
    return {
      service: this.service,
      uptime_s: r2((Date.now() - this.start) / 1000),
      requests_total: this.requestsTotal,
      status: { ...this.statusClass },
      latency_ms: {
        avg: r2(this.requestsTotal ? this.sumMs / this.requestsTotal : 0),
        p50: r2(this.quantile(0.5)),
        p90: r2(this.quantile(0.9)),
        p99: r2(this.quantile(0.99)),
        max: r2(this.maxMs),
      },
      routes: { ...this.routes },
      counters: { ...this.counters },
      gauges,
    };
  }
}

// Collapse dynamic id segments so route labels stay low-cardinality.
function routeLabel(method: string, p: string): string {
  const segs = p.replace(/^\/+|\/+$/g, "").split("/");
  const idLike = /^(user_\w+|u\d+|\d+|[0-9a-fA-F-]{8,})$/;
  return `${method} /${segs.map((s) => (idLike.test(s) ? "{id}" : s)).join("/")}`;
}

async function startServer() {
  const app = express();
  const PORT = process.env.PORT ? Number(process.env.PORT) : 3000;
  const SELF = `http://127.0.0.1:${PORT}`;

  app.use(express.json());

  const edgeMetrics = new MetricsRegistry("edge-bff");

  // Request-id + metrics middleware for every API call (both modes).
  app.use(/^\/(api|internal)\/.*/, (req, res, next) => {
    let id = req.headers["x-request-id"] as string | undefined;
    if (!id) {
      id = randomBytes(8).toString("hex");
      req.headers["x-request-id"] = id;
    }
    res.set("X-Request-ID", id);
    const start = process.hrtime.bigint();
    res.on("finish", () => {
      const durMs = Number(process.hrtime.bigint() - start) / 1e6;
      edgeMetrics.observe(routeLabel(req.method, req.path), res.statusCode, durMs);
    });
    next();
  });

  // -------------------------------------------------------------------------
  // Demo traffic generator (both modes). Fires realistic mixed traffic at the
  // BFF itself, so mock mode exercises the in-memory engine and proxy mode
  // exercises the full gateway -> microservices -> DB/Redis chain.
  // -------------------------------------------------------------------------
  const PERSONAS = ["user_123", "user_fashion", "user_foodie", "user_new"];
  const CATEGORIES = ["electronics", "tech", "fashion", "home", "food", "travel", "fitness"];
  const traffic = {
    enabled: false,
    rps: 5,
    sent: 0,
    errors: 0,
    started_at: null as string | null,
    timer: null as ReturnType<typeof setInterval> | null,
  };

  async function fireOne() {
    const user = PERSONAS[Math.floor(Math.random() * PERSONAS.length)];
    const roll = Math.random();
    try {
      if (roll < 0.6) {
        await fetch(`${SELF}/api/v1/recommendations?user_id=${user}`);
      } else if (roll < 0.8) {
        const category = CATEGORIES[Math.floor(Math.random() * CATEGORIES.length)];
        await fetch(`${SELF}/api/v1/users/${user}/interactions`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            event_type: Math.random() < 0.5 ? "view" : "like",
            metadata: { category, source: "traffic-generator" },
          }),
        });
      } else if (roll < 0.9) {
        await fetch(`${SELF}/api/v1/users/${user}/profile`);
      } else {
        await fetch(`${SELF}/api/v1/configs`);
      }
      traffic.sent++;
    } catch {
      traffic.errors++;
    }
  }

  function applyTrafficState(enabled: boolean, rps: number) {
    traffic.rps = Math.max(1, Math.min(50, Math.round(rps || traffic.rps)));
    if (traffic.timer) {
      clearInterval(traffic.timer);
      traffic.timer = null;
    }
    traffic.enabled = enabled;
    if (enabled) {
      if (!traffic.started_at) traffic.started_at = new Date().toISOString();
      // 4 ticks/second, each firing a slice of the target rate.
      const perTick = Math.max(1, Math.round(traffic.rps / 4));
      traffic.timer = setInterval(() => {
        for (let i = 0; i < perTick; i++) void fireOne();
      }, 250);
      // Never keep the process alive just for the generator.
      traffic.timer.unref?.();
    } else {
      traffic.started_at = null;
    }
  }

  app.get("/api/v1/simulator/traffic", (_req, res) => {
    res.json({
      code: 200,
      data: {
        enabled: traffic.enabled,
        rps: traffic.rps,
        sent: traffic.sent,
        errors: traffic.errors,
        started_at: traffic.started_at,
      },
    });
  });

  app.post("/api/v1/simulator/traffic", (req, res) => {
    const { enabled, rps } = req.body ?? {};
    applyTrafficState(Boolean(enabled), Number(rps) || traffic.rps);
    res.json({
      code: 200,
      message: traffic.enabled ? `continuous traffic ON @ ~${traffic.rps} rps` : "continuous traffic OFF",
      data: { enabled: traffic.enabled, rps: traffic.rps, sent: traffic.sent, errors: traffic.errors },
    });
  });

  // One-click load burst: N recommendation requests at fixed concurrency,
  // measured client-side so the response doubles as a mini load-test report.
  app.post("/api/v1/simulator/burst", async (req, res) => {
    const count = Math.max(10, Math.min(2000, Number(req.body?.count) || 300));
    const concurrency = Math.max(1, Math.min(100, Number(req.body?.concurrency) || 25));
    const durations: number[] = [];
    let errors = 0;
    let next = 0;
    const t0 = Date.now();

    async function worker() {
      while (next < count) {
        next++;
        const user = PERSONAS[Math.floor(Math.random() * PERSONAS.length)];
        const s = process.hrtime.bigint();
        try {
          const r = await fetch(`${SELF}/api/v1/recommendations?user_id=${user}`);
          if (!r.ok) errors++;
        } catch {
          errors++;
        }
        durations.push(Number(process.hrtime.bigint() - s) / 1e6);
      }
    }
    await Promise.all(Array.from({ length: concurrency }, worker));

    const elapsedMs = Date.now() - t0;
    durations.sort((a, b) => a - b);
    const pick = (q: number) => durations[Math.min(durations.length - 1, Math.floor(q * durations.length))] ?? 0;
    const r2 = (x: number) => Math.round(x * 100) / 100;
    res.json({
      code: 200,
      message: "burst complete",
      data: {
        requests: count,
        concurrency,
        errors,
        duration_ms: elapsedMs,
        achieved_rps: r2(count / (elapsedMs / 1000)),
        latency_ms: {
          avg: r2(durations.reduce((a, b) => a + b, 0) / (durations.length || 1)),
          p50: r2(pick(0.5)),
          p99: r2(pick(0.99)),
          max: r2(durations[durations.length - 1] ?? 0),
        },
      },
    });
  });

  // --- API layer: proxy to the real microservice gateway, or serve in-memory mocks ---
  const GATEWAY_URL = process.env.GATEWAY_URL;

  if (GATEWAY_URL) {
    // BFF convenience route: the consumer feed reads the user profile through
    // the public API; internally it lives on the user service's internal route.
    app.get("/api/v1/users/:id/profile", async (req, res) => {
      try {
        const upstream = await fetch(`${GATEWAY_URL}/internal/users/${encodeURIComponent(req.params.id)}/profile`, {
          headers: { "X-Request-ID": String(req.headers["x-request-id"] || "") },
        });
        res.status(upstream.status);
        res.set("content-type", upstream.headers.get("content-type") || "application/json");
        res.send(await upstream.text());
      } catch (err: any) {
        res.status(502).json({ code: 502, message: "Gateway unreachable: " + err.message });
      }
    });

    app.all(/^\/api\/v1\/.*/, async (req, res) => {
      try {
        const target = GATEWAY_URL + req.originalUrl;
        const headers: Record<string, string> = {};
        for (const [k, v] of Object.entries(req.headers)) {
          if (["host", "content-length", "connection"].includes(k)) continue;
          if (typeof v === "string") headers[k] = v;
        }
        const init: any = { method: req.method, headers };
        if (!["GET", "HEAD"].includes(req.method)) {
          headers["content-type"] = "application/json";
          init.body = JSON.stringify(req.body ?? {});
        }
        const upstream = await fetch(target, init);
        const body = await upstream.text();
        res.status(upstream.status);
        res.set("content-type", upstream.headers.get("content-type") || "application/json");
        res.send(body);
      } catch (err: any) {
        res.status(502).json({ code: 502, message: "Gateway unreachable: " + err.message });
      }
    });
    console.log("API mode: PROXY ->", GATEWAY_URL);
  } else {
    console.log("API mode: MOCK (in-memory recommendation engine)");

    // -----------------------------------------------------------------------
    // In-memory replica of the microservice stack. Mirrors postgres/init.sh
    // seed data and the Go ranking strategies so the closed loop
    // (interaction -> profile -> ranking) works on the single-service deploy.
    // -----------------------------------------------------------------------
    const now = Date.now();
    const hours = (h: number) => new Date(now - h * 3600_000).toISOString();
    const CATALOG = [
      { video_id: "v1", author: "StyleHouse", category: "fashion", title: "Autumn Streetwear Lookbook", created_at: hours(1) },
      { video_id: "v2", author: "TechMaster", category: "electronics", title: "Wireless Earbuds Deep Dive", created_at: hours(2) },
      { video_id: "v3", author: "HomeNest", category: "home", title: "Minimalist Ceramic Vase", created_at: hours(3) },
      { video_id: "v4", author: "FoodieIntl", category: "food", title: "Jakarta Street Food Tour", created_at: hours(4) },
      { video_id: "v5", author: "GadgetGuru", category: "tech", title: "Top Tech Gadgets 2026", created_at: hours(5) },
      { video_id: "v6", author: "FitLife", category: "fitness", title: "10-Minute Home Workout", created_at: hours(6) },
      { video_id: "v7", author: "TechMaster", category: "electronics", title: "Mechanical Keyboard Review", created_at: hours(7) },
      { video_id: "v8", author: "Wanderer", category: "travel", title: "Hidden Beaches of Bali", created_at: hours(8) },
      { video_id: "v9", author: "StyleHouse", category: "fashion", title: "Capsule Wardrobe Basics", created_at: hours(9) },
      { video_id: "v10", author: "GadgetGuru", category: "tech", title: "AI Phones Compared", created_at: hours(10) },
    ];

    type Interaction = { user_id: string; event_type: string; metadata: Record<string, any>; created_at: string };
    const interactions: Interaction[] = [
      { user_id: "user_123", event_type: "view", metadata: { category: "electronics" }, created_at: hours(24) },
      { user_id: "user_123", event_type: "like", metadata: { category: "electronics" }, created_at: hours(23) },
      { user_id: "user_123", event_type: "view", metadata: { category: "tech" }, created_at: hours(22) },
      { user_id: "user_123", event_type: "view", metadata: { category: "tech" }, created_at: hours(21) },
      { user_id: "user_fashion", event_type: "view", metadata: { category: "fashion" }, created_at: hours(24) },
      { user_id: "user_fashion", event_type: "like", metadata: { category: "fashion" }, created_at: hours(23) },
      { user_id: "user_fashion", event_type: "view", metadata: { category: "home" }, created_at: hours(22) },
      { user_id: "user_foodie", event_type: "view", metadata: { category: "food" }, created_at: hours(24) },
      { user_id: "user_foodie", event_type: "like", metadata: { category: "food" }, created_at: hours(23) },
      { user_id: "user_foodie", event_type: "view", metadata: { category: "travel" }, created_at: hours(22) },
    ];

    // Per-logical-service registries so /api/v1/metrics mirrors the gateway's
    // aggregate shape. They record the real work the mock engine performs.
    const svcMetrics = {
      user: new MetricsRegistry("user"),
      content: new MetricsRegistry("content"),
      recommendation: new MetricsRegistry("recommendation"),
    };
    for (const m of Object.values(svcMetrics)) {
      m.gauges["mode"] = () => "mock";
    }
    svcMetrics.recommendation.gauges["breaker_user"] = () => "closed";
    svcMetrics.recommendation.gauges["breaker_content"] = () => "closed";

    const timed = <T>(m: MetricsRegistry, route: string, fn: () => T): T => {
      const s = process.hrtime.bigint();
      const out = fn();
      m.observe(route, 200, Number(process.hrtime.bigint() - s) / 1e6);
      return out;
    };

    // Profile cache with TTL, mirroring the Redis profile:{user_id} cache.
    const PROFILE_TTL_MS = 60_000;
    const profileCache = new Map<string, { expires: number; profile: { user_id: string; tags: Record<string, number> } }>();

    function buildProfile(userId: string) {
      const cached = profileCache.get(userId);
      if (cached && cached.expires > Date.now()) {
        svcMetrics.user.inc("cache_hits");
        return cached.profile;
      }
      svcMetrics.user.inc("cache_misses");
      const tags: Record<string, number> = {};
      const recent = interactions.filter((it) => it.user_id === userId).slice(-50);
      for (const it of recent) {
        const cat = it.metadata?.category;
        if (typeof cat === "string" && cat) tags[cat] = (tags[cat] || 0) + 1;
      }
      const profile = { user_id: userId, tags };
      profileCache.set(userId, { expires: Date.now() + PROFILE_TTL_MS, profile });
      return profile;
    }

    // Ranking strategies — line-for-line mirrors of the Go domain strategies.
    function rankEngagement(tags: Record<string, number>) {
      const scored = CATALOG.map((v) => {
        const match = tags[v.category] || 0;
        const conf = Math.min(0.99, 0.6 + 0.08 * match);
        return {
          ...v,
          score: Math.round(conf * 100) / 100,
          reason: match > 0 ? `interest_match:${v.category}` : "globally_trending",
          _match: match,
        };
      });
      scored.sort((a, b) => b._match - a._match); // Array.sort is stable
      return scored.map(({ _match, ...v }) => v);
    }

    function rankChronological() {
      const out = [...CATALOG].sort((a, b) => b.created_at.localeCompare(a.created_at));
      return out.map((v, i) => ({
        ...v,
        score: Math.round(Math.max(0.3, 0.95 - 0.05 * i) * 100) / 100,
        reason: "recency",
      }));
    }

    let algorithmConfig = {
      strategy_name: "engagement",
      weight: 0.85,
      is_active: true,
      updated_at: new Date().toISOString(),
    };
    let configHistory: Array<{ strategy_name: string; weight: number; updated_at: string }> = [];

    const authMiddleware = (req: any, res: any, next: any) => {
      const authHeader = req.headers.authorization;
      if (authHeader && authHeader.startsWith("Bearer ")) {
        next();
      } else {
        res.status(401).json({ code: 401, message: "Unauthorized: Bearer token required" });
      }
    };

    app.post("/api/v1/login", (req, res) => {
      res.json({
        code: 200,
        message: "success",
        data: {
          token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.dummy_admin_payload",
          expires_in: 3600,
        },
      });
    });

    app.get("/api/v1/recommendations", (req, res) => {
      const userId = String(req.query.user_id || "guest");
      const profile = timed(svcMetrics.user, "GET /internal/users/{id}/profile", () => buildProfile(userId));
      timed(svcMetrics.content, "GET /internal/content/candidates", () => CATALOG.length);
      const videos = timed(svcMetrics.recommendation, "GET /api/v1/recommendations", () =>
        algorithmConfig.strategy_name === "chronological" ? rankChronological() : rankEngagement(profile.tags)
      );
      res.json({
        trace_id: String(req.headers["x-request-id"] || "trace-" + Date.now()),
        code: 200,
        message: "success",
        data: {
          user_id: userId,
          strategy: algorithmConfig.strategy_name,
          degraded: false,
          videos,
        },
      });
    });

    app.post("/api/v1/users/:id/interactions", (req, res) => {
      const { event_type, metadata } = req.body ?? {};
      if (!event_type) {
        return res.status(400).json({ code: 400, message: "event_type is required" });
      }
      timed(svcMetrics.user, "POST /api/v1/users/{id}/interactions", () => {
        interactions.push({
          user_id: req.params.id,
          event_type,
          metadata: metadata ?? {},
          created_at: new Date().toISOString(),
        });
        if (interactions.length > 2000) interactions.splice(0, interactions.length - 2000);
        profileCache.delete(req.params.id); // cache invalidation, like the Redis DEL
      });
      res.status(204).end();
    });

    app.get("/api/v1/users/:id/profile", (req, res) => {
      const profile = timed(svcMetrics.user, "GET /internal/users/{id}/profile", () => buildProfile(req.params.id));
      res.json(profile);
    });

    app.get("/api/v1/configs", (req, res) => {
      res.json({ code: 200, message: "success", data: algorithmConfig });
    });

    app.put("/api/v1/configs", authMiddleware, (req, res) => {
      const { strategy_name, weight } = req.body;
      if (!strategy_name || weight === undefined) {
        return res.status(400).json({ code: 400, message: "Invalid payload" });
      }
      algorithmConfig = { ...algorithmConfig, strategy_name, weight, updated_at: new Date().toISOString() };
      configHistory.unshift({ strategy_name, weight, updated_at: algorithmConfig.updated_at });
      res.json({
        code: 200,
        message: "Configuration deployed to Ranking Shards successfully",
        data: algorithmConfig,
      });
    });

    app.get("/api/v1/configs/history", (req, res) => {
      res.json({ code: 200, message: "success", data: configHistory });
    });

    app.get("/api/v1/health", (req, res) => {
      const uptime = Math.max(1, (Date.now() - edgeMetrics.start) / 1000);
      res.json({
        status: "healthy",
        instances: {
          rec_service_go: "UP",
          user_service: "UP",
          content_service: "UP",
          dashboard_fe: "UP",
          redis_shards: 1,
          postgres_primary: "ACTIVE",
        },
        metrics: {
          throughput_rps: Math.round(edgeMetrics.requestsTotal / uptime),
          avg_p99_latency_ms: Math.round(edgeMetrics.quantile(0.99)),
        },
      });
    });

    // Aggregate metrics feed, same shape the Go gateway serves in proxy mode.
    app.get("/api/v1/metrics", (req, res) => {
      res.json({
        mode: "mock",
        ts: Date.now(),
        gateway: edgeMetrics.snapshot(),
        services: {
          user: svcMetrics.user.snapshot(),
          content: svcMetrics.content.snapshot(),
          recommendation: svcMetrics.recommendation.snapshot(),
        },
      });
    });
  } // end mock mode

  // --- Vite / SPA Handling ---
  if (process.env.NODE_ENV !== "production") {
    const vite = await createViteServer({
      server: { middlewareMode: true },
      appType: "spa",
    });
    app.use(vite.middlewares);
  } else {
    const distPath = path.join(process.cwd(), "dist");
    app.use(express.static(distPath));
    app.get("*", (req, res) => {
      res.sendFile(path.join(distPath, "index.html"));
    });
  }

  app.listen(PORT, "0.0.0.0", () => {
    console.log(`Demo environment running on http://localhost:${PORT}`);
    console.log(`API endpoints simulated on /api/v1/*`);
  });
}

startServer();
