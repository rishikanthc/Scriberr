#!/bin/bash

# Platform-specific dependency installation for WhisperLive

echo "Installing WhisperLive dependencies..."

# Detect platform
PLATFORM=$(uname -s)
ARCH=$(uname -m)

echo "Platform: $PLATFORM"
echo "Architecture: $ARCH"

# Install base dependencies
echo "Installing base dependencies..."
if ! uv pip install -r requirements.txt; then
    echo "Full requirements failed, trying minimal requirements..."
    uv pip install -r requirements_minimal.txt
fi

# Handle macOS ARM64 specific issues
if [[ "$PLATFORM" == "Darwin" && "$ARCH" == "arm64" ]]; then
    echo "Detected macOS ARM64 - handling platform-specific dependencies..."
    
    # Try to install kaldialign with source compilation if needed
    if ! uv pip install kaldialign; then
        echo "Attempting to install kaldialign from source..."
        uv pip install kaldialign --no-binary :all:
    fi
    
    # Install additional macOS-specific dependencies if needed
    echo "Installing macOS-specific audio dependencies..."
    # Check if portaudio is installed via Homebrew
    if ! brew list portaudio &>/dev/null; then
        echo "Installing portaudio via Homebrew..."
        brew install portaudio
    fi
    
    # Try to install pyaudio with portaudio headers available
    if ! uv pip install pyaudio; then
        echo "Warning: pyaudio installation failed. Audio recording may not work."
        echo "You can try: brew install portaudio && uv pip install pyaudio"
    fi
fi

echo "Dependencies installation complete!" 