# Default recipe
default: check

# Setup the development environment (install tools, dependencies, and build)
setup: install-tools deps build
	@echo "✅ Setup complete! You can now run 'just check' to verify everything is working."
	@if [ ! -f ./stackit ]; then \
		echo "Warning: stackit binary not found. Try running 'just build'."; \
	fi

# Download dependencies for all modules
deps:
	@echo "📦 Downloading main dependencies..."
	go mod download
	go mod tidy
	@echo "📦 Downloading website dependencies..."
	cd website && go mod tidy

# Install development tools (gotestsum, golangci-lint, goimports)
install-tools:
	@echo "🛠️ Installing development tools..."
	go install gotest.tools/gotestsum@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Run all tests (with caching for faster repeated runs)
test:
	@echo "Running tests..."
	@if command -v gotestsum >/dev/null 2>&1; then \
		STACKIT_TEST_NO_INTERACTIVE=1 gotestsum --format pkgname-and-test-fails -- ./...; \
	else \
		STACKIT_TEST_NO_INTERACTIVE=1 go test ./...; \
	fi

# Run all tests without caching (for CI or debugging flaky tests)
test-fresh:
	@echo "Running tests (no cache)..."
	@if command -v gotestsum >/dev/null 2>&1; then \
		STACKIT_TEST_NO_INTERACTIVE=1 gotestsum --format pkgname-and-test-fails -- ./... -count=1; \
	else \
		STACKIT_TEST_NO_INTERACTIVE=1 go test ./... -count=1; \
	fi

# Run tests with verbose output
test-verbose:
	@if command -v gotestsum >/dev/null 2>&1; then \
		STACKIT_TEST_NO_INTERACTIVE=1 gotestsum --format standard-verbose -- ./...; \
	else \
		STACKIT_TEST_NO_INTERACTIVE=1 go test -v ./...; \
	fi

# Run tests with coverage
test-coverage:
	@if command -v gotestsum >/dev/null 2>&1; then \
		STACKIT_TEST_NO_INTERACTIVE=1 gotestsum --format pkgname-and-test-fails -- -coverprofile=coverage.out ./...; \
	else \
		STACKIT_TEST_NO_INTERACTIVE=1 go test -coverprofile=coverage.out ./...; \
	fi
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
test-race:
	@if command -v gotestsum >/dev/null 2>&1; then \
		STACKIT_TEST_NO_INTERACTIVE=1 gotestsum --format pkgname-and-test-fails -- -race ./...; \
	else \
		STACKIT_TEST_NO_INTERACTIVE=1 go test -race ./...; \
	fi

# Run tests for a specific package
# Usage: just test-pkg ./testhelpers
test-pkg pkg:
	@if [ -z "{{pkg}}" ]; then \
		echo "Usage: just test-pkg ./testhelpers"; \
		exit 1; \
	fi
	@if command -v gotestsum >/dev/null 2>&1; then \
		STACKIT_TEST_NO_INTERACTIVE=1 gotestsum --format standard-verbose -- {{pkg}}; \
	else \
		STACKIT_TEST_NO_INTERACTIVE=1 go test -v {{pkg}}; \
	fi

# Run tests in watch mode (requires gotestsum)
test-watch:
	@if command -v gotestsum >/dev/null 2>&1; then \
		STACKIT_TEST_NO_INTERACTIVE=1 gotestsum --watch --format pkgname-and-test-fails -- ./...; \
	else \
		echo "gotestsum not installed. Install with: go install gotest.tools/gotestsum@latest"; \
		exit 1; \
	fi

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html
	find . -type d -name "stackit-test-*" -exec rm -rf {} + 2>/dev/null || true

# Format code
fmt:
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	elif [ -f "$(go env GOPATH)/bin/goimports" ]; then \
		"$(go env GOPATH)/bin/goimports" -w .; \
	else \
		go fmt ./...; \
	fi

# Run linter (requires golangci-lint)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	elif [ -f "$(go env GOPATH)/bin/golangci-lint" ]; then \
		"$(go env GOPATH)/bin/golangci-lint" run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Run linter and fix issues (if supported by the linter)
lint-fix:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix; \
	elif [ -f "$(go env GOPATH)/bin/golangci-lint" ]; then \
		"$(go env GOPATH)/bin/golangci-lint" run --fix; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Run all checks (format, lint, test)
check:
	@echo "🎨 Formatting..."
	@just fmt
	@echo "🔍 Linting..."
	@just lint
	@echo "🧪 Testing..."
	@just test

# Build the stackit binary
build:
	go build -o stackit ./cmd/stackit
	@if [ ! -L ./st ] && [ ! -f ./st ]; then \
		ln -s ./stackit ./st; \
		echo "Created symlink: st -> stackit"; \
	elif [ -L ./st ]; then \
		echo "Symlink st already exists"; \
	else \
		echo "Warning: st already exists as a regular file, skipping symlink creation"; \
	fi

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




