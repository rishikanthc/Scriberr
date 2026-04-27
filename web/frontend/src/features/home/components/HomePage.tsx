import { useCallback, useMemo, useRef, type ChangeEvent } from "react";
import { Link } from "react-router-dom";
import { Check, ChevronDown, Clock3, FileAudio, Home, Loader2, Mic, Search, Settings, StopCircle, Trash2, UploadCloud, Video, Wand2, XCircle } from "lucide-react";
import { WandAdvancedIcon } from "@/components/icons/WandAdvancedIcon";
import { UploadProgressShelf } from "@/features/files/components/UploadProgressShelf";
import type { ScriberrFile } from "@/features/files/api/filesApi";
import { useFileEvents } from "@/features/files/hooks/useFileEvents";
import { importAccept, type UploadItem, useFileImport } from "@/features/files/hooks/useFileImport";
import { useFiles } from "@/features/files/hooks/useFiles";
import { EmptyState } from "@/shared/ui/EmptyState";
import { AppButton, IconButton } from "@/shared/ui/Button";

type RecordingStatus = "ready" | "processing" | "uploading" | "failed";

type Recording = {
  id: string;
  title: string;
  date: string;
  status: RecordingStatus;
  progress?: number;
};

type SidebarProps = {
  activeItem?: "home" | "settings";
};

export function Sidebar({ activeItem = "home" }: SidebarProps) {
  return (
    <aside className="scr-sidebar" aria-label="Primary navigation">
      <div className="scr-logo-row">
        <img className="scr-logo-mark" src="/logo.svg" alt="" aria-hidden="true" />
        <img className="scr-logo-text" src="/logo-text.svg" alt="Scriberr" />
      </div>
      <nav className="scr-nav">
        <Link className="scr-nav-item" data-active={activeItem === "home"} to="/">
          <Home size={18} aria-hidden="true" />
          <span className="scr-nav-label">Home</span>
        </Link>
        <Link className="scr-nav-item" data-active={activeItem === "settings"} to="/settings">
          <Settings size={18} aria-hidden="true" />
          <span className="scr-nav-label">Settings</span>
        </Link>
      </nav>
    </aside>
  );
}

type TopBarProps = {
  onImportClick: () => void;
};

function TopBar({ onImportClick }: TopBarProps) {
  return (
    <div className="scr-topbar">
      <div className="scr-search-shell" aria-hidden="true">
        <Search size={15} />
        <span>Ask or search</span>
        <kbd className="scr-kbd">⌘K</kbd>
      </div>

      <div className="scr-topbar-actions">
        <IconButton label="Video">
          <Video size={14} aria-hidden="true" />
        </IconButton>
        <AppButton type="button" variant="secondary" className="scr-topbar-button" onClick={onImportClick}>
          <UploadCloud size={14} aria-hidden="true" />
          Import
        </AppButton>
        <AppButton type="button" className="scr-topbar-button">
          <Mic size={14} aria-hidden="true" />
          Record
        </AppButton>
      </div>
    </div>
  );
}

function RecordingCard({ recording }: { recording: Recording }) {
  const isProcessing = recording.status === "processing" || recording.status === "uploading";

  return (
    <article className="scr-recording-card" tabIndex={0}>
      <div className="scr-recording-icon">
        <FileAudio size={24} aria-hidden="true" />
      </div>
      <div>
        <h2 className="scr-recording-title">{recording.title}</h2>
        <p className="scr-recording-date">{recording.date}</p>
      </div>
      <div className="scr-recording-meta-actions">
        <div className="scr-recording-actions" aria-label={`${recording.title} actions`}>
          <button className="scr-recording-action" type="button" aria-label="Transcribe" title="Transcribe">
            <Wand2 size={16} aria-hidden="true" />
          </button>
          <button className="scr-recording-action" type="button" aria-label="Transcribe advanced" title="Transcribe advanced">
            <WandAdvancedIcon className="scr-recording-action-icon" strokeWidth={2} />
          </button>
          <button
            className="scr-recording-action scr-recording-action-danger"
            type="button"
            aria-label={isProcessing ? "Stop transcription" : "Delete recording"}
            title={isProcessing ? "Stop transcription" : "Delete"}
          >
            {isProcessing ? <StopCircle size={16} aria-hidden="true" /> : <Trash2 size={16} aria-hidden="true" />}
          </button>
        </div>
        <div className="scr-recording-status" data-status={recording.status} aria-label={recording.status}>
          {statusIcon(recording)}
        </div>
      </div>
    </article>
  );
}

export function HomePage() {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const filesQuery = useFiles();
  const { uploadItems, importFiles, dismissItem, handleFileEvent } = useFileImport();
  useFileEvents(handleFileEvent);

  const recordings = useMemo(() => {
    const optimistic = uploadItems
      .filter((item) => !item.fileId)
      .map(uploadItemToRecording);
    const serverFiles = (filesQuery.data?.items || []).map(fileToRecording);
    return [...optimistic, ...serverFiles];
  }, [filesQuery.data?.items, uploadItems]);

  const handleImportClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleImportChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const selected = event.currentTarget.files;
    if (selected?.length) {
      void importFiles(selected);
    }
    event.currentTarget.value = "";
  }, [importFiles]);

  return (
    <div className="scr-app">
      <div className="scr-shell">
        <Sidebar />
        <main className="scr-main">
          <TopBar onImportClick={handleImportClick} />
          <input
            ref={fileInputRef}
            className="scr-visually-hidden"
            type="file"
            hidden
            aria-hidden="true"
            multiple
            accept={importAccept}
            onChange={handleImportChange}
          />
          <div className="scr-content">
            <div className="scr-feed-toolbar" aria-label="Recording view controls">
              <button className="scr-feed-select" type="button">
                Yesterday, Apr 25
                <ChevronDown size={13} aria-hidden="true" />
              </button>
            </div>

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
      <UploadProgressShelf items={uploadItems} onDismiss={dismissItem} />
    </div>
  );
}

function fileToRecording(file: ScriberrFile): Recording {
  return {
    id: file.id,
    title: file.title || "Untitled recording",
    date: formatRecordingDate(file.created_at),
    status: normalizeFileStatus(file.status),
  };
}

function uploadItemToRecording(item: UploadItem): Recording {
  return {
    id: item.id,
    title: item.fileName.replace(/\.[^/.]+$/, ""),
    date: item.status === "uploading" ? `Uploading ${item.progress}%` : itemLabel(item.status),
    status: item.status,
    progress: item.progress,
  };
}

function normalizeFileStatus(status: ScriberrFile["status"]): RecordingStatus {
  if (status === "ready" || status === "uploaded") return "ready";
  if (status === "failed") return "failed";
  return "processing";
}

function statusIcon(recording: Recording) {
  switch (recording.status) {
    case "ready":
      return <Check size={18} aria-hidden="true" />;
    case "failed":
      return <XCircle size={17} aria-hidden="true" />;
    case "uploading":
      return <Loader2 className="scr-spin" size={16} aria-hidden="true" />;
    case "processing":
      return <Clock3 size={16} aria-hidden="true" />;
  }
}

function itemLabel(status: UploadItem["status"]) {
  switch (status) {
    case "uploading":
      return "Uploading";
    case "processing":
      return "Extracting audio";
    case "ready":
      return "Ready";
    case "failed":
      return "Failed";
  }
}

function formatRecordingDate(value: string) {
  return new Date(value).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}
