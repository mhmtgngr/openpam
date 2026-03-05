# OpenPAM - Makefile
# Quick commands for Docker deployment

COMPOSE_DIR := deployments/docker
COMPOSE := docker compose -f $(COMPOSE_DIR)/docker-compose.yml

.PHONY: help setup build up up-dev up-prod down restart logs ps clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

setup: ## Generate JWT keys, TLS certs, and prepare environment
	@chmod +x $(COMPOSE_DIR)/setup.sh $(COMPOSE_DIR)/postgres-entrypoint.sh
	@bash $(COMPOSE_DIR)/setup.sh

build: ## Build all Docker images
	$(COMPOSE) build

up: setup ## Start all services (infrastructure + backend + frontend)
	$(COMPOSE) up -d --build

up-dev: setup ## Start with dev frontend (hot reload on port 3000)
	$(COMPOSE) --profile dev up -d --build

up-prod: setup ## Start with production frontend + nginx (ports 80/443)
	$(COMPOSE) --profile production up -d --build

down: ## Stop all services
	$(COMPOSE) --profile dev --profile production down

restart: ## Restart all services
	$(COMPOSE) restart

logs: ## Show logs (tail)
	$(COMPOSE) logs -f --tail=100

logs-service: ## Show logs for a specific service (usage: make logs-service SVC=gateway)
	$(COMPOSE) logs -f --tail=100 $(SVC)

ps: ## Show running services
	$(COMPOSE) ps

clean: ## Stop services and remove volumes (WARNING: deletes all data)
	$(COMPOSE) --profile dev --profile production down -v
	@echo "All volumes removed. Run 'make up' to start fresh."

health: ## Check health of all services
	@echo "=== Service Health ==="
	@for port in 8500 8501 8502 8503 8504 8505 8506 8507 8508; do \
		printf "Port %-5s: " $$port; \
		curl -sf http://localhost:$$port/health 2>/dev/null && echo "" || echo "DOWN"; \
	done
