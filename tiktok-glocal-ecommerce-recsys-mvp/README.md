TikTok Glocal Ecommerce Recommendation Architecture (Proxy MVP)

Minimal microservices MVP for MTech SE defense.

Contains:
- API Gateway (Go)
- User Service (Go) with PostgreSQL `user_db` and Redis caching
- Content Service (Go) with PostgreSQL `content_db`
- Recommendation Service (Go) with PostgreSQL `rec_db` and strict Strategy Pattern
- `docker-compose.yml` with one Postgres instance (creates three logical DBs) and Redis

Quick run:

```bash
docker compose up --build
```

API:
- Gateway routes `/api/v1/users`, `/api/v1/content`, `/api/v1/recommendations` to services.
