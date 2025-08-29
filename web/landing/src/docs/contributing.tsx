import React from 'react';
import { createRoot } from 'react-dom/client';
import DocsLayout from '../components/DocsLayout';
import '../styles.css';

function Contributing() {
  return (
    <DocsLayout active="contributing">
      <header>
        <h1>Contributing</h1>
        <p className="mt-2">Thanks for your interest in improving Scriberr! Here’s how to get set up and contribute.</p>
      </header>

      <section>
        <h2>Guidelines</h2>
        <ul className="list-disc pl-5 mt-2 space-y-1">
          <li>Open an issue first for large changes to discuss scope and approach.</li>
          <li>Keep pull requests focused and small; write clear descriptions.</li>
          <li>Use conventional, imperative commit messages (e.g., <code>add docs sidebar link</code>, <code>fix queue status endpoint</code>).</li>
          <li>Follow coding styles: run <code>go fmt ./...</code>, <code>go vet ./...</code> and <code>npm run lint</code> in the frontend.</li>
          <li>Add tests where appropriate (Go tests live under <code>tests/</code> or next to packages).</li>
          <li>Update docs (README, swagger) when you change API shapes.</li>
        </ul>
      </section>

      <section>
        <h2>Prerequisites</h2>
        <ul className="list-disc pl-5 mt-2 space-y-1">
          <li>Node.js 18+ and npm</li>
          <li>Go 1.24+</li>
          <li>Python 3.11+ and <a href="https://docs.astral.sh/uv/" target="_blank" rel="noopener noreferrer">uv</a> (for transcription features)</li>
        </ul>
        <div className="bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2 overflow-x-auto">
          <pre>{`# macOS (Homebrew)\nbrew install node go python\ncurl -LsSf https://astral.sh/uv/install.sh | sh\n\n# Verify\nnode -v\nnpm -v\ngo version\npython3 --version\nuv --version`}</pre>
        </div>
      </section>

      <section>
        <h2>Build and run locally</h2>
        <h3 className="mt-2">Backend (dev)</h3>
        <div className="bg-gray-50 rounded-lg p-4 font-mono text-sm mt-1 overflow-x-auto">
          <pre>{`# Copy and edit environment\ncp -n .env.example .env || true\n\n# Run API server\ngo run cmd/server/main.go`}</pre>
        </div>

        <h3 className="mt-4">Frontend (dev)</h3>
        <div className="bg-gray-50 rounded-lg p-4 font-mono text-sm mt-1 overflow-x-auto">
          <pre>{`cd web/frontend\nnpm ci\nnpm run dev`}</pre>
        </div>

        <h3 className="mt-4">Full build (embed UI)</h3>
        <p className="mt-1">Use the build script to bundle the React app and compile the Go binary with embedded assets.</p>
        <div className="bg-gray-50 rounded-lg p-4 font-mono text-sm mt-1 overflow-x-auto">
          <pre>{`# From repo root\nchmod +x ./build.sh\n./build.sh\n\n# Run the server\n./scriberr`}</pre>
        </div>
      </section>

      <section>
        <h2>Testing</h2>
        <div className="bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2 overflow-x-auto">
          <pre>{`# Run all Go tests with verbose output\ngo test ./... -v\n\n# Or target test suites\ngo test ./tests -run TestAPITestSuite -v\n\n# Lint frontend\ncd web/frontend && npm run lint`}</pre>
        </div>
      </section>

      <section>
        <h2>Submitting changes</h2>
        <ul className="list-disc pl-5 mt-2 space-y-1">
          <li>Create a feature branch from <code>main</code>.</li>
          <li>Ensure CI passes locally: build, test, lint.</li>
          <li>Open a pull request with a clear summary and screenshots/GIFs for UI changes.</li>
          <li>Link issues (e.g., <code>Closes #123</code>) when applicable.</li>
        </ul>
      </section>
    </DocsLayout>
  );
}

const root = createRoot(document.getElementById('root')!);
root.render(
  <React.StrictMode>
    <Contributing />
  </React.StrictMode>
);

