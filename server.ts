import express from "express";
import path from "path";
import { createServer as createViteServer } from "vite";

async function startServer() {
  const app = express();
  const PORT = process.env.PORT ? Number(process.env.PORT) : 3000;

  app.use(express.json());

  // --- API layer: proxy to the real microservice gateway, or serve in-memory mocks ---
  // Local full-stack: set GATEWAY_URL (e.g. http://localhost:8090) and every /api/v1/*
  // request is forwarded to the gateway sitting in front of the Go microservices.
  // Online single-service deploy (Render): GATEWAY_URL is unset, so the lightweight
  // in-memory mocks below are served instead.
  const GATEWAY_URL = process.env.GATEWAY_URL;

  if (GATEWAY_URL) {
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
  console.log("API mode: MOCK (in-memory)");

  const JWT_SECRET = "defense-secret-2026";
  
  // Mock Database in memory
  let algorithmConfig = {
    strategy_name: "engagement",
    weight: 0.85,
    is_active: true,
    updated_at: new Date().toISOString()
  };

  // In-memory deployment-log history (mirrors the DB-backed history used in full-stack mode).
  let configHistory: Array<{ strategy_name: string; weight: number; updated_at: string }> = [];

  // Auth Middleware Simulation
  const authMiddleware = (req: any, res: any, next: any) => {
    const authHeader = req.headers.authorization;
    if (authHeader && authHeader.startsWith('Bearer ')) {
      // In real Go backend, we verify JWT here. Mocking success for demo.
      next();
    } else {
      res.status(401).json({ code: 401, message: "Unauthorized: Bearer token required" });
    }
  };

  // Login (returns mock token)
  app.post("/api/v1/login", (req, res) => {
    res.json({
        code: 200,
        message: "success",
        data: {
            token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.dummy_admin_payload",
            expires_in: 3600
        }
    });
  });

  // GET Recommendations
  app.get("/api/v1/recommendations", (req, res) => {
    const userId = req.query.user_id || "guest";
    res.json({
      trace_id: "trace-" + Date.now(),
      code: 200,
      message: "success",
      data: {
        user_id: userId,
        videos: [
          { video_id: "v_01", title: "Top Tech 2026", score: 0.98, reason: "interest_match_tech", author: "TechMaster" },
          { video_id: "v_03", title: "Home Workout", score: 0.85, reason: "engagement_hot", author: "FitnessGuru" },
          { video_id: "v_02", title: "Jakarta Street Food", score: 0.72, reason: "globally_trending", author: "FoodieIntl" }
        ]
      }
    });
  });

  // GET Config
  app.get("/api/v1/configs", (req, res) => {
    res.json({
        code: 200,
        message: "success",
        data: algorithmConfig
    });
  });

  // PUT Config (Protected by JWT)
  app.put("/api/v1/configs", authMiddleware, (req, res) => {
    const { strategy_name, weight } = req.body;
    if (!strategy_name || weight === undefined) {
      return res.status(400).json({ code: 400, message: "Invalid payload" });
    }

    algorithmConfig = {
      ...algorithmConfig,
      strategy_name,
      weight,
      updated_at: new Date().toISOString()
    };
    configHistory.unshift({ strategy_name, weight, updated_at: algorithmConfig.updated_at });

    res.json({
      code: 200,
      message: "Configuration deployed to Ranking Shards successfully",
      data: algorithmConfig
    });
  });

  // Deployment-log history
  app.get("/api/v1/configs/history", (req, res) => {
    res.json({ code: 200, message: "success", data: configHistory });
  });

  // System Health
  app.get("/api/v1/health", (req, res) => {
    res.json({
        status: "healthy",
        instances: {
            rec_service_go: "UP",
            dashboard_fe: "UP",
            redis_shards: 3,
            postgres_primary: "ACTIVE"
        },
        metrics: {
            throughput_rps: 1250,
            avg_p99_latency_ms: 32
        }
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
