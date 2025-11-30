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

      <article className="docs-prose">
        <h2>What is Scriberr?</h2>
        <p className="mt-4">
          Scriberr is a powerful, self-hosted transcription application designed for privacy and performance. It converts audio files into text entirely offline, ensuring your data never leaves your machine. Whether you're a journalist, researcher, or developer, Scriberr provides a seamless workflow for transcribing, summarizing, and interacting with your audio content.
        </p>
        <p className="mt-4">
          Built with a robust Go backend and a modern React frontend, Scriberr is distributed as a single binary for easy deployment. It leverages state-of-the-art models like Whisper, NVIDIA Parakeet, and NVIDIA Canary to deliver high-accuracy transcriptions with word-level timestamps.
        </p>

        <h3 className="mt-8 text-xl font-semibold text-gray-900">Key Features</h3>
        <ul className="mt-4 list-disc pl-5 space-y-2">
          <li>
            <strong>Advanced Transcription Engines:</strong> Support for Whisper, NVIDIA Parakeet, and NVIDIA Canary models for superior accuracy and speed.
          </li>
          <li>
            <strong>Offline & Private:</strong> All processing happens locally on your device. No data is sent to the cloud.
          </li>
          <li>
            <strong>Hardware Acceleration:</strong> Optimized for NVIDIA GPUs (CUDA) with fallback to CPU execution.
          </li>
          <li>
            <strong>Speaker Diarization:</strong> Automatically identify and label different speakers in your audio (powered by PyAnnote).
          </li>
          <li>
            <strong>Scriberr Watcher CLI:</strong> A background service that automatically detects and transcribes new audio files in monitored directories.
          </li>
          <li>
            <strong>Interactive Player:</strong> Click-to-seek, waveform visualization, and synchronized playback.
          </li>
          <li>
            <strong>LLM Integration:</strong> Summarize and chat with your transcripts using your preferred LLM provider (Ollama, OpenAI, Anthropic, etc.).
          </li>
          <li>
            <strong>PWA Support:</strong> Install Scriberr as a Progressive Web App on mobile and desktop for a native-like experience.
          </li>
        </ul>

        <div className="mt-8">
          <Window src="/screenshots/scriberr-homepage.png" alt="Scriberr homepage" />
        </div>

        <p className="mt-8">
          Ready to get started? Check out the <a href="/docs/installation.html">Installation Guide</a> to set up Scriberr on your machine.
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
