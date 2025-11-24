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
	cd web/landing && npm run dev

website-build: docs ## Build project website for GitHub Pages
	@echo "Building project website..."
	cd web/landing && npm run build
	@echo "✓ Website built to /docs directory"

website-serve: website-build ## Build and preview project website locally
	@echo "Previewing website..."
	cd web/landing && npm run preview

docs-serve: website-serve ## Alias for website-serve
