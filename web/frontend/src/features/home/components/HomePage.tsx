import { Check, Clock3, Home, Mic, Search, UploadCloud, Video } from "lucide-react";
import { EmptyState } from "@/shared/ui/EmptyState";
import { AppButton, IconButton } from "@/shared/ui/Button";

type RecordingStatus = "completed" | "queued";

type Recording = {
  id: string;
  title: string;
  date: string;
  status: RecordingStatus;
};

const mockRecordings: Recording[] = [
  { id: "mock-1", title: "Stanford CS336 - Lecture 2", date: "Dec 11, 2025", status: "completed" },
  { id: "mock-2", title: "LTT - WAN Show", date: "Dec 10, 2025", status: "queued" },
  { id: "mock-3", title: "Stanford CS336 - Lecture 1", date: "Dec 7, 2025", status: "queued" },
  { id: "mock-4", title: "Plaud Audio", date: "Dec 7, 2025", status: "completed" },
  { id: "mock-5", title: "20min Recording", date: "Dec 7, 2025", status: "queued" },
  { id: "mock-6", title: "recording_2025-12-06T04-45-17.wav", date: "Dec 5, 2025", status: "queued" },
  { id: "mock-7", title: "LTT", date: "Dec 5, 2025", status: "completed" },
  { id: "mock-8", title: "40min", date: "Dec 5, 2025", status: "completed" },
];

function Sidebar() {
  return (
    <aside className="scr-sidebar" aria-label="Primary navigation">
      <div className="scr-logo-row">
        <img className="scr-logo-text" src="/logo-text.svg" alt="Scriberr" />
      </div>
      <nav className="scr-nav">
        <a className="scr-nav-item" data-active="true" href="/">
          <Home size={18} aria-hidden="true" />
          <span className="scr-nav-label">Home</span>
        </a>
      </nav>
    </aside>
  );
}

function TopBar() {
  return (
    <div className="scr-topbar">
      <div className="scr-search-shell" aria-hidden="true">
        <Search size={15} />
        <span>Ask or search</span>
        <kbd className="scr-kbd">⌘K</kbd>
      </div>

      <div className="scr-topbar-actions">
        <IconButton label="Video">
          <Video size={16} aria-hidden="true" />
        </IconButton>
        <AppButton type="button" variant="secondary">
          <UploadCloud size={16} aria-hidden="true" />
          Import
        </AppButton>
        <AppButton type="button">
          <Mic size={16} aria-hidden="true" />
          Record
        </AppButton>
      </div>
    </div>
  );
}

function RecordingCard({ recording }: { recording: Recording }) {
  return (
    <article className="scr-recording-card">
      <div className="scr-recording-icon">
        <img src="/logo.svg" alt="" width={30} height={30} aria-hidden="true" />
      </div>
      <div>
        <h2 className="scr-recording-title">{recording.title}</h2>
        <p className="scr-recording-date">{recording.date}</p>
      </div>
      <div className="scr-recording-status" data-status={recording.status} aria-label={recording.status}>
        {recording.status === "completed" ? <Check size={22} aria-hidden="true" /> : <Clock3 size={19} aria-hidden="true" />}
      </div>
    </article>
  );
}

export function HomePage() {
  const recordings = mockRecordings;

  return (
    <div className="scr-app">
      <div className="scr-shell">
        <Sidebar />
        <main className="scr-main">
          <TopBar />
          <div className="scr-content">
            <header className="scr-page-head">
              <h1 className="scr-page-title">Home</h1>
              <p className="scr-page-meta">{recordings.length} recordings</p>
            </header>

            {recordings.length > 0 ? (
              <section className="scr-recording-list" aria-label="Recordings">
                {recordings.map((recording) => (
                  <RecordingCard key={recording.id} recording={recording} />
                ))}
              </section>
            ) : (
              <EmptyState title="No recordings yet" description="Uploaded audio files will appear here." />
            )}
          </div>
        </main>
      </div>
    </div>
  );
}
