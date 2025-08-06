# Multi-stage build for frontend
FROM --platform=$BUILDPLATFORM node:20 AS frontend-builder

# Set working directory for frontend
WORKDIR /frontend

# Copy frontend files
COPY scriberr-frontend/package*.json ./

# Install frontend dependencies
RUN npm install

# Copy frontend source
COPY scriberr-frontend/ .

# Build frontend
RUN npm run build

# Create target directory for embedded assets
RUN mkdir -p /app/cmd/scriberr/embedded_assets

# Copy built frontend to embedded assets
RUN cp -r ./build/* /app/cmd/scriberr/embedded_assets/

# Build stage for Go application
FROM --platform=$BUILDPLATFORM golang:1.24 AS builder

# Install build dependencies (only what's needed for Go build)
RUN apt-get update && apt-get install -y \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy go mod files
COPY ./scriberr-backend/go.mod ./scriberr-backend/go.sum ./

# Dow load dependencies
RUN go mod download

# Copy source code (including embedded assets)
COPY ./scriberr-backend/ .

# Copy frontend assets from frontend-builder stage
COPY --from=frontend-builder /app/cmd/scriberr/embedded_assets ./cmd/scriberr/embedded_assets

# Build the binary with proper architecture flags
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

# Disable CGO for cross-compilation (using pure Go SQLite driver)
ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

# Build the binary with architecture-specific optimizations
RUN go build -ldflags="-s -w" -o scriberr ./cmd/scriberr

# Make the binary executable
RUN chmod +x /app/scriberr

# Final stage - use python base image for reliability
FROM --platform=$TARGETPLATFORM python:3.12-slim

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        sqlite3 \
        ffmpeg \
        curl \
        build-essential \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/*

# Copy the binary and Python dependencies
COPY --from=builder /app/scriberr /app/scriberr
COPY --from=builder /app/pyproject.toml /app/pyproject.toml
COPY --from=builder /app/uv.lock /app/uv.lock
COPY --from=builder /app/diarize.py /app/diarize.py

# Install uv and dependencies
RUN pip install --upgrade pip && \
    pip install --root-user-action ignore uv

# Set working directory for uv operations
WORKDIR /app

# Install Python dependencies
RUN uv sync --frozen

# Create storage directory for database and files
RUN mkdir -p /app/storage

# Expose port
EXPOSE 8080

# Set environment variables with defaults
ENV OPENAI_API_KEY=""
ENV SESSION_KEY=""
ENV HF_TOKEN=""
ENV SCRIBERR_USERNAME=""
ENV SCRIBERR_PASSWORD=""
ENV OLLAMA_BASE_URL=""

# Set the entrypoint to run the binary
ENTRYPOINT ["/app/scriberr"]
