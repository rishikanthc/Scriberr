# Scriberr_LL: A Customized Version of Scriberr for Language Learning

## Introduction
The repo is based on the fantastic [Scriberr](https://github.com/rishikanthc/Scriberr) project and I'm adding more features to make it better for language learning.

## Features
### New Features
The base features can be found on the [Scriberr](https://github.com/rishikanthc/Scriberr) project. Here are the additional features:
- On the audio detail page, users can use up / down arrow keys to navigate among segments.
- Active segment is highlighted and automatically focused when being played.
- Clicks no more start playing the segment, but the default behaviors are reserved to select texts. It's more convinient for users to copy the transcripts and paste them to other places such as dictionaries. An additional play button is placed in every segment so users can use that to jump among segments.
### Planned Features
- [ ] Add a function to scan the disk and discover the audios and transcripts, to let users import processed audios and transcripts.
- [ ] Add support for transcribing audios via external Whisper APIs, so it can be hosted on low end servers.
- [ ] Add support for using S3 as the storage backend.


## Quick Start
### Running with Docker Compose (CPU Only)
1. Download the [docker-compose.yml](./docker-compose.yml) file to your working directory.
2. Create a `.env` to include the essential environment variables
```
OPENAI_API_KEY=<your_openai_api_key>
SESSION_KEY=<your_session_key>
HF_TOKEN=<your_hf_token>
SCRIBERR_USERNAME=<your_scriberr_username>
SCRIBERR_PASSWORD=<your_scriberr_password>
```
3. Run `docker compose up -d` to start the container and you can access the application at port `8080`


## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Scriberr](https://github.com/rishikanthc/Scriberr)
- [OpenAI Whisper](https://github.com/openai/whisper)
- [WhisperX](https://github.com/m-bain/whisperX)
- [HuggingFace](https://huggingface.co/)
- [PyAnnote Speaker Diarization](https://huggingface.co/pyannote/speaker-diarization)
- [Ollama](https://ollama.ai/)

