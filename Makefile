.PHONY: help build run dev test clean docker-build docker-up docker-down docker-up-build css css-watch assets server/start server/restart server/stop migrate-up migrate-status migrate-rollback migrate-new migrate-up-docker

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
	docker build -t $(APP_NAME):latest .

docker-up: ## Start Docker Compose services (uses existing image; no rebuild)
	$(DOCKER_COMPOSE) up -d

docker-up-build: ## Rebuild app image and start Docker Compose services
	$(DOCKER_COMPOSE) up -d --build

docker-down: ## Stop Docker Compose services
	$(DOCKER_COMPOSE) down

docker-logs: ## View Docker Compose logs (api service)
	$(DOCKER_COMPOSE) logs -f api

docker-ps: ## Show running Docker containers
	$(DOCKER_COMPOSE) ps

server/start: ## Start Docker Compose (api, postgres, adminer)
	$(DOCKER_COMPOSE) up -d

server/restart: ## Restart api service
	$(DOCKER_COMPOSE) restart api

server/stop: ## Stop Docker Compose
	$(DOCKER_COMPOSE) down

# Dbmate migrations (db/migrations/*.sql). With Docker: db-migration runs on server/start.
migrate-up: ## Run pending dbmate migrations (requires dbmate: brew install dbmate, or use Docker)
	dbmate up

migrate-status: ## Show dbmate migration status
	dbmate status

migrate-rollback: ## Roll back the most recent dbmate migration
	dbmate rollback

migrate-new: ## Create a new dbmate migration (usage: make migrate-new name=add_foo)
	dbmate new $(name)

migrate-up-docker: ## Run dbmate up inside Docker (no local dbmate needed)
	$(DOCKER_COMPOSE) run --rm db-migration up

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

