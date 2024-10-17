# Build Whisper.cpp using the official process
ARG ARCH=
FROM ${ARCH}debian:bookworm-slim AS build_whisper

# Set working directory
WORKDIR /app

RUN apt-get update && \
    apt-get install -y build-essential \
    ca-certificates \
    git && \
    rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

RUN git clone https://github.com/ggerganov/whisper.cpp.git
WORKDIR /app/whisper.cpp

# Build Whisper.cpp
RUN make

# Final base image for the application (Node.js)
FROM ${ARCH}debian:bookworm-slim AS base

# Set ARG to dynamically handle architecture
ARG POCKETBASE_ADMIN_EMAIL
ARG POCKETBASE_ADMIN_PASSWORD
ARG SCRIBO_FILES
ARG REDIS_HOST
ARG REDIS_PORT
ARG OPENAI_API_KEY
ARG OPENAI_ENDPOINT="https://api.openai.com/v1"
ARG OPENAI_MODEL="gpt-4"
ARG OPENAI_ROLE="system"
ARG POCKETBASE_VERSION=0.22.21

# Set environment variables
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


RUN apt-get update && apt-get install -y --no-install-recommends \
    software-properties-common \
    gnupg2 \
    wget \
    ffmpeg \
    unzip \
    libgd3 \
    libmad0 \
    libid3tag0 \
    libboost-all-dev \
    libboost-filesystem-dev \
    libboost-program-options-dev \
    libboost-regex-dev \
    curl \
    python3 \
    python3-pip \
    python3-dev \
    pkg-config \
    libsndfile1 \
    && rm -rf /var/lib/apt/lists/*

RUN python -m pip install pyannote.audio

# Add the repository and install audiowaveform
# RUN add-apt-repository ppa:chris-needham/ppa && \
#     apt-get update && \
#     apt-get install -y audiowaveform
RUN ARCH="$(dpkg --print-architecture)" && \
    case "${ARCH}" in \
      "amd64") ARCH_URL="https://github.com/bbc/audiowaveform/releases/download/1.10.1/audiowaveform_1.10.1-1-12_amd64.deb" ;; \
      "arm64") ARCH_URL="https://github.com/bbc/audiowaveform/releases/download/1.10.1/audiowaveform_1.10.1-1-12_arm64.deb" ;; \
      *) echo "Unsupported architecture"; exit 1 ;; \
    esac && \
    wget ${ARCH_URL} -O audiowaveform.deb && \
    dpkg -i audiowaveform.deb && \
    apt-get -f install -y && \
    rm audiowaveform.deb

# Copy the whisper.cpp binary from the build stage
COPY --from=build_whisper /app/whisper.cpp/main /usr/local/bin/whisper
COPY --from=build_whisper /app/whisper.cpp/models/download-ggml-model.sh /usr/local/bin/download-ggml-model.sh

# Download Whisper models
WORKDIR /models
RUN download-ggml-model.sh base.en /models && \
    download-ggml-model.sh tiny.en /models && \
    download-ggml-model.sh small.en /models

# Download and unzip PocketBase
RUN ARCH="$(dpkg --print-architecture)" && \
    curl -L "https://github.com/pocketbase/pocketbase/releases/download/v${POCKETBASE_VERSION}/pocketbase_${POCKETBASE_VERSION}_linux_${ARCH}.zip" -o /tmp/pb.zip && \
    unzip /tmp/pb.zip pocketbase -d /usr/local/bin/ && \
    rm /tmp/pb.zip

# Set working directory back to /app
WORKDIR /app

# Copy application files
COPY . .

# Install Node.js and dependencies
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && apt-get install -y nodejs

# RUN apt install nodejs && apt install npm
RUN npm ci

# Expose necessary ports
EXPOSE 3000 8080 9243

# Start the services
CMD ["/bin/sh", "/app/start_services.sh"]
