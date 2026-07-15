# E-commerce Video Recommendation Platform

A TikTok-style e-commerce video recommendation system built as a set of Go
microservices behind an API gateway, with a React dashboard covering both the
operations side and the consumer experience.

## What it does

- **Consumer side** — a vertical "For You" video feed. Views and likes are
  posted as interaction events, update the user's interest profile, and change
  the ranking on the next refresh: a complete
  interaction → profile → ranking closed loop.
- **Operations side** — system health and topology, algorithm configuration
  (ranking strategy hot-swap with a persisted deployment log), a recommendation
  simulator, and a live monitoring center (throughput, latency percentiles,
  error rate, cache hit rate, circuit-breaker states) with a built-in demo
  traffic generator.

## Architecture

- **API Gateway (Go)** — routing/reverse proxy, JWT auth (HS256), per-IP rate
  limiting, per-downstream circuit breakers, health and metrics aggregation.
- **Domain services (Go)** — recommendation (clean/onion architecture,
  Strategy + Factory ranking, retry with backoff, cold-start graceful
  degradation), user profile (interactions + Redis-cached profile aggregation),
  content (video catalog).
- **Data** — PostgreSQL with database-per-service (`user_db`, `content_db`,
  `rec_db`) and Redis as the profile cache.
- **Edge/BFF (Node + Express)** — serves the SPA, proxies `/api/v1/*` to the
  gateway in full-stack mode, or runs a faithful in-memory replica of the whole
  stack for the single-service cloud deploy; hosts the demo traffic generator.
- **Observability** — every service exposes Prometheus-format `/metrics` and
  JSON `/metricsz`; one `X-Request-ID` traces a request across all services.

Full technical detail, diagrams included: [`docs/TECHNICAL_DOSSIER.md`](docs/TECHNICAL_DOSSIER.md).

## Tech Stack

Go · React 19 + Vite + TypeScript · Express (BFF) · PostgreSQL 15 · Redis 7 ·
Docker Compose · GitHub Actions · Terraform (AWS option) · Playwright · JMeter

## Run

```bash
# Demo mode (in-memory engine, no Docker needed)
npm install
npm run dev                # http://localhost:3000

# Full stack (gateway + 3 services + Postgres + Redis)
cd tiktok-glocal-ecommerce-recsys-mvp && docker compose up -d
GATEWAY_URL=http://localhost:8080 npm run dev
```

## Tests

```bash
npm run test:smoke          # BFF API contract, mock mode (18 checks)
npm run test:e2e            # Playwright UI flows (10 tests)
BASE=http://localhost:8080 npm run test:integration   # real stack (22 checks)
npm run test:stress         # load gate; JMeter plan in tests/stress/
# unit tests (66 cases): cd tiktok-glocal-ecommerce-recsys-mvp/services/<svc> && go test ./...
```

CI runs all of the above on every push (7 jobs: static analysis & security
scans, unit/smoke/integration/E2E/load tests, gated deploy).
