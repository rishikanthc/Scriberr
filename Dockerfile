# Use a specific version of Ubuntu as the base image
FROM ubuntu:24.04

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive \
    TZ="Etc/UTC" \
    PATH="/root/.local/bin/:$PATH"

# Install minimal runtime dependencies
RUN apt-get update && \
    apt-get install -y \
        python3 \
        python3-pip \
        python3-venv \
        postgresql-client \
        ffmpeg \
        curl \
        git && \
    # Add the PPA and install audiowaveform
    add-apt-repository ppa:chris-needham/ppa && \
    apt-get update && \
    apt-get install -y audiowaveform && \
    # Install UV
    curl -sSL https://astral.sh/uv/install.sh -o /uv-installer.sh && \
    sh /uv-installer.sh && \
    rm /uv-installer.sh && \
    # Install Node.js
    curl -fsSL https://deb.nodesource.com/setup_23.x | bash - && \
    apt-get install -y nodejs && \
    # Clean up
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Create deps directory
RUN mkdir -p /app/deps

# Copy only the files needed for dependency installation and runtime
COPY . .

# Install node dependencies and build frontend
RUN npm install && \
    npm run build

# Ensure entrypoint script is executable
RUN chmod +x docker-entrypoint.sh

# Expose port
EXPOSE 3000

# Define default command
CMD ["./docker-entrypoint.sh"]