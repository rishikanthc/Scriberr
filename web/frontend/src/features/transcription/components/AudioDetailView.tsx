import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties, type ChangeEvent } from "react";
import { Link, useParams } from "react-router-dom";
import { AlignJustify, CalendarDays, Clock3, MoreHorizontal, Pause, Pencil, Play } from "lucide-react";
import { Sidebar } from "@/features/home/components/HomePage";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useToast } from "@/components/ui/toast";
import { useFile, useUpdateFile } from "@/features/files/hooks/useFiles";
import type { FileStatus } from "@/features/files/api/filesApi";
import type { TranscriptionSummary } from "@/features/transcription/api/summariesApi";
import type { TranscriptSegment, TranscriptWord, Transcription, TranscriptionTranscript } from "@/features/transcription/api/transcriptionsApi";
import { useTranscriptionDetailEvents } from "@/features/transcription/hooks/useTranscriptionDetailEvents";
import { useTranscriptionListEvents } from "@/features/transcription/hooks/useTranscriptionListEvents";
import { useTranscriptionSummary } from "@/features/transcription/hooks/useTranscriptionSummaries";
import { useTranscriptionTranscript, useTranscriptions } from "@/features/transcription/hooks/useTranscriptions";

type DetailTab = "summary" | "transcript";

export function AudioDetailView() {
  const { audioId = "" } = useParams<{ audioId: string }>();
  const [activeTab, setActiveTab] = useState<DetailTab>("summary");
  const [audioDuration, setAudioDuration] = useState<number | null>(null);
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [draftTitle, setDraftTitle] = useState("");
  const fileQuery = useFile(audioId);
  const updateFileMutation = useUpdateFile(audioId);
  const transcriptionsQuery = useTranscriptions();
  const { toast } = useToast();
  const warnedSummaryIds = useRef<Set<string>>(new Set());

  const file = fileQuery.data;
  const title = file?.title?.trim() || "Untitled recording";
  const visibleDuration = file?.duration_seconds ?? audioDuration;
  const latestTranscription = useMemo(() => {
    if (!file) return undefined;
    return latestTranscriptionForFile(transcriptionsQuery.data?.items || [], file.id);
  }, [file, transcriptionsQuery.data?.items]);
  const transcriptQuery = useTranscriptionTranscript(latestTranscription?.id, latestTranscription?.status === "completed");
  const handleSummaryTruncated = useCallback((summaryId: string) => {
    if (warnedSummaryIds.current.has(summaryId)) return;
    warnedSummaryIds.current.add(summaryId);
    toast({
      title: "Transcript truncated for summary",
      description: "The transcript exceeds the small model context window, so summarization will use the first fitting portion.",
    });
  }, [toast]);

  useTranscriptionListEvents();
  useTranscriptionDetailEvents(latestTranscription?.id, { onSummaryTruncated: handleSummaryTruncated });

  const meta = useMemo(() => {
    return {
      createdAt: file?.created_at ? formatCreatedDate(file.created_at) : "",
      duration: formatDurationLabel(visibleDuration),
    };
  }, [file?.created_at, visibleDuration]);

  useEffect(() => {
    if (!isEditingTitle) {
      setDraftTitle(title);
    }
  }, [isEditingTitle, title]);

  const saveTitle = () => {
    if (!file) return;
    const nextTitle = draftTitle.trim();
    if (!nextTitle || nextTitle === title) {
      setDraftTitle(title);
      setIsEditingTitle(false);
      return;
    }
    updateFileMutation.mutate({ title: nextTitle });
    setIsEditingTitle(false);
  };

  if (fileQuery.isLoading) {
    return (
      <div className="scr-app">
        <div className="scr-shell">
          <Sidebar />
          <main className="scr-audio-detail-main">
            <div className="scr-audio-loading" aria-label="Loading recording">
              <span />
            </div>
          </main>
        </div>
      </div>
    );
  }

  if (fileQuery.error || !file) {
    return (
      <div className="scr-app">
        <div className="scr-shell">
          <Sidebar />
          <main className="scr-audio-detail-main">
            <div className="scr-audio-error">
              <h1>Recording unavailable</h1>
              <p>Could not load this recording.</p>
              <Link to="/">Back to Home</Link>
            </div>
          </main>
        </div>
      </div>
    );
  }

  return (
    <div className="scr-app">
      <div className="scr-shell">
        <Sidebar />
        <main className="scr-audio-detail-main">
          <article className="scr-audio-detail">
            <header className="scr-audio-hero">
              <div className="scr-audio-title-row">
                {isEditingTitle ? (
                  <input
                    className="scr-audio-title scr-audio-title-input"
                    value={draftTitle}
                    aria-label="Recording title"
                    autoFocus
                    onBlur={saveTitle}
                    onChange={(event) => setDraftTitle(event.currentTarget.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter") {
                        event.currentTarget.blur();
                      }
                      if (event.key === "Escape") {
                        setDraftTitle(title);
                        setIsEditingTitle(false);
                      }
                    }}
                  />
                ) : (
                  <button
                    className="scr-audio-title scr-audio-title-button"
                    type="button"
                    title="Edit title"
                    onClick={() => {
                      setDraftTitle(title);
                      setIsEditingTitle(true);
                    }}
                  >
                    {title}
                  </button>
                )}
                <button className="scr-audio-icon-action" type="button" aria-label="More actions" title="More actions">
                  <MoreHorizontal size={18} aria-hidden="true" />
                </button>
              </div>
              <div className="scr-audio-meta" aria-label="Recording metadata">
                <span>
                  <CalendarDays size={14} aria-hidden="true" />
                  {meta.createdAt}
                </span>
                <span>
                  <Clock3 size={14} aria-hidden="true" />
                  {meta.duration}
                </span>
              </div>
            </header>

            <div className="scr-audio-tabbar">
              <button type="button" data-active={activeTab === "summary"} onClick={() => setActiveTab("summary")}>
                Summary
              </button>
              <button type="button" data-active={activeTab === "transcript"} onClick={() => setActiveTab("transcript")}>
                Transcript
              </button>
              <button className="scr-audio-edit-icon" type="button" aria-label="Edit transcript" title="Edit transcript" disabled>
                <Pencil size={14} aria-hidden="true" />
              </button>
            </div>

            {activeTab === "summary" ? (
              <SummaryPanel transcription={latestTranscription} />
            ) : (
              <TranscriptPanel
                fileStatus={file.status}
                transcription={latestTranscription}
                transcript={transcriptQuery.data}
                isLoading={transcriptionsQuery.isLoading || transcriptQuery.isLoading}
                isError={transcriptionsQuery.isError || transcriptQuery.isError}
              />
            )}
          </article>
          <StreamingAudioPlayer
            fileId={file.id}
            durationSeconds={file.duration_seconds}
            title={title}
            onDurationChange={setAudioDuration}
          />
        </main>
      </div>
    </div>
  );
}

function SummaryPanel({ transcription }: { transcription?: Transcription }) {
  const summaryQuery = useTranscriptionSummary(transcription?.id, Boolean(transcription));
  const summary = summaryQuery.data;

  if (!transcription) {
    return (
      <section className="scr-audio-summary" aria-label="Summary">
        <p>No transcription has been queued for this recording yet.</p>
      </section>
    );
  }

  if (summaryQuery.isLoading) {
    return (
      <section className="scr-audio-summary" aria-label="Summary">
        <p>Checking summary status.</p>
      </section>
    );
  }

  if (summaryQuery.isError) {
    return (
      <section className="scr-audio-summary" aria-label="Summary">
        <p>Summary could not be loaded.</p>
      </section>
    );
  }

  if (summary?.status === "completed" && summary.content.trim()) {
    return <SummaryOverview summary={summary} />;
  }

  if (summary?.status === "pending" || summary?.status === "processing") {
    return (
      <section className="scr-audio-summary" aria-label="Summary">
        <p>Summary is being generated.</p>
      </section>
    );
  }

  if (summary?.status === "failed") {
    return (
      <section className="scr-audio-summary" aria-label="Summary">
        <p>{summary.error || "Summary generation failed."}</p>
      </section>
    );
  }

  if (transcription.status === "queued" || transcription.status === "processing") {
    return (
      <section className="scr-audio-summary" aria-label="Summary">
        <p>Summary will be generated when the transcript is ready.</p>
      </section>
    );
  }

  if (transcription.status !== "completed") {
    return (
      <section className="scr-audio-summary" aria-label="Summary">
        <p>Summary is not available for this transcription.</p>
      </section>
    );
  }

  return (
    <section className="scr-audio-summary" aria-label="Summary">
      <p>Summary is not available yet.</p>
    </section>
  );
}

function SummaryOverview({ summary }: { summary: TranscriptionSummary }) {
  return (
    <section className="scr-audio-summary" aria-label="Summary">
      <div className="scr-summary-heading">
        <AlignJustify size={18} aria-hidden="true" />
        <h2>Overview</h2>
      </div>
      <p className="scr-summary-body">{summary.content}</p>
    </section>
  );
}

type TranscriptPanelProps = {
  fileStatus: FileStatus;
  transcription?: Transcription;
  transcript?: TranscriptionTranscript;
  isLoading: boolean;
  isError: boolean;
};

function TranscriptPanel({ fileStatus, transcription, transcript, isLoading, isError }: TranscriptPanelProps) {
  if (isLoading) {
    return <TranscriptPlaceholder title="Loading transcript" description="Reading the latest transcription state." />;
  }

  if (isError) {
    return <TranscriptPlaceholder title="Transcript unavailable" description="Could not load transcript data." />;
  }

  if (!transcription) {
    if (fileStatus === "processing") {
      return <TranscriptPlaceholder title="Audio is processing" description="Transcript actions will be available when the imported audio is ready." />;
    }
    if (fileStatus === "failed") {
      return <TranscriptPlaceholder title="Audio import failed" description="This recording cannot be transcribed until the file import succeeds." />;
    }
    return <TranscriptPlaceholder title="Not transcribed yet" description="Queue a transcription from Home to generate transcript segments for this recording." />;
  }

  if (transcription.status === "queued" || transcription.status === "processing") {
    return (
      <TranscriptPlaceholder
        title={transcription.status === "queued" ? "Transcription queued" : "Transcription processing"}
        description={formatTranscriptionProgress(transcription)}
      />
    );
  }

  if (transcription.status === "failed") {
    return <TranscriptPlaceholder title="Transcription failed" description="Start another transcription from Home when you are ready to retry." />;
  }

  if (transcription.status === "canceled") {
    return <TranscriptPlaceholder title="Transcription canceled" description="Start another transcription from Home to generate transcript text." />;
  }

  const segments = normalizeTranscriptSegments(transcript);
  if (!segments.length) {
    return <TranscriptPlaceholder title="Transcript is empty" description="The completed transcription did not return any text." />;
  }

  return (
    <section className="scr-transcript" aria-label="Transcript">
      {segments.map((segment) => (
        <article className="scr-transcript-segment" key={segment.id || `${segment.start}-${segment.end}`}>
          {segment.speaker ? <span className="scr-transcript-speaker">{segment.speaker}</span> : null}
          <time className="scr-transcript-time">{formatSegmentTime(segment.start)}</time>
          <p className="scr-transcript-text">{segment.text}</p>
        </article>
      ))}
    </section>
  );
}

function TranscriptPlaceholder({ title, description }: { title: string; description: string }) {
  return (
    <section className="scr-transcript-placeholder" aria-label="Transcript status">
      <h2>{title}</h2>
      <p>{description}</p>
    </section>
  );
}

type StreamingAudioPlayerProps = {
  fileId: string;
  durationSeconds: number | null;
  title: string;
  onDurationChange?: (duration: number) => void;
};

function StreamingAudioPlayer({ fileId, durationSeconds, title, onDurationChange }: StreamingAudioPlayerProps) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const { token } = useAuth();
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(durationSeconds ?? 0);
  const [hasError, setHasError] = useState(false);

  useEffect(() => {
    setDuration(durationSeconds ?? 0);
  }, [durationSeconds]);

  useEffect(() => {
    if (!token) return;
    document.cookie = `scriberr_access_token=${encodeURIComponent(token)}; path=/; SameSite=Lax`;
  }, [token]);

  const streamUrl = `/api/v1/files/${fileId}/audio`;
  const progress = duration > 0 ? currentTime / duration : 0;

  const handleToggle = () => {
    const audio = audioRef.current;
    if (!audio) return;
    if (audio.paused) {
      void audio.play().catch(() => setHasError(true));
      return;
    }
    audio.pause();
  };

  const handleSeek = (event: ChangeEvent<HTMLInputElement>) => {
    const nextTime = Number(event.currentTarget.value);
    setCurrentTime(nextTime);
    if (audioRef.current) {
      audioRef.current.currentTime = nextTime;
    }
  };

  return (
    <div className="scr-player-shell" aria-label={`${title} audio player`}>
      <div className="scr-player-time scr-player-time-current">{formatSegmentTime(currentTime)}</div>
      <button className="scr-player-toggle" type="button" onClick={handleToggle} aria-label={isPlaying ? "Pause" : "Play"}>
        {isPlaying ? <Pause size={18} aria-hidden="true" /> : <Play size={18} aria-hidden="true" />}
      </button>
      <label className="scr-player-seek">
        <span className="scr-visually-hidden">Seek audio</span>
        <input
          type="range"
          min="0"
          max={Math.max(duration, 1)}
          step="0.1"
          value={Math.min(currentTime, Math.max(duration, 1))}
          onChange={handleSeek}
          style={{ "--scr-player-progress": `${progress * 100}%` } as CSSProperties}
        />
      </label>
      <div className="scr-player-time">{formatSegmentTime(duration)}</div>
      {hasError ? <div className="scr-player-error">Audio stream unavailable</div> : null}
      <audio
        ref={audioRef}
        src={streamUrl}
        preload="metadata"
        onLoadedMetadata={(event) => {
          const nextDuration = event.currentTarget.duration;
          if (Number.isFinite(nextDuration)) {
            setDuration(nextDuration);
            onDurationChange?.(nextDuration);
          }
          setHasError(false);
        }}
        onTimeUpdate={(event) => setCurrentTime(event.currentTarget.currentTime)}
        onPlay={() => setIsPlaying(true)}
        onPause={() => setIsPlaying(false)}
        onEnded={() => setIsPlaying(false)}
        onError={() => setHasError(true)}
      />
    </div>
  );
}

function latestTranscriptionForFile(transcriptions: Transcription[], fileId: string) {
  return transcriptions
    .filter((transcription) => transcription.file_id === fileId)
    .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())[0];
}

function normalizeTranscriptSegments(transcript?: TranscriptionTranscript): TranscriptSegment[] {
  if (!transcript) return [];
  if (transcript.segments.length > 1) return transcript.segments.filter((segment) => segment.text.trim());
  if (transcript.words.length > 0) return chunkWordsIntoDisplaySegments(transcript.words);
  if (transcript.segments.length === 1 && transcript.segments[0].text.trim()) return transcript.segments;
  const text = transcript.text.trim();
  if (!text) return [];
  return [{
    id: "full-transcript",
    start: 0,
    end: 0,
    text,
  }];
}

function chunkWordsIntoDisplaySegments(words: TranscriptWord[]): TranscriptSegment[] {
  const segments: TranscriptSegment[] = [];
  let current: TranscriptWord[] = [];

  const flush = () => {
    if (!current.length) return;
    const first = current[0];
    const last = current[current.length - 1];
    const text = current.map((word) => word.word.trim()).filter(Boolean).join(" ");
    if (text) {
      segments.push({
        id: `word-segment-${segments.length}`,
        start: first.start,
        end: last.end,
        speaker: first.speaker,
        text,
      });
    }
    current = [];
  };

  for (const word of words) {
    current.push(word);
    const cleaned = word.word.trim();
    const closesSentence = /[.!?]$/.test(cleaned);
    if ((closesSentence && current.length >= 10) || current.length >= 32) {
      flush();
    }
  }

  flush();
  return segments;
}

function formatTranscriptionProgress(transcription: Transcription) {
  const progress = normalizeProgress(transcription.progress);
  const stage = transcription.progress_stage && transcription.progress_stage !== "queued"
    ? transcription.progress_stage
    : transcription.status;
  if (progress === null) return sentenceCase(stage);
  return `${sentenceCase(stage)} · ${progress}%`;
}

function normalizeProgress(value: number) {
  if (!Number.isFinite(value) || value <= 0) return null;
  const percent = value <= 1 ? value * 100 : value;
  return Math.min(100, Math.max(1, Math.round(percent)));
}

function sentenceCase(value: string) {
  const normalized = value.replace(/[_-]+/g, " ").trim();
  if (!normalized) return "";
  return normalized.charAt(0).toUpperCase() + normalized.slice(1);
}

function formatCreatedDate(value: string) {
  return new Date(value).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function formatDurationLabel(value: number | null) {
  if (!value || value <= 0) return "Unknown duration";
  if (value < 60) return `${Math.round(value)} sec`;
  const minutes = Math.round(value / 60);
  return `${minutes} min`;
}

function formatSegmentTime(value: number) {
  if (!Number.isFinite(value) || value < 0) return "0:00";
  const hours = Math.floor(value / 3600);
  const minutes = Math.floor((value % 3600) / 60);
  const seconds = Math.floor(value % 60);
  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, "0")}:${seconds.toString().padStart(2, "0")}`;
  }
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}
