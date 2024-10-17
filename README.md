# Scriberr

This is Scriberr, a self-hostable AI audio transcription app. Scriberr uses the open-source [Whisper](https://github.com/openai/whisper) models from OpenAI,
to transcribe audio files locally on your hardware. It uses the [Whisper.cpp](https://github.com/ggerganov/whisper.cpp) high-performance inference engine
for OpenAI's Whisper. Scriberr also allows you to summarize transcripts using ollama or OpenAI's ChatGPT API, with your own custom prompts. From v0.2.0 Scriberr supports
offline speaker diarization.

## Features
- Fast transcription with support for hardware acceleration across a wide variety of platforms
- Customizable compute settings. Choose #threads, #cores and your model size
- Transcription happens locally on device
- Exposes API endpoints for automation pipelines and integrating with other tools
- Optionally summarize transcripts with ChatGPT or Ollama
- Use your own custom prompts for summarization
- Mobile ready
- Simple & Easy to use
- Speaker Diarization (**New**)

and more to come. Checkout the planned features section.

## Demo and Screenshots

> [!note]
> Demo was run locally on my Macbook Air M2 using docker.
> Performance depends on the size of the model used and also
> number of cores and threads you assign.  Was running a lot of things in the background and this is in dev mode so it's really slow.

https://github.com/user-attachments/assets/69d0c5a8-3412-4af5-a312-f3eddebc392e


![CleanShot 2024-10-04 at 14 42 54@2x](https://github.com/user-attachments/assets/90e68ebd-695e-4043-8d51-83c704a18c5c)
![CleanShot 2024-10-04 at 14 48 31@2x](https://github.com/user-attachments/assets/a8ecfa26-84aa-4091-8f22-481f0b5e67e6)
![CleanShot 2024-10-04 at 14 49 08@2x](https://github.com/user-attachments/assets/22820b96-f982-46da-8a71-79ea73559c79)
![CleanShot 2024-10-04 at 15 11 27@2x](https://github.com/user-attachments/assets/6e10b0c1-cf97-4cf6-ab47-591b6da607ef)




## Installation

Scriberr can be deployed using Docker. Use the docker-compose shown below with your configuration values.
Under the directory or volume you are mapping to `/scriberr`, please create the following 2 sub-directories,
`audio` and `transcripts`.

> [!warning]
> Make sure to create the sub-directories inside `SCRIBO_FILES` as transcription will fail silently without that.

> [!important]
> On first load, the app will throw a 500 Error because the database collection hasn't been created.
> Please reload the page for the app to start working. This only happens on the very first run after
> install.

```yaml
services:
  scriberr:
    image: ghcr.io/rishikanthc/scriberr:0.2.2 #use nightly for the latest cutting edge version (might be unstable)
    depends_on:
      redis:
        condition: service_started

    ports:
      - "3000:3000"
      - "8080:8080" # Optionally expose DB UI
      - "9243:9243" # Optionally expose JobQueue UI
    environment:
      - OPENAI_API_KEY=<reallylongsecretkey>
      - OPENAI_ENDPOINT=http://ollama:11434/v1
      - OPENAI_MODEL=llama3.2 # Ensure this model matches in `ollama-models` service
      - OPENAI_ROLE=user
      - POCKETBASE_ADMIN_EMAIL=admin@example.com
      - POCKETBASE_ADMIN_PASSWORD=password
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - SCRIBO_FILES=/scriberr
    volumes:
      - ./.dockerdata/pb_data:/app/db
      - ./.dockerdata/scriberr:/scriberr

  redis:
    image: redis:7-alpine
    volumes:
      - ./.dockerdata/redis:/data
```

### Full Local Stack

To run all components locally, including Ollama in place of OpenAI, see [`docker-compose.ollama.yaml`](./docker-compose.ollama.yaml).

```sh
$ mkdir -p .dockerdata/scriberr/audio .dockerdata/scriberr/transcripts
$ docker-compose -f docker-compose.ollama.yaml up
...
```

The app will be available in your browser: `http://localhost:3000`

Additionally, you can run the container against an external Ollama instance by passing in the appropriate values for these environment variables:

```env
OPENAI_ENDPOINT=<ollama service api url>
OPENAI_MODEL=<the ollama model> # must already be pulled
OPENAI_ROLE=user
```

> [!warning]
> This will be _very_ slow without an NVIDIA GPU to pass through.

> [!warning]
> If you have issues re-starting the stack (`403: 'Only admins can perform this action.'`), clear the Auth token cookie.

## Planned Features

- [x] Speaker diarization for speaker labels
- [x] File actions - rename, delete
- [ ] Provide multiple algorithms for speaker label corrections
- [ ] Hardware Acceleration setup wizard
- [ ] Youtube integration
- [ ] Subtitle generation
- [ ] Support for other languages
- [ ] Audio recording functionality
- [ ] Full text fuzzy search
- [ ] Tag based organization system
- [ ] Follow along text with playback
- [ ] Edit summaries
- [ ] Export options


## Known Bugs
- First app load will load a blank due to missing database. Reloading will fix it.
- ~~Requires page refresh to load audio for newly transcribed files~~
- ~~Automatic update of processed files is finnicky and might require a page refresh for update~~

## Note
This app is under development, so expect a few rough edges and minor bugs. Expect breaking changes
in the first few minor releases. Will smooth out and try to avoid it as best as I can

If you like this project I would really appreciate it if you could star this repository.
