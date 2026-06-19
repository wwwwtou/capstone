# Architecture Diagrams (PlantUML)

Authoritative architecture diagrams for the E-commerce Video Recommendation
System, kept **in sync with the actual code** (Go microservices + gateway,
React/Node BFF, PostgreSQL database-per-service, Redis cache, GitHub Actions
CI/CD, and the circuit-breaker / retry / rate-limit fault-tolerance layer).

> These supersede the illustrative Mermaid sketches in `../../PRD_ARCHITECTURE.md`,
> which were aspirational and no longer match the implementation.

## Diagrams

| File | Diagram | What it shows |
|------|---------|---------------|
| `logical-architecture.puml` | Logical architecture | Layered view: Presentation (React) → Edge/BFF (Node `server.ts`) → API Gateway (Go) → Domain services → Data (PostgreSQL per-service + Redis). Gateway cross-cutting concerns called out. |
| `physical-architecture.puml` | Physical / deployment (local) | The real `docker-compose` runtime: containers, images, ports, env wiring, volumes. |
| `deployment-cloud.puml` | Cloud deployment | Current Render single-service deploy (`render.yaml`) + the AWS EC2 Terraform option (`terraform/`). |
| `ddd-context-map.puml` | DDD context map | Bounded contexts (Recommendation = core; User/Content = supporting; Dashboard = generic), their relationships, and database-per-service ownership. |
| `er-diagram.puml` | ER diagram (detailed) | All tables grouped by owning database (`user_db`, `content_db`, `rec_db`) with columns, types, PK/index/unique, the Redis key-value structure, and app-level (no-FK) cross-service links. |
| `sequence-get-recommendations.puml` | Core use-case sequence | `GET /api/v1/recommendations` end to end, including rate-limit 429, circuit-breaker routing, concurrent downstream fetches, and the cold-start degraded fallback. |
| `cicd-pipeline.puml` | CI/CD pipeline | The six GitHub Actions jobs and the gated Render deploy. |
| `usecase.puml` | Use case diagram | Terminal User vs Admin/Operator actors and their use cases. |
| `recommendation-clean-architecture.puml` | Clean/onion architecture | The core recommendation service's layered code structure (`internal/domain` → `app` → `infra`/`transport`), showing the repository ports and the adapters that implement them. Mirrors the actual package layout. |

Pre-rendered PNGs live in `png/` for quick viewing / screenshots.

## Rendering

Requires the PlantUML jar (e.g. from the VS Code PlantUML extension) and
Graphviz `dot`. From the repo root:

```bash
java -jar /path/to/plantuml.jar -tpng -charset UTF-8 \
  -graphvizdot /path/to/dot -o png docs/architecture/*.puml
```

Or open any `.puml` in VS Code with the PlantUML extension and use **Preview**
(`Alt+D`). The sources are ASCII-only so they render identically regardless of
the host locale.
