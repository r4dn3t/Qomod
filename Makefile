.PHONY: all help dev-deps down test-all test-users test-notifications test-auth run-users run-notifications run-auth

help:
	@echo "Available commands:"
	@echo "  make dev-deps           Start PostgreSQL and RabbitMQ via Docker Compose"
	@echo "  make down               Stop all Docker Compose services"
	@echo "  make test-all           Run tests across all 3 microservices"
	@echo "  make test-users         Run tests for Users microservice (Go)"
	@echo "  make test-notifications Run tests for Notifications microservice (.NET)"
	@echo "  make test-auth          Run tests for Auth microservice (Python)"
	@echo "  make run-users          Run Users microservice (Go on port 8081)"
	@echo "  make run-notifications  Run Notifications microservice (.NET on port 8082)"
	@echo "  make run-auth           Run Auth microservice (Python on port 8083)"

dev-deps:
	docker compose up -d postgres rabbitmq

down:
	docker compose down

test-all: test-users test-notifications test-auth

test-users:
	@echo "=== Running Users Service Tests (Go) ==="
	cd services/users && go test -v -race ./...

test-notifications:
	@echo "=== Running Notifications Service Tests (.NET) ==="
	dotnet test services/notifications.tests

test-auth:
	@echo "=== Running Auth Service Tests (Python) ==="
	cd services/auth && .venv/bin/pytest -v

run-users:
	cd services/users && go run ./cmd/api

run-notifications:
	cd services/notifications && dotnet run --launch-profile ""

run-auth:
	cd services/auth && .venv/bin/uvicorn app.main:app --port 8083 --reload
