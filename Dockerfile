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
    PUID=10001 \
    PGID=10001

WORKDIR /app

# System deps: curl for uv install, ca-certs, ffmpeg for yt-dlp, git for git+ installs
RUN apt-get update \
  && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
       curl ca-certificates ffmpeg git \
  && rm -rf /var/lib/apt/lists/*

# Install uv (fast Python package manager) directly to system PATH
RUN curl -LsSf https://astral.sh/uv/install.sh | sh \
  && cp /root/.local/bin/uv /usr/local/bin/uv \
  && chmod 755 /usr/local/bin/uv \
  && uv --version

# Add non-root user and data directory using configurable UID/GID
RUN groupadd -g ${PGID} appuser \
  && useradd -m -u ${PUID} -g ${PGID} appuser \
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

USER appuser

# Verify uv is available for appuser
RUN uv --version

# Use entrypoint script that handles permissions
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["/app/scriberr"]
