# Default recipe
default: test

# Run all tests (ensures dependencies are up to date)
test:
	go mod tidy
	go test ./...

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
