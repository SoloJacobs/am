PROJECT_ROOT := $(shell git rev-parse --show-toplevel)
STACK_NAME := monitoring

.PHONY: up down build logs

up: build
	docker stack deploy -c $(PROJECT_ROOT)/assets/docker-compose.yaml $(STACK_NAME)

down:
	docker stack rm $(STACK_NAME)

build:
	docker build -t my-custom-prometheus:latest -f $(PROJECT_ROOT)/Dockerfile.prometheus .
	docker build -t my-custom-alertmanager:latest -f $(PROJECT_ROOT)/Dockerfile.alertmanager .

logs_prometheus:
	docker service logs -f $(STACK_NAME)_prometheus

logs_alertmanager:
	docker service logs -f $(STACK_NAME)_alertmanager
