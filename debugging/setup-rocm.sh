#!/bin/bash

# Manual ROCm setup script for Scriberr
echo "=== Manual ROCm Setup for Scriberr ==="

# Set default environment variables if not set
export HSA_OVERRIDE_GFX_VERSION=${HSA_OVERRIDE_GFX_VERSION:-11.0.0}
export PYTORCH_ROCM_ARCH=${PYTORCH_ROCM_ARCH:-gfx1100}
export ROCM_PATH=${ROCM_PATH:-/opt/rocm}

echo "Using HSA_OVERRIDE_GFX_VERSION: $HSA_OVERRIDE_GFX_VERSION"
echo "Using PYTORCH_ROCM_ARCH: $PYTORCH_ROCM_ARCH"
echo "Using ROCM_PATH: $ROCM_PATH"

# Create WhisperX environment directory
WHISPERX_ENV="./whisperx-env"
WHISPERX_PATH="$WHISPERX_ENV/WhisperX"

echo ""
echo "Setting up WhisperX environment at: $WHISPERX_PATH"

# Remove existing environment if it exists
if [ -d "$WHISPERX_ENV" ]; then
    echo "Removing existing environment..."
    rm -rf "$WHISPERX_ENV"
fi

# Create directory
mkdir -p "$WHISPERX_ENV"

# Clone WhisperX
echo "Cloning WhisperX repository..."
cd "$WHISPERX_ENV"
git clone https://github.com/m-bain/WhisperX.git

# Check if ROCm is available
echo ""
echo "Checking ROCm availability..."
ROCM_AVAILABLE=false
if python3 -c "import torch; print(hasattr(torch, 'hip') and torch.hip.is_available())" 2>/dev/null | grep -q "True"; then
    ROCM_AVAILABLE=true
    echo "ROCm detected - using ROCm-compatible ctranslate2 fork"
else
    echo "ROCm not detected - using standard ctranslate2"
fi

# Update pyproject.toml
echo ""
echo "Updating pyproject.toml dependencies..."
if [ "$ROCM_AVAILABLE" = true ]; then
    # Use ROCm fork
    sed -i 's/ctranslate2<4.5.0/ctranslate2 @ git+https:\/\/github.com\/arlo-phoenix\/CTranslate2.git@rocm/' "$WHISPERX_PATH/pyproject.toml"
    sed -i 's/ctranslate2==4.6.0/ctranslate2 @ git+https:\/\/github.com\/arlo-phoenix\/CTranslate2.git@rocm/' "$WHISPERX_PATH/pyproject.toml"
else
    # Use standard ctranslate2
    sed -i 's/ctranslate2<4.5.0/ctranslate2==4.6.0/' "$WHISPERX_PATH/pyproject.toml"
fi

# Add yt-dlp if not present
if ! grep -q "yt-dlp" "$WHISPERX_PATH/pyproject.toml"; then
    echo "Adding yt-dlp dependency..."
    sed -i 's/"transformers>=4.48.0",/"transformers>=4.48.0",\n    "yt-dlp",/' "$WHISPERX_PATH/pyproject.toml"
fi

# Install dependencies
echo ""
echo "Installing dependencies with uv sync..."
cd "$WHISPERX_PATH"
uv sync --all-extras --dev --native-tls

echo ""
echo "Setup complete! You can now test with:"
echo "uv run --project $WHISPERX_PATH python -c \"import whisperx; print('WhisperX ready')\""
echo ""
echo "For transcription, use:"
echo "uv run --project $WHISPERX_PATH python -m whisperx <audio_file> --device cuda"