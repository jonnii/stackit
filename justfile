# Default recipe
default: test

# Run all tests (with caching for faster repeated runs)
test:
	go mod tidy
	@echo "Running tests..."
	go test ./...

# Run all tests without caching (for CI or debugging flaky tests)
test-fresh:
	go mod tidy
	@echo "Running tests (no cache)..."
	go test ./... -count=1

# Run tests with verbose output
test-verbose:
	go mod tidy
	go test -v ./...

# Run tests with coverage
test-coverage:
	go mod tidy
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
test-race:
	go mod tidy
	go test -race ./...

# Run tests for a specific package
# Usage: just test-pkg ./testhelpers
test-pkg pkg:
	@if [ -z "{{pkg}}" ]; then \
		echo "Usage: just test-pkg ./testhelpers"; \
		exit 1; \
	fi
	go mod tidy
	go test -v {{pkg}}

# Download dependencies
deps:
	go mod download
	go mod tidy

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html
	find . -type d -name "stackit-test-*" -exec rm -rf {} + 2>/dev/null || true

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Run all checks (format, lint, test)
check: fmt lint test

# Build the stackit binary
build:
	go build -o stackit ./cmd/stackit

# Install stackit binary (builds and copies to current directory)
install: build
	@echo "Built stackit binary in current directory"

# Run stackit command (builds first, then runs)
# Usage: just run log
# Usage: just run init
run cmd:
	@echo "Building stackit..."; \
	just build
	./stackit {{cmd}}

# Initialize stackit in this repository
init:
	@if [ ! -f ./stackit ]; then \
		echo "Building stackit..."; \
		just build; \
	fi
	./stackit init

# Run the website server
website:
	cd website && make run

# Run the website in dev mode with live reload
website-dev:
	cd website && make dev

