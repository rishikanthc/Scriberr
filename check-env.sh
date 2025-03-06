#!/bin/bash

# This script checks for required directories and creates them if they don't exist
# It also ensures models are downloaded correctly

# Set default paths if not defined in environment
MODELS_DIR=${MODELS_DIR:-"/scriberr/models"}
WORK_DIR=${WORK_DIR:-"/scriberr/temp"}
AUDIO_DIR=${AUDIO_DIR:-"/scriberr/uploads"}

echo "Checking and creating required directories..."

# Create directories if they don't exist
mkdir -p "$MODELS_DIR"
mkdir -p "$WORK_DIR"
mkdir -p "$AUDIO_DIR"

echo "Setting proper permissions..."
chmod -R 755 "$MODELS_DIR"
chmod -R 755 "$WORK_DIR"
chmod -R 755 "$AUDIO_DIR"

# Check for PyTorch and WhisperX installation
echo "Checking Python dependencies..."
pip list | grep -E "torch|whisperx" || {
  echo "Installing required Python packages..."
  pip install torch torchaudio --index-url https://download.pytorch.org/whl/cpu
  pip install whisperx
}

echo "Environment check complete. Ready to transcribe!"