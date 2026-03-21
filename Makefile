COMPOSE_FILE=infrastructure/docker-compose.yml

.PHONY: up down build logs ps restart clean fmt

up:
	docker compose -f $(COMPOSE_FILE) up --build -d

down:
	docker compose -f $(COMPOSE_FILE) down

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
