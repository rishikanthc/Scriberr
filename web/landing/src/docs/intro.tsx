import React from 'react';
import { createRoot } from 'react-dom/client';
import DocsLayout from '../components/DocsLayout';
import Window from '../components/Window';
import '../styles.css';

function Introduction() {
  return (
    <DocsLayout active="intro">
      <header>
        <h1>Introduction</h1>
        <p className="mt-2">A quick overview of Scriberr and what you can do with it.</p>
      </header>

      <article>
        <h2>What is Scriberr?</h2>
        <p className="mt-2">
          Scriberr is a self-hosted, offline transcription app for converting audio files into text. Record or upload audio, get it transcribed, and quickly summarize or chat using your preferred LLM provider. Scriberr doesn’t require GPUs (although GPUs can be used for acceleration) and runs on modern CPUs, offering a range of trade-offs between speed and transcription quality.
        </p>
        <p className="mt-2">
          Scriberr is built with React on the frontend and Go on the backend, compiled into a single binary. It uses the WhisperX engine and open-source Whisper models for transcription. Some key features include:
        </p>
        <ul className="mt-2 list-disc pl-5 space-y-1">
          <li>Fine-tune advanced transcription parameters for precise control over quality</li>
          <li>Built-in recorder to capture audio directly in‑app</li>
          <li>Speaker diarization to identify and label different speakers</li>
          <li>Summarize &amp; chat with your audio using LLMs</li>
          <li>Highlight, annotate, and tag notes</li>
          <li>Save configurations as profiles for different audio scenarios</li>
          <li>API endpoints for building your own automations and applications</li>
        </ul>

        <div className="mt-5">
          <Window src="/screenshots/scriberr-homepage.png" alt="Scriberr homepage" />
        </div>

        <p className="mt-4">
          To install Scriberr, check the <a href="/docs/installation.html">installation page</a> for setup instructions.
        </p>
      </article>
    </DocsLayout>
  );
}

const root = createRoot(document.getElementById('root')!);
root.render(
  <React.StrictMode>
    <Introduction />
  </React.StrictMode>
);
