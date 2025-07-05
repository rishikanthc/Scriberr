# Scriberr - Self-hosted AI Transcription App




## About
Scriberr is a self-hostable AI audio transcription app. It leverages the open-source [Whisper](https://github.com/openai/whisper) models from OpenAI, utilizing the high-performance [WhisperX](https://github.com/m-bain/whisperX) transcription engine to transcribe audio files locally on your hardware. Scriberr also allows you to summarize transcripts using Ollama or OpenAI's ChatGPT API, with your own custom prompts. From v0.2.0, Scriberr supports offline speaker diarization with significant improvements.

**Note**: This app is under active development, and this release includes **breaking changes**. You will lose your old data. Please read the installation instructions carefully.

## Call for Beta Testers
Hi all, It's been several months since I started this project. The project has come a long way since then, and now, I'm about to release the first stable release v1.0.0. In light of this, I am releasing a beta version for seeking feedback before the release to smooth out any bugs. I request anyone interested to please try out the beta version and provide quality feedback so I can smooth any bugs out before the stable release.

## Updates
The stable version brings a lot of updates to the app. The app has been rebuilt from the ground up and also introduces a bunch of cool new features.

### Under the hood
The app has been rebuilt with Go for the backend and Svelte5 for the frontend and runs as a single binary file. The frontend is compiled to static website (plain HTML and JS) and this static website is embedded into the Go binary to provide a fast and highly responsive app. It uses Python for the actual AI transcription by leveraging the WhisperX engine for running Whisper models. This release is a breaking release and moves to using SQLite for the database. Audio files are stored to disk as is.

### New Features and improvements
* Fast transcription with support for all model sizes
* Automatic language detection
* Uses VAD and ASR models for better alignment and speech detection to remove silence periods
* Speaker diarization (Speaker detection and identification)
* Automatic summarization using OpenAI/Ollama endpoints
* Markdown rendering of Summaries (NEW)
* AI Chat with transcript using OpenAI/Ollama endpoints (NEW)
  * Multiple chat sessions for each transcript (NEW)
* Built-in audio recorder
* YouTube video transcription (NEW)
* Download transcript as plaintext / JSON / SRT file (NEW)
* Save and reuse summarization prompt templates
* Tweak advanced parameters for transcription and diarization models (NEW)
* Audio playback follow (highlights transcript segment currently being played) (NEW)
* Stop or terminate running transcription jobs (NEW)
* Better reactivity and responsiveness (NEW)
* Toast notifications for all actions to provide instant status (NEW)
* Simplified deployment - single binary (Single container) (NEW)

## Installing the Beta version
You can install and try the new beta version with the following docker compose:

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

### Build Status
**Main Branch:**
[![Main Docker](https://github.com/rishikanthc/Scriberr/actions/workflows/Main%20Docker%20Build.yml/badge.svg)](https://github.com/rishikanthc/Scriberr/actions/workflows/Main%20Docker%20Build.yml)
[![Main CUDA Docker](https://github.com/rishikanthc/Scriberr/actions/workflows/Main%20Cuda%20Docker%20Build.yml/badge.svg)](https://github.com/rishikanthc/Scriberr/actions/workflows/Main%20Cuda%20Docker%20Build.yml)

**Nightly Branch:**
[![Nightly Docker](https://github.com/rishikanthc/Scriberr/actions/workflows/Nightly%20Docker%20Build.yml/badge.svg)](https://github.com/rishikanthc/Scriberr/actions/workflows/Nightly%20Docker%20Build.yml)
[![Nightly CUDA Docker](https://github.com/rishikanthc/Scriberr/actions/workflows/Nightly%20Cuda%20Docker%20Build.yml/badge.svg)](https://github.com/rishikanthc/Scriberr/actions/workflows/Nightly%20Cuda%20Docker%20Build.yml)

## Table of Contents

- [Features](#features)
- [Demo and Screenshots](#demo-and-screenshots)
- [Installation](#installation)
    - [Requirements](#requirements)
    - [Quick Start](#quick-start)
        - [Clone the Repository](#clone-the-repository)
        - [Configure Environment Variables](#configure-environment-variables)
        - [Running with Docker Compose (CPU Only)](#running-with-docker-compose-cpu-only)
        - [Running with Docker Compose (GPU Support)](#running-with-docker-compose-gpu-support)
        - [Access the Application](#access-the-application)
    - [Building Docker Images Manually](#building-docker-images-manually)
        - [CPU Image](#cpu-image)
        - [GPU Image](#gpu-image)
    - [Advanced Configuration](#advanced-configuration)
    - [Updating from Previous Versions](#updating-from-previous-versions)
    - [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgments](#acknowledgments)

## Features

- **Fast Local Transcription**: Transcribe audio files locally using WhisperX for high performance.
- **Hardware Acceleration**: Supports both CPU and GPU (NVIDIA) acceleration.
- **Customizable Compute Settings**: Configure the number of threads, cores, and model size.
- **Speaker Diarization**: Improved speaker identification with HuggingFace models.
- **Multilingual Support**: Supports all languages that the Whisper model supports.
- **Customize Summarization**: Optionally summarize transcripts with ChatGPT or Ollama using custom prompts.
- **API Access**: Exposes API endpoints for automation and integration.
- **User-Friendly Interface**: New UI with glassmorphism design.
- **Mobile Ready**: Responsive design suitable for mobile devices.

## Demo and Screenshots

> **Note:** \
> Demo was run locally on a MacBook Air M2 using Docker.
> Performance depends on the size of the model used and the number of cores and threads assigned.
> The demo was running in development mode, so performance may be slower than production.

https://github.com/user-attachments/assets/69d0c5a8-3412-4af5-a312-f3eddebc392e


![CleanShot 2024-10-04 at 14 42 54@2x](https://github.com/user-attachments/assets/90e68ebd-695e-4043-8d51-83c704a18c5c)
![CleanShot 2024-10-04 at 14 48 31@2x](https://github.com/user-attachments/assets/a8ecfa26-84aa-4091-8f22-481f0b5e67e6)
![CleanShot 2024-10-04 at 14 49 08@2x](https://github.com/user-attachments/assets/22820b96-f982-46da-8a71-79ea73559c79)
![CleanShot 2024-10-04 at 15 11 27@2x](https://github.com/user-attachments/assets/6e10b0c1-cf97-4cf6-ab47-591b6da607ef)




## Installation

### Requirements

- **Docker** and **Docker Compose** installed on your system. [Install Docker](https://docs.docker.com/get-docker/).
- **NVIDIA GPU** (optional): If you plan to use GPU acceleration, ensure you have an NVIDIA GPU and the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html) installed.
- **HuggingFace API Key** (required for speaker diarization): You'll need a free API key from [HuggingFace](https://huggingface.co/settings/tokens) to download diarization models.

### Quick Start

#### Clone the Repository

```bash
git clone https://github.com/rishikanthc/Scriberr.git
cd Scriberr
```

#### Configure Environment Variables

Copy the example `.env` file and adjust the settings as needed:

```bash
cp env.example .env
```

Edit the `.env` file to set your desired configuration, including:

- **`ADMIN_USERNAME`** and **`ADMIN_PASSWORD`** for accessing the web interface.
- **`OPENAI_API_KEY`** if you plan to use OpenAI's GPT models for summarization.
- **`HARDWARE_ACCEL`** set to `gpu` if you have an NVIDIA GPU.
- Other configurations as needed.

#### Running with Docker Compose (CPU Only)

To run Scriberr without GPU acceleration:

```bash
docker-compose up -d
```

This command uses the `docker-compose.yml` file and builds the Docker image using the `Dockerfile`.

#### Running with Docker Compose (GPU Support)

To run Scriberr with GPU acceleration:

```bash
docker-compose -f docker-compose.yml -f docker-compose.gpu.yml up -d
```

This command uses both `docker-compose.yml` and `docker-compose.gpu.yml` files and builds the Docker image using the `Dockerfile-gpu`.

**Note**: Ensure that you have the NVIDIA Container Toolkit installed and properly configured.

#### Access the Application

Once the containers are up and running, access the Scriberr web interface at `http://localhost:3000` (or the port you specified in the `.env` file).

### Building Docker Images Manually

If you wish to build the Docker images yourself, you can use the provided `Dockerfile` and `Dockerfile-gpu`.

#### CPU Image

```bash
docker build -t scriberr:main -f Dockerfile .
```

#### GPU Image

```bash
docker build -t scriberr:main-cuda128 -f Dockerfile-cuda128 .
```

### Advanced Configuration

The application can be customized using the following environment variables in your `.env` file.

- **`ADMIN_USERNAME`**: Username for the admin user in the web interface.
- **`ADMIN_PASSWORD`**: Password for the admin user.
- **`AI_MODEL`**: Default model to use for summarization (e.g., `"gpt-3.5-turbo"`).
- **`OLLAMA_BASE_URL`**: Base URL of your OpenAI API-compatible server if not using OpenAI (e.g., your Ollama server).
- **`OPENAI_API_KEY`**: Your OpenAI API key if using OpenAI for summarization (Or Ollama if `OLLAMA_BASE_URL` is set)
- **`DIARIZATION_MODEL`**: Default model for speaker diarization (e.g., `"pyannote/speaker-diarization@3.1"`).
- **`MODELS_DIR`**, **`WORK_DIR`**, **`AUDIO_DIR`**: Directories for models, temporary files, and uploads.
- **`BODY_SIZE_LIMIT`**: Maximum request body size (e.g., `"1G"`).
- **`HARDWARE_ACCEL`**: Set to `gpu` for GPU acceleration (NVIDIA GPU required), defaults to `cpu`.

#### Speaker Diarization Setup

##### Required Models
The application requires access to the following Hugging Face models:

* pyannote/speaker-diarization-3.1
* pyannote/segmentation-3.0
###### Setup Steps
1. Create a free account at HuggingFace if you don’t already have one.
2. Generate an API token at HuggingFace Tokens.
3. Accept user conditions for the required models on Hugging Face:
    - Visit pyannote/speaker-diarization-3.1 and accept the conditions.
    - Visit pyannote/segmentation-3.0 and accept the conditions.
4. Enter the API token in the setup wizard when prompted. The token is only used during initial setup and is not stored permanently.
Storage and Usage


The diarization models are downloaded once and stored locally, so you won’t need to provide the API key again after the initial setup.


### Updating from Previous Versions

**Important**: This release includes **breaking changes** and is **not backward compatible** with previous versions. You will lose your existing data. Please back up your data before proceeding.

Changes include:

- **Performance Improvements**: The rewrite takes advantage of Svelte 5 reactivity features.
- **Transcription Engine Change**: Switched from Whisper.cpp to **WhisperX**.
- **Improved Diarization**: Significant improvements to the diarization pipeline.
- **Simplified Setup**: Streamlined setup process with improved wizard.
- **New UI**: Implemented a new UI design with glassmorphism.
- **Multilingual Support**: Transcription and diarization now support all languages that Whisper models support.

### Troubleshooting

- **Database Connection Issues**: Ensure that the PostgreSQL container is running and accessible.
- **GPU Not Detected**: Ensure that the NVIDIA Container Toolkit is installed and that Docker is configured correctly.
- **Permission Issues**: Running Docker commands may require root permissions or being part of the `docker` group.
- **Diarization Model Download Failure**: Make sure you've entered a valid HuggingFace API key during setup.
- **CUDA failed with error out of memory**: Ensure that your GPU has enough memory to run the models. You can try reducing the batch size by adding WHISPER_BATCH_SIZE= to your .env file. The default is 16, you can reduce to 8, 4, 2, etc. (Got large-v2 running on a 1.5 hour audio file on a 3070 with 8GB VRAM using batch size of 1. Anything higher and it died.)

Check the logs for more details:

```bash
docker-compose logs -f
```

### Need Help?

If you encounter issues or have questions, feel free to open an [issue](https://github.com/rishikanthc/Scriberr/issues).

## Contributing

Contributions are welcome! Feel free to submit pull requests or open issues.

- **Fork the Repository**: Create a personal fork of the repository on GitHub.
- **Clone Your Fork**: Clone your forked repository to your local machine.
- **Create a Feature Branch**: Make a branch for your feature or fix.
- **Commit Changes**: Make your changes and commit them.
- **Push to Your Fork**: Push your changes to your fork on GitHub.
- **Submit a Pull Request**: Create a pull request to merge your changes into the main repository.

For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [OpenAI Whisper](https://github.com/openai/whisper)
- [WhisperX](https://github.com/m-bain/whisperX)
- [HuggingFace](https://huggingface.co/)
- [PyAnnote Speaker Diarization](https://huggingface.co/pyannote/speaker-diarization)
- [Ollama](https://ollama.ai/)
- Community contributors who have submitted great PRs and helped the app evolve.

---

*Thank you for your patience, support, and interest in the project. Looking forward to any and all feedback.*
