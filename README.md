<div align="center">

<img alt="Scriberr" src="cropped-main-logo.png" width="480" />

Self‑hostable, secure & private offline transcription. Drop in a recording, get clean transcripts, highlight key moments, take notes or chat with your audio using your favorite LLM — all without sending your data to the cloud.

[Website](https://scriberr.app) • [Docs](https://scriberr.app/docs/intro.html) • [API Reference](https://scriberr.app/api.html) • [Changelog](https://scriberr.app/changelog.html)

<p align="center">
<a href='https://ko-fi.com/H2H41KQZA3' target='_blank'><img height='36' style='border:0px;height:36px;' src='https://storage.ko-fi.com/cdn/kofi6.png?v=6' border='0' alt='Buy Me a Coffee at ko-fi.com' /></a>
</p>
</div>

**Collecting feedback on new feature. Drop by https://github.com/rishikanthc/Scriberr/discussions/200 to share your opinions.**

---

## Sponsors

![recall.ai-logo](https://cdn.prod.website-files.com/620d732b1f1f7b244ac89f0e/66b294e51ee15f18dd2b171e_recall-logo.svg) Meeting Transcription API   
If you're looking for a transcription API for meetings, consider checking out [Recall.ai](https://www.recall.ai/?utm_source=github&utm_medium=sponsorship&utm_campaign=rishikanthc-scriberr), an API that works with Zoom, Google Meet, Microsoft Teams, and more.
Recall.ai diarizes by pulling the speaker data and seperate audio streams from the meeting platforms, which means 100% accurate speaker diarization with actual speaker names.

# Introduction

Scriberr is a self‑hosted offline transcription app for converting audio into text. Record or upload audio, get it transcribed, and quickly summarize or chat using your preferred LLM provider. Scriberr runs on modern CPUs (no GPU required, though GPUs can accelerate processing) and offers a range of trade‑offs between speed and transcription quality.

- Built with React (frontend) and Go (backend), packaged as a single binary
- Uses WhisperX with open‑source Whisper models for accurate transcription
- Clean, distraction‑free UI optimized for reading and working with transcripts

<p align="center">
  <img alt="Scriberr homepage" src="screenshots/scriberr-homepage.png" width="720" />
</p>

## Features

- Accurate transcription with word‑level timing
- Speaker diarization (identify and label speakers)
- Transcript reader with playback follow‑along and seek‑from‑text
- Highlights and lightweight note‑taking (jump note → audio/transcript)
- Summarize and chat over transcripts (OpenAI or local models via Ollama)
- Transcription profiles for re‑usable configurations
- YouTube video transcription (paste a link and transcribe)
- Quick transcribe (ephemeral) and batch upload
- REST API coverage for all major features + API key management
- Download transcripts as JSON/SRT/TXT (and more)
- Support for Nvidia GPUs [New - Experimental]

## Screenshots

<details>
  <summary>Show screenshots</summary>

  <p align="center">
    <img alt="Transcript view" src="screenshots/scriberr-transcript page.png" width="720" />
  </p>
  <p align="center"><em>Minimal transcript reader with playback follow‑along and seek‑from‑text.</em></p>

  <p align="center">
    <img alt="Summarize transcripts" src="screenshots/scriberr-summarize transcripts.png" width="720" />
  </p>
  <p align="center"><em>Summarize long recordings and use custom prompts.</em></p>

  <p align="center">
    <img alt="API key management" src="screenshots/scriberr-api-key-management.png" width="720" />
  </p>
  <p align="center"><em>Generate and manage API keys for the REST API.</em></p>

  <p align="center">
    <img alt="YouTube video transcription" src="screenshots/scriberr-youtube-video.png" width="720" />
  </p>
  <p align="center"><em>Transcribe audio directly from a YouTube link.</em></p>

</details>

## Installation

Visit the website for the full guide: https://scriberr.app/docs/installation.html

### Homebrew (macOS & Linux)

```bash
brew tap rishikanthc/scriberr
brew install scriberr

# Start the server
scriberr
```

Open http://localhost:8080 in your browser.

Optional configuration via .env (sensible defaults provided):

```env
# Server
HOST=localhost
PORT=8080

# Storage
DATABASE_PATH=./data/scriberr.db
UPLOAD_DIR=./data/uploads
WHISPERX_ENV=./data/whisperx-env

# Custom paths (if needed)
UV_PATH=/custom/path/to/uv
```

### Docker

Run the command below in a shell:

```bash
docker run -d \
  --name scriberr \
  -p 8080:8080 \
  -v scriberr_data:/app/data \
  --restart unless-stopped \
  ghcr.io/rishikanthc/scriberr:latest
```

#### Docker Compose:

```yaml
version: '3.9'
services:
  scriberr:
    image: ghcr.io/rishikanthc/scriberr:latest
    container_name: scriberr
    ports:
      - "8080:8080"
    volumes:
      - scriberr_data:/app/data
    restart: unless-stopped

volumes:
  scriberr_data:
```

#### With GPU (CUDA)
```yaml
version: "3.9"
services:
  scriberr:
    image: ghcr.io/rishikanthc/scriberr:v1.0.4-cuda
    ports:
      - "8080:8080"
    volumes:
      - scriberr_data:/app/data
    restart: unless-stopped
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities:
                - gpu
    environment:
      - NVIDIA_VISIBLE_DEVICES=all
      - NVIDIA_DRIVER_CAPABILITIES=compute,utility

volumes:
  scriberr_data: {}
```

Then open http://localhost:8080.

## Diarization (speaker identification)

Scriberr uses the open‑source pyannote models for local speaker diarization. Models are hosted on Hugging Face and require an access token (only used to download models — diarization runs locally).

1) Create an account on https://huggingface.co

2) Visit and accept the user conditions for these repositories:
   - https://huggingface.co/pyannote/speaker-diarization-3.0
   - https://huggingface.co/pyannote/speaker-diarization
   - https://huggingface.co/pyannote/speaker-diarization-3.1
   - https://huggingface.co/pyannote/segmentation-3.0

   Verify they appear here: https://huggingface.co/settings/gated-repos

3) Create an access token under Settings → Access Tokens and enable all permissions under “Repositories”. Keep it safe.

4) In Scriberr, when creating a profile or using Transcribe+, open the Diarization tab and paste the token into the “Hugging Face Token” field.

See the full guide: https://scriberr.app/docs/diarization.html

<p align="center">
  <img alt="Diarization setup" src="screenshots/scriberr-diarization-setup.png" width="420" />
</p>

## API

Scriberr exposes a clean REST API for most features (transcription, chat, notes, summaries, admin, and more). Authentication supports JWT or API keys depending on endpoint.

- API Reference: https://scriberr.app/api.html
- Quick start examples (cURL and JS) on the API page
- Generate or manage API keys in the app

## Contributing

Issues and PRs are welcome. Please open an issue to discuss large changes first and keep PRs focused.

Local dev overview:

```bash
# Backend (dev)
cp -n .env.example .env || true
go run cmd/server/main.go

# Frontend (dev)
cd web/frontend
npm ci
npm run dev

# Full build (embeds UI in Go binary)
./build.sh
./scriberr
```

Coding style: `go fmt ./...`, `go vet ./...`, and `cd web/frontend && npm run lint`.

## Donating

<a href='https://ko-fi.com/H2H41KQZA3' target='_blank'><img height='36' style='border:0px;height:36px;' src='https://storage.ko-fi.com/cdn/kofi6.png?v=6' border='0' alt='Buy Me a Coffee at ko-fi.com' /></a>
