import React from 'react';
import { createRoot } from 'react-dom/client';
import DocsLayout from '../components/DocsLayout';
import '../styles.css';

function Usage() {
    return (
        <DocsLayout active="usage">
            <header>
                <h1>Usage Guide</h1>
                <p className="mt-2">Tips and tricks to get the most out of Scriberr.</p>
            </header>

            <article className="docs-prose mt-8">
                <section className="mb-12">
                    <h2>First Run</h2>
                    <p className="mt-4">
                        When you first launch Scriberr, it may take a few minutes to initialize. This is because the application needs to download the necessary transcription models (Whisper) to your local machine.
                    </p>
                    <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mt-4">
                        <h4 className="text-yellow-800 font-semibold mt-0">Startup Time</h4>
                        <p className="text-yellow-700 mt-2 mb-0">
                            Please be patient during the first run. You can check the application logs to see the download progress. Once the models are cached, subsequent startups will be instant.
                        </p>
                    </div>
                </section>

                <section className="mb-12">
                    <h2>Interactive Player Features</h2>

                    <h3 className="mt-6">Click-to-Seek</h3>
                    <p className="mt-2">
                        Navigate your audio effortlessly by holding <kbd className="kbd">Cmd</kbd> (macOS) or <kbd className="kbd">Ctrl</kbd> (Windows/Linux) and clicking on any word in the transcript. The audio player will instantly jump to that timestamp.
                    </p>

                    <h3 className="mt-6">Highlighting & Notes</h3>
                    <p className="mt-2">
                        Select any text in the transcript to highlight it and add a note.
                    </p>
                </section>

                <section className="mb-12">
                    <h2>Scriberr Watcher CLI</h2>
                    <p className="mt-4">
                        The Watcher CLI allows you to automatically upload files from a local folder to Scriberr.
                    </p>

                    <h3 className="mt-6">Installation & Setup</h3>
                    <div className="bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2 overflow-x-auto">
                        <div className="text-gray-500 mb-2"># 1. Login to your Scriberr instance</div>
                        <div className="text-gray-900 mb-4">scriberr login --server http://localhost:8080</div>

                        <div className="text-gray-500 mb-2"># 2. Install the watcher service for a specific folder</div>
                        <div className="text-gray-900 mb-4">scriberr install /path/to/watch/folder</div>

                        <div className="text-gray-500 mb-2"># 3. Start the background service</div>
                        <div className="text-gray-900">scriberr start</div>
                    </div>

                    <h3 className="mt-6">Commands Reference</h3>
                    <div className="overflow-x-auto mt-4">
                        <table className="w-full text-sm text-left">
                            <thead className="text-xs text-gray-500 uppercase bg-gray-50">
                                <tr>
                                    <th className="px-4 py-3">Command</th>
                                    <th className="px-4 py-3">Description</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-gray-100">
                                <tr>
                                    <td className="px-4 py-3 font-mono">login</td>
                                    <td className="px-4 py-3">Authenticate with the server. Opens a browser window.</td>
                                </tr>
                                <tr>
                                    <td className="px-4 py-3 font-mono">install [folder]</td>
                                    <td className="px-4 py-3">Register the watcher as a system service for the specified folder.</td>
                                </tr>
                                <tr>
                                    <td className="px-4 py-3 font-mono">start</td>
                                    <td className="px-4 py-3">Start the background watcher service.</td>
                                </tr>
                                <tr>
                                    <td className="px-4 py-3 font-mono">stop</td>
                                    <td className="px-4 py-3">Stop the background service.</td>
                                </tr>
                                <tr>
                                    <td className="px-4 py-3 font-mono">logs</td>
                                    <td className="px-4 py-3">View (tail) the service logs to debug issues.</td>
                                </tr>
                                <tr>
                                    <td className="px-4 py-3 font-mono">uninstall</td>
                                    <td className="px-4 py-3">Remove the background service.</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </section>

                <section className="mb-12">
                    <h2>Model Selection Guide</h2>
                    <p className="mt-4">
                        Scriberr supports various Whisper models. Choose based on your hardware and accuracy needs.
                    </p>

                    <div className="overflow-x-auto mt-4">
                        <table className="min-w-full text-left text-sm">
                            <thead>
                                <tr className="border-b border-gray-200">
                                    <th className="py-2 font-semibold">Model</th>
                                    <th className="py-2 font-semibold">Best For</th>
                                    <th className="py-2 font-semibold">Notes</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr className="border-b border-gray-100">
                                    <td className="py-2"><strong>Whisper Tiny / Base</strong></td>
                                    <td className="py-2">Fastest transcription, low resource usage</td>
                                    <td className="py-2">Good for clear audio. <code>.en</code> versions are English-only and slightly more accurate.</td>
                                </tr>
                                <tr className="border-b border-gray-100">
                                    <td className="py-2"><strong>Whisper Small / Medium</strong></td>
                                    <td className="py-2">Balanced speed and accuracy</td>
                                    <td className="py-2">Recommended for most users on standard CPUs.</td>
                                </tr>
                                <tr className="border-b border-gray-100">
                                    <td className="py-2"><strong>Whisper Large v3</strong></td>
                                    <td className="py-2">Highest accuracy, multilingual</td>
                                    <td className="py-2">Slower on CPU, recommended if you have a GPU.</td>
                                </tr>
                                <tr className="border-b border-gray-100">
                                    <td className="py-2"><strong>NVIDIA Canary / Parakeet</strong></td>
                                    <td className="py-2">State-of-the-art accuracy and speed</td>
                                    <td className="py-2">Runs on CPU, but significantly faster with NVIDIA GPU (CUDA).</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </section>

                <section className="mb-12">
                    <h2>Progressive Web App (PWA)</h2>
                    <p className="mt-4">
                        You can install Scriberr as a standalone app on your device.
                    </p>
                    <ul className="mt-2 list-disc pl-5">
                        <li><strong>Desktop (Chrome/Edge):</strong> Click the install icon in the address bar.</li>
                        <li><strong>iOS:</strong> Open in Safari, tap "Share", then "Add to Home Screen".</li>
                        <li><strong>Android:</strong> Open in Chrome, tap the menu, then "Install App".</li>
                    </ul>
                </section>

                <section className="mb-12">
                    <h2>Viewing Logs</h2>
                    <p className="mt-4">
                        If you encounter issues, checking the logs is the first step.
                    </p>
                    <ul className="mt-2 list-disc pl-5">
                        <li><strong>Docker:</strong> Run <code>docker logs scriberr</code></li>
                        <li><strong>Binary:</strong> Check the terminal output where you launched the app.</li>
                    </ul>
                </section>
            </article>
        </DocsLayout>
    );
}

const root = createRoot(document.getElementById('root')!);
root.render(
    <React.StrictMode>
        <Usage />
    </React.StrictMode>
);
