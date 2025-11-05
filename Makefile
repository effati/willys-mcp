.PHONY: build test

build:
	@echo "Building..."
	@go build -o willys-mcp ./cmd/server

test:
	@echo "Running integration tests..."
	@echo "Note: tests make real API calls to Willys.se"
	@echo "Set WILLYS_USERNAME and WILLYS_PASSWORD"
	@go test -v ./test -timeout 10m
