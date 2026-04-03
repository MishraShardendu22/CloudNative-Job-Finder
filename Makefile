DOCKER_NAMESPACE ?= shardendumishra22
IMAGE_TAG ?= v2

.PHONY: up down build pull logs ps restart clean fmt stop reset-kafka docker-login docker-build-images docker-push-images docker-release-images

up: pull
	cd infrastructure && docker compose up -d

down:
	cd infrastructure && docker compose down

stop: down

build:
	DOCKERHUB_NAMESPACE=$(DOCKER_NAMESPACE) IMAGE_TAG=$(IMAGE_TAG) ./scripts/docker-images.sh build

pull:
	cd infrastructure && docker compose pull

logs:
	cd infrastructure && docker compose logs -f --tail=200

ps:
	cd infrastructure && docker compose ps

restart:
	cd infrastructure && docker compose restart

clean:
	cd infrastructure && docker compose down -v --remove-orphans

reset-kafka:
	cd infrastructure && docker compose rm -sf kafka kafka-ui zookeeper
	-docker volume rm infrastructure_kafka_data infrastructure_zookeeper_data
	cd infrastructure && docker compose up -d zookeeper kafka kafka-ui

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')

docker-login:
	./scripts/docker-images.sh login

docker-build-images:
	DOCKERHUB_NAMESPACE=$(DOCKER_NAMESPACE) IMAGE_TAG=$(IMAGE_TAG) ./scripts/docker-images.sh build

docker-push-images:
	DOCKERHUB_NAMESPACE=$(DOCKER_NAMESPACE) IMAGE_TAG=$(IMAGE_TAG) ./scripts/docker-images.sh push

docker-release-images:
	DOCKERHUB_NAMESPACE=$(DOCKER_NAMESPACE) IMAGE_TAG=$(IMAGE_TAG) ./scripts/docker-images.sh release
