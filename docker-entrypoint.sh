#!/bin/bash
set -e

# Function to safely create directory
ensure_dir() {
    local dir="$1"
    
    # Try to create directory if it doesn't exist
    if [ ! -d "$dir" ]; then
        if ! mkdir -p "$dir" 2>/dev/null; then
            echo "Warning: Could not create directory $dir"
            return 1
        fi
    fi
    
    # Check if we can write to the directory
    if [ ! -w "$dir" ]; then
        echo "Error: No write permission to $dir"
        echo "For bind mounts, please ensure the host directory has correct permissions:"
        echo "  sudo chown -R $(id -u):$(id -g) /path/to/host/directory"
        echo "Or set container user ID to match your host user:"
        echo "  docker run -e PUID=\$(id -u) -e PGID=\$(id -g) ..."
        echo "Or run container with --user root to auto-fix permissions"
        return 1
    fi
    
    echo "✓ Directory $dir is writable"
    return 0
}

echo "=== Scriberr Container Setup ==="
echo "Running as UID=$(id -u), GID=$(id -g)"

# Ensure required directories exist and are writable
echo "Setting up data directories..."

if ! ensure_dir "/app/data"; then
    echo "Failed to set up /app/data directory"
    exit 1
fi

ensure_dir "/app/data/uploads" || true
ensure_dir "/app/data/transcripts" || true

# Create whisperx-env in working directory (not under mounted volume)
# This avoids permission issues with bind mounts
echo "Setting up Python environment directory..."
if ! ensure_dir "/app/whisperx-env"; then
    echo "Failed to set up Python environment directory"
    exit 1
fi

echo "=== Setup Complete ==="
echo "Starting Scriberr application..."

# Execute the main command
exec "$@"