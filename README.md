# Scriberr

Ever wished you could just record something and get an accurate transcript with speaker identification? That's exactly what Scriberr does.

## What is this?

Scriberr is a web app that takes your audio files and turns them into detailed transcripts. It uses WhisperX under the hood, so you get:

- **Accurate transcription** - WhisperX is really good at this
- **Speaker diarization** - It figures out who said what
- **Multiple formats** - Get your transcript as text, SRT, VTT, or JSON
- **Chat with your audio** - Ask questions about what was said using AI
- **Take notes** - Annotate important parts as you listen

## Quick start

The easiest way to get started:

```bash
brew tap rishikanthc/scriberr
brew install scriberr
```

Then just run `scriberr` and open http://localhost:8080 in your browser.

## What you can do

- Upload audio files or record directly in the app
- Get transcripts with timestamps and speaker labels
- Download transcripts in whatever format you need
- Chat with AI about your recordings
- Create summaries of long audio
- Manage everything through a clean web interface

## Requirements

- Python 3.11+ (for the transcription engine)
- A few GB of disk space for the AI models

That's it. Everything else is handled for you.

## Built with

Go backend, React frontend, WhisperX for transcription, and a lot of coffee.