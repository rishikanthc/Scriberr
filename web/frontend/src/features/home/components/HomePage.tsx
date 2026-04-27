import { useCallback, useMemo, useRef, type ChangeEvent } from "react";
import { Link } from "react-router-dom";
import { ChevronDown, FileAudio, Home, Mic, Search, Settings, StopCircle, Trash2, UploadCloud, Video, Wand2 } from "lucide-react";
import { WandAdvancedIcon } from "@/components/icons/WandAdvancedIcon";
import { UploadProgressShelf } from "@/features/files/components/UploadProgressShelf";
import type { ScriberrFile } from "@/features/files/api/filesApi";
import { useFileEvents } from "@/features/files/hooks/useFileEvents";
import { importAccept, type UploadItem, useFileImport } from "@/features/files/hooks/useFileImport";
import { useFiles } from "@/features/files/hooks/useFiles";
import { useProfiles } from "@/features/settings/hooks/useProfiles";
import type { Transcription, TranscriptionStatus } from "@/features/transcription/api/transcriptionsApi";
import { useTranscriptionListEvents } from "@/features/transcription/hooks/useTranscriptionListEvents";
import { useCreateTranscription, useTranscriptions } from "@/features/transcription/hooks/useTranscriptions";
import { EmptyState } from "@/shared/ui/EmptyState";
import { AppButton, IconButton } from "@/shared/ui/Button";

type RecordingStatus = "ready" | "uploading" | "file-processing" | "queued" | "transcribing" | "transcribed" | "failed" | "canceled";

type Recording = {
  id: string;
  title: string;
  date: string;
  status: RecordingStatus;
  fileStatus: ScriberrFile["status"] | UploadItem["status"];
  transcriptionId?: string;
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

type RecordingCardProps = {
  recording: Recording;
  canTranscribe: boolean;
  isSubmitting: boolean;
  onTranscribe: (recording: Recording) => void;
};

function RecordingCard({ recording, canTranscribe, isSubmitting, onTranscribe }: RecordingCardProps) {
  const isProcessing = recording.status === "file-processing" || recording.status === "uploading" || recording.status === "queued" || recording.status === "transcribing";
  const isFileReady = recording.fileStatus === "ready" || recording.fileStatus === "uploaded";
  const hasActiveTranscription = recording.status === "queued" || recording.status === "transcribing";
  const transcribeDisabled = !canTranscribe || !isFileReady || hasActiveTranscription || isSubmitting;
  const transcribeTitle = !canTranscribe
    ? "Set a default profile in Settings"
    : !isFileReady
      ? "File is not ready yet"
      : hasActiveTranscription
        ? "Transcription already running"
        : isSubmitting
          ? "Submitting transcription"
          : "Transcribe with default profile";

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
          <span className="scr-recording-action-tip" title={transcribeTitle}>
            <button
              className="scr-recording-action"
              type="button"
              aria-label={transcribeTitle}
              disabled={transcribeDisabled}
              onClick={(event) => {
                event.stopPropagation();
                onTranscribe(recording);
              }}
            >
              <Wand2 size={16} aria-hidden="true" />
            </button>
          </span>
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
        <div className="scr-recording-status" data-status={recording.status}>
          {statusText(recording)}
        </div>
      </div>
    </article>
  );
}

export function HomePage() {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const filesQuery = useFiles();
  const profilesQuery = useProfiles();
  const transcriptionsQuery = useTranscriptions();
  const createTranscriptionMutation = useCreateTranscription();
  const { uploadItems, importFiles, dismissItem, handleFileEvent } = useFileImport();
  useFileEvents(handleFileEvent);
  useTranscriptionListEvents();

  const defaultProfile = useMemo(() => {
    return (profilesQuery.data || []).find((profile) => profile.is_default);
  }, [profilesQuery.data]);

  const latestTranscriptionByFileId = useMemo(() => {
    const byFileId = new Map<string, Transcription>();
    for (const transcription of transcriptionsQuery.data?.items || []) {
      const current = byFileId.get(transcription.file_id);
      if (!current || new Date(transcription.updated_at).getTime() > new Date(current.updated_at).getTime()) {
        byFileId.set(transcription.file_id, transcription);
      }
    }
    return byFileId;
  }, [transcriptionsQuery.data?.items]);

  const recordings = useMemo(() => {
    const optimistic = uploadItems
      .filter((item) => !item.fileId)
      .map(uploadItemToRecording);
    const serverFiles = (filesQuery.data?.items || []).map((file) => fileToRecording(file, latestTranscriptionByFileId.get(file.id)));
    return [...optimistic, ...serverFiles];
  }, [filesQuery.data?.items, latestTranscriptionByFileId, uploadItems]);

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

  const handleTranscribe = useCallback((recording: Recording) => {
    if (!defaultProfile || recording.fileStatus !== "ready" && recording.fileStatus !== "uploaded") return;
    createTranscriptionMutation.mutate({
      fileId: recording.id,
      profileId: defaultProfile.id,
      title: recording.title,
    });
  }, [createTranscriptionMutation, defaultProfile]);

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
                  <RecordingCard
                    key={recording.id}
                    recording={recording}
                    canTranscribe={Boolean(defaultProfile)}
                    isSubmitting={createTranscriptionMutation.isPending && createTranscriptionMutation.variables?.fileId === recording.id}
                    onTranscribe={handleTranscribe}
                  />
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

function fileToRecording(file: ScriberrFile, transcription?: Transcription): Recording {
  return {
    id: file.id,
    title: file.title || "Untitled recording",
    date: formatRecordingDate(file.created_at),
    status: normalizeRecordingStatus(file.status, transcription?.status),
    fileStatus: file.status,
    transcriptionId: transcription?.id,
    progress: transcription?.progress,
  };
}

function uploadItemToRecording(item: UploadItem): Recording {
  return {
    id: item.id,
    title: item.fileName.replace(/\.[^/.]+$/, ""),
    date: item.status === "uploading" ? `Uploading ${item.progress}%` : itemLabel(item.status),
    status: item.status === "processing" ? "file-processing" : item.status,
    fileStatus: item.status,
    progress: item.progress,
  };
}

function normalizeRecordingStatus(fileStatus: ScriberrFile["status"], transcriptionStatus?: TranscriptionStatus): RecordingStatus {
  if (transcriptionStatus) return normalizeTranscriptionStatus(transcriptionStatus);
  if (fileStatus === "ready" || fileStatus === "uploaded") return "ready";
  if (fileStatus === "failed") return "failed";
  return "file-processing";
}

function normalizeTranscriptionStatus(status: TranscriptionStatus): RecordingStatus {
  switch (status) {
    case "queued":
      return "queued";
    case "processing":
      return "transcribing";
    case "completed":
      return "transcribed";
    case "failed":
      return "failed";
    case "canceled":
      return "canceled";
  }
}

function statusText(recording: Recording) {
  switch (recording.status) {
    case "ready":
      return "Ready";
    case "transcribed":
      return "Done";
    case "failed":
      return "Failed";
    case "uploading":
      return recording.progress ? formatProgress(recording.progress) : "Uploading";
    case "file-processing":
      return "Processing";
    case "queued":
      return "Queued";
    case "transcribing":
      return (recording.progress ?? 0) > 0 ? formatProgress(recording.progress ?? 0) : "Transcribing";
    case "canceled":
      return "Canceled";
  }
}

function formatProgress(progress: number) {
  const percent = progress <= 1 ? progress * 100 : progress;
  if (percent >= 99.5) return "100%";
  if (percent < 1) return "<1%";
  return `${Math.round(percent)}%`;
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
