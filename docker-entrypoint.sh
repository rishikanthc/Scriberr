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

  VENV_LOCATION_DIR="/app/deps/"
  VENV_DIR="/app/deps/.venv"
  MARKER_FILE="/app/deps/requirements_installed"

  # Create venv if it doesn't exist
  if [ ! -d "$VENV_DIR" ]; then
    echo "Creating virtual environment..."
    uv venv --directory "$VENV_LOCATION_DIR"
     echo "Virtual environment created."
  else
    echo "Virtual environment already exists."
  fi

  # Activate venv
  source "$VENV_DIR/bin/activate"

  # Install Python dependencies if not already installed
  if [ ! -f "$MARKER_FILE" ]; then
    echo "Installing Python dependencies..."

    # Install PyTorch based on hardware
    if [ "$HARDWARE_ACCEL" = "cuda" ]; then
      uv pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu126
      uv pip install nvidia-cudnn-cu11==8.9.6.50
      echo "PyTorch with CUDA installed."
    else
      uv pip install torch==2.0.0 torchvision==0.15.1 torchaudio==2.0.1 --index-url https://download.pytorch.org/whl/cpu
    fi

    # Install remaining dependencies from requirements.txt
    uv pip install -r requirements.txt

    # Create marker file
    touch "$MARKER_FILE"
  else
    echo "Python dependencies already installed."
  fi
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
export VENV_DIR="/app/deps/.venv"
export LD_LIBRARY_PATH=$VENV_DIR/lib/python3.12/site-packages/nvidia/cudnn/lib
export NODE_ENV=production

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