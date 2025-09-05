import React, { useMemo, useState } from 'react';
import GithubBadge from './components/GithubBadge';
import { createRoot } from 'react-dom/client';
import './styles.css';

type ChangeItem = { type: 'Added' | 'Changed' | 'Fixed' | 'Deprecated' | 'Removed' | 'Security'; text: string };
type Release = { version: string; date: string; tag?: 'Latest' | 'Beta'; notes: ChangeItem[] };

const RELEASES: Release[] = [
  {
    version: '1.1.0',
    date: '2025-09-05',
    tag: 'Latest',
    notes: [
      // Added
      { type: 'Added', text: 'Donation link to README and ko-fi badge for project support.' },
      { type: 'Added', text: 'Table for storing transcription statistics.' },
      { type: 'Added', text: 'Animation during summary generation to indicate processing.' },
      { type: 'Added', text: 'Default transcription profiles.' },
      { type: 'Added', text: 'Info to transcripts to view parameters and stats.' },
      { type: 'Added', text: 'Optional auto-transcription on upload.' },
      { type: 'Added', text: 'Support for renaming speakers.' },
      { type: 'Added', text: 'Transcription report in transcript section.' },
      { type: 'Added', text: 'Sponsors section.' },
      
      // Changed
      { type: 'Changed', text: 'Increase timeout to query OpenAI from 30 to 300 seconds.' },
      { type: 'Changed', text: 'Makes summary template dialogue larger to provide more space for text input.' },
      { type: 'Changed', text: 'Makes summary template text input scrollable.' },
      { type: 'Changed', text: 'Moves auto transcription settings to transcription tab.' },
      { type: 'Changed', text: 'Update README with Nvidia GPU support and Docker example.' },
      { type: 'Changed', text: 'Updates Dockerfile for CUDA and compose files.' },
      
      // Fixed
      { type: 'Fixed', text: 'Chat with GPT-5 models support (fixes #173).' },
      { type: 'Fixed', text: 'Mobile transcript view toolbar for better visibility and UX.' },
      { type: 'Fixed', text: 'CI/CD for project website - API and changelog pages not being updated.' },
      { type: 'Fixed', text: 'Go releaser syscall related errors during packaging.' },
      { type: 'Fixed', text: 'Incorrect persistence of old speaker data when re-transcribed.' },
      { type: 'Fixed', text: 'Job termination issues.' },
      { type: 'Fixed', text: 'Missing parameters in transcription info.' },
      { type: 'Fixed', text: 'Project page issues.' },
      
      // Removed
      { type: 'Removed', text: 'Build files from git repository.' },
      { type: 'Removed', text: 'Data tracking files.' },
    ],
  },
  {
    version: '1.0.4',
    date: '2025-09-01',
    notes: [
      { type: 'Fixed', text: 'Fix Homebrew package.' },
    ],
  },
  {
    version: '1.0.3',
    date: '2025-08-31',
    notes: [
      { type: 'Fixed', text: 'Fixes #163.' },
    ],
  },
  {
    version: '1.0.2',
    date: '2025-08-31',
    notes: [
      { type: 'Fixed', text: 'Fix translate issues on Arch Linux and Ubuntu.' },
    ],
  },
  {
    version: '1.0.1',
    date: '2025-08-30',
    notes: [
      { type: 'Changed', text: 'Default container UID/GID to 1000.' },
      { type: 'Fixed', text: 'Docker now respects environment-provided UID/GID.' },
      { type: 'Fixed', text: 'Resolve permission errors for Docker bind mounts.' },
      { type: 'Changed', text: 'Diarization uses whisperX-cli instead of Python script (fixes #158, #160).' },
      { type: 'Removed', text: 'Remove Python scripts no longer used (fixes #161).' },
    ],
  },
  {
    version: '1.0.0',
    date: '2025-08-29',
    notes: [
      // Changed
      { type: 'Changed', text: 'Migrated to React for the frontend and Go for the backend — improves responsiveness and enables simple single-binary packaging.' },
      { type: 'Changed', text: 'Revamped frontend, designed from scratch.' },
      { type: 'Changed', text: 'Major UI and UX improvements across the app.' },

      // Added
      { type: 'Added', text: 'Dedicated settings page to manage all settings.' },
      { type: 'Added', text: 'Transcription profiles — save specific configurations for reuse.' },
      { type: 'Added', text: 'Ability to stop running jobs.' },
      { type: 'Added', text: 'Chat with your transcript — supports multiple chat sessions.' },
      { type: 'Added', text: 'Summary profiles — save custom prompts.' },
      { type: 'Added', text: 'Automatic titling of chat sessions.' },
      { type: 'Added', text: 'Highlights and note taking — jump from note to audio and transcript segment.' },
      { type: 'Added', text: 'Playback follow-along — highlights the current word being played.' },
      { type: 'Added', text: 'Seek from text — Cmd + click a word to jump to the corresponding audio timestamp.' },
      { type: 'Added', text: 'Support for controlling and fine-tuning advanced transcription parameters.' },
      { type: 'Added', text: 'REST API endpoints for all app features.' },
      { type: 'Added', text: 'Secure management of API keys.' },
      { type: 'Added', text: 'YouTube video transcription.' },
      { type: 'Added', text: 'Quick transcribe — transcribe without saving audio or transcript to the database.' },
      { type: 'Added', text: 'Batch upload audio files.' },
      { type: 'Added', text: 'Search audio files.' },
      { type: 'Added', text: 'Sort by column.' },
      { type: 'Added', text: 'Admin user credentials management.' },
      { type: 'Added', text: 'Download transcripts in JSON/SRT/TXT formats.' },
    ],
  },
];

const ORDER: ChangeItem['type'][] = ['Added', 'Changed', 'Fixed', 'Deprecated', 'Removed', 'Security'];

function Changelog() {
  const byVersion = useMemo(() => RELEASES, []);
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <div className="min-h-screen bg-white">
      <header className="api-topbar">
        <div className="container-narrow py-3 flex items-center justify-between gap-3">
          <a href="/" className="flex items-center gap-2 select-none">
            <span className="logo-font-poiret text-lg text-gray-900">Scriberr</span>
            <span className="text-gray-300">/</span>
            <span className="text-sm text-gray-600">Changelog</span>
          </a>
          <div className="flex items-center gap-2">
            <button
              className="md:hidden inline-flex items-center justify-center rounded-md border border-gray-200 bg-white px-2.5 py-1.5 text-gray-700 hover:bg-gray-50"
              aria-label="Toggle versions"
              onClick={() => setMobileOpen((v) => !v)}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="size-5">
                <path d="M4 7h16M4 12h16M4 17h16" />
              </svg>
            </button>
            <div className="hidden md:block"><GithubBadge /></div>
          </div>
        </div>
      </header>

      <main className="container-narrow py-10">
        {mobileOpen && (
          <div className="md:hidden mb-4 border border-gray-200 rounded-lg p-3">
            <div className="text-[11px] font-medium text-gray-500 mb-2">Versions</div>
            <ul className="grid grid-cols-3 gap-2 text-sm">
              {byVersion.map((rel) => (
                <li key={`m-${rel.version}`}>
                  <a href={`#v${rel.version}`} className="text-gray-700 hover:text-gray-900" onClick={() => setMobileOpen(false)}>
                    {rel.tag ? `${rel.version} (${rel.tag})` : rel.version}
                  </a>
                </li>
              ))}
            </ul>
          </div>
        )}
        <div className="grid grid-cols-1 md:grid-cols-[260px_minmax(0,1fr)] gap-8">
          <aside className="api-sidebar">
            <div className="sticky top-24 pr-6">
              <div className="text-[11px] font-medium text-gray-500 mb-2">Versions</div>
              <ul className="space-y-2">
                {byVersion.map((rel) => (
                  <li key={rel.version}>
                    <a href={`#v${rel.version}`} className="text-gray-600 hover:text-gray-900 flex items-center gap-2">
                      <span className="font-medium">{rel.version}</span>
                      {rel.tag && <span className="status-pill">{rel.tag}</span>}
                    </a>
                    <div className="text-[11px] text-gray-500 ml-0.5">{formatDate(rel.date)}</div>
                  </li>
                ))}
              </ul>
            </div>
          </aside>

          <section className="space-y-6 changelog-prose">
            {byVersion.map((rel) => (
              <article key={rel.version} id={`v${rel.version}`} className="api-card">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h2>v{rel.version}</h2>
                    <div className="mt-0.5 text-gray-600">Released {formatDate(rel.date)}</div>
                  </div>
                  {rel.tag && <span className="status-pill">{rel.tag}</span>}
                </div>

                <div className="mt-4 space-y-5">
                  {groupNotes(rel.notes).map(([type, items]) => (
                    <section key={type}>
                      <div className="api-section-title mb-2">{type}</div>
                      <ul className="list-disc pl-5 text-gray-700 space-y-1">
                        {items.map((it, i) => (
                          <li key={i}>{it.text}</li>
                        ))}
                      </ul>
                    </section>
                  ))}
                </div>
              </article>
            ))}
          </section>
        </div>
      </main>
    </div>
  );
}

function groupNotes(notes: ChangeItem[]): [ChangeItem['type'], ChangeItem[] ][] {
  const map = new Map<ChangeItem['type'], ChangeItem[]>();
  for (const t of ORDER) map.set(t, []);
  for (const n of notes) map.get(n.type)?.push(n);
  return Array.from(map.entries()).filter(([, arr]) => arr.length > 0);
}

function formatDate(s: string) {
  try {
    const d = new Date(s + 'T00:00:00Z');
    return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
  } catch { return s; }
}

const root = createRoot(document.getElementById('root')!);
root.render(
  <React.StrictMode>
    <Changelog />
  </React.StrictMode>
);
