FROM ubuntu:22.04

# Set environment variables to make the installation non-interactive
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ="Etc/UTC"

RUN apt-get update && apt-get install -y \
    python3 \
    python3-dev \
    python3-pip \
    software-properties-common \
    tzdata \
    ffmpeg \
    curl \
    unzip \
    git

# Install Python packages
# RUN python3 -m pip install --no-cache-dir pyannote.audio

# Add the required PPA for audiowaveform
RUN add-apt-repository ppa:chris-needham/ppa \
    && apt-get update \
    && apt-get install -y audiowaveform

# Clean up the apt cache to reduce image size
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

ARG POCKETBASE_ADMIN_EMAIL
ARG POCKETBASE_ADMIN_PASSWORD
ARG POCKETBASE_URL
ARG SCRIBO_FILES
ARG REDIS_HOST
ARG REDIS_PORT
ARG OPENAI_API_KEY
ARG OPENAI_ENDPOINT=https://api.openai.com/v1
ARG OPENAI_MODEL="gpt-4"
ARG OPENAI_ROLE="system"
ARG POCKETBASE_VERSION=0.22.21
ARG DEV_MODE
ARG NVIDIA
ARG CONCURRENCY
 
ENV POCKETBASE_ADMIN_EMAIL=$POCKETBASE_ADMIN_EMAIL
ENV POCKETBASE_ADMIN_PASSWORD=$POCKETBASE_ADMIN_PASSWORD
ENV SCRIBO_FILES=$SCRIBO_FILES
ENV REDIS_HOST=$REDIS_HOST
ENV REDIS_PORT=$REDIS_PORT
ENV OPENAI_API_KEY=$OPENAI_API_KEY
ENV OPENAI_ENDPOINT=$OPENAI_ENDPOINT
ENV OPENAI_MODEL=$OPENAI_MODEL
ENV OPENAI_ROLE=$OPENAI_ROLE
ENV BODY_SIZE_LIMIT=512M
ENV DEV_MODE=$DEV_MODE
ENV POCKETBASE_URL=$POCKETBASE_URL
ENV NVIDIA=$NVIDIA
ENV CONCURRENCY=$CONCURRENCY
# ENV LD_PRELOAD=/usr/local/lib/python3.8/dist-packages/sklearn/__check_build/../../scikit_learn.libs/libgomp-d22c30c5.so.1.0.0 
 
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && apt-get install -y nodejs
	

WORKDIR /app

COPY . .

RUN git clone https://github.com/ggerganov/whisper.cpp.git

RUN npm ci

# Expose necessary ports
EXPOSE 3000 8080 9243

CMD ["/bin/sh", "/app/start_services.sh"]
