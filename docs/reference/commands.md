# Commands reference

## Root make targets
```bash
make test
make backend-test
make backend-run
make backend-fmt
make backend-vet
make backend-tidy
make backend-lint
make backend-build
make backend-auth-bootstrap
APP_ENV=local CONFIRM=YES make backend-auth-reset
ALLOW_PUBLIC_REGISTER=true make backend-e2e
make frontend-install
make frontend-run
make frontend-build
make frontend-preview
make frontend-test
```

Notes:
- `make test` and `make backend-e2e` require the API to already be running.
- E2E requires `ALLOW_PUBLIC_REGISTER=true`.
- `make backend-lint` uses `golangci-lint` if it is installed.

## Go commands (API module)
```bash
cd services/api
go test ./...
go run ./cmd/api
go fmt ./...
go vet ./...
go mod tidy
```

## Lint (optional)
```bash
cd services/api
golangci-lint run
```

## Local dependencies
```bash
docker compose -f deployments/local/docker-compose.yml up --build
docker compose -f deployments/local/docker-compose.yml down -v
```
