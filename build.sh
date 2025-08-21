#!/bin/bash

# Scriberr Build Script
# This script builds the React frontend and embeds it in the Go binary

set -e  # Exit on any error

echo "🚀 Starting Scriberr build process..."

# Step 1: Clean up old files
echo "🧹 Cleaning up old build files..."
rm -f scriberr
rm -rf internal/web/dist
cd web/frontend

# Remove old build output and copied files
rm -rf dist/
rm -rf assets/ 2>/dev/null || true

echo "✅ Old files cleaned"

# Step 2: Build React frontend
echo "📦 Building React frontend..."
npm run build
echo "✅ React frontend built successfully"

# Step 3: Copy dist files to internal/web for embedding
echo "📁 Copying dist files for Go embedding..."
cd ../..
rm -rf internal/web/dist
cp -r web/frontend/dist internal/web/
echo "✅ Dist files copied to internal/web"

# Step 4: Clean Go build cache and rebuild binary
echo "🔨 Building Go binary with embedded static files..."
go clean -cache
go build -o scriberr cmd/server/main.go
echo "✅ Go binary built successfully"

echo "🎉 Build complete! Run './scriberr' to start the server"