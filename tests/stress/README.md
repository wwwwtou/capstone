# Stress / Load Testing

Two complementary load tests target `GET /api/v1/recommendations` through the
gateway (gateway → recommendation → user/content → Postgres/Redis):

| Artifact | Tool | Role |
|---|---|---|
| `recommend.jmx` | Apache JMeter | Primary load test for **evidence** (HTML dashboard + screenshots) |
| `recommend.load.mjs` | Node (global `fetch`) | Lightweight **CI gate** — fails the pipeline if error rate > 5% |

## Prerequisite: the stack must be up

```bash
cd tiktok-glocal-ecommerce-recsys-mvp && docker compose up -d
# gateway is on http://localhost:8080
```

(If host 8080 is taken, start it on another port, e.g.
`GATEWAY_HOST_PORT=18080 docker compose up -d`, and pass that port below.)

## JMeter (evidence)

### Option A — GUI (quickest screenshot)
1. Open JMeter, **File → Open** `tests/stress/recommend.jmx`.
2. Add a listener for the numbers: right-click *Recommendation Users* →
   **Add → Listener → Summary Report** (and/or *Aggregate Report*).
3. Click the green **Start** button.
4. Screenshot the Summary/Aggregate Report — it shows samples, throughput
   (req/s), average/P90/P99, and error %.

### Option B — Headless + HTML dashboard (best evidence)
Run non-GUI and generate a full HTML report you can screenshot:

```bash
# from repo root; adjust the jmeter path to your install
jmeter -n -t tests/stress/recommend.jmx \
  -Jhost=localhost -Jport=8080 -Jthreads=50 -Jrampup=20 -Jduration=60 \
  -l tests/stress/jmeter-results.jtl \
  -e -o tests/stress/jmeter-report
```

Then open `tests/stress/jmeter-report/index.html` in a browser and screenshot
the **Statistics** table + the **Response Times Over Time** / **Throughput**
charts. `jmeter-results.jtl` and `jmeter-report/` are git-ignored (raw run
output), so commit only the screenshots / a summary into `RESULTS.md`.

Overridable properties (all have defaults): `host` (localhost), `port` (8080),
`threads` (50), `rampup` (20s), `duration` (60s).

## Node CI gate (also runnable locally)

```bash
BASE=http://localhost:8080 TOTAL=2000 CONCURRENCY=50 node tests/stress/recommend.load.mjs
```

Prints a throughput + latency-percentile JSON summary and appends a row to
`RESULTS.md`. Exits non-zero if more than 5% of requests error. This is what CI
job *5) Microservice Integration Tests* runs (with `TOTAL=300 CONCURRENCY=20`).
