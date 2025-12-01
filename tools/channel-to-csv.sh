#!/bin/bash
#
# YouTube Channel to CSV Exporter
# Wrapper script that uses the Scriberr environment
#
# Usage:
#   ./channel-to-csv.sh <channel_url> [options]
#
# Examples:
#   ./channel-to-csv.sh https://www.youtube.com/@ChannelName
#   ./channel-to-csv.sh https://www.youtube.com/@ChannelName -o videos.csv -v
#   ./channel-to-csv.sh https://www.youtube.com/@ChannelName --limit 100
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Check if yt-dlp is available directly
if command -v yt-dlp &> /dev/null; then
    python3 "$SCRIPT_DIR/channel-to-csv.py" "$@"
# Otherwise try using UV with whisperx environment
elif [ -f "$PROJECT_ROOT/data/whisperx-env/pyproject.toml" ]; then
    UV_PATH="${UV_PATH:-uv}"
    "$UV_PATH" run --native-tls --project "$PROJECT_ROOT/data/whisperx-env" \
        python "$SCRIPT_DIR/channel-to-csv.py" "$@"
else
    echo "Error: yt-dlp not found. Please install it:" >&2
    echo "  pip install yt-dlp" >&2
    echo "  # or" >&2
    echo "  brew install yt-dlp" >&2
    exit 1
fi
