FROM alpine:3.20 AS build_flac

ENV FLAC_VERSION=1.3.3
ENV GOOGLE_TEST_VERSION=1.12.1
ENV AUDIOWAVEFORM_VERSION=1.10.1
RUN apk update && apk add --no-cache \
    autoconf \
    automake \
    boost-dev \
    boost-static \
    cmake \
    g++ \
    gcc \
    gd-dev \
    gettext \
    git \
    libgd \
    libid3tag-dev \
    libmad-dev \
    libpng-dev \
    libpng-static \
    libsndfile-dev \
    libtool \
    make \
    zlib-dev \
    zlib-static

WORKDIR /tmp
# Build/Install FLAC
ADD https://github.com/xiph/flac/archive/${FLAC_VERSION}.tar.gz /tmp/${FLAC_VERSION}.tar.gz
RUN tar xzf "/tmp/${FLAC_VERSION}.tar.gz"

WORKDIR /tmp/flac-${FLAC_VERSION}
RUN ./autogen.sh
RUN ./configure --enable-shared=no
RUN make
RUN make install

# Build/Install AudioWaveform
WORKDIR /tmp
ADD https://github.com/google/googletest/archive/release-${GOOGLE_TEST_VERSION}.tar.gz /tmp/googletest.tar.gz
RUN tar xzf /tmp/googletest.tar.gz

WORKDIR /tmp/aw
ADD https://github.com/bbc/audiowaveform/archive/refs/tags/${AUDIOWAVEFORM_VERSION}.tar.gz /tmp/aw.tar.gz
RUN tar xzf /tmp/aw.tar.gz

RUN ln -s /tmp/googletest-release-${GOOGLE_TEST_VERSION} /tmp/aw/audiowaveform-${AUDIOWAVEFORM_VERSION}/googletest

WORKDIR /tmp/aw/audiowaveform-${AUDIOWAVEFORM_VERSION}/build
RUN cmake ..
RUN make
RUN make install

FROM alpine:3.20 AS build_whisper

RUN apk update && apk add git wget make gcc g++

# Download and unzip Whisper.cpp
# ADD https://github.com/ggerganov/whisper.cpp/archive/refs/heads/master.zip /tmp/whisper.zip
# RUN unzip /tmp/whisper.zip -d /app/
WORKDIR /app
RUN git clone https://github.com/ggerganov/whisper.cpp.git

# Set the Whisper directory as the working directory
WORKDIR /app/whisper.cpp

# Use bash to download the models
# RUN bash ./models/download-ggml-model.sh base.en && \
#     bash ./models/download-ggml-model.sh tiny.en && \
#     bash ./models/download-ggml-model.sh small.en

# Compile Whisper.cpp with make
RUN make

FROM node:22.9.0-alpine AS base
# FROM arm64v8/node:22.9.0-alpine

ARG POCKETBASE_ADMIN_EMAIL
ARG POCKETBASE_ADMIN_PASSWORD
ARG SCRIBO_FILES
ARG REDIS_HOST
ARG REDIS_PORT
ARG OPENAI_API_KEY
ARG OPENAI_ENDPOINT="https://api.openai.com/v1"
ARG OPENAI_MODEL="gpt-4"
ARG OPENAI_ROLE="system"


# Set environment variables to be overridden at runtime
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

ENV POCKETBASE_VERSION=0.22.21
# Install required packages
RUN apk update && apk add --no-cache wget ffmpeg unzip libgd libmad libid3tag boost-static boost-build

COPY --from=build_flac /usr/local/bin/* /usr/local/bin/
COPY --from=build_flac /usr/local/share/man/man1/* /usr/local/share/man/man1/
COPY --from=build_flac /usr/local/share/man/man5/* /usr/local/share/man/man5/

COPY --from=build_whisper /app/whisper.cpp/models/download-ggml-model.sh /usr/local/bin/download-ggml-model.sh
COPY --from=build_whisper /app/whisper.cpp/main /usr/local/bin/whisper

WORKDIR /models
RUN download-ggml-model.sh base.en /models && \
    download-ggml-model.sh tiny.en /models && \
    download-ggml-model.sh small.en /models

# Download and unzip PocketBase
ADD https://github.com/pocketbase/pocketbase/releases/download/v${POCKETBASE_VERSION}/pocketbase_${POCKETBASE_VERSION}_linux_amd64.zip /tmp/pb.zip
RUN unzip /tmp/pb.zip pocketbase -d /usr/local/bin/ && rm /tmp/pb.zip

# Set the working directory back to /app
WORKDIR /app

# Copy the application files
COPY . .

# Install Node.js dependencies
RUN npm ci

# Expose necessary ports
EXPOSE 3000 8080 9243 5173

# Start the services
CMD ["/bin/sh", "/app/start_services.sh"]
