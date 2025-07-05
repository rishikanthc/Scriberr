---
---

# Usage

## Audio sources

Scriberr accepts audio for transcription in 3 ways:

* You can bulk upload audio files directly from your computer

* You can use the in-built audio recorder to record your own audio and upload it

* You can directly transcribe a youtube video by pasting the youtube link

For the audio source options click on the _New Recording_ button which will show you a popover with the above 3 options to choose from.&#x20;

Once you select an option and provide the required inputs, the audio will start getting uploaded or downloaded (if youtube) and you will get a notification when the audio is available in the app and you will see it listed in the home page under the _Audio_ tab.

If recording audio using the in-built audio recorder, then you can optionally set the title for the recorded audio. This option is available for the youtube video transcription as well.

<br />

## Actions

On the audio tab you can right click on a audio entry to:

* Transcribe the audio

* Generate a summary of the transcript using a template prompt  (only if transcript is available)

* Chat with the transcript (only if a transcript is available)

* Delete the audio and related files

<br />

Left-Clicking on the audio entry will open the Audio details dialogue which will show a audio player, transcript and summary if available and options to download the transcript and summary.

<br />

### Transcribe

Clicking on the transcribe option will open the transcription settings dialog. Under the _Basic Settings_ tab you will see inputs to choose model size and enable diarization. Enabling diarization will reveal the diarization parameters to configure speaker diarization.&#x20;

<br />

> \[!NOTE]
>
> Please always input min and max speakers for diarization

<br />

### Summarize

You can summarize transcripts using OpenAI/Ollama API endpoints.

> \[!IMPORTANT]
>
> Either the OpenAI or Ollama credentials must be configured as environment variables for summarization and chat to work

<br />

Clicking the summarize option will open up the summarizer dialog which will allow you to choose the model for summarization and the template prompt from available templates,  to use for summarization. This will show up as a job for summarization and once it completes it updates the summary.

The summary can be viewed by clicking on an audio entry from the _Audio_ tab.

### Chat

Clicking on the chat option will open a chat window. The chat window, on the right has inputs for creating a new chat session and below it all previous chat sessions. For each new chat session, the transcript is added to the context. 