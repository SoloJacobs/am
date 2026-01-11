.PHONY: up

up: bin/alert-receiver
	docker compose -f assets/docker-compose.yaml up --build

bin/alert-receiver: cmd/alert-receiver/main.go
	go build -o bin/alert-receiver cmd/alert-receiver/main.go
