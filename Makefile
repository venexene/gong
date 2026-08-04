.PHONY: up down test lint build run clean clean-db

help:
	@echo "Usage: make [target]"
	@echo "Targets:"
	@echo "  up       - Build and start the Docker containers"
	@echo "  down     - Stop and remove the Docker containers"
	@echo "  test     - Run tests with race detection"
	@echo "  lint     - Run golangci-lint"
	@echo "  build    - Build the Go application"
	@echo "  run      - Run the Go application"
	@echo "  clean    - Remove the built application"
	@echo "  clean-db - Stop containers and remove database volumes"

up:
	@docker compose up --build -d

down:
	@docker compose down

test:
	@go test ./internal/... -race -count=1

lint:
	@golangci-lint run

build:
	@go build -o gong ./cmd

run:
	@go run ./cmd

clean:
	@rm -f gong

clean-db:
	@docker compose down -v