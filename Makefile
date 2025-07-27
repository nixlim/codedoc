.PHONY: help
help: ## Display this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: deps
deps: ## Install dependencies
	go mod download
	go mod tidy

.PHONY: build
build: ## Build the application
	go build -o bin/codedoc-mcp-server ./cmd/server

.PHONY: run
run: ## Run the application
	go run ./cmd/server

.PHONY: test
test: ## Run tests
	go test -v -race ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: lint
lint: ## Run linter
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

.PHONY: docker-up
docker-up: ## Start Docker services
	docker-compose up -d

.PHONY: docker-down
docker-down: ## Stop Docker services
	docker-compose down

.PHONY: docker-logs
docker-logs: ## View Docker logs
	docker-compose logs -f

.PHONY: db-create
db-create: ## Create database
	createdb -h localhost -U codedoc codedoc_dev || true

.PHONY: db-drop
db-drop: ## Drop database
	dropdb -h localhost -U codedoc codedoc_dev || true

.PHONY: db-migrate
db-migrate: ## Run database migrations
	migrate -path migrations -database $(DATABASE_URL) up

.PHONY: db-rollback
db-rollback: ## Rollback database migrations
	migrate -path migrations -database $(DATABASE_URL) down 1

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/ coverage.out coverage.html