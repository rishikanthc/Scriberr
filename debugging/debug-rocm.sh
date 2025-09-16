#!/bin/bash

# Debug script for ROCm setup in Scriberr
echo "=== ROCm Debug Script ==="
echo "Current directory: $(pwd)"
echo "User: $(whoami)"
echo "Date: $(date)"

# Check environment variables
echo ""
echo "=== Environment Variables ==="
echo "HSA_OVERRIDE_GFX_VERSION: ${HSA_OVERRIDE_GFX_VERSION:-not set}"
echo "PYTORCH_ROCM_ARCH: ${PYTORCH_ROCM_ARCH:-not set}"
echo "ROCM_PATH: ${ROCM_PATH:-not set}"
echo "CUDA_VISIBLE_DEVICES: ${CUDA_VISIBLE_DEVICES:-not set}"

# Check if ROCm is available
echo ""
echo "=== ROCm Detection ==="
if command -v python3 &> /dev/null; then
    echo "Testing ROCm with system Python3:"
    python3 -c "import torch; print('PyTorch version:', torch.__version__); print('CUDA available:', torch.cuda.is_available()); print('ROCm available:', hasattr(torch, 'hip') and torch.hip.is_available() if hasattr(torch, 'hip') else False)"
else
    echo "Python3 not found in PATH"
fi

# Check if uv is available
echo ""
echo "=== UV Check ==="
if command -v uv &> /dev/null; then
    echo "UV found: $(uv --version)"
else
    echo "UV not found in PATH"
fi

# Check WhisperX environment
echo ""
echo "=== WhisperX Environment ==="
WHISPERX_PATH="./whisperx-env/WhisperX"
if [ -d "$WHISPERX_PATH" ]; then
    echo "WhisperX directory exists at: $WHISPERX_PATH"
    if [ -f "$WHISPERX_PATH/pyproject.toml" ]; then
        echo "pyproject.toml exists"
        echo "Current ctranslate2 dependency:"
        grep -i ctranslate2 "$WHISPERX_PATH/pyproject.toml" || echo "No ctranslate2 dependency found"
    else
        echo "pyproject.toml not found"
    fi
else
    echo "WhisperX directory not found at: $WHISPERX_PATH"
fi

# Check if we can run WhisperX
echo ""
echo "=== WhisperX Test ==="
if [ -d "$WHISPERX_PATH" ] && command -v uv &> /dev/null; then
    echo "Testing WhisperX import:"
    uv run --project "$WHISPERX_PATH" python -c "import whisperx; print('WhisperX import successful')" 2>&1
else
    echo "Cannot test WhisperX (missing directory or uv)"
fi

echo ""
echo "=== Debug Complete ==="