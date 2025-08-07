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

# New stage for Python dependencies
FROM --platform=$TARGETPLATFORM python:3.12-slim AS python-deps

# Install build dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        build-essential \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/*

# Install uv
RUN pip install --no-cache-dir uv

# Set working directory
WORKDIR /app

# Copy Python dependency files
COPY --from=builder /app/pyproject.toml /app/uv.lock ./

# Create and activate virtual environment
RUN python -m venv /app/.venv
ENV PATH="/app/.venv/bin:$PATH"

# Install Python dependencies using uv sync
RUN uv sync --frozen

# Final stage
FROM --platform=$TARGETPLATFORM python:3.12-slim

# Install runtime dependencies and build tools for uv
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        sqlite3 \
        ffmpeg \
        curl \
        python3-pip \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/*

# Install uv system-wide
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir --root-user-action ignore uv

# Copy the binary from builder
COPY --from=builder /app/scriberr /app/scriberr

# Copy virtual environment and lock file from python-deps stage
COPY --from=python-deps /app/.venv /app/.venv
COPY --from=python-deps /app/uv.lock /app/uv.lock

# Ensure virtual environment is in PATH
ENV PATH="/app/.venv/bin:$PATH"

# Copy necessary Python files
COPY --from=builder /app/diarize.py /app/diarize.py

# Set working directory
WORKDIR /app

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
