# Architecture Decision Records (ADR) - E-commerce Video Recommendation Platform

## ADR-001: Choice of Language - Golang for Recommendation Service
*   **Context:** The recommendation engine requires high concurrency and low latency for scoring thousands of candidates.
*   **Decision:** Use Golang.
*   **Rationale:** Goroutines provide lightweight concurrency; Static typing ensures safety; Faster cold start vs Java JRE.
*   **Consequences:** Requires manual handling of some DDD boilerplate, but provides superior performance for the ranking stage.

## ADR-002: Clean Architecture Pattern
*   **Context:** Need to keep business logic (ranking/scoring) independent of infrastructure (DB/Redis/Nginx).
*   **Decision:** Implement Clean Architecture (Independent of Frameworks, Testable).
*   **Rationale:** Allows swapping PostgreSQL with ScyllaDB/Mongo in Phase 3 without changing the Ranking Engine core. Interface-driven design enables easy Mocking for Unit Tests.

## ADR-003: Redis as "Feature Store" (Hot Data Cache)
*   **Context:** Fetching User Profiles from Disk (Postgres) is too slow for real-time recommendation.
*   **Decision:** Use Redis (In-memory) for User Interest Tags.
*   **Rationale:** Sub-millisecond read time. The system follows a "Write-through Cache" strategy where updates happen in Postgres and invalidate Redis keys.

## ADR-004: Strategy Pattern for Ranking Engine
*   **Context:** Admins need to switch between "Engagement-based" and "Diversity-based" algorithms.
*   **Decision:** Strategy Pattern.
*   **Rationale:** Decouples the recommendation controller from the specific scoring algorithm. Allows dynamic weight injection from the Admin Dashboard without service restarts.
