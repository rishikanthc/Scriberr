#!/bin/sh

# cd whisper.cpp && ./main -f ./samples/jfk.wav

# uname -m

# sleep 5

# redis-server --daemonize yes &

redis-server &
# redis-cli -p 6379 ping
# Ensure the environment variables are correctly set
echo "Creating admin with email: ${POCKETBASE_ADMIN_EMAIL}"
echo "PocketBase URL: ${POCKETBASE_URL}"

# Start PocketBase in the background
/pb/pocketbase serve --http=0.0.0.0:8080 --dir /app/db &

# Wait for PocketBase to start (adjust the sleep time if necessary)
sleep 5

# Create the admin user using environment variables
/pb/pocketbase admin create "${POCKETBASE_ADMIN_EMAIL}" "${POCKETBASE_ADMIN_PASSWORD}" --dir /app/db

sleep 2

npm run build

node build

# npm run dev -- --host 0.0.0.0
