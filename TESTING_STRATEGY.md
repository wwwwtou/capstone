# Testing Strategy & Execution Proof

## 1. Unit Testing (Backend)
*   **Methodology:** Focus on the `RankingEngine` logic without DB dependencies. Use Go's built-in `testing` package.
*   **Proof of Concept:** See `backend-go/internal/application/service_test.go`.
*   **Goal:** 80%+ test coverage for Core Domain logic.

## 2. Integration Testing
*   **Methodology:** Use `testcontainers-go` or Docker Compose to spin up Redis/Postgres in ephemeral containers.
*   **Test Case:** Verify that a profile updated in Postgres correctly reflects in the Recommendation response.

## 3. Stress Testing (Performance Defense)
*   **Tool:** `k6` or `Vegeta`.
*   **Scenario:** 1000 Concurrent Users fetching recommendations simultaneously.
*   **Expectations:**
    *   P99 Latency < 50ms.
    *   Throughput > 500 RPS on a single microservice instance.

## 4. Security Testing
*   **Static Analysis:** Use `gosec` for Go and `npm audit` for React to find vulnerabilities in dependencies.
*   **RBAC Proof:** Attempt to post new weights without an Admin JWT; verify `403 Forbidden` response.
