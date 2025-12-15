.PHONY: build run clean dev test

# Build the server
build:
	go build -o server ./cmd/server

# Run the server (builds first)
run: build
	./server

# Development mode - run with live reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Running without live reload..."; \
		$(MAKE) run; \
	fi

# Clean build artifacts
clean:
	rm -f server
	rm -f *.log

# Test the server builds
test:
	go build -o /dev/null ./cmd/server
	@echo "Build test passed!"

# Format code
fmt:
	go fmt ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Deploy to Railway (requires railway CLI)
deploy:
	@if command -v railway >/dev/null 2>&1; then \
		railway up; \
	else \
		echo "Railway CLI not installed. Visit: https://docs.railway.app/develop/cli"; \
	fi

# Help
help:
	@echo "Available targets:"
	@echo "  build   - Build the server binary"
	@echo "  run     - Build and run the server"
	@echo "  dev     - Run with live reload (requires air)"
	@echo "  clean   - Remove build artifacts"
	@echo "  test    - Test that the server builds"
	@echo "  fmt     - Format Go code"
	@echo "  deps    - Download and tidy dependencies"
	@echo "  deploy  - Deploy to Railway"
