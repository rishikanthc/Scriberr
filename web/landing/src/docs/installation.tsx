import React from 'react';
import { createRoot } from 'react-dom/client';
import DocsLayout from '../components/DocsLayout';
import '../styles.css';

function Installation() {
  return (
    <DocsLayout active="installation">
      <header>
        <h1>Installation</h1>
        <p className="mt-2">Get Scriberr running on your system in a few minutes.</p>
      </header>

      <section>
        <h2>Install with Homebrew (macOS & Linux)</h2>
        <p className="mt-2">
          The easiest way to install Scriberr is using Homebrew. If you don’t have Homebrew installed,
          <a href="https://brew.sh" target="_blank" rel="noopener noreferrer" className="ml-1">get it here first</a>.
        </p>

        <div className="bg-gray-50 rounded-lg p-4 font-mono text-sm mt-3">
          <div className="text-gray-800">
            <span className="text-green-600"># Add the Scriberr tap</span><br />
            brew tap rishikanthc/scriberr<br /><br />

            <span className="text-green-600"># Install Scriberr (automatically installs UV dependency)</span><br />
            brew install scriberr<br /><br />

            <span className="text-green-600"># Start the server</span><br />
            scriberr
          </div>
        </div>

        <p className="mt-3">Open <code className="bg-gray-100 px-1 rounded">http://localhost:8080</code> in your browser.</p>

        <h3 className="mt-8">Configuration</h3>
        <p className="mt-2">Scriberr works out of the box. To customize settings, create a <code className="bg-gray-100 px-1 rounded">.env</code> file:</p>
        <div className="bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2">
          <div className="text-gray-800">
            <span className="text-green-600"># Server settings</span><br />
            HOST=localhost<br />
            PORT=8080<br /><br />

            <span className="text-green-600"># Data storage (optional)</span><br />
            DATABASE_PATH=./data/scriberr.db<br />
            UPLOAD_DIR=./data/uploads<br />
            WHISPERX_ENV=./data/whisperx-env<br /><br />

            <span className="text-green-600"># Custom paths (if needed)</span><br />
            UV_PATH=/custom/path/to/uv
          </div>
        </div>

        <h3 className="mt-8">Troubleshooting</h3>
        <div className="space-y-3 mt-2">
          <div>
            <strong>Command not found</strong>
            <p className="mt-1">Make sure the binary is in your PATH or run it with the full path: <code className="bg-gray-100 px-1 rounded">./scriberr</code></p>
          </div>
          <div>
            <strong>Transcription not working</strong>
            <p className="mt-1">Ensure Python 3.11+ and UV are installed. Check logs on start for Python environment issues.</p>
          </div>
          <div>
            <strong>Port already in use</strong>
            <p className="mt-1">Set a different port with <code className="bg-gray-100 px-1 rounded">PORT=8081 scriberr</code> or add it to your .env file.</p>
          </div>
        </div>
      </section>

      <section>
        <h2 className="mt-12">Install with Docker</h2>
        <p className="mt-2">Run Scriberr in a container with all dependencies included. We provide images for both CPU and NVIDIA GPU (CUDA) environments.</p>

        <h3 className="mt-4">CPU Version (Standard)</h3>
        <div className="bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2 overflow-x-auto">
          <span className="text-green-600"># Run with Docker (data persisted in volume)</span>
          <pre className="mt-2">{`docker run -d \\
  --name scriberr \\
  -p 8080:8080 \\
  -v scriberr_data:/app/data \\
  --restart unless-stopped \\
  ghcr.io/rishikanthc/scriberr:latest`}</pre>
        </div>

        <h3 className="mt-8">NVIDIA GPU Version (CUDA)</h3>
        <p className="mt-2">
          For hardware acceleration, use the CUDA image. Requires <a href="https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html" target="_blank" rel="noopener noreferrer">NVIDIA Container Toolkit</a> to be installed on your host.
        </p>
        <div className="bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2 overflow-x-auto">
          <pre className="mt-2">{`docker run -d \\
  --name scriberr-cuda \\
  --gpus all \\
  -p 8080:8080 \\
  -v scriberr_data:/app/data \\
  --restart unless-stopped \\
  ghcr.io/rishikanthc/scriberr:cuda-latest`}</pre>
        </div>

        <h3 className="mt-8">Docker Compose</h3>
        <p className="mt-2">Create a <code className="bg-gray-100 px-1 rounded">docker-compose.yml</code> with the following:</p>
        <div className="bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2 overflow-x-auto">
          <pre>{`version: '3.9'
services:
  scriberr:
    # Use ghcr.io/rishikanthc/scriberr:cuda-latest for GPU support
    image: ghcr.io/rishikanthc/scriberr:latest
    container_name: scriberr
    ports:
      - "8080:8080"
    volumes:
      - scriberr_data:/app/data
    # Uncomment for GPU support
    # deploy:
    #   resources:
    #     reservations:
    #       devices:
    #         - driver: nvidia
    #           count: 1
    #           capabilities: [gpu]
    restart: unless-stopped

volumes:
  scriberr_data:`}</pre>
        </div>
        <div className="bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2">
          <div className="text-gray-800">
            <span className="text-green-600"># Start the service</span><br />
            docker compose up -d
          </div>
        </div>

        <p className="mt-3">Access the web interface at <code className="bg-gray-100 px-1 rounded">http://localhost:8080</code>.</p>
      </section>

      <section>
        <p className="mt-10">
          To configure speaker diarization, see the <a href="/docs/configuration.html">Configuration guide</a>.
        </p>
      </section>
    </DocsLayout>
  );
}

const root = createRoot(document.getElementById('root')!);
root.render(
  <React.StrictMode>
    <Installation />
  </React.StrictMode>
);
