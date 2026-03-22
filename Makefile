COMPOSE_FILE=infrastructure/docker-compose.yml
FRONTEND_DIR=frontend
FRONTEND_NEXT_PATTERN=[n]ext/dist/bin/next dev

.PHONY: up down build logs ps restart clean fmt dev frontend-dev frontend-stop stop

up:
	docker compose -f $(COMPOSE_FILE) up --build -d

dev: up frontend-dev

frontend-dev: frontend-stop
	cd $(FRONTEND_DIR) && pnpm dev

frontend-stop:
	@# Stop stale Next.js dev process using PID file when available.
	@if [ -f "$(FRONTEND_DIR)/.next/dev/next-dev.pid" ]; then \
		PID=$$(cat "$(FRONTEND_DIR)/.next/dev/next-dev.pid"); \
		if ps -p $$PID > /dev/null 2>&1; then \
			echo "Stopping existing Next.js dev server (PID $$PID)"; \
			kill $$PID >/dev/null 2>&1 || true; \
		fi; \
		rm -f "$(FRONTEND_DIR)/.next/dev/next-dev.pid"; \
	fi
	@# Fallback for older/stale sessions without a PID file.
	@pkill -f "$(FRONTEND_NEXT_PATTERN)" >/dev/null 2>&1 || true

down:
	docker compose -f $(COMPOSE_FILE) down

stop: frontend-stop down

build:
	docker compose -f $(COMPOSE_FILE) build

logs:
	docker compose -f $(COMPOSE_FILE) logs -f --tail=200

ps:
	docker compose -f $(COMPOSE_FILE) ps

restart:
	docker compose -f $(COMPOSE_FILE) restart

clean:
	docker compose -f $(COMPOSE_FILE) down -v

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')
