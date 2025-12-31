# Local docker stack

Spin up Postgres for local development:
```bash
docker compose -f deployments/local/docker-compose.yml up --build
```

Service:
- `postgres` (image: `postgres:16-alpine`)
  - user: `ultimate_ticket`
  - password: `ultimate_ticket`
  - database: `ultimate_ticket`
  - port: `5432`

Data persists in the named volume `postgres_data`.
