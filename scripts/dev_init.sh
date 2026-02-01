#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

NODE_VERSION_DEFAULT="20"

log() {
  echo "$*"
}

warn() {
  echo "⚠️  $*"
}

ensure_command() {
  local cmd="$1"
  if command -v "$cmd" >/dev/null 2>&1; then
    return 0
  fi
  return 1
}

detect_os() {
  local uname_out
  uname_out="$(uname -s)"
  case "$uname_out" in
    Darwin) echo "macos" ;;
    Linux) echo "linux" ;;
    *) echo "unknown" ;;
  esac
}

install_brew() {
  if ensure_command brew; then
    return 0
  fi
  log "Installing Homebrew..."
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  if [[ -f /opt/homebrew/bin/brew ]]; then
    eval "$(/opt/homebrew/bin/brew shellenv)"
  elif [[ -f /usr/local/bin/brew ]]; then
    eval "$(/usr/local/bin/brew shellenv)"
  fi
}

install_go_macos() {
  if ensure_command go; then
    return 0
  fi
  log "Installing Go via Homebrew..."
  brew install go
}

install_go_linux() {
  if ensure_command go; then
    return 0
  fi
  log "Installing Go via apt..."
  sudo apt-get update -y
  sudo apt-get install -y golang-go
}

install_linux_base() {
  log "Installing Linux base dependencies (curl, git, build tools)..."
  sudo apt-get update -y
  sudo apt-get install -y curl git build-essential
}

install_nvm() {
  if [[ -d "${NVM_DIR:-$HOME/.nvm}" ]]; then
    return 0
  fi
  log "Installing nvm..."
  curl -fsSL https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
}

load_nvm() {
  export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
  if [[ -s "$NVM_DIR/nvm.sh" ]]; then
    # shellcheck disable=SC1090
    . "$NVM_DIR/nvm.sh"
  fi
}

install_node() {
  load_nvm
  if ! ensure_command nvm; then
    warn "nvm not available in this shell. Ensure your shell profile loads nvm."
    return 1
  fi
  local version="${NODE_VERSION:-$NODE_VERSION_DEFAULT}"
  log "Installing Node.js v${version} via nvm..."
  nvm install "$version"
  nvm use "$version"
  nvm alias default "$version"
}

install_uv() {
  if ensure_command uv; then
    return 0
  fi
  log "Installing uv..."
  curl -LsSf https://astral.sh/uv/install.sh | sh
}

install_node_deps() {
  log "Installing frontend dependencies..."
  (cd "$ROOT_DIR/web/frontend" && npm install)
  log "Installing project site dependencies..."
  (cd "$ROOT_DIR/web/project-site" && npm install)
}

setup_engines() {
  log "Setting up ASR engine dependencies..."
  (cd "$ROOT_DIR" && make asr-engine-setup)
  log "Setting up diarization engine dependencies..."
  (cd "$ROOT_DIR" && make diar-engine-setup)
}

main() {
  if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    if [[ "${ALLOW_ROOT:-}" != "1" ]]; then
      warn "Running as root is not recommended. Re-run with ALLOW_ROOT=1 if this is intentional."
      exit 1
    fi
    warn "Running as root (ALLOW_ROOT=1 set). Proceeding."
  fi
  local os
  os="$(detect_os)"
  if [[ "$os" == "unknown" ]]; then
    warn "Unsupported OS. Please install Go, Node (nvm), uv manually."
    exit 1
  fi

  if [[ "$os" == "macos" ]]; then
    install_brew
    install_go_macos
  else
    install_linux_base
    install_go_linux
  fi

  install_nvm
  install_node
  install_uv
  install_node_deps
  setup_engines

  log "Dev environment setup complete."
  log "Run: make dev"
}

main "$@"
