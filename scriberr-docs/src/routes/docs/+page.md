# Scriberr - Introduction

Scriberr is a self-hostable, offline audio transcription app. It uses the WhisperX engine for fast transcription with automatic language detection, speaker diarization and support for all whisper models. Built with Go and Svelte, the server runs as a single binary and is fast and highly responsive.

#### Features

* Fast transcription with support for all model sizes

* Automatic language detection

* Uses VAD and ASR models for better alignment and speech detection to remove silence periods

* Speaker diarization (Speaker detection and identification)

* Automatic summarization using OpenAI/Ollama endpoints

* AI Chat with notes using OpenAI/Ollama endpoints

  * Multiple chat sessions for each transcript

* Built-in audio recorder

* YouTube video transcription

* Download transcript as plaintext / JSON / SRT file

* Save and reuse summarization prompt templates

* Tweak advanced parameters for transcription and diarization models

* Audio playback follow (highlights transcript segment currently being played)

* (Coming soon) GPU support - (need to compile docker image for it. The binary will work on GPUs)

<br />

### Under the hood

Scriberr uses Go for the backend, Svelte for the frontend and Python for AI transcription. The frontend is compiled to a static SPA (plain html and js) which is then embedded into the Go backend binary to provide a single binary. 