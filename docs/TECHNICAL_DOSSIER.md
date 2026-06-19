# Technical Dossier — E-commerce Video Recommendation System

> Single source of technical detail for the project report / defense PPT.
> Everything below reflects the **actual code** in this repository (verified via
> tests + CI, not aspirational). Diagrams live in [`docs/architecture/`](architecture/).
> Last updated: 2026-06-19 (HEAD `935c277`).

---

## 1. Executive Summary

A microservices recommendation platform: a React admin dashboard talks to a Go
API gateway that fronts three Go domain services (recommendation, user, content)
backed by per-service PostgreSQL databases and a Redis profile cache. The system
demonstrates DDD/clean architecture, fault tolerance (circuit breaker + retry +
rate limiting), a full four-dimension test suite, and an automated GitHub Actions
CI/CD pipeline that deploys to Render.

- **Languages:** Go (services), TypeScript/React (frontend + BFF), SQL.
- **Infra:** Docker Compose (local/self-host), Render (cloud), Terraform/AWS (IaC option).
- **Repo layout:** frontend at repo root (`src/`, `server.ts`); microservices under
  `tiktok-glocal-ecommerce-recsys-mvp/`.

---

## 2. Architecture

Authoritative PlantUML diagrams (source `.puml` + rendered `png/`) in
`docs/architecture/`:

| Diagram | File |
|---|---|
| Logical architecture (layered) | `logical-architecture.puml` |
| Physical architecture (docker-compose) | `physical-architecture.puml` |
| Cloud deployment (Render + AWS IaC) | `deployment-cloud.puml` |
| DDD context map | `ddd-context-map.puml` |
| ER diagram (detailed, per-service DBs) | `er-diagram.puml` |
| Core sequence: GET /recommendations | `sequence-get-recommendations.puml` |
| Clean/onion architecture (recommendation) | `recommendation-clean-architecture.puml` |
| CI/CD pipeline | `cicd-pipeline.puml` |
| Use case diagram | `usecase.puml` |

### 2.1 Layers (logical)

1. **Presentation** — React SPA (Vite): `DashboardHome`, `AlgoConfig`, `Simulator`
   (`src/pages/*`, `src/App.tsx`).
2. **Edge / BFF** — Node `server.ts` (Express). Dual-mode: serves an in-memory
   **mock** API (online single-service / demo) or **reverse-proxies** `/api/v1/*`
   to the gateway when `GATEWAY_URL` is set (full-stack).
3. **API Gateway (Go)** — `services/gateway`: routing/reverse-proxy, JWT auth,
   per-IP rate limiting, per-downstream circuit breaker, health aggregation.
4. **Domain services (Go)** — recommendation (core), user, content.
5. **Data** — PostgreSQL (database-per-service) + Redis (profile cache).

### 2.2 Microservices

| Service | Port | Responsibility | Key endpoints |
|---|---|---|---|
| gateway | 8080 | Edge: auth, rate limit, breaker, health, routing | `/api/v1/login`, `/api/v1/health`, proxies `/api/v1/*` |
| recommendation | 8083 | Core ranking; config; deployment history | `GET /api/v1/recommendations`, `GET/PUT /api/v1/configs`, `GET /api/v1/configs/history` |
| user | 8081 | Interactions + profile aggregation | `POST /api/v1/users/{id}/interactions`, `GET /internal/users/{id}/profile` |
| content | 8082 | Video catalog / candidates | `GET /internal/content/candidates` |

---

## 3. Technology Stack

- **Backend:** Go 1.20+ (`net/http`, `gorilla/mux`, `lib/pq`), standard-library
  JWT (HS256), standard-library resilience primitives (no heavy frameworks).
- **Frontend:** React 19 + Vite 6 + TypeScript, Tailwind, Framer Motion, axios.
- **BFF:** Express on Node (run with `tsx`).
- **Data:** PostgreSQL 15, Redis 7.
- **Testing:** Go `testing`, Node scripts, Playwright (E2E), Apache JMeter (load).
- **CI/CD:** GitHub Actions; Docker; Render deploy hook.
- **IaC:** Terraform (AWS EC2 + Docker bootstrap).

---

## 4. Domain Model & Database Design

**Database-per-service** (no cross-service DB foreign keys; integration is HTTP-only).
Schema defined in `tiktok-glocal-ecommerce-recsys-mvp/postgres/init.sh`. See
`docs/architecture/er-diagram.puml` for the full ER diagram.

### user_db (User Profile Service)
- `interactions(id PK, user_id, event_type, metadata jsonb, created_at)` — index `idx_interactions_user(user_id)`.
- `profiles(user_id PK, tags jsonb)`.

### content_db (Content Service)
- `videos(id PK, video_id UNIQUE, author, category, title, created_at)` — index `idx_videos_category(category)`.

### rec_db (Recommendation Service)
- `configs(id PK, key UNIQUE, value jsonb)` — holds the `active_strategy` config.
- `config_history(id PK, strategy_name, weight, created_at)` — persisted deployment log.

### Redis (NoSQL, key-value)
- Key `profile:{user_id}` → JSON `{user_id, tags:{category:count}}`, TTL 10 min.
  Write-through cache of the derived profile; invalidated (DEL) on new interaction.

**SQL injection defense:** all queries are parameterized (`$1, $2`).

---

## 5. Design Patterns, DDD & Clean Architecture (in code)

- **DDD bounded contexts** → microservices: Recommendation (core), User Profile
  (supporting), Content (supporting), Dashboard/Monitoring (generic). Context map:
  `docs/architecture/ddd-context-map.puml`.
- **Clean / onion architecture** in the core recommendation service
  (`services/recommendation/internal/`), dependencies pointing inward:
  - `domain/` — entities, `RankingStrategy` (Strategy pattern) + `StrategyFactory`
    (Factory pattern), repository **ports** (`ProfileRepository`,
    `ContentRepository`, `ConfigRepository`). No external dependencies.
  - `app/` — `Service` use cases, depend only on domain ports → unit-testable with fakes.
  - `infra/` — adapters implementing the ports (HTTP repos with breaker+retry,
    Postgres config repo, resilience).
  - `transport/` — HTTP handlers. `main.go` is the composition root.
  - Diagram: `docs/architecture/recommendation-clean-architecture.puml`.
- **Strategy + Factory** — `domain/strategy.go` (`EngagementStrategy`,
  `ChronologicalStrategy`, `StrategyFactory`).
- **Reverse Proxy / API Gateway** — `services/gateway`.
- **Cache-aside / write-through** — Redis profile cache in the user service.
- **Repository / Ports & Adapters** — the `domain` interfaces above.

---

## 6. Fault Tolerance & Resilience (Technical Added Value)

All implemented with the standard library only.

| Mechanism | Where | Behavior |
|---|---|---|
| **Circuit breaker** (3-state: closed→open→half-open) | recommendation (`infra/resilience.go`) per downstream; gateway (`resilience.go`) per service via a `breakerTransport` wrapping the reverse-proxy | Trips after N consecutive failures (or a 5xx for the gateway), fails fast while open, half-open probe to recover. |
| **Retry with exponential backoff** | recommendation `callResilient` around user/content calls | 1×, 2×, 4× backoff; stops immediately if the breaker is open. |
| **Graceful degradation (cold-start fallback)** | recommendation `app/service.go` | A failed **user-profile** fetch falls back to an empty profile served as `globally_trending`, with `degraded:true`, HTTP 200 — instead of failing. Content is essential → 503 on outage. |
| **Rate limiting** | gateway `ratelimit.go` | Per-IP token bucket → HTTP 429 + `Retry-After`. Configurable via `RATE_LIMIT_RPS` / `RATE_LIMIT_BURST` (defaults 1000/2000). |

**Verified by live fault injection:** with the stack up, `docker compose stop user`
→ recommendations still return **HTTP 200 with `degraded:true`** (cold-start
fallback); after `docker compose start user` the breaker recovers (half-open →
closed) and personalized results resume. Unit tests cover the breaker state
machine, retry fail-fast, breaker transport, rate limiter, and the degraded use case.

---

## 7. Security

| Control | Status | Detail |
|---|---|---|
| JWT authentication (HS256) | ✅ | Standard-library signing in the gateway (`services/gateway/main.go`); `PUT /api/v1/configs` requires a valid Bearer token (401 otherwise). |
| SQL injection defense | ✅ | All DB access uses parameterized queries. |
| Dependency scanning | ✅ | `gosec`, `golangci-lint`, `govulncheck` (Go, advisory) + `npm audit --audit-level=high` (frontend, hard gate, currently 0 vulns). |
| RBAC | 🟡 | Binary admin/non-admin via JWT presence; no role hierarchy. |
| JWT secret management | ✅ | Read from `JWT_SECRET` env; **no hardcoded secret in source** — generates an ephemeral random secret (with a warning) if unset. Production sets `JWT_SECRET` for stable, multi-instance tokens. |
| Rate limiting / abuse cap | ✅ | Per-IP token bucket at the gateway (see §6). |
| XSS | 🟡 | React escapes by default; no explicit CSP / security headers. |
| Sensitive-data encryption at rest | ❌ | Not implemented (no sensitive PII stored in this MVP). |

---

## 8. Scalability

- **Stateless services** — all Go services are stateless; user state lives in
  Postgres/Redis, so services scale horizontally behind a load balancer.
- **Database-per-service** — independent data ownership, no shared schema.
- **Cache** — Redis offloads profile reads.
- **Documented horizontal-scaling path** — ALB/ECS in the cloud deployment
  diagram; current MVP runs single instances (no autoscaling configured yet).

---

## 9. Testing — Four Dimensions

| Dimension | Count | Location | Run | CI job / evidence |
|---|---|---|---|---|
| **Unit** | 57 | `services/*/**/*_test.go` (recommendation 35, gateway 11, user 9, content 2) | `cd services/<svc> && go test ./...` | Job 3 → artifact `unit-test-report.md` |
| **Integration** | 14 | `tests/integration/gateway.integration.mjs` | `BASE=http://localhost:8080 npm run test:integration` (stack up) | Job 5 (spins up real stack) |
| **E2E** | 4 | `tests/e2e/admin-flows.spec.ts` (Playwright) | `npm run test:e2e` → `npx playwright show-report` | Job 6 → artifact `playwright-report/` |
| **Load / Stress** | — | `tests/stress/recommend.jmx` (JMeter) + `recommend.load.mjs` (Node CI gate) | JMeter: see `tests/stress/README.md`; Node: `npm run test:stress` | Job 5 runs the Node gate; JMeter report `tests/stress/jmeter-report/index.html` |
| (Smoke) | 9 | `tests/smoke/server.smoke.mjs` | `npm run test:smoke` | Job 4 |

**Unit-test highlight (DDD payoff):** the recommendation use cases are tested
with **fake repositories** (`internal/app/service_test.go`) — no DB/HTTP needed —
covering the happy path, the degraded cold-start fallback, and the
content-failure error path.

**Load-test results (recorded in `tests/stress/RESULTS.md`):**
- **JMeter** (50 threads, 30s, 80 ms think time, full chain via gateway):
  **12,841 samples, 0 errors, ~430 req/s, p99 ≈ 38 ms**. HTML dashboard at
  `tests/stress/jmeter-report/index.html`.
- **Node peak-throughput run** (2,000 req @ concurrency 50, no think time):
  **~1,803 req/s, 0 errors, p99 ≈ 100 ms**.

---

## 10. CI/CD Pipeline

GitHub Actions `.github/workflows/ci.yml` — 7 jobs (diagram:
`docs/architecture/cicd-pipeline.puml`):

1. **Go Quality & Security** — `go vet` (hard gate) across all `services/*/go.mod`;
   gosec / golangci-lint / govulncheck (advisory report artifact).
2. **Frontend Dependency Security** — `npm audit --audit-level=high` (hard gate).
3. **Unit Tests - Report Artifact** — `go test -v` across all services →
   machine-readable `unit-test-report.md` (total/passed/failed/pass-rate).
4. **Lint & Build Checks** — `tsc --noEmit`, `vite build`, API smoke test, docker build.
5. **Microservice Integration Tests** — `docker compose up --build` → integration
   (14) + Node load gate → teardown.
6. **End-to-End Tests (Playwright)** — install Chromium, run the UI E2E suite,
   upload `playwright-report`.
7. **Deploy to Render** — on `main`, after all jobs green; triggers the Render
   deploy hook (skips cleanly if `DEPLOY_HOOK_URL` secret is unset).

---

## 11. Deployment

- **Local / self-host:** `cd tiktok-glocal-ecommerce-recsys-mvp && docker compose up -d`
  (gateway + 3 services + Postgres + Redis); frontend `GATEWAY_URL=http://localhost:8080 npm run dev`.
  Diagram: `physical-architecture.puml`.
- **Cloud (current):** Render single web service from the root `Dockerfile`,
  running `server.ts` in mock mode (`render.yaml`, auto-deploy on push to main).
- **AWS (IaC option):** `terraform/main.tf` provisions an EC2 instance that
  bootstraps Docker + the compose stack. Diagram: `deployment-cloud.puml`.

---

## 12. Requirements Compliance Matrix

Legend: ✅ met (in code) · 🟡 partial · ❌ not done · ➖ N/A

| # | Requirement | Status | Evidence |
|---|---|---|---|
| 1 | Logical / physical / deployment diagrams | ✅ | `docs/architecture/{logical,physical,deployment-cloud}.puml` |
| 1 | Microservices + DDD bounded contexts (in code structure) | ✅ | 4 services, database-per-service; clean arch in recommendation; `ddd-context-map.puml` |
| 1 | Multi-release / decoupling | 🟡 | HTTP-decoupled services, `/api/v1` versioning; no multi-version coexistence |
| 2 | Class & sequence design | ✅ | `recommendation-clean-architecture.puml`, `sequence-get-recommendations.puml` |
| 2 | FE/BE/communication code flow | ✅ | React → BFF → gateway → services → DB/Redis |
| 2 | Reusable components + design patterns in code | ✅ | Strategy+Factory, Repository/Ports, Gateway, Cache-aside |
| 3 | Relational schema + indexes | ✅ | `postgres/init.sh`; indexes on user_id, category |
| 3 | NoSQL structure | ✅ | Redis key-value profile cache |
| 3 | Data lake / pipeline | ➖ | Not applicable to this MVP |
| 4 | CI/CD pipeline in repo, with build/scan/test/deploy | ✅ | `.github/workflows/ci.yml` (7 jobs) |
| 4 | SonarQube specifically | 🟡 | Equivalent static analysis (gosec/golangci/govulncheck/npm audit); SonarQube not wired |
| 5 | Unit testing | ✅ | 57 cases, all services |
| 5 | Integration testing | ✅ | 14 cases, real stack in CI |
| 5 | End-to-end testing | ✅ | 4 Playwright flows |
| 5 | Stress / performance testing (JMeter) | ✅ | `recommend.jmx`; results in `RESULTS.md` |
| 6 | JWT auth, SQL-injection defense | ✅ | Gateway JWT; parameterized queries |
| 6 | Encryption at rest / advanced security | 🟡 | Rate limit ✅; JWT secret env-only ✅; encryption-at-rest ❌ (no PII stored) |
| 6 | Horizontal scaling design | 🟡 | Stateless services ✅; LB/autoscaling documented but not configured |
| 7 | Technical added value (≥1) | ✅ | **Fault tolerance**: circuit breaker, retry, rate limiting, graceful degradation |

---

## 13. Known Gaps / Future Work

1. **SonarQube** integration (current static analysis is equivalent but not Sonar).
2. **Security headers / CSP**, and **encryption at rest** if real PII is introduced.
3. **Horizontal scaling** — add load balancer + autoscaling config (currently single instances).
4. **Apply clean-architecture layering** to the user/content/gateway services
   (currently flatter than the recommendation service).
5. **Note:** `@google/genai` is a declared dependency but is **not used** in code —
   there is no LLM integration; the technical added value is the fault-tolerance layer.

---

## 14. Quick Reference

**Ports:** gateway 8080 · user 8081 · content 8082 · recommendation 8083 ·
Postgres 5432 · Redis 6379 · frontend dev 3000 (E2E uses 3101).

**Common commands:**
```bash
# Full stack
cd tiktok-glocal-ecommerce-recsys-mvp && docker compose up -d
GATEWAY_URL=http://localhost:8080 npm run dev        # frontend on :3000

# Tests
go test ./...                                        # per service dir
npm run test:smoke
BASE=http://localhost:8080 npm run test:integration
npm run test:e2e                                     # Playwright
npm run test:stress                                  # Node load gate
# JMeter: see tests/stress/README.md

# Render PlantUML diagrams
java -jar <plantuml.jar> -tpng -charset UTF-8 -graphvizdot <dot> -o png docs/architecture/*.puml
```

**Key source paths:**
- Frontend: `src/`, `server.ts`
- Services: `tiktok-glocal-ecommerce-recsys-mvp/services/{gateway,recommendation,user,content}`
- Recommendation layers: `services/recommendation/internal/{domain,app,infra,transport}`
- Schema: `tiktok-glocal-ecommerce-recsys-mvp/postgres/init.sh`
- CI: `.github/workflows/ci.yml`
- Diagrams: `docs/architecture/`
- Test suites: `tests/{smoke,integration,e2e,stress}`
