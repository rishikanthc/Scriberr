.PHONY: help docs docs-serve docs-clean website website-dev website-build

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

docs: ## Generate API documentation from Go code annotations
	@echo "Generating API documentation..."
	@command -v swag >/dev/null 2>&1 || { echo "Error: swag not installed. Run: go install github.com/swaggo/swag/cmd/swag@latest"; exit 1; }
	swag init -g cmd/server/main.go -o api-docs
	@echo "✓ API documentation generated in api-docs/"

docs-clean: ## Clean generated API documentation
	@echo "Cleaning API documentation..."
	rm -rf api-docs/docs.go api-docs/swagger.json api-docs/swagger.yaml
	@echo "✓ API documentation cleaned"

website-dev: docs ## Start local development server for project website
	@echo "Starting website development server..."
	cd web/project-site && npm run dev

website-build: docs ## Build project website for GitHub Pages
	@echo "Building project website..."
	cd web/project-site && npm run build
	@echo "✓ Website built to /docs directory"

website-serve: website-build ## Build and preview project website locally
	@echo "Previewing website..."
	cd web/project-site && npm run preview

docs-serve: website-serve ## Alias for website-serve

build: ## Build Scriberr binary with embedded frontend
	@echo "Starting Scriberr build process..."
	@echo "Cleaning old build files..."
	@rm -f scriberr
	@rm -rf internal/web/dist
	@cd web/frontend && rm -rf dist/ && rm -rf assets/ 2>/dev/null || true
	@echo "✓ Build files cleaned"
	@echo "Building React frontend..."
	@cd web/frontend && npm run build
	@echo "✓ Frontend built"
	@echo "Copying frontend assets for embedding..."
	@rm -rf internal/web/dist
	@cp -r web/frontend/dist internal/web/
	@echo "✓ Assets copied"
	@echo "Building Go binary..."
	@go clean -cache
	@go build -o scriberr cmd/server/main.go
	@echo "✓ Binary built successfully"
	@echo "Build complete. Run './scriberr' to start the server"

build-cli: ## Build CLI binaries for Linux, macOS, and Windows
	@echo "Building CLI binaries..."
	@mkdir -p bin/cli
	GOOS=linux GOARCH=amd64 go build -o bin/cli/scriberr-linux-amd64 ./cmd/scriberr-cli
	GOOS=darwin GOARCH=amd64 go build -o bin/cli/scriberr-darwin-amd64 ./cmd/scriberr-cli
	GOOS=darwin GOARCH=arm64 go build -o bin/cli/scriberr-darwin-arm64 ./cmd/scriberr-cli
	GOOS=windows GOARCH=amd64 go build -o bin/cli/scriberr-windows-amd64.exe ./cmd/scriberr-cli
	@echo "✓ CLI binaries built in bin/cli/"

test: ## Run tests using gotestsum (via go tool)
	@echo "Running tests..."
	go tool gotestsum --format pkgname -- -v ./...

test-watch: ## Run tests in watch mode using gotestsum (via go tool)
	@echo "Running tests in watch mode..."
	go tool gotestsum --watch -- -v ./...
