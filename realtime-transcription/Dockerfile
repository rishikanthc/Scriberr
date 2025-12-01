FROM python:3.10-slim

WORKDIR /app

# Install system dependencies (ffmpeg is usually required for audio processing)
RUN apt-get update && apt-get install -y \
    ffmpeg \
    git \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements
COPY requirements.txt .

# Install dependencies
# We install torch CPU version explicitly to save space if needed, 
# but for simplicity we rely on requirements.txt or default pip behavior.
# However, for CPU only, it's often better to specify the index url.
RUN pip install --no-cache-dir -r requirements.txt

# Copy server code
COPY server.py .

# Expose port
EXPOSE 8000

# Run server
CMD ["uvicorn", "server:app", "--host", "0.0.0.0", "--port", "8000"]
