FROM node:22.9.0-alpine
# FROM arm64v8/node:22.9.0-alpine

ARG POCKETBASE_ADMIN_EMAIL
ARG POCKETBASE_ADMIN_PASSWORD
ARG SCRIBO_FILES
ARG REDIS_HOST
ARG REDIS_PORT
ARG OPENAI_API_KEY

# Set environment variables to be overridden at runtime
ENV POCKETBASE_ADMIN_EMAIL=$POCKETBASE_ADMIN_EMAIL
ENV POCKETBASE_ADMIN_PASSWORD=$POCKETBASE_ADMIN_PASSWORD
ENV SCRIBO_FILES=$SCRIBO_FILES
ENV REDIS_HOST=$REDIS_HOST
ENV REDIS_PORT=$REDIS_PORT
ENV OPENAI_API_KEY=$OPENAI_API_KEY
ENV BODY_SIZE_LIMIT=512M 

# Install required packages
RUN apk update && apk add --no-cache \
    unzip \
    curl \
    redis \
    bash \
    wget \
    ffmpeg \
    build-base  # Required for 'make'

WORKDIR /tmp
COPY ./install_audiowaveform.sh .
RUN /bin/sh /tmp/install_audiowaveform.sh


# Download and unzip PocketBase
ADD https://github.com/pocketbase/pocketbase/releases/download/v0.22.21/pocketbase_0.22.21_linux_amd64.zip /tmp/pb.zip
RUN unzip /tmp/pb.zip -d /pb/

# Download and unzip Whisper.cpp
# ADD https://github.com/ggerganov/whisper.cpp/archive/refs/heads/master.zip /tmp/whisper.zip
# RUN unzip /tmp/whisper.zip -d /app/

ADD https://github.com/bbc/audiowaveform/archive/refs/heads/master.zip /tmp/aw/aw.zip
RUN unzip /tmp/aw/aw.zip -d /tmp/aw/

WORKDIR /tmp/aw
COPY ./install_aw.sh .
RUN /bin/sh /tmp/aw/install_aw.sh

WORKDIR /app
RUN git clone https://github.com/ggerganov/whisper.cpp.git

# Set the Whisper directory as the working directory
WORKDIR /app/whisper.cpp

# Use bash to download the models
RUN bash ./models/download-ggml-model.sh base.en && \
    bash ./models/download-ggml-model.sh tiny.en && \
    bash ./models/download-ggml-model.sh small.en

# # Compile Whisper.cpp with make
RUN make

# Set the working directory back to /app
WORKDIR /app

# Copy the application files
COPY . .

# Copy the startup script
COPY start_services.sh /app/start.sh

# Install Node.js dependencies
RUN npm ci

# Expose necessary ports
EXPOSE 3000 8080 9243 6379 5173

# Start the services
CMD ["/bin/sh", "/app/start.sh"]
