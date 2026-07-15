# TikTok Glocal Ecommerce Recommendation Architecture (Proxy MVP)

This project is a proxy MVP for a TikTok-style ecommerce recommendation system.

It demonstrates:
- a web dashboard with an admin side (health, algorithm config, simulator) and a
  consumer side (a TikTok-style "For You" feed that closes the
  interaction → profile → ranking loop)
- a microservices recommendation flow (Go gateway + recommendation/user/content
  services, database-per-service PostgreSQL, Redis profile cache)
- fault tolerance (circuit breakers, retry with backoff, rate limiting,
  graceful degradation) and observability (per-service Prometheus-format
  metrics, X-Request-ID tracing, a live monitoring page, a built-in demo
  traffic generator)
- Docker-based local and deployment workflows with a 7-job GitHub Actions
  CI/CD pipeline

Full technical detail: [`docs/TECHNICAL_DOSSIER.md`](docs/TECHNICAL_DOSSIER.md).

## Tech Stack

- Node.js + Vite + React (dashboard + BFF)
- Go microservices
- PostgreSQL
- Redis
- Docker

## Run Locally

```bash
# Demo mode (in-memory engine, no Docker needed)
npm install
npm run dev

# Full stack (microservices + Postgres + Redis)
cd tiktok-glocal-ecommerce-recsys-mvp && docker compose up -d
GATEWAY_URL=http://localhost:8080 npm run dev
```

## Tests

```bash
npm run test:smoke          # BFF mock-mode API contract (18 checks)
npm run test:e2e            # Playwright UI flows (10 tests)
BASE=http://localhost:8080 npm run test:integration   # real stack (22 checks)
npm run test:stress         # Node load gate; JMeter plan in tests/stress/
```

## Project Goal

The repository is designed for architecture demonstration, CI/CD practice, and deployment experiments.
