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
  # Skip if database variables aren't set
  if [ -z "$POSTGRES_USER" ] || [ -z "$POSTGRES_PASSWORD" ] || [ -z "$POSTGRES_DB" ]; then
    # For production, we should require these variables
    if [ "$NODE_ENV" = "production" ]; then
      echo "ERROR: Database credentials required in production but not set"
      echo "Please set POSTGRES_USER, POSTGRES_PASSWORD, and POSTGRES_DB environment variables"
      exit 1
    else
      echo "Database credentials not fully set, skipping database check"
      return 0
    fi
  fi

  echo "Waiting for database to be ready..."
  echo "Current DATABASE_URL: $DATABASE_URL"

  until PGPASSWORD="$POSTGRES_PASSWORD" pg_isready -h $POSTGRES_HOST -p 5432 -U "$POSTGRES_USER" -d "$POSTGRES_DB"
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

  # Create venv if it doesn't exist
  if [ ! -d "$VENV_DIR" ]; then
    echo "Creating virtual environment..."
    uv venv --directory "$VENV_LOCATION_DIR"
    echo "Virtual environment created."
  else
    echo "Virtual environment already exists."
  fi
  
  # Create necessary directories for model storage
  echo "Setting up model directories..."
  mkdir -p "${MODELS_DIR:-/scriberr/models}"
  mkdir -p "${WORK_DIR:-/scriberr/temp}"
  mkdir -p "${AUDIO_DIR:-/scriberr/uploads}"
  chmod -R 755 "${MODELS_DIR:-/scriberr/models}"
  
  # Set environment variables for HuggingFace
  export HF_HUB_DISABLE_TELEMETRY=1
  export TRUST_REMOTE_CODE=1
  
  # Pass through HuggingFace token if provided in environment
  if [ -n "$HUGGINGFACE_TOKEN" ]; then
    echo "HuggingFace token provided, will be used for model downloads"
  else
    echo "WARNING: No HuggingFace token provided. Speaker diarization may not work correctly."
  fi

  # Activate venv
  source "$VENV_DIR/bin/activate"

  # Install Python dependencies
  echo "Installing Python dependencies..."

  # Install PyTorch based on hardware
  if [ "$HARDWARE_ACCEL" = "cuda" ]; then
    uv pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/nightly/cu128
    uv pip install nvidia-cudnn-cu11==8.9.6.50
    echo "PyTorch with CUDA installed."
  else
    uv pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cpu
  fi

  # Install remaining dependencies from requirements.txt
  uv pip install -r requirements.txt
  # Note: Venv remains activated so PATH is set for the Node.js application

  # Install Node.js dependencies if directory is empty
  echo "Checking for Node.js dependencies..."
 if [ ! "$(ls -A /app/node_modules)" ]; then
    echo "Installing Node.js dependencies..."
  else
    echo "Node.js dependencies already installed."
  fi

  # Build the application
  if [ ! "$(ls -A /app/build)" ]; then
    echo "Building the application..."
    NODE_ENV=development npm run build
  else
    echo "Application already built."
  fi
}
export VENV_DIR="/scriberr/.venv"
export LD_LIBRARY_PATH=$VENV_DIR/lib/python3.12/site-packages/nvidia/cudnn/lib
export NODE_ENV=production

# Execute the setup steps
install_dependencies
wait_for_db


# Run database migrations if DATABASE_URL is set
if [ -n "$DATABASE_URL" ]; then
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
else
  # For production, DATABASE_URL is required
  if [ "$NODE_ENV" = "production" ]; then
    echo "ERROR: DATABASE_URL is required in production but not set"
    exit 1
  else
    echo "DATABASE_URL not set, skipping database migrations"
  fi
fi

# Start the application
echo "Starting the application..."
node build