#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if ! command -v npm >/dev/null 2>&1; then
  echo "npm is required but was not found in PATH." >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go is required but was not found in PATH." >&2
  exit 1
fi

echo "Building frontend for embed..."
cd "$ROOT_DIR/web/frontend"
npm ci
npm run build

echo "Copying frontend assets into Go embed path..."
cd "$ROOT_DIR"
rm -rf internal/web/dist
cp -r web/frontend/dist internal/web/

echo "Building backend binary for Electron packaging..."
mkdir -p dist/desktop-backend
go build -o dist/desktop-backend/scriberr cmd/server/main.go
chmod +x dist/desktop-backend/scriberr

echo "Desktop backend prepared at dist/desktop-backend/scriberr"
