.PHONY: help build run test clean docker migrate

help:
	@echo "Available targets:"
	@echo "  build         - Build all binaries"
	@echo "  build-server  - Build server binary"
	@echo "  build-worker  - Build worker binary"
	@echo "  build-cli     - Build CLI binary"
	@echo "  run-server    - Run server"
	@echo "  run-worker    - Run worker"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  migrate       - Run database migrations"
	@echo "  migrate-down  - Rollback database migrations"
	@echo "  docker-build  - Build Docker images"
	@echo "  docker-up     - Start all services"
	@echo "  docker-down   - Stop all services"
	@echo "  clean         - Clean build artifacts"

build: build-server build-worker build-cli

build-server:
	@echo "Building server..."
	@go build -o bin/server cmd/server/main.go

build-worker:
	@echo "Building worker..."
	@go build -o bin/worker cmd/worker/main.go

build-cli:
	@echo "Building CLI..."
	@go build -o bin/xgent-cli cmd/cli/main.go

run-server:
	@echo "Running server..."
	@go run cmd/server/main.go

run-worker:
	@echo "Running worker..."
	@go run cmd/worker/main.go

test:
	@echo "Running tests..."
	@go test -v -race ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

migrate:
	@echo "Running migrations..."
	@go run cmd/migrate/main.go up

migrate-down:
	@echo "Rolling back migrations..."
	@go run cmd/migrate/main.go down

docker-build:
	@echo "Building Docker images..."
	@docker-compose build

docker-up:
	@echo "Starting services..."
	@docker-compose up -d

docker-down:
	@echo "Stopping services..."
	@docker-compose down

docker-logs:
	@docker-compose logs -f

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean

fmt:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Running go vet..."
	@go vet ./...

lint:
	@echo "Running golangci-lint..."
	@golangci-lint run

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

dev: docker-up run-server

.DEFAULT_GOAL := help
