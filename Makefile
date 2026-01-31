.PHONY: help docs docs-serve docs-clean website website-dev website-build dev init asr-engine-dev asr-engine-setup diar-engine-dev diar-engine-setup

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

init: ## Install dev prerequisites (Go, Node via nvm, uv) and setup engines
	@bash scripts/dev_init.sh

dev: ## Start development environment with Air (backend) and Vite (frontend)
	@echo "Starting development environment..."
	@# Ensure air is installed
	@GOPATH=$$(go env GOPATH); \
	if [[ ":$$PATH:" != *":$$GOPATH/bin:"* ]]; then \
		echo "⚠️  $$GOPATH/bin is not in your PATH. Adding it temporarily..."; \
		export PATH=$$PATH:$$GOPATH/bin; \
	fi; \
	if ! command -v air >/dev/null 2>&1; then \
		echo "⚠️  'air' command not found."; \
		echo "Auto-installing 'air' for live reload..."; \
		go install github.com/air-verse/air@latest; \
		if ! command -v air >/dev/null 2>&1; then \
			echo "❌ Failed to install 'air'. Falling back to 'go run'..."; \
			USE_GO_RUN=true; \
		else \
			echo "✅ 'air' installed successfully."; \
			USE_GO_RUN=false; \
		fi; \
	else \
		USE_GO_RUN=false; \
	fi; \
	\
	mkdir -p internal/web/dist; \
	if [ -z "$$(ls -A internal/web/dist)" ]; then \
		echo "Creating placeholder files for Go embed..."; \
		echo "<!-- Placeholder for development -->" > internal/web/dist/index.html; \
		echo "placeholder" > internal/web/dist/dummy_asset; \
	fi; \
	\
	pids=""; \
	trap 'echo ""; echo "Stopping development servers..."; for pid in $$pids; do kill $$pid 2>/dev/null || true; done; wait $$pids 2>/dev/null || true; exit 0' INT TERM; \
	\
	echo "Starting ASR engine..."; \
	ASR_ENGINE_SOCKET=$${ASR_ENGINE_SOCKET:-/tmp/scriberr-asr.sock}; \
	ASR_ENGINE_CMD=$${ASR_ENGINE_CMD:-"uv run --project asr-engines/scriberr-asr-onnx asr-engine-server"}; \
	if command -v uv >/dev/null 2>&1; then \
		if [ -z "$${ASR_ENGINE_SKIP_SYNC}" ]; then \
			echo "Syncing ASR engine deps (set ASR_ENGINE_SKIP_SYNC=1 to skip)..."; \
			( cd asr-engines/scriberr-asr-onnx && uv sync ) || true; \
		fi; \
		( cd asr-engines/scriberr-asr-onnx && uv run asr-engine-server --socket $$ASR_ENGINE_SOCKET ) & \
		pids="$$pids $$!"; \
	else \
		echo "⚠️  'uv' not found. ASR engine will not start. Install uv or run make asr-engine-dev separately."; \
	fi; \
	\
	echo "Starting diarization engine..."; \
	DIAR_ENGINE_SOCKET=$${DIAR_ENGINE_SOCKET:-/tmp/scriberr-diar.sock}; \
	DIAR_ENGINE_CMD=$${DIAR_ENGINE_CMD:-"uv run --project asr-engines/scriberr-diariz-torch diar-engine-server"}; \
	if command -v uv >/dev/null 2>&1; then \
		if [ -z "$${DIAR_ENGINE_SKIP_SYNC}" ]; then \
			echo "Syncing diarization engine deps (set DIAR_ENGINE_SKIP_SYNC=1 to skip)..."; \
			( cd asr-engines/scriberr-diariz-torch && uv sync ) || true; \
		fi; \
		( cd asr-engines/scriberr-diariz-torch && uv run diar-engine-server --socket $$DIAR_ENGINE_SOCKET ) & \
		pids="$$pids $$!"; \
	else \
		echo "⚠️  'uv' not found. Diarization engine will not start. Install uv or run make diar-engine-dev separately."; \
	fi; \
	\
	if [ "$$USE_GO_RUN" = true ]; then \
		echo "Starting Go backend (standard run)..."; \
		ASR_ENGINE_SOCKET=$$ASR_ENGINE_SOCKET ASR_ENGINE_CMD="$$ASR_ENGINE_CMD" \
		DIAR_ENGINE_SOCKET=$$DIAR_ENGINE_SOCKET DIAR_ENGINE_CMD="$$DIAR_ENGINE_CMD" \
		go run cmd/server/main.go & \
		pids="$$pids $$!"; \
	else \
		echo "Starting Go backend (with Air live reload)..."; \
		ASR_ENGINE_SOCKET=$$ASR_ENGINE_SOCKET ASR_ENGINE_CMD="$$ASR_ENGINE_CMD" \
		DIAR_ENGINE_SOCKET=$$DIAR_ENGINE_SOCKET DIAR_ENGINE_CMD="$$DIAR_ENGINE_CMD" \
		air & \
		pids="$$pids $$!"; \
	fi; \
	\
	echo "⚛️  Starting React frontend (Vite)..."; \
	cd web/frontend && npm run dev & \
	pids="$$pids $$!"; \
	\
	wait

asr-engine-dev: ## Start ASR engine daemon for local development
	@echo "Starting ASR engine (onnx-asr)..."
	@if ! command -v uv >/dev/null 2>&1; then \
		echo "⚠️  'uv' not found. Install uv to run the ASR engine locally."; \
		exit 1; \
	fi; \
	ASR_ENGINE_SOCKET=$${ASR_ENGINE_SOCKET:-/tmp/scriberr-asr.sock}; \
	if [ -z "$${ASR_ENGINE_SKIP_SYNC}" ]; then \
		echo "Syncing ASR engine deps (set ASR_ENGINE_SKIP_SYNC=1 to skip)..."; \
		cd asr-engines/scriberr-asr-onnx && uv sync; \
	fi; \
	echo "ASR engine socket: $$ASR_ENGINE_SOCKET"; \
	cd asr-engines/scriberr-asr-onnx && uv run asr-engine-server --socket $$ASR_ENGINE_SOCKET

asr-engine-setup: ## Install uv and sync ASR engine dependencies
	@echo "Setting up ASR engine dev environment..."
	@bash scripts/dev_setup_asr_engine.sh

diar-engine-dev: ## Start diarization engine daemon for local development
	@echo "Starting diarization engine..."
	@if ! command -v uv >/dev/null 2>&1; then \
		echo "⚠️  'uv' not found. Install uv to run the diarization engine locally."; \
		exit 1; \
	fi; \
	DIAR_ENGINE_SOCKET=$${DIAR_ENGINE_SOCKET:-/tmp/scriberr-diar.sock}; \
	if [ -z "$${DIAR_ENGINE_SKIP_SYNC}" ]; then \
		echo "Syncing diarization engine deps (set DIAR_ENGINE_SKIP_SYNC=1 to skip)..."; \
		cd asr-engines/scriberr-diariz-torch && uv sync; \
	fi; \
	echo "Diarization engine socket: $$DIAR_ENGINE_SOCKET"; \
	cd asr-engines/scriberr-diariz-torch && uv run diar-engine-server --socket $$DIAR_ENGINE_SOCKET

diar-engine-setup: ## Install uv and sync diarization engine dependencies
	@echo "Setting up diarization engine dev environment..."
	@bash scripts/dev_setup_diar_engine.sh

docs: ## Generate API documentation from Go code annotations
	@echo "Generating API documentation..."
	@command -v swag >/dev/null 2>&1 || { echo "Error: swag not installed. Run: go install github.com/swaggo/swag/cmd/swag@latest"; exit 1; }
	swag init -g main.go -o api-docs --dir cmd/server,internal
	@echo "Syncing to project site..."
	swag init -g main.go -o web/project-site/public/api --outputTypes json --dir cmd/server,internal
	@echo "✓ API documentation generated in api-docs/ and web/project-site/public/api/"

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
