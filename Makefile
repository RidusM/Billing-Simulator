PROJECT_NAME := bill-stripe-sim
BINARY_PATH  := ./bin

SIM_IMG      := $(PROJECT_NAME):latest

BASE_STACK   := docker compose -f docker-compose.yml

GOBUILD      := CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"
GOTEST       := go test -v -race
GOCOVER      := -covermode=atomic -coverprofile=coverage.txt

.DEFAULT_GOAL := help

##@ General

.PHONY: help
help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: check-requirements
check-requirements: ## Check all requirements
	@echo "Checking requirements..."
	@command -v go         >/dev/null 2>&1 || { echo "Go not found";         exit 1; }
	@command -v docker     >/dev/null 2>&1 || { echo "Docker not found";     exit 1; }
	@echo "All requirements met"

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing tools..."
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/daixiang0/gci@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/segmentio/golines@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install go.uber.org/mock/mockgen@latest
	@echo "Tools installed"

##@ Development

.PHONY: deps
deps: ## Tidy and verify Go modules
	go mod tidy && go mod verify

.PHONY: deps-audit
deps-audit: ## Check dependencies for vulnerabilities
	govulncheck ./...

.PHONY: build
build: deps ## Build all binaries for linux/amd64
	@echo "Building binaries..."
	@mkdir -p $(BINARY_PATH)
	$(GOBUILD) -o $(BINARY_PATH)/bill-stripe-sim ./cmd/bill-stripe-sim
	@echo "Binaries built in $(BINARY_PATH)/"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf ./bin/ coverage*.txt
	@find . -type f -name '*_mock.go' -path '*/mock/*' -delete 2>/dev/null || true
	@echo "Done"

.PHONY: clean-cache
clean-cache: ## Clean test and linter caches
	go clean -testcache
	golangci-lint cache clean 2>/dev/null || true

##@ Code Quality

.PHONY: format
format: ## Format code (gofumpt, gci, golines, goimports)
	@echo "Formatting..."
	gofumpt  -l -w .
	gci write . --skip-generated -s standard -s default
	goimports -w .
	golines  -w --max-len=120 .
	@echo "Done"

.PHONY: lint
lint: ## Run golangci-lint
	@echo "Linting..."
	golangci-lint run
	@echo "Lint passed"

.PHONY: lint-hadolint
lint-hadolint: ## Run hadolint on Dockerfiles
	@for df in deployments/*/Dockerfile; do \
		echo "Linting $$df..."; \
		hadolint $$df; \
	done

.PHONY: lint-dotenv
lint-dotenv: ## Run dotenv-linter on config files
	dotenv-linter check -r configs/

.PHONY: lint-dotenv-fix
lint-dotenv-fix: ## Fix .env files
	dotenv-linter fix --no-backup -r configs/

.PHONY: pre-commit
pre-commit: format lint lint-hadolint lint-dotenv ## Run all checks before commit
	@echo "Pre-commit checks passed!"

##@ Testing

.PHONY: test
test: ## Run unit tests with race detector and coverage
	@echo "Running unit tests..."
	$(GOTEST) $(GOCOVER) ./internal/...
	go tool cover -func=coverage.txt | tail -1
	@echo "Done"

.PHONY: test-verbose
test-verbose: ## Run verbose unit tests
	$(GOTEST) -v -race -cover ./internal/...

.PHONY: test-all
test-all: test ## Run all tests
	@echo "All tests completed"

##@ Docker Compose

.PHONY: infra-up
infra-up: ## Start infrastructure only (postgres, redis, kafka)
	$(BASE_STACK) up -d \
		bill-stripe-sim-postgres \
		bill-stripe-sim-redis \
		zookeeper kafka kafka-init
	@echo "Infrastructure started"

.PHONY: infra-down
infra-down: ## Stop infrastructure
	$(BASE_STACK) stop \
		bill-stripe-sim-postgres \
		bill-stripe-sim-redis \
		zookeeper kafka

.PHONY: compose-up
compose-up: ## Start all services via docker-compose
	@echo "Starting all services..."
	$(BASE_STACK) up --build -d
	@echo "Done"
	$(BASE_STACK) logs -f bill-stripe-sim

.PHONY: compose-down
compose-down: ## Stop and remove all containers and volumes
	$(BASE_STACK) down --remove-orphans --volumes

.PHONY: compose-logs
compose-logs: ## Show logs (usage: make compose-logs service=bill-stripe-sim)
	$(BASE_STACK) logs -f $(service)

##@ Migrations (docker-compose)

.PHONY: migrate-up
migrate-up: ## Apply all migrations
	@echo "Running migrations..."
	$(BASE_STACK) run --rm bill-stripe-sim-migrator
	@echo "Migrations applied"

.PHONY: migrate-down
migrate-down: ## Rollback last migration
	@echo "Rolling back..."
	$(BASE_STACK) run --rm bill-stripe-sim-migrator down 1

.PHONY: open-grafana
open-grafana: ## Open Grafana in browser
	open http://localhost:3000

.PHONY: open-jaeger
open-jaeger: ## Open Jaeger in browser
	open http://localhost:16686

.PHONY: open-prometheus
open-prometheus: ## Open Prometheus in browser
	open http://localhost:9090

##@ Utilities

.PHONY: docker-prune
docker-prune: ## Prune docker system
	docker system prune -af

.PHONY: docker-logs
docker-logs: ## Show all docker-compose logs
	$(BASE_STACK) logs -f