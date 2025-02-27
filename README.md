# Important
im in the process of doing a full rewrite of the app.There are a few reasons for this:
- Svelte 5 which brings a lot of improvements especially for reactivity
- The previous implementation was a bit more hacky than I wanted it to be
- Diarization wasn't great

I had to take some time off this project due to some commitments. I'll be back on working on this project regularly. you can expect weekly updates from here on and ill clean up things for a new release. 
in the meantime as tou might have noticed the existing docker images aren't valid. this is because currently docker pulls whisper.cpp from their official repo and sets it up. unfortunately this turned out to be a bad move as whisper.cpp changed their build process and hence the current setup no longer works. 
I have already moved the main branch ahead for the new release. hence if you want to try out the new release please download the repo and run docker build to create your own image. 
My sincere apologies for the inconvenience and ill fix this up soon. 

in the meantime folks who have the time and resources to build and try the new release any feedback would be greatly appreciated. 
also a warning this release is a breaking change, and you will lose your old data. 

The new release brings these changes
- Performance Improvements. rewrite takes advantage of svelte 5 reactivity features
- Changed the transcription engine from whisper.cpp to whisperX
- Significant improvements to the diarization pipeline. diarization will be vastly better.
- Streamlined and simplified setup process. removes the wizard altogether. 
- New UI. I Tried playing around with glassmorphism. appreciate feedback on UI. I'm no frontend designer :P
- Support for multilingual transcription. both transcription and diarization now support all languages that whisper model supports

looking forward to any and all feedback. thank you for your patience, support and interest in the project. 
Folks have submitted some great PRs and im excited to see how the app evolves. 
# Scriberr

[![ci](https://github.com/rishikanthc/Scriberr/actions/workflows/nightly-docker.yml/badge.svg?event=push)](https://github.com/rishikanthc/Scriberr/actions/workflows/nightly-docker.yml)
[![ci](https://github.com/rishikanthc/Scriberr/actions/workflows/nightly-cuda-docker.yml/badge.svg?event=push)](https://github.com/rishikanthc/Scriberr/actions/workflows/nightly-cuda-docker.yml)
[![ci](https://github.com/rishikanthc/Scriberr/actions/workflows/main-docker.yml/badge.svg?event=push)](https://github.com/rishikanthc/Scriberr/actions/workflows/main-docker.yml)
[![ci](https://github.com/rishikanthc/Scriberr/actions/workflows/main-cuda-docker.yml/badge.svg?event=push)](https://github.com/rishikanthc/Scriberr/actions/workflows/main-cuda-docker.yml)

Scriberr is a self-hostable AI audio transcription app. It leverages the open-source [Whisper](https://github.com/openai/whisper) models from OpenAI, utilizing the high-performance [WhisperX](https://github.com/m-bain/whisperX) transcription engine to transcribe audio files locally on your hardware. Scriberr also allows you to summarize transcripts using Ollama or OpenAI's ChatGPT API, with your own custom prompts. From v0.2.0, Scriberr supports offline speaker diarization with significant improvements.

**Note**: This app is under active development, and this release includes **breaking changes**. You will lose your old data. Please read the installation instructions carefully.

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
- **Offline Speaker Diarization**: Improved speaker identification without internet dependency.
- **Multilingual Support**: Supports all languages that the Whisper model supports.
- **Customize Summarization**: Optionally summarize transcripts with ChatGPT or Ollama using custom prompts.
- **API Access**: Exposes API endpoints for automation and integration.
- **User-Friendly Interface**: New UI with glassmorphism design.
- **Mobile Ready**: Responsive design suitable for mobile devices.

And more to come. Checkout the planned features section.

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
- **`HF_API_KEY`** if you plan to use HuggingFace models for diarization.
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
docker build -t scriberr:latest -f Dockerfile .
```

#### GPU Image

```bash
docker build -t scriberr:latest-gpu -f Dockerfile-cuda128 .
```

### Advanced Configuration

The application can be customized using the following environment variables in your `.env` file.

- **`ADMIN_USERNAME`**: Username for the admin user in the web interface.
- **`ADMIN_PASSWORD`**: Password for the admin user.
- **`AI_MODEL`**: Default model to use for summarization (e.g., `"gpt-3.5-turbo"`).
- **`OLLAMA_BASE_URL`**: Base URL of your OpenAI API-compatible server if not using OpenAI (e.g., your Ollama server).
- **`OPENAI_API_KEY`**: Your OpenAI API key if using OpenAI for summarization (Or Ollama if `OLLAMA_BASE_URL` is set)
- **`HF_API_KEY`**: Your HuggingFace API key if using HuggingFace models for diarization.
- **`DIARIZATION_MODEL`**: Default model for speaker diarization (e.g., `"pyannote/speaker-diarization"`).
- **`MODELS_DIR`**, **`WORK_DIR`**, **`AUDIO_DIR`**: Directories for models, temporary files, and uploads.
- **`BODY_SIZE_LIMIT`**: Maximum request body size (e.g., `"1G"`).
- **`HARDWARE_ACCEL`**: Set to `gpu` for GPU acceleration (NVIDIA GPU required), defaults to `cpu`.

#### Customizing Docker Compose Files

If needed, you can modify the `docker-compose.yml` or `docker-compose.gpu.yml` files to suit your environment.

- **Volumes**: By default, data is stored in Docker volumes. If you prefer to store data in local directories, uncomment the lines in the `volumes` section and specify your paths.

### Updating from Previous Versions

**Important**: This release includes **breaking changes** and is **not backward compatible** with previous versions. You will lose your existing data. Please back up your data before proceeding.

Changes include:

- **Performance Improvements**: The rewrite takes advantage of Svelte 5 reactivity features.
- **Transcription Engine Change**: Switched from Whisper.cpp to **WhisperX**.
- **Improved Diarization**: Significant improvements to the diarization pipeline.
- **Simplified Setup**: Streamlined setup process; the wizard has been removed.
- **New UI**: Implemented a new UI design with glassmorphism.
- **Multilingual Support**: Transcription and diarization now support all languages that Whisper models support.

### Troubleshooting

- **Database Connection Issues**: Ensure that the PostgreSQL container is running and accessible.
- **GPU Not Detected**: Ensure that the NVIDIA Container Toolkit is installed and that Docker is configured correctly.
- **Permission Issues**: Running Docker commands may require root permissions or being part of the `docker` group.
- **Docker Images Not Valid**: If you encounter issues with pre-built Docker images, consider building the images locally using the provided Dockerfiles.

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
- [Ollama](https://ollama.ai/)
- Community contributors who have submitted great PRs and helped the app evolve.

---

*Thank you for your patience, support, and interest in the project. Looking forward to any and all feedback.*
