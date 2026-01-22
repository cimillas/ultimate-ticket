# Quickstart (local)

Get the API and frontend running locally with Docker-backed dependencies.

## Prerequisites
- Docker + Docker Compose
- Go 1.22+ (for the API)
- Node.js (see `.nvmrc`) for the frontend

## Steps
1) Copy env examples:
```bash
cp services/api/.env.example services/api/.env
cp frontend/.env.example frontend/.env
```

2) Start dependencies:
```bash
docker compose -f deployments/local/docker-compose.yml up --build
```

3) Run the API (terminal 1):
```bash
make backend-run
```

4) Run the frontend (terminal 2):
```bash
make frontend-install
make frontend-run
```

5) Visit:
- API: `http://localhost:8080/health`
- Frontend: `http://localhost:5173/`

## Next
- Tests: `docs/how-to/run-tests.md`
- API reference: `docs/reference/api/api-inventory.md`
