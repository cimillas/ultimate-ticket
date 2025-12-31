.PHONY: help test run fmt vet tidy lint build

API_DIR := services/api
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

test:
	@cd $(API_DIR) && $(GO) test ./...

run:
	@cd $(API_DIR) && $(GO) run ./cmd/api

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
