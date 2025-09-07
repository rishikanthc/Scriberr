# Multi-stage build for Scriberr: builds React UI and Go server, then
# ships a slim runtime with Python, uv, and ffmpeg for WhisperX/yt-dlp.

########################
# UI build stage
########################
FROM node:20-alpine AS ui-builder
WORKDIR /web

# Install deps and build web/frontend
COPY web/frontend/package*.json ./frontend/
RUN cd frontend \
  && npm ci

COPY web/frontend ./frontend
RUN cd frontend \
  && npm run build


########################
# Go build stage
########################
FROM golang:1.24-bookworm AS go-builder
WORKDIR /src

# Pre-cache modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Copy built UI into embed path
RUN rm -rf internal/web/dist && mkdir -p internal/web
COPY --from=ui-builder /web/frontend/dist internal/web/dist

# Build binary (arch matches builder platform)
RUN CGO_ENABLED=0 \
  go build -o /out/scriberr cmd/server/main.go


########################
# Runtime stage
########################
FROM python:3.11-slim AS runtime

ENV PYTHONUNBUFFERED=1 \
    HOST=0.0.0.0 \
    PORT=8080 \
    DATABASE_PATH=/app/data/scriberr.db \
    UPLOAD_DIR=/app/data/uploads \
    PUID=1000 \
    PGID=1000

WORKDIR /app

# System deps: curl for uv install, ca-certs, ffmpeg for yt-dlp, git for git+ installs, gosu for user switching
# Build tools: gcc, g++, make for compiling Python C extensions (needed for NeMo dependencies like texterrors)
RUN apt-get update \
  && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
       curl ca-certificates ffmpeg git gosu \
       build-essential gcc g++ make python3-dev \
  && rm -rf /var/lib/apt/lists/*

# Install uv (fast Python package manager) directly to system PATH
RUN curl -LsSf https://astral.sh/uv/install.sh | sh \
  && cp /root/.local/bin/uv /usr/local/bin/uv \
  && chmod 755 /usr/local/bin/uv \
  && uv --version

# Create default user (will be modified at runtime if needed)
RUN groupadd -g 1000 appuser \
  && useradd -m -u 1000 -g 1000 appuser \
  && mkdir -p /app/data/uploads /app/data/transcripts \
  && chown -R appuser:appuser /app

# Copy binary and entrypoint script
COPY --from=go-builder /out/scriberr /app/scriberr
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

# Make entrypoint script executable and set up basic permissions
RUN chmod +x /usr/local/bin/docker-entrypoint.sh \
  && chown appuser:appuser /app/scriberr

# Expose port and declare volume for persistence
EXPOSE 8080
VOLUME ["/app/data"]

# Start as root to allow user ID changes, entrypoint script will switch users
# Verify uv is available
RUN uv --version

# Use entrypoint script that handles user switching and permissions
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["/app/scriberr"]
