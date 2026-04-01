.PHONY: up down build logs ps restart clean fmt stop

up:
	cd infrastructure && docker compose up --build -d

down:
	cd infrastructure && docker compose down

stop: down

build:
	cd infrastructure && docker compose build

logs:
	cd infrastructure && docker compose logs -f --tail=200

ps:
	cd infrastructure && docker compose ps

restart:
	cd infrastructure && docker compose restart

clean:
	cd infrastructure && docker compose down -v

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')
