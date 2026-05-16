# Project Management & Sprint Records (TikTok Glocal MVP)

## 1. Project Backlog (Product Level)

| ID | User Story | Priority | Estimate (Days) | Status |
| :--- | :--- | :--- | :--- | :--- |
| US01 | As an admin, I want to deploy new ranking weights via JWT. | Critical | 3 | Done |
| US02 | As a user, I want optimized video feeds based on my Glocal tags. | Critical | 5 | Done |
| US03 | As a dev, I want a SAST-enabled CI/CD to prevent security leaks. | High | 3 | Done |
| US04 | As a system, I want concurrent data fetching to lower P99 latencies. | High | 4 | Done |
| US05 | As an architect, I want IaC code to define cloud environments. | Medium | 2 | Done |

## 2. Sprint Effort Tracking (MTech Degree Evidence)

### Sprint 2: Architecture & Concurrency (Weeks 3-4)
*   **Focus:** Clean Architecture implementation, Goroutine synchronization.
*   **Actual Effort:** 28 SP (Complexity in Strategy Pattern integration).
*   **Velocity:** 14 SP/Week.

### Sprint 3: Security & DevOps (Weeks 5-6 - Current)
*   **Focus:** JWT Middleware, GitHub Actions (SAST), Terraform.
*   **Actual Effort (to date):** 22 SP.

## 3. Risks & Mitigations (Evidence for Point 15-17)

| Risk Type | Description | Impact | Mitigation Strategy |
| :--- | :--- | :--- | :--- |
| **Technical** | Golang GC pauses affecting tail latency. | Medium | Use sync.Pool for high-frequency objects; monitor memory limits in Docker. |
| **Security** | Insecure API access for Admin Dashboard. | High | Implement JWT with short expiry + RBAC middleware in Go. |
| **Management**| Scope creep of "Real-time" AI features. | Medium | Strictly follow MVP blueprint; push complex ML features to Phase 3. |
