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
    UV_INSTALL_DIR=/usr/local/bin \
    UV_PATH=/usr/local/bin/uv \
    HOST=0.0.0.0 \
    PORT=8080 \
    DATABASE_PATH=/app/data/scriberr.db \
    UPLOAD_DIR=/app/data/uploads \
    WHISPERX_ENV=/app/data/whisperx-env

WORKDIR /app

# System deps: curl for uv install, ca-certs, ffmpeg for yt-dlp, git for git+ installs
RUN apt-get update \
  && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
       curl ca-certificates ffmpeg git \
  && rm -rf /var/lib/apt/lists/*

# Install uv (fast Python package manager). The script installs under
# /root/.local/bin by default; ensure it's available at /usr/local/bin/uv.
RUN curl -LsSf https://astral.sh/uv/install.sh | sh \
  && if [ -x /root/.local/bin/uv ]; then ln -sf /root/.local/bin/uv ${UV_INSTALL_DIR}/uv; fi

# Add non-root user and data directory
RUN useradd -m -u 10001 appuser \
  && mkdir -p /app/data/uploads /app/data/transcripts /app/data/whisperx-env \
  && chown -R appuser:appuser /app

# Copy binary
COPY --from=go-builder /out/scriberr /app/scriberr

# Expose port and declare volume for persistence
EXPOSE 8080
VOLUME ["/app/data"]

USER appuser

# Default command
ENTRYPOINT ["/app/scriberr"]
