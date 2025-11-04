# Makefile for MQTT Server

.PHONY: help build run test clean fmt lint

# Default target
help:
	@echo "Available targets:"
	@echo "  build    - Build the server binary"
	@echo "  run      - Run the server"
	@echo "  test     - Run tests"
	@echo "  clean    - Remove build artifacts"
	@echo "  fmt      - Format code"
	@echo "  lint     - Run linter"

# Build the server
build:
	go build -o mqtt-server.exe ./cmd/server

# Run the server
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	del /Q mqtt-server.exe 2>nul || true
	go clean

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run
