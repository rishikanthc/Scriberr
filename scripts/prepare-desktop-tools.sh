#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$ROOT_DIR/dist/desktop-tools"

mkdir -p "$OUT_DIR"
rm -rf "$OUT_DIR"/*

resolve_tool() {
  local tool_name="$1"
  local env_var_name="$2"
  local source_path="${!env_var_name:-}"

  if [[ -z "$source_path" ]]; then
    source_path="$(command -v "$tool_name" || true)"
  fi

  if [[ -z "$source_path" ]]; then
    if [[ "$tool_name" == "yt-dlp" ]]; then
      local yt_dlp_url="${SCRIBERR_YTDLP_DOWNLOAD_URL:-https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_macos}"
      if command -v curl >/dev/null 2>&1; then
        echo "yt-dlp not found in PATH; downloading from $yt_dlp_url"
        if ! curl -fsSL "$yt_dlp_url" -o "$OUT_DIR/yt-dlp"; then
          echo "Failed to download yt-dlp automatically. Set SCRIBERR_YTDLP_SOURCE to a local yt-dlp binary path." >&2
          exit 1
        fi
        chmod +x "$OUT_DIR/yt-dlp"
        echo "Bundled yt-dlp from $yt_dlp_url"
        return
      fi
    fi

    echo "Missing required tool '$tool_name'. Install it or set $env_var_name to an absolute path." >&2
    exit 1
  fi

  if [[ ! -f "$source_path" ]]; then
    echo "Tool path for '$tool_name' is not a file: $source_path" >&2
    exit 1
  fi

  cp -L "$source_path" "$OUT_DIR/$tool_name"
  chmod +x "$OUT_DIR/$tool_name"
  echo "Bundled $tool_name from $source_path"
}

resolve_tool "uv" "SCRIBERR_UV_SOURCE"
resolve_tool "ffmpeg" "SCRIBERR_FFMPEG_SOURCE"
resolve_tool "ffprobe" "SCRIBERR_FFPROBE_SOURCE"
resolve_tool "yt-dlp" "SCRIBERR_YTDLP_SOURCE"

echo "Desktop tools prepared in $OUT_DIR"
