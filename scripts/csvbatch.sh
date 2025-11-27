#!/bin/bash
set -euo pipefail

# CSV Batch Processing Script for Scriberr
# Simple wrapper for REST API batch transcription

# Configuration
SERVER="${SERVER:-http://localhost:8080}"
LOG_FILE="${LOG_FILE:-csvbatch.log}"

# Colors
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'

# Variables
CSV_FILE=""; OUTPUT_DIR=""; MODEL=""; DEVICE=""; BATCH_ID=""; RESUME_MODE=false; LIST_MODE=false

# Logging
log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"; }
error() { echo -e "${RED}Error: $*${NC}" >&2; log "ERROR: $*"; }
success() { echo -e "${GREEN}$*${NC}"; log "SUCCESS: $*"; }
info() { echo -e "${YELLOW}$*${NC}"; log "INFO: $*"; }

# Cleanup handler
cleanup() {
  echo ""
  if [ -n "$BATCH_ID" ]; then
    info "Stopping batch $BATCH_ID..."
    curl -s -X POST -H "Authorization: Bearer $API_KEY" \
      "$SERVER/api/v1/csv-batch/$BATCH_ID/stop" > /dev/null 2>&1 || true
    info "Batch stopped. Use --resume $BATCH_ID to continue."
  fi
  exit 1
}
trap cleanup SIGINT SIGTERM

# Usage
usage() {
  cat << EOF
${BLUE}CSV Batch Processing Script for Scriberr${NC}

Usage: $0 [OPTIONS]

Options:
  --csv FILE          CSV file with YouTube URLs (one per line)
  --output-dir DIR    Output directory for transcripts
  --model MODEL       Whisper model (tiny/base/small/medium/large)
  --device DEVICE     Device (cpu/cuda)
  --resume ID         Resume existing batch
  --list              List all batches
  -h, --help          Show help

Environment:
  API_KEY             Required. API key for authentication
  SERVER              Server URL (default: http://localhost:8080)
  LOG_FILE            Log file path (default: csvbatch.log)

Examples:
  $0 --csv urls.csv
  $0 --csv urls.csv --model medium --device cuda
  $0 --resume abc123
  $0 --list
EOF
  exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --csv) CSV_FILE="$2"; shift 2 ;;
    --output-dir) OUTPUT_DIR="$2"; shift 2 ;;
    --model) MODEL="$2"; shift 2 ;;
    --device) DEVICE="$2"; shift 2 ;;
    --resume) RESUME_MODE=true; BATCH_ID="$2"; shift 2 ;;
    --list) LIST_MODE=true; shift ;;
    -h|--help) usage ;;
    *) error "Unknown option: $1"; usage ;;
  esac
done

# Check dependencies
for cmd in curl jq; do
  if ! command -v "$cmd" &> /dev/null; then
    error "$cmd is required but not installed"
    exit 1
  fi
done

# Check API_KEY
if [ -z "${API_KEY:-}" ]; then
  error "API_KEY environment variable not set"
  echo "  Example: export API_KEY=your-api-key-here"
  exit 1
fi

# List mode
if [ "$LIST_MODE" = true ]; then
  log "Listing all batches..."
  curl -s -H "Authorization: Bearer $API_KEY" "$SERVER/api/v1/csv-batch" | jq .
  exit 0
fi

# Resume mode
if [ "$RESUME_MODE" = true ]; then
  [ -z "$BATCH_ID" ] && { error "--resume requires batch ID"; exit 1; }
  log "Resuming batch $BATCH_ID..."
else
  # Normal mode - upload CSV
  [ -z "$CSV_FILE" ] && read -p "Enter CSV file path: " CSV_FILE
  [ ! -f "$CSV_FILE" ] && { error "CSV file not found: $CSV_FILE"; exit 1; }

  log "Uploading CSV file: $CSV_FILE"
  UPLOAD_RESPONSE=$(curl -s -X POST -H "Authorization: Bearer $API_KEY" \
    -F "file=@$CSV_FILE" "$SERVER/api/v1/csv-batch/upload")

  BATCH_ID=$(echo "$UPLOAD_RESPONSE" | jq -r '.id // empty')
  if [ -z "$BATCH_ID" ]; then
    error "Failed to upload CSV"
    echo "$UPLOAD_RESPONSE" | jq . 2>/dev/null || echo "$UPLOAD_RESPONSE"
    exit 1
  fi

  success "Batch created with ID: $BATCH_ID"

  # Build start payload
  PAYLOAD="{}"
  [ -n "$MODEL" ] && PAYLOAD=$(echo "$PAYLOAD" | jq --arg m "$MODEL" '. + {model: $m}')
  [ -n "$DEVICE" ] && PAYLOAD=$(echo "$PAYLOAD" | jq --arg d "$DEVICE" '. + {device: $d}')
  [ -n "$OUTPUT_DIR" ] && PAYLOAD=$(echo "$PAYLOAD" | jq --arg o "$OUTPUT_DIR" '. + {outputDir: $o}')

  # Start batch
  log "Starting batch processing..."
  START_RESPONSE=$(curl -s -X POST -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" -d "$PAYLOAD" \
    "$SERVER/api/v1/csv-batch/$BATCH_ID/start")

  START_STATUS=$(echo "$START_RESPONSE" | jq -r '.status // empty')
  if [ -z "$START_STATUS" ]; then
    error "Failed to start batch"
    echo "$START_RESPONSE" | jq . 2>/dev/null || echo "$START_RESPONSE"
    exit 1
  fi
fi

# Poll status
info "Polling status (Ctrl+C to stop)..."
LAST_CURRENT=0

while true; do
  STATUS_RESPONSE=$(curl -s -H "Authorization: Bearer $API_KEY" \
    "$SERVER/api/v1/csv-batch/$BATCH_ID/status")

  STATUS=$(echo "$STATUS_RESPONSE" | jq -r '.status // "unknown"')
  TOTAL=$(echo "$STATUS_RESPONSE" | jq -r '.total_rows // 0')
  CURRENT=$(echo "$STATUS_RESPONSE" | jq -r '.current_row // 0')
  SUCCESS=$(echo "$STATUS_RESPONSE" | jq -r '.success_rows // 0')
  FAILED=$(echo "$STATUS_RESPONSE" | jq -r '.failed_rows // 0')

  # Log progress changes
  if [ "$CURRENT" -ne "$LAST_CURRENT" ]; then
    log "Status: $STATUS | Progress: $CURRENT/$TOTAL (Success: $SUCCESS, Failed: $FAILED)"
    LAST_CURRENT=$CURRENT
  fi

  # Check completion
  case "$STATUS" in
    completed)
      success "Batch completed! Total: $TOTAL, Success: $SUCCESS, Failed: $FAILED"
      exit 0
      ;;
    failed)
      error "Batch processing failed"
      exit 1
      ;;
    cancelled)
      info "Batch cancelled. Resume with: $0 --resume $BATCH_ID"
      exit 0
      ;;
  esac

  sleep 5
done
