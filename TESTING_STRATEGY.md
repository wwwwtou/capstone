# Testing Strategy & Execution Proof

## 1. Unit Testing (Backend)
*   **Methodology:** Test pure logic without DB dependencies, using Go's built-in `testing` package with table-driven subtests.
*   **Scope (all four Go microservices):**
    *   `services/recommendation` — ranking strategies (engagement/chronological), scoring, config, strategy factory, circuit breaker/retry, use cases with fake repositories, request-id propagation on outbound calls.
    *   `services/gateway` — JWT issue/validate, tampered/expired/malformed token rejection, auth middleware, rate limiter, breaker transport, metrics registry (counters/quantiles/Prometheus format), X-Request-ID middleware.
    *   `services/user` — `categoryFromMetadata` profile-tag aggregation + malformed-body rejection.
    *   `services/content` — health contract + `Video` JSON field contract.
*   **Proof:** CI job *3) Unit Tests - Report Artifact* runs `go test -v` across every `services/*/go.mod`, builds `unit-test-report.md` (total / passed / failed / pass-rate) and uploads it as an artifact. Currently **66** cases (incl. subtests).
*   **Run locally:** `cd tiktok-glocal-ecommerce-recsys-mvp/services/<svc> && go test -v ./...`

## 2. Integration Testing
*   **Methodology:** Docker Compose spins up the full stack (gateway → recommendation → user/content → Postgres/Redis) in CI job *5) Microservice Integration Tests*.
*   **Test cases (22):** `tests/integration/gateway.integration.mjs` verifies cross-service behavior — per-user recommendation differentiation, config read/write persistence, JWT-gated PUT (401 without token), deployment-history persistence, health aggregation, interactions POST through the gateway, X-Request-ID tracing end to end, Prometheus `/metrics`, aggregated `/api/v1/metrics` (breaker states, Redis cache-hit counters).
*   **Run locally:** `cd tiktok-glocal-ecommerce-recsys-mvp && docker compose up -d`, then `BASE=http://localhost:8080 npm run test:integration`.

## 3. Stress Testing (Performance Defense)
*   **Tool:** **Apache JMeter** — plan at `tests/stress/recommend.jmx`. A lightweight Node gate (`tests/stress/recommend.load.mjs`) also runs inside CI so the pipeline enforces an error-rate threshold without needing a JMeter/Java install on the runner.
*   **Scenario:** ramp to 50 virtual users hammering `GET /api/v1/recommendations` across multiple user_ids; assert HTTP 200 + non-empty `data.videos`.
*   **How to run JMeter + capture evidence:** see `tests/stress/README.md`.
*   **Expectations:** error rate < 5%; throughput and P99 recorded in `tests/stress/RESULTS.md` (latest local node run: 1803 req/s, 0 errors, P99 ≈ 100ms).

## 4. End-to-End Testing (UI)
*   **Tool:** Playwright (Chromium) against the deterministic mock-mode server on a dedicated port (3101).
*   **Test cases (10):** `tests/e2e/admin-flows.spec.ts` (dashboard health, login, simulator ranking, config deploy + log), `tests/e2e/feed.spec.ts` (consumer feed render, like → profile update, re-rank), `tests/e2e/monitoring.spec.ts` (stat cards/charts/service table, one-click burst report, continuous-traffic toggle).
*   **Proof:** CI job *6) End-to-End Tests (Playwright)* uploads the `playwright-report` artifact. **Run locally:** `npm run test:e2e`.

## 5. Security Testing
*   **Static Analysis:** `gosec` / `golangci-lint` / `govulncheck` for Go (CI job 1, advisory report artifact) and `npm audit --audit-level=high` for the frontend (CI job 2, hard gate — currently 0 vulnerabilities).
*   **RBAC Proof:** PUT `/api/v1/configs` without an admin JWT returns **401 Unauthorized** (covered by the integration suite).
