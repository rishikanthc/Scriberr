#!/bin/bash
set -e

# If the HARDWARE_ACCEL environment variable is not set, default to 'cpu', if set to GPU, use 'cuda'
if [ "$HARDWARE_ACCEL" = "gpu" ]; then
  export HARDWARE_ACCEL='cuda'
fi


# Function to wait for database to be ready
wait_for_db() {
  echo "Waiting for database to be ready..."
  
  # For debugging
  echo "Current DATABASE_URL: $DATABASE_URL"
  
  until PGPASSWORD="$POSTGRES_PASSWORD" pg_isready -h db -p 5432 -U "$POSTGRES_USER" -d "$POSTGRES_DB"
  do
    echo "Database connection attempt failed. Retrying in 2 seconds..."
    sleep 2
  done
  
  echo "Database is ready!"
}

# Wait for database
wait_for_db

# Run database migrations
echo "Creating database ..."
if ! npx drizzle-kit generate; then
    echo "Migration generation failed, but continuing..."
fi

if ! npx drizzle-kit migrate; then
    echo "Migration generation failed, but continuing..."
fi

# Run database push
echo "Running database push..."
if ! npx drizzle-kit push; then
    echo "Database push failed, but continuing..."
fi

# Start the application
echo "Building the application..."
exec "$@"

# Uncomment these lines if needed
# npm run build

echo "Starting the application..."
node build

