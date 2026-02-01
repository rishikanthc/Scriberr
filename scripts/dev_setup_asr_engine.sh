#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE_DIR="$ROOT_DIR/asr-engines/scriberr-asr-onnx"

echo "ASR engine dev setup"

if ! command -v uv >/dev/null 2>&1; then
  echo "⚠️  'uv' not found. Installing via official installer..."
  curl -LsSf https://astral.sh/uv/install.sh | sh
  if ! command -v uv >/dev/null 2>&1; then
    echo "❌ 'uv' still not found. Add it to PATH and re-run."
    exit 1
  fi
  echo "✅ 'uv' installed"
fi

resolve_asr_extra() {
  if [ -n "${ASR_ENGINE_EXTRA:-}" ]; then
    echo "$ASR_ENGINE_EXTRA"
    return
  fi
  case "${ASR_ENGINE_DEVICE:-}" in
    cpu|gpu)
      echo "$ASR_ENGINE_DEVICE"
      return
      ;;
  esac
  if command -v nvidia-smi >/dev/null 2>&1 || [ -e /dev/nvidia0 ] || [ -e /dev/nvidiactl ]; then
    echo "gpu"
  else
    echo "cpu"
  fi
}

if [ ! -d "$ENGINE_DIR" ]; then
  echo "❌ Missing ASR engine directory: $ENGINE_DIR"
  exit 1
fi

if [ -d "$ENGINE_DIR/.venv" ]; then
  if ! uv run --project "$ENGINE_DIR" python -c "import sys" >/dev/null 2>&1; then
    echo "⚠️  Detected broken .venv. Recreating..."
    rm -rf "$ENGINE_DIR/.venv"
  fi
fi

echo "Syncing ASR engine deps..."
cd "$ENGINE_DIR"
uv python install 3.11 >/dev/null 2>&1 || true
uv venv --python 3.11 >/dev/null 2>&1 || true
ASR_EXTRA="$(resolve_asr_extra)"
echo "Using ASR engine dependency profile: $ASR_EXTRA"
uv sync --extra "$ASR_EXTRA"

echo "✅ ASR engine dev setup complete"
