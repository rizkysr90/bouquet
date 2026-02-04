.PHONY: help build run dev test clean docker-build docker-up docker-down migrate-up migrate-down css css-watch assets

# Variables
APP_NAME=aslam-flower
DOCKER_COMPOSE=docker-compose
GO=go

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application (using vendor). Run 'make css' first for production styles.
	$(GO) build -mod=vendor -o bin/$(APP_NAME) ./cmd/server

css: ## Build Tailwind CSS for production (run before deploy)
	@mkdir -p web/static/css
	npm run build:css

css-watch: ## Watch and rebuild Tailwind CSS during development
	@mkdir -p web/static/css
	npm run watch:css

assets: ## Build all static assets (Tailwind CSS + copy htmx). Run before deploy.
	@mkdir -p web/static/css web/static/js
	npm run build:assets

run: ## Run the application (using vendor)
	$(GO) run -mod=vendor ./cmd/server

dev: ## Run with hot reload (requires air)
	air -c .air.toml

test: ## Run tests (using vendor)
	$(GO) test -mod=vendor -v ./...

test-coverage: ## Run tests with coverage (using vendor)
	$(GO) test -mod=vendor -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

docker-build: ## Build Docker image
	docker build -f docker/Dockerfile -t $(APP_NAME):latest .

docker-up: ## Start Docker Compose services
	$(DOCKER_COMPOSE) up -d

docker-down: ## Stop Docker Compose services
	$(DOCKER_COMPOSE) down

docker-logs: ## View Docker Compose logs
	$(DOCKER_COMPOSE) logs -f app

docker-ps: ## Show running Docker containers
	$(DOCKER_COMPOSE) ps

migrate-up: ## Run database migration 004_move_flags_to_variants.sql
	@echo "Running migration: 004_move_flags_to_variants.sql"
	@$(DOCKER_COMPOSE) exec -T postgres psql -U postgres -d flower_catalog < migrations/004_move_flags_to_variants.sql

migrate-up-all: ## Run all migrations in order (for fresh database)
	@echo "Running all database migrations..."
	@$(DOCKER_COMPOSE) exec -T postgres psql -U postgres -d flower_catalog < migrations/001_init.sql
	@$(DOCKER_COMPOSE) exec -T postgres psql -U postgres -d flower_catalog < migrations/002_seed_categories.sql
	@$(DOCKER_COMPOSE) exec -T postgres psql -U postgres -d flower_catalog < migrations/003_create_admin.sql
	@$(DOCKER_COMPOSE) exec -T postgres psql -U postgres -d flower_catalog < migrations/004_move_flags_to_variants.sql
	@echo "All migrations completed!"

migrate-down: ## Rollback last migration (manual - edit as needed)
	@echo "Warning: Manual rollback required. Edit this target for specific rollback SQL."

db-shell: ## Open PostgreSQL shell
	$(DOCKER_COMPOSE) exec postgres psql -U postgres -d flower_catalog

deps: ## Download Go dependencies and vendor
	$(GO) mod download
	$(GO) mod tidy
	$(GO) mod vendor

deps-update: ## Update Go dependencies and vendor
	$(GO) get -u ./...
	$(GO) mod tidy
	$(GO) mod vendor

vendor: ## Create/update vendor directory
	$(GO) mod vendor

lint: ## Run linter
	@echo "TODO: Add golangci-lint"

fmt: ## Format Go code
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

