<div align="center">
  <img src="logo.svg" height="100" style="vertical-align: middle;" />
  <img src="logo-text.svg" height="80" style="vertical-align: middle;" />
</div>

<p align="center">
Scriberr is an open-source, and completely offline audio transcription application designed for self-hosters who value privacy and performance.
</p>

<p align="center">
  <a href="https://scriberr.app">Website</a> •
  <a href="https://scriberr.app/docs/">Docs</a> •
  <a href="https://scriberr.app/api">API Reference</a>
</p>

<p align="center">
<a href='https://ko-fi.com/H2H41KQZA3' target='_blank'><img height='36' style='border:0px;height:36px;' src='https://storage.ko-fi.com/cdn/kofi6.png?v=6' border='0' alt='Buy Me a Coffee at ko-fi.com' /></a>
</p>

<div align="center">
  <img src="screenshots/hero.png" alt="Scriberr Desktop App" width="800" />
</div>

## Sponsors

![recall.ai-logo](https://cdn.prod.website-files.com/620d732b1f1f7b244ac89f0e/66b294e51ee15f18dd2b171e_recall-logo.svg) Meeting Transcription API   
If you're looking for a transcription API for meetings, consider checking out [Recall.ai](https://www.recall.ai/?utm_source=github&utm_medium=sponsorship&utm_campaign=rishikanthc-scriberr), an API that works with Zoom, Google Meet, Microsoft Teams, and more.
Recall.ai diarizes by pulling the speaker data and seperate audio streams from the meeting platforms, which means 100% accurate speaker diarization with actual speaker names.

## Introduction

At its core, Scriberr allows you to transcribe audio and video locally on your machine, ensuring no data is ever sent to a third-party cloud provider.
Leveraging state-of-the-art machine learning models (such as **Whisper**, **NVIDIA Parakeet**, and **Canary**), it delivers high-accuracy text with word-level timing.

Scriberr goes beyond simple transcription and provides various advanced capabilities.
It combines powerful under-the-hood AI with a polished, fluid user interface that makes managing your recordings feel effortless. Whether you are sorting through voice notes or analyzing long meetings, Scriberr provides a beautiful environment to get work done:

- **Smart Speaker Detection**: Scriberr automatically detects different speakers (Diarization) and labels exactly who said what.
- **Chat with your Audio**: Connect seamlessly with Ollama or OpenAI API compatible providers. You can generate summaries, ask questions, or have a full conversation with your transcripts right inside the app.
- **Built for your Workflow**: With extensive APIs and Folder Watcher that automatically processes new files in a folder, Scriberr fits right into your existing automations (like n8n).
- **Capture & Organize**: Use the built-in audio recorder to capture thoughts on the fly, and the integrated note-taking features to annotate your transcripts as you listen.
- **Native Experience everywhere**: Scriberr supports PWA (Progressive Web App) installation, giving you a native app experience on your desktop or mobile device.
- **A Polished UI**: I’ve focused on the little UI niceties that make the app feel responsive and satisfying to use.

[View full list of features →](https://scriberr.app/docs/features)

### Why I built this

The inspiration for Scriberr was born out of privacy paranoia and not wanting to pay for subscription.
About a year ago, I purchased a [Plaud Note](https://www.plaud.ai/) for recording voice memos. I loved the device itself; the form factor, microphone quality, and workflow were excellent.

However, transcription was done on their cloud servers. As someone who is paranoid about privacy I wasn't comfortable with uploading my recordings to a third party provider.
Moreover I was hit with subscription costs: $100 a year for 20 hours of transcription per month, or $240 a year for unlimited access. As an avid self-hoster with a background in ML and AI, it felt wrong to pay such a premium for a service I knew I could engineer myself.

I decided to build Scriberr to bridge that gap, creating a powerful, private, and free alternative for everyone.

## Screenshots

<details>
  <summary>Click to expand</summary>

  <p align="center">
    <img alt="Transcript view" src="screenshots/transcript-light.png" width="720" />
  </p>
  <p align="center"><em>Transcript reader with playback follow‑along and seek‑from‑text.</em></p>

  <p align="center">
    <img alt="Chat with Audio" src="screenshots/chat.png" width="720" />
  </p>
  <p align="center"><em>Chat with your transcripts using local LLMs or OpenAI.</em></p>

  <p align="center">
    <img alt="Notes and Highlights" src="screenshots/notes.png" width="720" />
  </p>
  <p align="center"><em>Highlight key moments and take notes while listening.</em></p>

  <p align="center">
    <img alt="AI Summaries" src="screenshots/ai-summary.png" width="720" />
  </p>
  <p align="center"><em>Generate comprehensive summaries of your recordings.</em></p>

  <p align="center">
    <strong style="font-size: 1.2em;">Dark Mode</strong>
  </p>

  <p align="center">
    <img alt="Homepage Dark Mode" src="screenshots/homepage-dark.png" width="720" />
  </p>
  <p align="center"><em>Homepage in Dark Mode.</em></p>

  <p align="center">
    <img alt="Transcript Dark Mode" src="screenshots/transcript-dark.png" width="720" />
  </p>
  <p align="center"><em>Transcript view in Dark Mode.</em></p>

  ### Mobile

  <p align="center">
    <img alt="Mobile Homepage" src="screenshots/homepage-mobile.PNG" width="300" />
    <img alt="Mobile Homepage Dark" src="screenshots/homepage-mobile-dark.PNG" width="300" />
  </p>
  <p align="center"><em>PWA mobile app (Light & Dark).</em></p>

  <p align="center">
    <img alt="Mobile Transcript" src="screenshots/transcript-mobile.PNG" width="300" />
    <img alt="Mobile Transcript Dark" src="screenshots/transcript-mobile-dark.PNG" width="300" />
  </p>
  <p align="center"><em>Mobile transcript reading experience.</em></p>

</details>

## Installation

Get Scriberr running on your system in a few minutes.

### Install with Homebrew (macOS & Linux)

The easiest way to install Scriberr is using Homebrew. If you don’t have Homebrew installed, [get it here first](https://brew.sh/).

```bash
# Add the Scriberr tap
brew tap rishikanthc/scriberr

# Install Scriberr (automatically installs UV dependency)
brew install scriberr

# Start the server
scriberr
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

### Configuration

Scriberr works out of the box. To customize settings, create a `.env` file:

```bash
# Server settings
HOST=localhost
PORT=8080

# Data storage (optional)
DATABASE_PATH=./data/scriberr.db
UPLOAD_DIR=./data/uploads
WHISPERX_ENV=./data/whisperx-env
```

### Docker Deployment

For a containerized setup, you can use Docker. We provide two configurations: one for standard CPU usage and one optimized for NVIDIA GPUs (CUDA).

#### Standard Deployment (CPU)

Use this configuration for running Scriberr on any machine without a dedicated NVIDIA GPU.

1.  Create a file named `docker-compose.yml`:

```yaml
services:
  scriberr:
    image: ghcr.io/rishikanthc/scriberr:latest
    ports:
      - "8080:8080"
    volumes:
      - scriberr_data:/app/data
    restart: unless-stopped

volumes:
  scriberr_data:
```

2.  Run the container:

```bash
docker compose up -d
```

#### NVIDIA GPU Deployment (CUDA)

If you have a compatible NVIDIA GPU, this configuration enables hardware acceleration for significantly faster transcription.

1.  Ensure you have the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html) installed.
2.  Create a file named `docker-compose.cuda.yml`:

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

3.  Run the container with the CUDA configuration:

```bash
docker compose -f docker-compose.cuda.yml up -d
```

## Post installation

Once you have Scriberr up and running:

- **Configure Diarization**: To enable speaker identification, visit the [Configuration page](https://scriberr.app/docs/configuration).
- **Usage Guide**: For a detailed usage guide, visit [https://scriberr.app/docs/usage](https://scriberr.app/docs/usage).

## Donating

<a href='https://ko-fi.com/H2H41KQZA3' target='_blank'><img height='36' style='border:0px;height:36px;' src='https://storage.ko-fi.com/cdn/kofi6.png?v=6' border='0' alt='Buy Me a Coffee at ko-fi.com' /></a>

