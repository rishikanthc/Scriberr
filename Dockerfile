# Use a specific version of Ubuntu as the base image
FROM ubuntu:24.04

# Set only basic environment variables needed for build
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
        software-properties-common \
        tzdata \
        ffmpeg \
        curl \
        unzip \
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

# Copy everything except node_modules
COPY . .
RUN rm -rf node_modules

# Clean install of all dependencies
RUN npm ci

# Build the application
RUN NODE_ENV=production npm run build

# Ensure entrypoint script is executable
RUN chmod +x docker-entrypoint.sh

# Create directory for Python virtual environment
RUN mkdir -p /scriberr

# Expose port
EXPOSE 3000

# Define default command
CMD ["./docker-entrypoint.sh"]
