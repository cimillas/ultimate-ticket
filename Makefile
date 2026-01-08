.PHONY: help test backend-test backend-run backend-fmt backend-vet backend-tidy backend-lint backend-build backend-auth-bootstrap backend-auth-reset backend-e2e run fmt vet tidy lint build frontend-install frontend-run frontend-build frontend-preview frontend-test

API_DIR := services/api
GO := go
NPM := ./scripts/npm.sh

help:
	@printf "Targets:\n"
	@printf "  test           - run backend + frontend tests + E2E\n"
	@printf "  backend-test   - run Go tests\n"
	@printf "  backend-run    - run the API locally\n"
	@printf "  backend-fmt    - format Go code\n"
	@printf "  backend-vet    - run go vet\n"
	@printf "  backend-tidy   - tidy Go modules\n"
	@printf "  backend-lint   - run golangci-lint if installed\n"
	@printf "  backend-build  - build the API binary\n"
	@printf "  backend-auth-bootstrap - create admin user from env vars\n"
	@printf "  backend-auth-reset     - reset auth tables and create admin (APP_ENV=local, CONFIRM=YES)\n"
	@printf "  backend-e2e            - run curl-based E2E validation script\n"
	@printf "  frontend-test  - run frontend tests\n"
	@printf "  frontend-install - install frontend deps\n"
	@printf "  frontend-run     - run the frontend dev server\n"
	@printf "  frontend-build   - build the frontend\n"
	@printf "  frontend-preview - preview the frontend build\n"
	@printf "  aliases: run=backend-run, fmt=backend-fmt, vet=backend-vet, tidy=backend-tidy, lint=backend-lint, build=backend-build\n"

test: backend-test frontend-test backend-e2e

backend-test:
	@cd $(API_DIR) && $(GO) test ./...

backend-run:
	@cd $(API_DIR) && \
	$(GO) run ./cmd/api & pid=$$!; \
	trap 'kill -INT $$pid; wait $$pid; exit 0' INT TERM; \
	wait $$pid

backend-fmt:
	@cd $(API_DIR) && $(GO) fmt ./...

backend-vet:
	@cd $(API_DIR) && $(GO) vet ./...

backend-tidy:
	@cd $(API_DIR) && $(GO) mod tidy

backend-lint:
	@cd $(API_DIR) && \
	if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed"; \
	fi

backend-build:
	@cd $(API_DIR) && $(GO) build ./cmd/api

backend-auth-bootstrap:
	@cd $(API_DIR) && $(GO) run ./cmd/authctl bootstrap-admin

backend-auth-reset:
	@cd $(API_DIR) && $(GO) run ./cmd/authctl reset-auth

backend-e2e:
	@./scripts/e2e/run.sh

run: backend-run
fmt: backend-fmt
vet: backend-vet
tidy: backend-tidy
lint: backend-lint
build: backend-build

frontend-install:
	@$(NPM) install

frontend-run:
	@$(NPM) run dev

frontend-build:
	@$(NPM) run build

frontend-preview:
	@$(NPM) run preview

frontend-test:
	@$(NPM) test
