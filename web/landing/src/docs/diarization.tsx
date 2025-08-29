import React from 'react';
import { createRoot } from 'react-dom/client';
import DocsLayout from '../components/DocsLayout';
import Window from '../components/Window';
import '../styles.css';

function DiarizationSetup() {
  return (
    <DocsLayout active="diarization">
      <header>
        <h1>Setting up diarization</h1>
        <p className="mt-2">Enable local speaker diarization using pyannote models.</p>
      </header>

      <section>
        <p className="mt-2">
          Diarization uses the open‑source pyannote models for speaker segmentation. These models are hosted on Hugging Face.
          To access them, you will need a Hugging Face account. The access token is only used to download the models —
          diarization runs locally and does NOT use any third‑party cloud services. Follow the steps below to enable diarization.
        </p>

        <p className="mt-4">Create an account on <a href="https://huggingface.co" target="_blank" rel="noopener noreferrer">https://huggingface.co</a></p>
        <p className="mt-2">Navigate to the following repositories:</p>
        <ul className="list-disc pl-5 mt-1 space-y-1">
          <li><a href="https://huggingface.co/pyannote/speaker-diarization-3.0" target="_blank" rel="noopener noreferrer">https://huggingface.co/pyannote/speaker-diarization-3.0</a></li>
          <li><a href="https://huggingface.co/pyannote/speaker-diarization" target="_blank" rel="noopener noreferrer">https://huggingface.co/pyannote/speaker-diarization</a></li>
          <li><a href="https://huggingface.co/pyannote/speaker-diarization-3.1" target="_blank" rel="noopener noreferrer">https://huggingface.co/pyannote/speaker-diarization-3.1</a></li>
          <li><a href="https://huggingface.co/pyannote/segmentation-3.0" target="_blank" rel="noopener noreferrer">https://huggingface.co/pyannote/segmentation-3.0</a></li>
        </ul>

        <p className="mt-3">
          Accept their user conditions. Navigate to <a href="https://huggingface.co/settings/gated-repos" target="_blank" rel="noopener noreferrer">https://huggingface.co/settings/gated-repos</a> to ensure these repos show up on that page as gated repos.
        </p>
        <p className="mt-2">
          Then navigate to Settings -&gt; Access Tokens and create a new token. Under permissions, enable all under the
          "Repositories" section. Copy and save the token somewhere securely. You will need this access token for using diarization.
        </p>

        <p className="mt-3">
          In the app, to enable diarization — either when creating a new profile or using the Transcribe+ action — open the
          Diarization tab in the dialog and paste your token into the Hugging Face Token field.
        </p>
      </section>

      <div className="mt-6">
        <Window className="w-[300px] mx-auto" src="/screenshots/scriberr-diarization-setup.png" alt="Diarization setup in Scriberr" />
      </div>
    </DocsLayout>
  );
}

const root = createRoot(document.getElementById('root')!);
root.render(
  <React.StrictMode>
    <DiarizationSetup />
  </React.StrictMode>
);
