#!/bin/bash
set -e

# Set HARDWARE_ACCEL: 'cuda' for GPU, 'cpu' otherwise
if [ "$HARDWARE_ACCEL" = "gpu" ]; then
  export HARDWARE_ACCEL='cuda'
else
  export HARDWARE_ACCEL='cpu'
fi

# Function to wait for the database to be ready
wait_for_db() {
  echo "Waiting for database to be ready..."
  echo "Current DATABASE_URL: $DATABASE_URL"

  until PGPASSWORD="$POSTGRES_PASSWORD" pg_isready -h db -p 5432 -U "$POSTGRES_USER" -d "$POSTGRES_DB"
  do
    echo "Database connection attempt failed. Retrying in 2 seconds..."
    sleep 2
  done

  echo "Database is ready!"
}

# Function to install dependencies based on hardware type
install_dependencies() {
  echo "Checking and installing dependencies..."

  VENV_LOCATION_DIR="/scriberr/"
  VENV_DIR="/scriberr/.venv"
  DEPS_MARKER="/scriberr/.deps_installed"

  # Only install dependencies if not already installed (for faster restarts)
  if [ ! -f "$DEPS_MARKER" ]; then
    echo "First run detected - installing Python dependencies..."
    
    # Create venv if it doesn't exist
    if [ ! -d "$VENV_DIR" ]; then
      echo "Creating virtual environment..."
      uv venv --directory "$VENV_LOCATION_DIR"
      echo "Virtual environment created."
    fi

    # Activate venv
    source "$VENV_DIR/bin/activate"

    # Install Python dependencies
    echo "Installing Python dependencies..."

    # Install PyTorch based on hardware
    if [ "$HARDWARE_ACCEL" = "cuda" ]; then
      uv pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu126
      uv pip install nvidia-cudnn-cu11==8.9.6.50
      echo "PyTorch with CUDA installed."
    else
      uv pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cpu
    fi

    # Install remaining dependencies from requirements.txt
    uv pip install -r requirements.txt
    
    # Create marker file to indicate dependencies are installed
    touch "$DEPS_MARKER"
    echo "Python dependencies installation complete."
  else
    echo "Dependencies already installed. Skipping installation for faster startup."
  fi

  # Activate venv (needed even if dependencies were already installed)
  source "$VENV_DIR/bin/activate"
  
  # Skip rebuilding during startup to use pre-built files
  echo "Using pre-built application files..."
}
export VENV_DIR="/scriberr/.venv"
export LD_LIBRARY_PATH=$VENV_DIR/lib/python3.12/site-packages/nvidia/cudnn/lib
export NODE_ENV=production

# Auto-generate DATABASE_URL if not provided
if [ -z "$DATABASE_URL" ] && [ -n "$POSTGRES_USER" ] && [ -n "$POSTGRES_PASSWORD" ] && [ -n "$POSTGRES_DB" ]; then
  export DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@db:5432/${POSTGRES_DB}"
  echo "Auto-generated DATABASE_URL: $DATABASE_URL"
fi

# Debug environment variables if debug variable is enabled
if [ "$DEBUG" = "true" ]; then
  echo "Runtime environment variables:"
  echo "OLLAMA_BASE_URL=$OLLAMA_BASE_URL"
  echo "AI_MODEL=$AI_MODEL"
  echo "POSTGRES_USER=$POSTGRES_USER"
  echo "POSTGRES_DB=$POSTGRES_DB"
  echo "DATABASE_URL=$DATABASE_URL"
  echo "ADMIN_USERNAME=$ADMIN_USERNAME"
  echo "AUDIO_DIR=$AUDIO_DIR"
  echo "HARDWARE_ACCEL=$HARDWARE_ACCEL"
  echo "DIARIZATION_MODEL=$DIARIZATION_MODEL"
  echo "HF_API_KEY=$HF_API_KEY"
  echo "OPENAI_API_KEY=$OPENAI_API_KEY"
fi # End of debug block

# Ensure all runtime environment variables are correctly set for the app process
export NODE_ENV=production
export RUNTIME_CHECK=true

# Print the full environment variable list (redacted for security)
echo "Full list of environment variables available at runtime:"
env | grep -v PASSWORD | grep -v TOKEN | grep -v KEY | grep -v SECRET | sort

# Mark as runtime check to enable DB connection validation
export RUNTIME_CHECK=true

# Execute the setup steps
install_dependencies
wait_for_db


# Run database migrations
echo "Creating database..."
if ! npx drizzle-kit generate; then
  echo "Migration generation failed, but continuing..."
fi

if ! npx drizzle-kit migrate; then
  echo "Migration failed, but continuing..."
fi

echo "Running database push..."
if ! npx drizzle-kit push; then
  echo "Database push failed, but continuing..."
fi

# Start the application
echo "Starting the application..."
node build