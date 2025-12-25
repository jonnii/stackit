.PHONY: build run dev clean test lint format

# Build the website for production
build:
	npm run build

# Run the production server
run: build
	npm start

# Development server with hot reload
dev:
	npm run dev

# Clean build artifacts
clean:
	rm -rf .next out node_modules/.cache

# Test the build (lint and type check)
test:
	npm run build

# Lint code
lint:
	npm run lint

# Format code
format:
	npm run lint -- --fix

# Install dependencies
install:
	npm install

# Deploy to static hosting (exports to 'out' directory)
export: build
	@echo "Static files exported to 'out' directory"

# Help
help:
	@echo "Available targets:"
	@echo "  build     - Build the website for production"
	@echo "  run       - Build and run the production server"
	@echo "  dev       - Run development server with hot reload"
	@echo "  clean     - Remove build artifacts"
	@echo "  test      - Test the build (lint and type check)"
	@echo "  lint      - Lint code"
	@echo "  format    - Format code"
	@echo "  install   - Install dependencies"
	@echo "  export    - Export static files for deployment"