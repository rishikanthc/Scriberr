import React from 'react';
import { createRoot } from 'react-dom/client';
import DocsLayout from '../components/DocsLayout';
import '../styles.css';

function Configuration() {
    return (
        <DocsLayout active="configuration">
            <header>
                <h1>Configuration</h1>
                <p className="mt-2">Setting up advanced features like Speaker Diarization.</p>
            </header>

            <article className="docs-prose mt-8">
                <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-8">
                    <h3 className="text-blue-800 font-semibold mt-0">Note on Local Execution</h3>
                    <p className="text-blue-700 mt-2 mb-0">
                        While you need to accept user agreements on Hugging Face to download the models, the actual diarization process happens entirely <strong>locally on your machine</strong>. No audio data is sent to Hugging Face or any third party.
                    </p>
                </div>

                <h2>Speaker Diarization Setup</h2>
                <p className="mt-4">
                    Scriberr uses <a href="https://github.com/pyannote/pyannote-audio" target="_blank" rel="noopener noreferrer">pyannote.audio</a> for speaker diarization. To use this feature, you need to obtain an access token from Hugging Face.
                </p>

                <h3 className="mt-6">Step 1: Hugging Face Account</h3>
                <p className="mt-2">
                    If you don't have one, create an account on <a href="https://huggingface.co/join" target="_blank" rel="noopener noreferrer">Hugging Face</a>.
                </p>

                <h3 className="mt-6">Step 2: Accept User Agreement</h3>
                <p className="mt-2">
                    Visit the following model page and accept the user agreement:
                </p>
                <ul className="mt-2 list-disc pl-5">
                    <li>
                        <a href="https://huggingface.co/pyannote/speaker-diarization-community-1" target="_blank" rel="noopener noreferrer">pyannote/speaker-diarization-community-1</a>
                    </li>
                </ul>
                <p className="mt-2 text-sm text-gray-600">
                    <em>Note: Previous versions required accepting agreements for multiple models. You now only need access to this single community model.</em>
                </p>

                <h3 className="mt-6">Step 3: Create Access Token</h3>
                <ol className="mt-2 list-decimal pl-5 space-y-2">
                    <li>Go to your <a href="https://huggingface.co/settings/tokens" target="_blank" rel="noopener noreferrer">Access Tokens settings</a>.</li>
                    <li>Create a new token with <strong>Read</strong> permissions.</li>
                    <li>Copy the token.</li>
                </ol>

                <h3 className="mt-6">Step 4: Configure Scriberr</h3>
                <p className="mt-2">
                    Open Scriberr, go to <strong>Settings</strong>, and paste your token into the <strong>Hugging Face Token</strong> field. Save the settings.
                </p>
                <p className="mt-2">
                    Scriberr will now be able to download the diarization model the first time you run a transcription with diarization enabled.
                </p>

                <hr className="my-12 border-gray-200" />

                <h2>NVIDIA Sortformer</h2>
                <p className="mt-4">
                    If you are using the NVIDIA Docker image, Sortformer is supported out of the box for diarization and does not require a Hugging Face token.
                </p>
            </article>
        </DocsLayout>
    );
}

const root = createRoot(document.getElementById('root')!);
root.render(
    <React.StrictMode>
        <Configuration />
    </React.StrictMode>
);
