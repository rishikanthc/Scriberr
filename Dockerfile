# ARG TARGETARCH
# ADD "https://github.com/bbc/audiowaveform/releases/download/1.10.1/audiowaveform_1.10.1-1-12_$TARGETARCH.deb" /pre/audiowaveform.deb

# RUN echo "I'm building for $TARGETARCH" && sleep 4
# RUN ls -l /pre && sleep 4

FROM ubuntu:22.04

# Set environment variables to make the installation non-interactive
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ="Etc/UTC"

RUN apt-get update && apt-get install -y \
    python3 \
    python3-dev \
    python3-pip \
    postgresql-client \
    software-properties-common \
    build-essential \
    cmake \
    tzdata \
    ffmpeg \
    curl \
    unzip \
    git
    
RUN add-apt-repository ppa:chris-needham/ppa \
    && apt-get update \
    && apt-get install -y audiowaveform

# Clean up the apt cache to reduce image size
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://deb.nodesource.com/setup_23.x -o nodesource_setup.sh && bash nodesource_setup.sh
RUN apt-get install -y nodejs

# Install audiowaveform and its dependencies
# RUN apt-get update && \
#     apt-get install -y libmad0 libid3tag0 libsndfile1 libgd3 && \
#     dpkg -i /pre/audiowaveform.deb || true && \
#     apt-get install -f -y && \
#     rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy application code
COPY . .

# Temporarily set NODE_ENV to development to install dev dependencies
ENV NODE_ENV=development
RUN npm install

# Copy and set up entrypoint script
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

EXPOSE 3000

CMD ["/bin/sh", "/usr/local/bin/docker-entrypoint.sh"]

