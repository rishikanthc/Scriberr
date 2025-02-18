# Important
im in the process of doing a full rewrite of the app.There are a few reasona for this:
- Svelte 5 which brings a lot of improvements especially for reactivity
- the previous implementation was a bit more backy than i wanted it to be
- diarization wasnt great

i had to take some time off this project due to some commitments. Ill be back on working on this project regularly. you can wxpect weekly updates from here on and ill clean up things for a new release. 
in the meantime as tou might have noticed the existing docker images arent valid. this is because currently docker pulls whisper.cpp from their official repo and sets it up. unfortunately this turned out to be a bad move as whisper.cpp changed thier build process and hence the current setup no longer works. 
i have already moved the main branch ahead for the new release. hence if you want to try out the new release please download the repo and run docker build to create your own image. 
My sincere apologies for the inconvenience and ill fix this up soon. 

in the meantime folks who have the time and resources to build and try the new release any feedback would be greatlt appreciated. 
also a warning this release is a breaking change and you will loose your old data. 


# Scriberr
[![ci](https://github.com/rishikanthc/Scriberr/actions/workflows/github-actions-docker.yml/badge.svg?event=push)](https://github.com/rishikanthc/Scriberr/actions/workflows/github-actions-docker.yml)

This is Scriberr, a self-hostable AI audio transcription app. Scriberr uses the open-source [Whisper](https://github.com/openai/whisper) models from OpenAI,
to transcribe audio files locally on your hardware. It uses the [Whisper.cpp](https://github.com/ggerganov/whisper.cpp) high-performance inference engine
for OpenAI's Whisper. Scriberr also allows you to summarize transcripts using ollama or OpenAI's ChatGPT API, with your own custom prompts. From v0.2.0 Scriberr supports
offline speaker diarization. Check out the documentation [website](https://scriberr.app) for more details and instructions.

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
For installation and usage instruction refer the documentation website at [scriberr.app](https://scriberr.app)

## Note
This app is under development, so expect a few rough edges and minor bugs. Expect breaking changes
in the first few minor releases. Will smooth out and try to avoid it as best as I can

If you like this project I would really appreciate it if you could star this repository.
