.PHONY: help test test-verbose test-coverage test-auth test-storage test-watch clean
#.DEFAULT_GOAL:=run

help:
	@echo "Available targets:"
	@echo "  test              - Run all tests"
	@echo "  test-verbose      - Run all tests with verbose output"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  test-auth         - Run only auth service tests"
	@echo "  test-storage      - Run only storage tests"
	@echo "  clean             - Remove coverage files"
	@echo "  run               - Run the application"
	@echo "  build             - Build the application"


# Run the application
run-auth:
	go run ./cmd/sso/main.go --config=./config/local.yaml

# Run the application
run-ads:
	go run cmd/server/main.go

run:
	go run ./cmd/sso/main.go --config=./config/local.yaml
	go run cmd/server/main.go

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover ./internal/...

# Run only auth service tests
test-auth:
	go test -v ./internal/services/auth/...

# Run only storage tests
test-storage:
	go test -v ./internal/storage/...

# Build the application
build:
	go build -o bin/sso ./cmd/sso

# Clean up generated files
clean:
	go clean
	rm -f coverage.out coverage.html
	rm -rf bin/

# Run linter (requires: golangci-lint)
lint:
	golangci-lint run --fix ./...

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...
