# Scriberr - Installation

Installation can be done in 2 ways either using the provided docker image or by compiling from source.&#x20;

<br />

## Docker

Use the below `docker-compose.yaml` to deploy using docker.

```yaml
services:
  scriberr:
    image: ghcr.io/rishikanthc/scriberr:v1.0.0-beta1
    ports:
      - "8080:8080"
    environment:
      # OpenAI API Key - Set this to your actual API key
      - OPENAI_API_KEY=${OPENAI_API_KEY:-}
      # Ollama config for using Ollama for summarization and chat
      - OLLAMA_BASE_URL=${OLLAMA_BASE_URL:-}
      # Session Key - Generate a secure random key for production
      # You can generate one with: openssl rand -base64 32
      - SESSION_KEY=${SESSION_KEY:-}
      # Hugging face token needed for speaker diarization
      - HF_TOKEN=${HF_TOKEN:-}
      # Authentication credentials - Set these for custom admin credentials
      - SCRIBERR_USERNAME=${SCRIBERR_USERNAME:-admin}
      - SCRIBERR_PASSWORD=${SCRIBERR_PASSWORD:-password}
      
    volumes:
      # Persist database and storage files
      - ./scriberr-storage:/app/storage
    restart: unless-stopped
```

<br />

## Building from source

First clone the repository using `git clone https://github.com/rishikanthc/Scriberr.git`. You need to have `npm, uv, ffmpeg, yt-dlp and go` installed.&#x20;

* Navigate to `scriberr-frontend` and run `npm install`.

* Then run `npm run build`

* After the build successfully create a directory `mkdir ../scriberr-backend/cmd/scriberr/embedded_assets`

* Then copy the build files to this directory. `cp -r ./build/* ../scriberr-backend/cmd/scriberr/embedded_assets`

* Then navigate to `scriberr-backend` directory

* Run `uv sync --native-tls` which will install all the required python packages

* Install the go modules with `go mod tidy`

* then compile the binary with `go build -o scriberr ./cmd/scriberr/`

<br />

You will now be able to access the UI at `http://localhost:8080`. 