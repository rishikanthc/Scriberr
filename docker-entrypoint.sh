#!/bin/bash
set -euo pipefail

PUID=${PUID:-1000}
PGID=${PGID:-1000}

SCRIBERR_DATA_DIR=${SCRIBERR_DATA_DIR:-/var/lib/scriberr}
SCRIBERR_STATE_DIR=${SCRIBERR_STATE_DIR:-/opt/scriberr}
SCRIBERR_ENV_DIR=${SCRIBERR_ENV_DIR:-/opt/scriberr/env}
SCRIBERR_MODELS_DIR=${SCRIBERR_MODELS_DIR:-/opt/scriberr/models}
SCRIBERR_ENGINES_DIR=${SCRIBERR_ENGINES_DIR:-/opt/scriberr/engines-src}
UV_CACHE_DIR=${UV_CACHE_DIR:-/opt/scriberr/uv-cache}

ASR_ENGINE_EXTRA=${ASR_ENGINE_EXTRA:-cpu}
DIAR_ENGINE_EXTRA=${DIAR_ENGINE_EXTRA:-cpu}

ASR_ENV_PATH=${ASR_ENV_PATH:-$SCRIBERR_ENV_DIR/asr}
DIAR_ENV_PATH=${DIAR_ENV_PATH:-$SCRIBERR_ENV_DIR/diar}

SKIP_ENV_SETUP=${SCRIBERR_SKIP_ENV_SETUP:-0}
ENV_HASH_FILE=${SCRIBERR_ENV_HASH_FILE:-$SCRIBERR_ENV_DIR/.lockhash}

mkdir -p \
  "$SCRIBERR_DATA_DIR" \
  "$SCRIBERR_DATA_DIR/uploads" \
  "$SCRIBERR_DATA_DIR/transcripts" \
  "$SCRIBERR_DATA_DIR/temp" \
  "$SCRIBERR_STATE_DIR" \
  "$SCRIBERR_ENV_DIR" \
  "$SCRIBERR_MODELS_DIR" \
  "$UV_CACHE_DIR" \
  /run/scriberr/engines

export DATABASE_PATH=${DATABASE_PATH:-$SCRIBERR_DATA_DIR/scriberr.db}
export UPLOAD_DIR=${UPLOAD_DIR:-$SCRIBERR_DATA_DIR/uploads}
export TRANSCRIPTS_DIR=${TRANSCRIPTS_DIR:-$SCRIBERR_DATA_DIR/transcripts}
export TEMP_DIR=${TEMP_DIR:-$SCRIBERR_DATA_DIR/temp}

export ASR_ENGINE_CMD=${ASR_ENGINE_CMD:-"uv run --project $SCRIBERR_ENGINES_DIR/scriberr-asr-onnx --venv $ASR_ENV_PATH asr-engine-server"}
export DIAR_ENGINE_CMD=${DIAR_ENGINE_CMD:-"uv run --project $SCRIBERR_ENGINES_DIR/scriberr-diariz-torch --venv $DIAR_ENV_PATH diar-engine-server"}

setup_user() {
  local target_uid=$1
  local target_gid=$2

  if [ "$target_uid" != "1000" ] || [ "$target_gid" != "1000" ]; then
    if getent group "$target_gid" >/dev/null 2>&1; then
      true
    else
      groupmod -g "$target_gid" appuser 2>/dev/null || {
        groupadd -g "$target_gid" appgroup
        usermod -g "$target_gid" appuser
      }
    fi

    usermod -u "$target_uid" appuser 2>/dev/null || true
  fi

  chown -R "$target_uid:$target_gid" \
    "$SCRIBERR_DATA_DIR" \
    "$SCRIBERR_STATE_DIR" \
    "$SCRIBERR_ENV_DIR" \
    "$SCRIBERR_MODELS_DIR" \
    "$UV_CACHE_DIR" \
    /run/scriberr || true
}

install_envs() {
  if [ "$SKIP_ENV_SETUP" = "1" ]; then
    echo "Skipping env setup (SCRIBERR_SKIP_ENV_SETUP=1)"
    return 0
  fi

  export UV_CACHE_DIR

  if ! uv python list >/dev/null 2>&1; then
    uv python install 3.12 || true
  fi

  local asr_lock="$SCRIBERR_ENGINES_DIR/scriberr-asr-onnx/uv.lock"
  local diar_lock="$SCRIBERR_ENGINES_DIR/scriberr-diariz-torch/uv.lock"
  local lock_hash=""
  if command -v sha256sum >/dev/null 2>&1; then
    lock_hash="$(sha256sum "$asr_lock" "$diar_lock" | sha256sum | awk '{print $1}')"
  else
    lock_hash="$(python - <<PY
import hashlib, sys
paths = [r"$asr_lock", r"$diar_lock"]
h = hashlib.sha256()
for p in paths:
    with open(p, "rb") as f:
        h.update(f.read())
print(h.hexdigest())
PY
)"
  fi

  local current_hash=""
  if [ -f "$ENV_HASH_FILE" ]; then
    current_hash="$(cat "$ENV_HASH_FILE" 2>/dev/null || true)"
  fi

  if [ -n "$lock_hash" ] && [ "$lock_hash" != "$current_hash" ]; then
    echo "Lockfile changed; rebuilding envs"
    rm -rf "$ASR_ENV_PATH" "$DIAR_ENV_PATH"
  fi

  if [ ! -x "$ASR_ENV_PATH/bin/python" ]; then
    echo "Setting up ASR engine env ($ASR_ENGINE_EXTRA) in $ASR_ENV_PATH"
    uv venv --python 3.12 "$ASR_ENV_PATH"
    uv sync --project "$SCRIBERR_ENGINES_DIR/scriberr-asr-onnx" --extra "$ASR_ENGINE_EXTRA" --frozen --venv "$ASR_ENV_PATH"
  fi

  if [ ! -x "$DIAR_ENV_PATH/bin/python" ]; then
    echo "Setting up diarization env ($DIAR_ENGINE_EXTRA) in $DIAR_ENV_PATH"
    uv venv --python 3.12 "$DIAR_ENV_PATH"
    uv sync --project "$SCRIBERR_ENGINES_DIR/scriberr-diariz-torch" --extra "$DIAR_ENGINE_EXTRA" --frozen --venv "$DIAR_ENV_PATH"
  fi

  if [ -n "$lock_hash" ]; then
    echo "$lock_hash" > "$ENV_HASH_FILE"
  fi
}

if [ "$(id -u)" = "0" ]; then
  setup_user "$PUID" "$PGID"
  install_envs
  exec gosu appuser "$@"
else
  install_envs
  exec "$@"
fi
