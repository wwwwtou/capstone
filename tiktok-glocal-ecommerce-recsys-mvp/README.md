# Microservices Backend — E-commerce Video Recommendation Platform

The Go microservices monorepo behind the recommendation platform.

Contains:
- **API Gateway (Go)** — routing, JWT auth, per-IP rate limiting,
  per-downstream circuit breakers, health + metrics aggregation
- **User Service (Go)** — interactions + profile aggregation, PostgreSQL
  `user_db`, Redis profile cache (TTL + write-invalidation)
- **Content Service (Go)** — video catalog, PostgreSQL `content_db`
- **Recommendation Service (Go)** — clean/onion architecture
  (`internal/{domain,app,infra,transport}`), Strategy + Factory ranking,
  retry with backoff and cold-start graceful degradation, PostgreSQL `rec_db`
- `docker-compose.yml` — one Postgres instance (creates the three logical DBs,
  seeded by `postgres/init.sh`) plus Redis

Every service exposes `/metrics` (Prometheus text) and `/metricsz` (JSON);
the gateway aggregates them on `GET /api/v1/metrics`.

Quick run:

```bash
docker compose up -d --build
# host port 8080 busy? -> GATEWAY_HOST_PORT=18080 docker compose up -d gateway
```

API (through the gateway, default `:8080`):
- `POST /api/v1/login` · `GET /api/v1/health` · `GET /api/v1/metrics`
- `GET /api/v1/recommendations?user_id=...`
- `GET|PUT /api/v1/configs` (PUT requires a Bearer JWT) · `GET /api/v1/configs/history`
- `POST /api/v1/users/{id}/interactions` · `GET /internal/users/{id}/profile`
