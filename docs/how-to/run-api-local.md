# Run the API locally

This guide focuses on running the API against local dependencies.

## Prerequisites
- Docker + Docker Compose
- Go 1.22+

## Steps
1) Start local dependencies:
```bash
docker compose -f deployments/local/docker-compose.yml up --build
```

2) Create the API env file:
```bash
cp services/api/.env.example services/api/.env
```

3) Run the API:
```bash
make backend-run
```

## Optional: run directly with Go
```bash
cd services/api
go run ./cmd/api
```

## Verify
- Health: `GET http://localhost:8080/health`
- Metrics: `GET http://localhost:8080/metrics`

## Reference
- Environment variables: `docs/reference/env.md`
- API surface: `docs/reference/api/api-inventory.md`
