.PHONY: help test run fmt vet tidy lint build frontend-install frontend-run frontend-build frontend-preview

API_DIR := services/api
FRONTEND_DIR := frontend
GO := go

help:
	@printf "Targets:\n"
	@printf "  test  - run unit tests\n"
	@printf "  run   - run the API locally\n"
	@printf "  fmt   - format Go code\n"
	@printf "  vet   - run go vet\n"
	@printf "  tidy  - tidy Go modules\n"
	@printf "  lint  - run golangci-lint if installed\n"
	@printf "  build - build the API binary\n"
	@printf "  frontend-install - install frontend deps via nvm\n"
	@printf "  frontend-run     - run the frontend dev server via nvm\n"
	@printf "  frontend-build   - build the frontend via nvm\n"
	@printf "  frontend-preview - preview the frontend build via nvm\n"

test:
	@cd $(API_DIR) && $(GO) test ./...

run:
	@cd $(API_DIR) && \
	$(GO) run ./cmd/api & pid=$$!; \
	trap 'kill -INT $$pid; wait $$pid; exit 0' INT TERM; \
	wait $$pid

fmt:
	@cd $(API_DIR) && $(GO) fmt ./...

vet:
	@cd $(API_DIR) && $(GO) vet ./...

tidy:
	@cd $(API_DIR) && $(GO) mod tidy

lint:
	@cd $(API_DIR) && \
	if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed"; \
	fi

build:
	@cd $(API_DIR) && $(GO) build ./cmd/api

frontend-install:
	@bash -lc 'if [ -s "$$HOME/.nvm/nvm.sh" ]; then source "$$HOME/.nvm/nvm.sh"; nvm use >/dev/null; cd $(FRONTEND_DIR) && npm install; else echo "nvm not found in $$HOME/.nvm"; exit 1; fi'

frontend-run:
	@bash -lc 'if [ -s "$$HOME/.nvm/nvm.sh" ]; then source "$$HOME/.nvm/nvm.sh"; nvm use >/dev/null; cd $(FRONTEND_DIR) && npm run dev; else echo "nvm not found in $$HOME/.nvm"; exit 1; fi'

frontend-build:
	@bash -lc 'if [ -s "$$HOME/.nvm/nvm.sh" ]; then source "$$HOME/.nvm/nvm.sh"; nvm use >/dev/null; cd $(FRONTEND_DIR) && npm run build; else echo "nvm not found in $$HOME/.nvm"; exit 1; fi'

frontend-preview:
	@bash -lc 'if [ -s "$$HOME/.nvm/nvm.sh" ]; then source "$$HOME/.nvm/nvm.sh"; nvm use >/dev/null; cd $(FRONTEND_DIR) && npm run preview; else echo "nvm not found in $$HOME/.nvm"; exit 1; fi'
