# Ultimate-ticket

High-scale ticketing prototype focused on:
- Zone-based inventory (no seat selection)
- Holds with TTL
- Evolution path: waiting room, payments, anti-abuse, SRE hardening

## Documentation
Start here: `docs/README.md` (Diataxis index).

## Quickstart (local)
Prerequisites:
- Docker + Docker Compose
- Go 1.22+
- Node.js (see `.nvmrc`) for the frontend

```bash
cp services/api/.env.example services/api/.env
cp frontend/.env.example frontend/.env
docker compose -f deployments/local/docker-compose.yml up --build
```

Run the API in one terminal:
```bash
make backend-run
```

Run the frontend in another terminal:
```bash
make frontend-install
make frontend-run
```

Visit:
- API health: `http://localhost:8080/health`
- Frontend: `http://localhost:5173/`

More detail: `docs/tutorials/quickstart-local.md`

## Key references
- API inventory: `docs/reference/api/api-inventory.md`
- Error codes: `docs/reference/api/error-codes.md`
- Metrics: `docs/reference/observability/metrics.md`
- Env variables: `docs/reference/env.md`
- Tests: `docs/how-to/run-tests.md`
