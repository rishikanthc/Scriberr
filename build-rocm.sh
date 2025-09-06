#!/bin/bash

# Build ROCm Docker image for Scriberr
# This script builds the ROCm-compatible Docker image

set -e

echo "🏗️ Building Scriberr ROCm Docker image..."

# Get version from git tag or use default
VERSION=${1:-"v1.0.4-rocm"}

echo "📦 Building image: ghcr.io/rishikanthc/scriberr:$VERSION"

# Build the Docker image
docker build -f Dockerfile.rocm -t ghcr.io/rishikanthc/scriberr:$VERSION .

echo "✅ ROCm Docker image built successfully!"
echo "🚀 To run: docker run -d -p 8080:8080 --device=/dev/kfd --device=/dev/dri ghcr.io/rishikanthc/scriberr:$VERSION"
