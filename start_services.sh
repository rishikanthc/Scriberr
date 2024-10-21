#!/bin/sh

# Ensure the environment variables are correctly set
echo "Creating admin with email: ${POCKETBASE_ADMIN_EMAIL}"
echo "PocketBase URL: ${POCKETBASE_URL}"

cp -r /app/whisper.cpp /models/
# Start PocketBase in the background
# pocketbase serve --http=0.0.0.0:8080 --dir /app/db &

# Create the admin user using environment variables
# pocketbase admin create "${POCKETBASE_ADMIN_EMAIL}" "${POCKETBASE_ADMIN_PASSWORD}" --dir /app/db

npm run build
node build
