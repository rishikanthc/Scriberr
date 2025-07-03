#!/bin/bash

# WhisperLiveKit Server Startup Script

echo "🚀 Starting WhisperLiveKit Transcription Server..."

# Check if Docker is available
if command -v docker &> /dev/null; then
    echo "🐳 Using Docker to run the server..."
    
    # Build and run with docker-compose
    docker-compose up --build
else
    echo "🐍 Running with Python directly..."
    
    # Check if requirements are installed
    if ! python -c "import whisperlivekit" 2>/dev/null; then
        echo "📦 Installing requirements..."
        pip install -r requirements.txt
    fi
    
    # Run the server
    python whisperlivekit_server.py
fi 