import React from 'react';
import { createRoot } from 'react-dom/client';
import DocsLayout from '../components/DocsLayout';
import '../styles.css';

function Features() {
    return (
        <DocsLayout active="features">
            <header>
                <h1>Features</h1>
                <p className="mt-2">A comprehensive list of everything Scriberr can do.</p>
            </header>

            <article className="docs-prose mt-8">
                <section className="mb-12">
                    <h2>Transcription & Audio</h2>
                    <ul className="mt-4 space-y-4">
                        <li>
                            <strong>Multi-Model Support:</strong> Choose between Whisper, NVIDIA Parakeet, and NVIDIA Canary models to balance speed and accuracy.
                        </li>
                        <li>
                            <strong>GPU Acceleration:</strong> Full CUDA support for NVIDIA GPUs, ensuring lightning-fast transcription.
                        </li>
                        <li>
                            <strong>Speaker Diarization:</strong> Identify "who said what" with high accuracy using PyAnnote audio models (requires configuration).
                        </li>
                        <li>
                            <strong>Word-Level Timestamps:</strong> Precise alignment of text with audio for accurate seeking and editing.
                        </li>
                        <li>
                            <strong>In-App Recording:</strong> Record audio directly from your microphone without needing external tools.
                        </li>
                        <li>
                            <strong>Batch Processing:</strong> Upload multiple files at once and let Scriberr handle the queue.
                        </li>
                    </ul>
                </section>

                <section className="mb-12">
                    <h2>Productivity & Organization</h2>
                    <ul className="mt-4 space-y-4">
                        <li>
                            <strong>Click-to-Seek:</strong> CMD/CTRL + Click on any word to jump the audio to that exact moment.
                        </li>
                        <li>
                            <strong>Highlighting & Notes:</strong> Highlight text and add notes for easy reference.
                        </li>
                        <li>
                            <strong>Export Options:</strong> Export transcripts to TXT, SRT, or JSON formats.
                        </li>
                    </ul>
                </section>

                <section className="mb-12">
                    <h2>AI & Automation</h2>
                    <ul className="mt-4 space-y-4">
                        <li>
                            <strong>LLM Chat:</strong> Chat with your transcripts to ask questions, extract insights, or generate summaries.
                        </li>
                        <li>
                            <strong>Custom Prompts:</strong> Create and save custom prompts for repetitive tasks like "Summarize meeting" or "Extract action items".
                        </li>
                        <li>
                            <strong>Scriberr Watcher:</strong> A CLI tool that monitors folders and automatically transcribes new audio files in the background.
                        </li>
                    </ul>
                </section>

                <section className="mb-12">
                    <h2>Deployment & Access</h2>
                    <ul className="mt-4 space-y-4">
                        <li>
                            <strong>Single Binary:</strong> Easy to deploy with no complex dependencies (for CPU mode).
                        </li>
                        <li>
                            <strong>Docker Support:</strong> Ready-to-use Docker images for both CPU and CUDA environments.
                        </li>
                        <li>
                            <strong>PWA:</strong> Install as a Progressive Web App on iOS, Android, macOS, and Windows.
                        </li>
                        <li>
                            <strong>API Access:</strong> Full REST API for integrating Scriberr into your own workflows.
                        </li>
                    </ul>
                </section>
            </article>
        </DocsLayout>
    );
}

const root = createRoot(document.getElementById('root')!);
root.render(
    <React.StrictMode>
        <Features />
    </React.StrictMode>
);
