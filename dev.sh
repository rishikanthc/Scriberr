#!/bin/bash

# Scriberr Development Environment Script
# This script starts both the Go backend (with Air live reload) and the React frontend (with Vite HMR) concurrently.

set -e

# --- Helper Functions ---

cleanup() {
    echo ""
    echo "🛑 Stopping development servers..."
    kill $(jobs -p) 2>/dev/null || true
    echo "✅ Stopped."
    exit 0
}

# Trap signals for cleanup
trap cleanup SIGINT SIGTERM

# --- Step 1: Ensure Air is installed ---

if ! command -v air &> /dev/null; then
    echo "⚠️  'air' command not found."
    echo "📦 Auto-installing 'air' for live reload..."
    
    # Check if GOPATH/bin is in PATH
    GOPATH=$(go env GOPATH)
    if [[ ":$PATH:" != *":$GOPATH/bin:"* ]]; then
        echo "⚠️  $GOPATH/bin is not in your PATH. Adding it temporarily..."
        export PATH=$PATH:$GOPATH/bin
    fi

    go install github.com/air-verse/air@latest

    if ! command -v air &> /dev/null; then
        echo "❌ Failed to install 'air'. Please install it manually: go install github.com/air-verse/air@latest"
        echo "🔄 Falling back to 'go run' (no live reload for backend)..."
        USE_GO_RUN=true
    else
        echo "✅ 'air' installed successfully."
        USE_GO_RUN=false
    fi
else
    USE_GO_RUN=false
fi

# --- Step 2: Ensure Embed Directory Exists & is Populated ---

DIST_DIR="internal/web/dist"
if [ ! -d "$DIST_DIR" ]; then
    echo "📁 Creating placeholder dist directory for Go embed..."
    mkdir -p "$DIST_DIR"
fi

# Go's embed directive requires at least one file to match "dist/*" 
# If the directory is empty, create a dummy file to prevent compilation errors.
if [ -z "$(ls -A $DIST_DIR)" ]; then
    echo "📄 Creating dummy index.html to satisfy embed directive..."
    echo "<!-- Placeholder for development. In dev mode, the frontend is served by Vite proxy. -->" > "$DIST_DIR/index.html"
    # Also add a dummy asset to satisfy the static routes if needed
    echo "placeholder" > "$DIST_DIR/dummy_asset"
fi

# --- Step 3: Start Servers ---

echo "🚀 Starting development environment..."

# Start Backend
if [ "$USE_GO_RUN" = true ]; then
    echo "🔧 Starting Go backend (standard run)..."
    go run cmd/server/main.go &
else
    echo "🔥 Starting Go backend (with Air live reload)..."
    air &
fi

# Start Frontend
echo "⚛️  Starting React frontend (Vite)..."
cd web/frontend
npm run dev &

# Wait for all background processes
wait
