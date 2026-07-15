NETWORK_NAME := go-mongodb
MONGODB_CONTAINER_NAME := mongodb
MONGODB_VERSION := 7.0
MONGODB_PORT := 27017
APP_PORT := 8080

.PHONY: docker.create.network
docker.create.network:
	docker network create go-mongodb

.PHONY: docker.build.app
docker.build.app:
	docker build -t app .

.PHONY: docker.run.app
docker.run.app:
	docker run --rm \
		-p $(APP_PORT):$(APP_PORT) \
		-e PORT=$(APP_PORT) \
		-e MONGODB_HOST=$(MONGODB_CONTAINER_NAME) \
		-e MONGODB_PORT=$(MONGODB_PORT) \
		--network=$(NETWORK_NAME) \
		--name go-app app

.PHONY: docker.run.mongodb
docker.run.mongodb:
	docker run --rm \
		-p $(MONGODB_PORT):$(MONGODB_PORT) \
		-d \
		--network=$(NETWORK_NAME) \
		--name $(MONGODB_CONTAINER_NAME) \
		mongo:$(MONGODB_VERSION)
