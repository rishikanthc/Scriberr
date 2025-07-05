# Scriberr - Introduction

Scriberr is a self-hostable, offline audio transcription app. It uses the WhisperX engine for fast transcription with automatic language detection, speaker diarization and support for all whisper models. Built with Go and Svelte, the server runs as a single binary and is fast and highly responsive.

#### New Features and improvements
* Fast transcription with support for all model sizes
* Automatic language detection
* Uses VAD and ASR models for better alignment and speech detection to remove silence periods
* Speaker diarization (Speaker detection and identification)
* Automatic summarization using OpenAI/Ollama endpoints
* Markdown rendering of Summaries <span class="new-tag">(NEW)</span>
* AI Chat with transcript using OpenAI/Ollama endpoints <span class="new-tag">(NEW)</span>
  * Multiple chat sessions for each transcript <span class="new-tag">(NEW)</span>
* Built-in audio recorder
* YouTube video transcription <span class="new-tag">(NEW)</span>
* Download transcript as plaintext / JSON / SRT file <span class="new-tag">(NEW)</span>
* Save and reuse summarization prompt templates
* Tweak advanced parameters for transcription and diarization models <span class="new-tag">(NEW)</span>
* Audio playback follow (highlights transcript segment currently being played) <span class="new-tag">(NEW)</span>
* Stop or terminate running transcription jobs <span class="new-tag">(NEW)</span>
* Better reactivity and responsiveness <span class="new-tag">(NEW)</span>
* Toast notifications for all actions to provide instant status <span class="new-tag">(NEW)</span>
* Simplified deployment - single binary (Single container) <span class="new-tag">(NEW)</span>

<style>
  .new-tag {
    background: linear-gradient(-45deg, #ee7752, #e73c7e, #23a6d5, #23d5ab);
    background-size: 400% 400%;
    animation: gradient 3s ease infinite;
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    font-weight: bold;
    padding: 2px 6px;
    border-radius: 4px;
  }

  @keyframes gradient {
    0% {
      background-position: 0% 50%;
    }
    50% {
      background-position: 100% 50%;
    }
    100% {
      background-position: 0% 50%;
    }
  }

  /* Make the feature list text gray-100 */
  ul li {
    color: rgb(243 244 246) !important;
  }
</style>

<br />

### Under the hood

Scriberr uses Go for the backend, Svelte for the frontend and Python for AI transcription. The frontend is compiled to a static SPA (plain html and js) which is then embedded into the Go backend binary to provide a single binary. 