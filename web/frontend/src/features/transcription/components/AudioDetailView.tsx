import { useEffect, useMemo, useRef, useState, type CSSProperties, type ChangeEvent } from "react";
import { Link, useParams } from "react-router-dom";
import { CalendarDays, Clock3, MoreHorizontal, Pause, Pencil, Play } from "lucide-react";
import { Sidebar } from "@/features/home/components/HomePage";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useFile, useUpdateFile } from "@/features/files/hooks/useFiles";

type DetailTab = "summary" | "transcript";

type MockSegment = {
  id: string;
  start: number;
  text: string;
  speaker?: string;
};

const mockSegments: MockSegment[] = [
  {
    id: "seg-1",
    start: 1,
    text: "About adding event listeners",
  },
  {
    id: "seg-2",
    start: 3,
    text: "On your element. We are going to dive deeper and explain the things you need to know about adding event listeners on your elements in your app component.",
  },
  {
    id: "seg-3",
    start: 17,
    text: "So let's take a look at the repo.",
  },
  {
    id: "seg-4",
    start: 18,
    text: "Over here, I have a heading and if I want to add event listeners, this is how I would have done it. If I write vanilla JavaScript, I would have a reference to the element and I will say add event listener. Then I will pass in the event name and the event handler. There are two ways to handle this: create a function inline, or create the function ahead of time and pass the reference to this event listener method.",
  },
  {
    id: "seg-5",
    start: 93,
    text: "So that's how you say this.",
  },
];

export function AudioDetailView() {
  const { audioId = "" } = useParams<{ audioId: string }>();
  const [activeTab, setActiveTab] = useState<DetailTab>("transcript");
  const [audioDuration, setAudioDuration] = useState<number | null>(null);
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [draftTitle, setDraftTitle] = useState("");
  const fileQuery = useFile(audioId);
  const updateFileMutation = useUpdateFile(audioId);

  const file = fileQuery.data;
  const title = file?.title?.trim() || "Untitled recording";
  const visibleDuration = file?.duration_seconds ?? audioDuration;

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

            {activeTab === "summary" ? <MockSummary /> : <TranscriptMock segments={mockSegments} />}
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

function MockSummary() {
  return (
    <section className="scr-audio-summary" aria-label="Summary">
      <p>
        This recording walks through adding event listeners, how event names and handlers are passed, and the difference
        between inline callbacks and pre-defined handler functions.
      </p>
      <p>
        The key workflow is to identify the target element, choose the relevant event, and pass a stable function reference
        when the handler will be reused or benefits from being named.
      </p>
    </section>
  );
}

function TranscriptMock({ segments }: { segments: MockSegment[] }) {
  return (
    <section className="scr-transcript" aria-label="Transcript">
      {segments.map((segment) => (
        <article className="scr-transcript-segment" key={segment.id}>
          {segment.speaker ? <span className="scr-transcript-speaker">{segment.speaker}</span> : null}
          <time className="scr-transcript-time">{formatSegmentTime(segment.start)}</time>
          <p className="scr-transcript-text">{segment.text}</p>
        </article>
      ))}
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
