PROJECT_ROOT := $(shell git rev-parse --show-toplevel)
STACK_NAME := monitoring

.PHONY: up down build logs

up: build
	docker stack deploy -c $(PROJECT_ROOT)/assets/docker-compose.yaml $(STACK_NAME)

down:
	docker stack rm $(STACK_NAME)

build: bin/alert-receiver
	docker build -t my-custom-alertmanager:latest -f $(PROJECT_ROOT)/Dockerfile.alertmanager .
	docker build -t my-custom-alert-receiver:latest -f $(PROJECT_ROOT)/Dockerfile.alert-receiver .

bin/alert-receiver: cmd/alert-receiver/main.go
	go build -o bin/alert-receiver cmd/alert-receiver/main.go

logs_prometheus:
	docker service logs -f $(STACK_NAME)_prometheus

logs_alertmanager:
	docker service logs -f $(STACK_NAME)_alertmanager

logs_alert_receiver:
	docker service logs -f $(STACK_NAME)_alert_receiver
