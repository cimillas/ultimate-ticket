# Run tests

This project uses a mix of Go tests, frontend tests, and curl-based E2E flows.

## Full suite
```bash
make test
```

Notes:
- E2E requires the API to already be running.
- E2E requires `ALLOW_PUBLIC_REGISTER=true`.

## Backend tests
```bash
make backend-test
```

Single package / test example:
```bash
cd services/api
go test ./internal/transport/http -run TestRequestLogger_LogsServiceName
```

## Frontend tests
```bash
make frontend-test
```

## E2E only
```bash
ALLOW_PUBLIC_REGISTER=true make backend-e2e
```

Optional clean auth state for E2E:
```bash
E2E_RESET_AUTH=1 ALLOW_PUBLIC_REGISTER=true make backend-e2e
```

## Reference
- Commands: `docs/reference/commands.md`
- E2E plan: `docs/explanation/e2e-plan.md`
