#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE_DIR="$ROOT_DIR/asr-engines/scriberr-asr-onnx"

echo "🧠 ASR engine dev setup"

if ! command -v uv >/dev/null 2>&1; then
  echo "⚠️  'uv' not found. Installing via official installer..."
  curl -LsSf https://astral.sh/uv/install.sh | sh
  if ! command -v uv >/dev/null 2>&1; then
    echo "❌ 'uv' still not found. Add it to PATH and re-run."
    exit 1
  fi
  echo "✅ 'uv' installed"
fi

if [ ! -d "$ENGINE_DIR" ]; then
  echo "❌ Missing ASR engine directory: $ENGINE_DIR"
  exit 1
fi

echo "📦 Syncing ASR engine deps..."
cd "$ENGINE_DIR"
uv sync

echo "✅ ASR engine dev setup complete"
