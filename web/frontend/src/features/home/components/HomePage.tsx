import { useCallback, useMemo, useRef, useState, type ChangeEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ChevronDown, FileAudio, Home, Mic, Search, Settings, StopCircle, Trash2, UploadCloud, Video, Wand2, Youtube } from "lucide-react";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { WandAdvancedIcon } from "@/components/icons/WandAdvancedIcon";
import { UploadProgressShelf } from "@/features/files/components/UploadProgressShelf";
import { YouTubeImportDialog } from "@/features/files/components/YouTubeImportDialog";
import type { ScriberrFile } from "@/features/files/api/filesApi";
import { useFileEvents } from "@/features/files/hooks/useFileEvents";
import { importAccept, type UploadItem, useFileImport } from "@/features/files/hooks/useFileImport";
import { useFiles } from "@/features/files/hooks/useFiles";
import { useProfiles } from "@/features/settings/hooks/useProfiles";
import type { TranscriptionProfile } from "@/features/settings/api/profilesApi";
import type { Transcription, TranscriptionStatus } from "@/features/transcription/api/transcriptionsApi";
import { useTranscriptionListEvents } from "@/features/transcription/hooks/useTranscriptionListEvents";
import { preferVisibleTranscription, useCreateTranscription, useStopTranscription, useTranscriptions } from "@/features/transcription/hooks/useTranscriptions";
import { EmptyState } from "@/shared/ui/EmptyState";
import { AppButton, IconButton } from "@/shared/ui/Button";

type RecordingStatus = "ready" | "uploading" | "file-processing" | "queued" | "transcribing" | "transcribed" | "failed" | "stopped" | "canceled";

type Recording = {
  id: string;
  title: string;
  description: string;
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
  onUploadFilesClick: () => void;
  onYouTubeImportClick: () => void;
};

function TopBar({ onUploadFilesClick, onYouTubeImportClick }: TopBarProps) {
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
        <DropdownMenu>
          <DropdownMenuTrigger className="scr-button scr-button-secondary scr-topbar-button scr-import-trigger">
            <UploadCloud size={14} aria-hidden="true" />
            Import
            <ChevronDown size={13} aria-hidden="true" />
          </DropdownMenuTrigger>
          <DropdownMenuContent className="scr-import-menu" align="end">
            <DropdownMenuItem className="scr-import-menu-item" onSelect={onUploadFilesClick}>
              <UploadCloud size={15} aria-hidden="true" />
              Upload files
            </DropdownMenuItem>
            <DropdownMenuItem className="scr-import-menu-item" onSelect={onYouTubeImportClick}>
              <Youtube size={15} aria-hidden="true" />
              Import from YouTube
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
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
  profiles: TranscriptionProfile[];
  profilesLoading: boolean;
  isSubmitting: boolean;
  isStopping: boolean;
  onTranscribe: (recording: Recording) => void;
  onTranscribeWithProfile: (recording: Recording, profile: TranscriptionProfile) => void;
  onStop: (recording: Recording) => void;
  onOpen: (recording: Recording) => void;
};

function RecordingCard({
  recording,
  canTranscribe,
  profiles,
  profilesLoading,
  isSubmitting,
  isStopping,
  onTranscribe,
  onTranscribeWithProfile,
  onStop,
  onOpen,
}: RecordingCardProps) {
  const [profilePickerOpen, setProfilePickerOpen] = useState(false);
  const isProcessing = recording.status === "file-processing" || recording.status === "uploading" || recording.status === "queued" || recording.status === "transcribing";
  const isFileReady = recording.fileStatus === "ready" || recording.fileStatus === "uploaded";
  const hasActiveTranscription = recording.status === "queued" || recording.status === "transcribing";
  const transcribeDisabled = !canTranscribe || !isFileReady || hasActiveTranscription || isSubmitting;
  const profileTranscribeDisabled = (!profilesLoading && profiles.length === 0) || !isFileReady || hasActiveTranscription || isSubmitting;
  const transcribeTitle = !canTranscribe
    ? "Set a default profile in Settings"
    : !isFileReady
      ? "File is not ready yet"
      : hasActiveTranscription
        ? "Transcription already running"
        : isSubmitting
          ? "Submitting transcription"
          : "Transcribe with default profile";
  const profileTranscribeTitle = profilesLoading
    ? "Loading ASR profiles"
    : profiles.length === 0
    ? "Create an ASR profile in Settings"
    : !isFileReady
      ? "File is not ready yet"
      : hasActiveTranscription
        ? "Transcription already running"
        : isSubmitting
          ? "Submitting transcription"
          : "Choose ASR profile";

  const openTitle = recording.id.startsWith("file_") ? `Open ${recording.title}` : undefined;

  return (
    <article
      className="scr-recording-card"
      tabIndex={recording.id.startsWith("file_") ? 0 : -1}
      role={recording.id.startsWith("file_") ? "button" : undefined}
      aria-label={openTitle}
      onClick={() => onOpen(recording)}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onOpen(recording);
        }
      }}
    >
      <div className="scr-recording-icon">
        <FileAudio size={24} aria-hidden="true" />
      </div>
      <div>
        <h2 className="scr-recording-title">{recording.title}</h2>
        {recording.description ? <p className="scr-recording-description">{recording.description}</p> : null}
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
          <Popover open={profilePickerOpen} onOpenChange={setProfilePickerOpen}>
            <PopoverTrigger asChild>
              <button
                className="scr-recording-action"
                type="button"
                aria-label={profileTranscribeTitle}
                title={profileTranscribeTitle}
                disabled={profileTranscribeDisabled}
                onClick={(event) => event.stopPropagation()}
              >
                <WandAdvancedIcon className="scr-recording-action-icon" strokeWidth={2} />
              </button>
            </PopoverTrigger>
            <PopoverContent
              className="scr-asr-profile-popover"
              align="end"
              side="bottom"
              onClick={(event) => event.stopPropagation()}
            >
              <Command>
                <CommandInput placeholder="Search ASR profiles" />
                <CommandList>
                  <CommandEmpty>No ASR profiles found.</CommandEmpty>
                  <CommandGroup heading="ASR profiles">
                    {profilesLoading ? (
                      <p className="scr-asr-profile-status">Loading profiles.</p>
                    ) : profiles.map((profile) => (
                      <CommandItem
                        key={profile.id}
                        value={`${profile.name} ${profile.description} ${profile.options.model} ${profile.is_default ? "default" : ""}`}
                        onSelect={() => {
                          onTranscribeWithProfile(recording, profile);
                          setProfilePickerOpen(false);
                        }}
                      >
                        <span className="scr-asr-profile-option">
                          <span>
                            {profile.name}
                            {profile.is_default ? <small>Default</small> : null}
                          </span>
                          <small>{formatProfileDescription(profile)}</small>
                        </span>
                      </CommandItem>
                    ))}
                  </CommandGroup>
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>
          <button
            className="scr-recording-action scr-recording-action-danger"
            type="button"
            aria-label={isProcessing ? "Stop transcription" : "Delete recording"}
            title={isProcessing ? "Stop transcription" : "Delete"}
            disabled={isProcessing && (!recording.transcriptionId || isStopping)}
            onClick={(event) => {
              event.stopPropagation();
              if (isProcessing) onStop(recording);
            }}
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
  const navigate = useNavigate();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [youtubeDialogOpen, setYoutubeDialogOpen] = useState(false);
  const filesQuery = useFiles();
  const profilesQuery = useProfiles();
  const transcriptionsQuery = useTranscriptions();
  const createTranscriptionMutation = useCreateTranscription();
  const stopTranscriptionMutation = useStopTranscription();
  const { uploadItems, importFiles, importFromYouTube, dismissItem, handleFileEvent } = useFileImport();
  useFileEvents(handleFileEvent);
  useTranscriptionListEvents();

  const defaultProfile = useMemo(() => {
    return (profilesQuery.data || []).find((profile) => profile.is_default);
  }, [profilesQuery.data]);
  const profiles = profilesQuery.data || [];

  const latestTranscriptionByFileId = useMemo(() => {
    const byFileId = new Map<string, Transcription>();
    for (const transcription of transcriptionsQuery.data?.items || []) {
      const current = byFileId.get(transcription.file_id);
      byFileId.set(transcription.file_id, preferVisibleTranscription(transcription, current));
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

  const handleUploadFilesClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleYouTubeImportClick = useCallback(() => {
    setYoutubeDialogOpen(true);
  }, []);

  const handleYouTubeImport = useCallback(async (url: string) => {
    await importFromYouTube(url);
  }, [importFromYouTube]);

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

  const handleTranscribeWithProfile = useCallback((recording: Recording, profile: TranscriptionProfile) => {
    if (recording.fileStatus !== "ready" && recording.fileStatus !== "uploaded") return;
    createTranscriptionMutation.mutate({
      fileId: recording.id,
      profileId: profile.id,
      title: recording.title,
    });
  }, [createTranscriptionMutation]);

  const handleStopTranscription = useCallback((recording: Recording) => {
    if (!recording.transcriptionId) return;
    stopTranscriptionMutation.mutate(recording.transcriptionId);
  }, [stopTranscriptionMutation]);

  const handleOpenRecording = useCallback((recording: Recording) => {
    if (recording.id.startsWith("file_")) {
      navigate(`/audio/${recording.id}`);
    }
  }, [navigate]);

  return (
    <div className="scr-app">
      <div className="scr-shell">
        <Sidebar />
        <main className="scr-main">
          <TopBar onUploadFilesClick={handleUploadFilesClick} onYouTubeImportClick={handleYouTubeImportClick} />
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
                    profiles={profiles}
                    profilesLoading={profilesQuery.isLoading}
                    isSubmitting={createTranscriptionMutation.isPending && createTranscriptionMutation.variables?.fileId === recording.id}
                    isStopping={stopTranscriptionMutation.isPending && stopTranscriptionMutation.variables === recording.transcriptionId}
                    onTranscribe={handleTranscribe}
                    onTranscribeWithProfile={handleTranscribeWithProfile}
                    onStop={handleStopTranscription}
                    onOpen={handleOpenRecording}
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
      <YouTubeImportDialog
        open={youtubeDialogOpen}
        importing={uploadItems.some((item) => item.source === "youtube" && item.status === "processing" && !item.fileId)}
        onOpenChange={setYoutubeDialogOpen}
        onSubmit={handleYouTubeImport}
      />
    </div>
  );
}

function fileToRecording(file: ScriberrFile, transcription?: Transcription): Recording {
  return {
    id: file.id,
    title: file.title || "Untitled recording",
    description: file.description,
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
    description: "",
    date: item.status === "uploading" ? `Uploading ${item.progress}%` : itemLabel(item),
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
    case "stopped":
      return "stopped";
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
    case "stopped":
      return "Stopped";
    case "uploading":
      return recording.progress ? formatProgress(recording.progress) : "Uploading";
    case "file-processing":
      return "Processing";
    case "queued":
      return "Queued";
    case "transcribing":
      return (recording.progress ?? 0) > 0 ? formatProgress(recording.progress ?? 0) : "Transcribing";
    case "canceled":
      return "Stopped";
  }
}

function formatProfileDescription(profile: TranscriptionProfile) {
  const parts = [profile.options.model];
  if (profile.options.language) parts.push(profile.options.language.toUpperCase());
  if (profile.options.diarize) parts.push("Diarization");
  return parts.join(" · ");
}

function formatProgress(progress: number) {
  const percent = progress <= 1 ? progress * 100 : progress;
  if (percent >= 99.5) return "100%";
  if (percent < 1) return "<1%";
  return `${Math.round(percent)}%`;
}

function itemLabel(item: UploadItem) {
  switch (item.status) {
    case "uploading":
      return "Uploading";
    case "processing":
      return item.source === "youtube" ? "Importing from YouTube" : "Extracting audio";
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
