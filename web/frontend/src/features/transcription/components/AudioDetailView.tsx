import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties, type ChangeEvent, type ReactNode, type SyntheticEvent } from "react";
import { Link, useParams } from "react-router-dom";
import { AlignJustify, CalendarDays, CheckSquare, Clock3, FileText, MoreHorizontal, Pause, Pencil, Play } from "lucide-react";
import { Sidebar } from "@/features/home/components/HomePage";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useToast } from "@/components/ui/toast";
import { useFile, useUpdateFile } from "@/features/files/hooks/useFiles";
import type { FileStatus } from "@/features/files/api/filesApi";
import type { TranscriptAnnotation } from "@/features/transcription/api/annotationsApi";
import type { SummaryWidgetRun, TranscriptionSummary } from "@/features/transcription/api/summariesApi";
import type { TranscriptWord, Transcription, TranscriptionTranscript } from "@/features/transcription/api/transcriptionsApi";
import { TranscriptHighlightMenu } from "@/features/transcription/components/TranscriptHighlightMenu";
import { TranscriptNoteComposer, type TranscriptNoteComposerSelection } from "@/features/transcription/components/TranscriptNoteComposer";
import { TranscriptNotesSidebar } from "@/features/transcription/components/TranscriptNotesSidebar";
import { TranscriptSelectionMenu } from "@/features/transcription/components/TranscriptSelectionMenu";
import { AudioTagSection } from "@/features/tags/components/AudioTagSection";
import { useFileEvents } from "@/features/files/hooks/useFileEvents";
import { useTranscriptionDetailEvents } from "@/features/transcription/hooks/useTranscriptionDetailEvents";
import { computeWordOffsets, computeWordOffsetsInText, createPlaybackSync, useTranscriptKaraokeHighlight, type KaraokeHighlightSegment, type PlaybackSync } from "@/features/transcription/hooks/useKaraokeHighlight";
import { useTranscriptClickSeek } from "@/features/transcription/hooks/useTranscriptClickSeek";
import { useTranscriptTextSelection } from "@/features/transcription/hooks/useTranscriptTextSelection";
import { selectTranscriptNotes, useCreateTranscriptHighlight, useCreateTranscriptNote, useCreateTranscriptNoteEntry, useDeleteTranscriptHighlight, useDeleteTranscriptNoteEntry, useTranscriptAnnotations, useUpdateTranscriptNoteEntry } from "@/features/transcription/hooks/useTranscriptAnnotations";
import { useTranscriptionListEvents } from "@/features/transcription/hooks/useTranscriptionListEvents";
import { useTranscriptionSummary, useTranscriptionSummaryWidgets } from "@/features/transcription/hooks/useTranscriptionSummaries";
import { preferVisibleTranscription, useTranscriptionTranscript, useTranscriptions } from "@/features/transcription/hooks/useTranscriptions";
import { ReadOnlyMarkdown } from "@/features/transcription/components/ReadOnlyMarkdown";
import { buildHighlightRangesBySegment, hasDuplicateActiveHighlight, type SegmentHighlightRange } from "@/features/transcription/utils/transcriptHighlighting";
import type { SelectionMenuRect } from "@/features/transcription/utils/transcriptHighlighting";
import { buildWordSeekTargetsFromOffsets } from "@/features/transcription/utils/wordSeekIndex";

type DetailTab = "summary" | "transcript";

type AudioSeekRequest = {
  seconds: number;
  token: number;
};

const notesSidebarMinWidth = 320;
const notesSidebarDefaultMaxWidth = 720;
const notesSidebarDefaultViewportRatio = 0.2;

function clampNotesSidebarWidth(width: number) {
  const viewportMax = typeof window === "undefined" ? notesSidebarDefaultMaxWidth : Math.floor(window.innerWidth * 0.4);
  const maxWidth = Math.max(notesSidebarMinWidth, viewportMax);
  return Math.min(Math.max(Math.round(width), notesSidebarMinWidth), maxWidth);
}

function getDefaultNotesSidebarWidth() {
  if (typeof window === "undefined") return 360;
  return clampNotesSidebarWidth(window.innerWidth * notesSidebarDefaultViewportRatio);
}

export function AudioDetailView() {
  const { audioId = "" } = useParams<{ audioId: string }>();
  const [activeTab, setActiveTab] = useState<DetailTab>("summary");
  const [audioDuration, setAudioDuration] = useState<number | null>(null);
  const [audioSeekRequest, setAudioSeekRequest] = useState<AudioSeekRequest | null>(null);
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [draftTitle, setDraftTitle] = useState("");
  const [notesSidebarOpen, setNotesSidebarOpen] = useState(true);
  const [notesSidebarWidth, setNotesSidebarWidth] = useState(getDefaultNotesSidebarWidth);
  const playbackSync = useMemo(() => createPlaybackSync(), [audioId]);
  const fileQuery = useFile(audioId);
  const updateFileMutation = useUpdateFile(audioId);
  const transcriptionsQuery = useTranscriptions();
  const { toast } = useToast();
  const warnedSummaryIds = useRef<Set<string>>(new Set());
  const nextSeekToken = useRef(0);

  const file = fileQuery.data;
  const title = file?.title?.trim() || "Untitled recording";
  const visibleDuration = file?.duration_seconds ?? audioDuration;
  const latestTranscription = useMemo(() => {
    if (!file) return undefined;
    return latestTranscriptionForFile(transcriptionsQuery.data?.items || [], file.id);
  }, [file, transcriptionsQuery.data?.items]);
  const transcriptQuery = useTranscriptionTranscript(latestTranscription?.id, latestTranscription?.status === "completed");
  const annotationsQuery = useTranscriptAnnotations(
    latestTranscription?.id,
    Boolean(latestTranscription?.status === "completed")
  );
  const createNoteEntryMutation = useCreateTranscriptNoteEntry(latestTranscription?.id || "");
  const updateNoteEntryMutation = useUpdateTranscriptNoteEntry(latestTranscription?.id || "");
  const deleteNoteEntryMutation = useDeleteTranscriptNoteEntry(latestTranscription?.id || "");
  const notes = useMemo(
    () => selectTranscriptNotes(annotationsQuery.data?.items),
    [annotationsQuery.data?.items]
  );
  const handleSummaryTruncated = useCallback((summaryId: string) => {
    if (warnedSummaryIds.current.has(summaryId)) return;
    warnedSummaryIds.current.add(summaryId);
    toast({
      title: "Transcript truncated for summary",
      description: "The transcript exceeds the small model context window, so summarization will use the first fitting portion.",
    });
  }, [toast]);

  const handleNoteSeekRequest = useCallback((seconds: number) => {
    nextSeekToken.current += 1;
    setAudioSeekRequest({ seconds, token: nextSeekToken.current });
  }, []);

  const handleCreateNoteEntry = useCallback(async (annotationId: string, content: string) => {
    await createNoteEntryMutation.mutateAsync({ annotationId, content });
  }, [createNoteEntryMutation]);

  const handleUpdateNoteEntry = useCallback(async (annotationId: string, entryId: string, content: string) => {
    await updateNoteEntryMutation.mutateAsync({ annotationId, entryId, content });
  }, [updateNoteEntryMutation]);

  const handleDeleteNoteEntry = useCallback(async (annotationId: string, entryId: string) => {
    await deleteNoteEntryMutation.mutateAsync({ annotationId, entryId });
  }, [deleteNoteEntryMutation]);

  const handleNotesSidebarWidthChange = useCallback((width: number) => {
    setNotesSidebarWidth(clampNotesSidebarWidth(width));
  }, []);

  useEffect(() => {
    const handleResize = () => {
      setNotesSidebarWidth((current) => clampNotesSidebarWidth(current));
    };
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  useTranscriptionListEvents();
  useFileEvents();
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
        <main
          className="scr-audio-detail-main"
          data-notes-open={notesSidebarOpen}
          style={{ "--scr-detail-sidebar-width": `${notesSidebarOpen ? notesSidebarWidth : 48}px` } as CSSProperties}
        >
          <div className="scr-audio-detail-layout" data-notes-open={notesSidebarOpen}>
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

              <AudioTagSection
                transcriptionId={latestTranscription?.id}
                enabled={Boolean(latestTranscription?.status === "completed")}
              />

              {activeTab === "summary" ? (
                <SummaryPanel transcription={latestTranscription} />
              ) : (
                <TranscriptPanel
                  fileStatus={file.status}
                  transcription={latestTranscription}
                  transcript={transcriptQuery.data}
                  playbackSync={playbackSync}
                  annotations={annotationsQuery.data?.items || []}
                  isLoading={transcriptionsQuery.isLoading || transcriptQuery.isLoading}
                  isError={transcriptionsQuery.isError || transcriptQuery.isError}
                  onSeekRequest={handleNoteSeekRequest}
                  onNoteSaved={() => setNotesSidebarOpen(true)}
                />
              )}
            </article>
            <TranscriptNotesSidebar
              notes={notes}
              parentTranscriptionId={latestTranscription?.status === "completed" ? latestTranscription.id : undefined}
              isOpen={notesSidebarOpen}
              isLoading={annotationsQuery.isLoading}
              isError={annotationsQuery.isError}
              isCreatingEntry={createNoteEntryMutation.isPending}
              isUpdatingEntry={updateNoteEntryMutation.isPending}
              isDeletingEntry={deleteNoteEntryMutation.isPending}
              width={notesSidebarWidth}
              onWidthChange={handleNotesSidebarWidthChange}
              onCreateEntry={handleCreateNoteEntry}
              onUpdateEntry={handleUpdateNoteEntry}
              onDeleteEntry={handleDeleteNoteEntry}
              onSeekRequest={handleNoteSeekRequest}
              onOpenChange={setNotesSidebarOpen}
            />
          </div>
          <StreamingAudioPlayer
            fileId={file.id}
            durationSeconds={file.duration_seconds}
            title={title}
            playbackSync={playbackSync}
            seekRequest={audioSeekRequest}
            onDurationChange={setAudioDuration}
          />
        </main>
      </div>
    </div>
  );
}

function SummaryPanel({ transcription }: { transcription?: Transcription }) {
  const summaryQuery = useTranscriptionSummary(transcription?.id, Boolean(transcription));
  const widgetRunsQuery = useTranscriptionSummaryWidgets(transcription?.id, Boolean(transcription && summaryQuery.data?.status === "completed"));
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
    return <SummaryOverview summary={summary} widgetRuns={widgetRunsQuery.data || []} widgetsLoading={widgetRunsQuery.isLoading} widgetsError={widgetRunsQuery.isError} />;
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

function SummaryOverview({ summary, widgetRuns, widgetsLoading, widgetsError }: { summary: TranscriptionSummary; widgetRuns: SummaryWidgetRun[]; widgetsLoading: boolean; widgetsError: boolean }) {
  return (
    <section className="scr-audio-summary" aria-label="Summary">
      <section className="scr-summary-overview-section" aria-label="Overview">
        <SummarySectionHeading icon="overview" title="Overview" />
        <p className="scr-summary-body">{summary.content}</p>
      </section>
      {widgetsLoading ? <p className="scr-summary-status">Checking summary widgets.</p> : null}
      {widgetsError ? <p className="scr-summary-status">Summary widgets could not be loaded.</p> : null}
      {widgetRuns.map((run) => <SummaryWidgetSection key={run.id} run={run} />)}
    </section>
  );
}

function SummaryWidgetSection({ run }: { run: SummaryWidgetRun }) {
  if (run.status === "pending" || run.status === "processing") {
    return (
      <section className="scr-summary-widget-section" aria-label={run.display_title}>
        <SummarySectionHeading icon="widget" title={run.display_title} />
        <p className="scr-summary-status">{run.status === "pending" ? "Widget is queued." : "Widget is being generated."}</p>
      </section>
    );
  }

  if (run.status === "failed") {
    return (
      <section className="scr-summary-widget-section" aria-label={run.display_title}>
        <SummarySectionHeading icon="widget" title={run.display_title} />
        <p className="scr-summary-status">{run.error || "Widget generation failed."}</p>
      </section>
    );
  }

  if (!run.output.trim()) return null;

  return (
    <section className="scr-summary-widget-section" aria-label={run.display_title}>
      <SummarySectionHeading icon={run.render_markdown ? "markdown" : "widget"} title={run.display_title} />
      {run.render_markdown ? (
        <ReadOnlyMarkdown content={run.output} />
      ) : (
        <p className="scr-summary-body">{run.output}</p>
      )}
      {run.context_truncated ? <p className="scr-summary-note">Context was truncated to fit the model window.</p> : null}
    </section>
  );
}

function SummarySectionHeading({ icon, title }: { icon: "overview" | "widget" | "markdown"; title: string }) {
  const Icon = icon === "overview" ? AlignJustify : icon === "markdown" ? FileText : CheckSquare;
  return (
    <div className="scr-summary-heading">
      <Icon size={18} aria-hidden="true" />
      <h2>{title}</h2>
    </div>
  );
}

type TranscriptPanelProps = {
  fileStatus: FileStatus;
  transcription?: Transcription;
  transcript?: TranscriptionTranscript;
  playbackSync: PlaybackSync;
  annotations: TranscriptAnnotation[];
  isLoading: boolean;
  isError: boolean;
  onSeekRequest: (seconds: number) => void;
  onNoteSaved: () => void;
};

type TranscriptDisplaySegment = KaraokeHighlightSegment & {
  id: string;
  start: number;
  end: number;
  speaker?: string;
  startChar: number;
  endChar: number;
  charAnchorReliable: boolean;
  wordStartIndex: number;
};

function TranscriptPanel({ fileStatus, transcription, transcript, playbackSync, annotations, isLoading, isError, onSeekRequest, onNoteSaved }: TranscriptPanelProps) {
  const transcriptRef = useRef<HTMLElement | null>(null);
  const textElementsRef = useRef<(HTMLElement | null)[]>([]);
  const hideHighlightMenuTimer = useRef<number | undefined>(undefined);
  const [activeHighlight, setActiveHighlight] = useState<{ annotationId: string; rect: SelectionMenuRect } | null>(null);
  const [noteComposerSelection, setNoteComposerSelection] = useState<TranscriptNoteComposerSelection | null>(null);
  const segments = useMemo(() => buildTranscriptDisplaySegments(transcript), [transcript]);
  const hasTimedWords = segments.some((segment) => segment.offsets.length > 0);
  const createHighlightMutation = useCreateTranscriptHighlight(transcription?.id || "");
  const createNoteMutation = useCreateTranscriptNote(transcription?.id || "");
  const deleteHighlightMutation = useDeleteTranscriptHighlight(transcription?.id || "");
  const { toast } = useToast();
  const highlightRangesBySegment = useMemo(
    () => buildHighlightRangesBySegment(segments, annotations),
    [segments, annotations]
  );
  const selectionSegments = useMemo(() => (
    segments.map((segment, index) => ({
      index,
      start: segment.start,
      end: segment.end,
      startChar: segment.startChar,
      endChar: segment.endChar,
      charAnchorReliable: segment.charAnchorReliable,
      wordStartIndex: segment.wordStartIndex,
      offsets: segment.offsets,
    }))
  ), [segments]);
  const clickSeekSegments = useMemo(() => (
    segments.map((segment, index) => ({
      index,
      targets: buildWordSeekTargetsFromOffsets(segment.offsets, segment.wordStartIndex),
    }))
  ), [segments]);
  const { selection: pendingSelection, clearSelection } = useTranscriptTextSelection(
    transcriptRef,
    selectionSegments,
    Boolean(transcription?.status === "completed" && segments.length)
  );
  const isDuplicateHighlightSelection = Boolean(
    pendingSelection &&
    hasDuplicateActiveHighlight(annotations, pendingSelection.anchor, pendingSelection.quote)
  );

  const handleCreateHighlight = () => {
    if (!pendingSelection || !transcription?.id || createHighlightMutation.isPending || isDuplicateHighlightSelection) return;
    setNoteComposerSelection(null);
    createHighlightMutation.mutate({
      quote: pendingSelection.quote,
      anchor: pendingSelection.anchor,
    }, {
      onSuccess: clearSelection,
      onError: (error) => {
        toast({
          title: "Highlight was not saved",
          description: error instanceof Error ? error.message : "Try selecting the text again.",
        });
      },
    });
  };

  const handleOpenNoteComposer = () => {
    if (!pendingSelection || createNoteMutation.isPending) return;
    setNoteComposerSelection({
      quote: pendingSelection.quote,
      anchor: { ...pendingSelection.anchor },
      rect: {
        left: pendingSelection.rect.left,
        top: pendingSelection.rect.top,
        bottom: pendingSelection.rect.bottom,
        width: pendingSelection.rect.width,
      },
    });
  };

  const handleCancelNoteComposer = () => {
    setNoteComposerSelection(null);
  };

  const handleSaveNote = (content: string) => {
    if (!noteComposerSelection || !transcription?.id || createNoteMutation.isPending) return;
    createNoteMutation.mutate({
      content,
      quote: noteComposerSelection.quote,
      anchor: noteComposerSelection.anchor,
    }, {
      onSuccess: () => {
        setNoteComposerSelection(null);
        onNoteSaved();
        clearSelection();
      },
      onError: (error) => {
        toast({
          title: "Note was not saved",
          description: error instanceof Error ? error.message : "Try selecting the text again.",
        });
      },
    });
  };

  const showHighlightMenu = (annotationId: string, rect: DOMRect) => {
    if (hideHighlightMenuTimer.current) window.clearTimeout(hideHighlightMenuTimer.current);
    setActiveHighlight({
      annotationId,
      rect: {
        left: rect.left,
        top: rect.top,
        bottom: rect.bottom,
        width: rect.width,
      },
    });
  };

  const scheduleHideHighlightMenu = () => {
    if (hideHighlightMenuTimer.current) window.clearTimeout(hideHighlightMenuTimer.current);
    hideHighlightMenuTimer.current = window.setTimeout(() => setActiveHighlight(null), 140);
  };

  const keepHighlightMenuOpen = () => {
    if (hideHighlightMenuTimer.current) window.clearTimeout(hideHighlightMenuTimer.current);
  };

  const handleDeleteHighlight = (annotationId: string) => {
    if (!transcription?.id || deleteHighlightMutation.isPending) return;
    deleteHighlightMutation.mutate(annotationId, {
      onSuccess: () => setActiveHighlight(null),
      onError: (error) => {
        toast({
          title: "Highlight was not removed",
          description: error instanceof Error ? error.message : "Try again.",
        });
      },
    });
  };

  useEffect(() => {
    textElementsRef.current.length = segments.length;
  }, [segments.length]);

  useEffect(() => () => {
    if (hideHighlightMenuTimer.current) window.clearTimeout(hideHighlightMenuTimer.current);
  }, []);

  useTranscriptKaraokeHighlight(
    playbackSync,
    segments,
    textElementsRef,
    Boolean(transcription?.status === "completed" && hasTimedWords)
  );
  useTranscriptClickSeek(
    transcriptRef,
    clickSeekSegments,
    Boolean(transcription?.status === "completed" && hasTimedWords),
    onSeekRequest
  );

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

  if (transcription.status === "stopped" || transcription.status === "canceled") {
    return <TranscriptPlaceholder title="Transcription stopped" description="Start another transcription from Home to generate transcript text." />;
  }

  if (!segments.length) {
    return <TranscriptPlaceholder title="Transcript is empty" description="The completed transcription did not return any text." />;
  }

  return (
    <>
      <section ref={transcriptRef} className="scr-transcript" aria-label="Transcript">
        {segments.map((segment, index) => (
          <article className="scr-transcript-segment" key={segment.id || `${segment.start}-${segment.end}`}>
            {segment.speaker ? <span className="scr-transcript-speaker">{segment.speaker}</span> : null}
            <time className="scr-transcript-time">{formatSegmentTime(segment.start)}</time>
            <p
              ref={(element) => {
                textElementsRef.current[index] = element;
              }}
              className="scr-transcript-text"
              data-transcript-text
              data-click-seek-enabled={hasTimedWords ? "true" : undefined}
              data-transcript-segment-index={index}
              data-start-char={segment.startChar}
              data-end-char={segment.endChar}
            >
              {renderTranscriptTextWithHighlights(
                segment,
                highlightRangesBySegment.get(index) || [],
                showHighlightMenu,
                scheduleHideHighlightMenu
              )}
            </p>
          </article>
        ))}
      </section>
      <TranscriptSelectionMenu
        selection={noteComposerSelection ? null : pendingSelection}
        isCreatingHighlight={createHighlightMutation.isPending}
        isDuplicateHighlight={isDuplicateHighlightSelection}
        onCreateHighlight={handleCreateHighlight}
        onOpenNoteComposer={handleOpenNoteComposer}
      />
      <TranscriptNoteComposer
        selection={noteComposerSelection}
        isSaving={createNoteMutation.isPending}
        onCancel={handleCancelNoteComposer}
        onSave={handleSaveNote}
      />
      <TranscriptHighlightMenu
        activeHighlight={activeHighlight}
        isDeleting={deleteHighlightMutation.isPending}
        onDeleteHighlight={handleDeleteHighlight}
        onMouseEnter={keepHighlightMenuOpen}
        onMouseLeave={scheduleHideHighlightMenu}
      />
    </>
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
  playbackSync: PlaybackSync;
  seekRequest?: AudioSeekRequest | null;
  onDurationChange?: (duration: number) => void;
};

function StreamingAudioPlayer({ fileId, durationSeconds, title, playbackSync, seekRequest, onDurationChange }: StreamingAudioPlayerProps) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const handledSeekToken = useRef<number | null>(null);
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

  useEffect(() => {
    playbackSync.publish({ currentTime: 0, isPlaying: false });
    return () => playbackSync.publish({ isPlaying: false });
  }, [fileId, playbackSync]);

  useEffect(() => {
    if (!isPlaying) return;
    let frameId = 0;
    const publishFrame = () => {
      const audio = audioRef.current;
      if (!audio || audio.paused || audio.ended) return;
      playbackSync.publish({ currentTime: audio.currentTime, isPlaying: true });
      frameId = window.requestAnimationFrame(publishFrame);
    };
    frameId = window.requestAnimationFrame(publishFrame);
    return () => window.cancelAnimationFrame(frameId);
  }, [isPlaying, playbackSync]);

  useEffect(() => {
    if (!seekRequest) return;
    if (handledSeekToken.current === seekRequest.token) return;
    handledSeekToken.current = seekRequest.token;
    const audio = audioRef.current;
    const maxDuration = Number.isFinite(duration) && duration > 0 ? duration : Number.POSITIVE_INFINITY;
    const nextTime = Math.max(0, Math.min(seekRequest.seconds, maxDuration));
    setCurrentTime(nextTime);
    playbackSync.publish({ currentTime: nextTime, isPlaying });
    if (audio) {
      audio.currentTime = nextTime;
    }
  }, [duration, isPlaying, playbackSync, seekRequest]);

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
    const maxDuration = duration > 0 ? duration : Number.POSITIVE_INFINITY;
    const nextTime = Math.max(0, Math.min(Number(event.currentTarget.value), maxDuration));
    setCurrentTime(nextTime);
    playbackSync.publish({ currentTime: nextTime, isPlaying });
    if (audioRef.current) {
      audioRef.current.currentTime = nextTime;
    }
  };

  const handleTimeUpdate = (event: SyntheticEvent<HTMLAudioElement>) => {
    const nextTime = event.currentTarget.currentTime;
    setCurrentTime(nextTime);
    playbackSync.publish({ currentTime: nextTime, isPlaying: !event.currentTarget.paused });
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
        onTimeUpdate={handleTimeUpdate}
        onPlay={(event) => {
          setIsPlaying(true);
          playbackSync.publish({ currentTime: event.currentTarget.currentTime, isPlaying: true });
        }}
        onPause={(event) => {
          setIsPlaying(false);
          playbackSync.publish({ currentTime: event.currentTarget.currentTime, isPlaying: false });
        }}
        onEnded={(event) => {
          setIsPlaying(false);
          playbackSync.publish({ currentTime: event.currentTarget.currentTime, isPlaying: false });
        }}
        onError={() => setHasError(true)}
      />
    </div>
  );
}

function latestTranscriptionForFile(transcriptions: Transcription[], fileId: string) {
  return transcriptions
    .filter((transcription) => transcription.file_id === fileId)
    .reduce<Transcription | undefined>((current, transcription) => preferVisibleTranscription(transcription, current), undefined);
}

function buildTranscriptDisplaySegments(transcript?: TranscriptionTranscript): TranscriptDisplaySegment[] {
  if (!transcript) return [];
  let nextStartChar = 0;
  let searchFrom = 0;
  const sourceText = transcript.text || "";
  const appendSegment = (segment: Omit<TranscriptDisplaySegment, "startChar" | "endChar" | "charAnchorReliable">) => {
    const sourceIndex = sourceText ? sourceText.indexOf(segment.text, searchFrom) : -1;
    if (sourceIndex !== -1) {
      const endChar = sourceIndex + segment.text.length;
      searchFrom = endChar;
      nextStartChar = endChar + 1;
      return { ...segment, startChar: sourceIndex, endChar, charAnchorReliable: true };
    }

    const startChar = nextStartChar;
    const endChar = startChar + segment.text.length;
    nextStartChar = endChar + 1;
    return { ...segment, startChar, endChar, charAnchorReliable: false };
  };

  if (transcript.segments.length > 1) {
    const words = transcript.words || [];
    return transcript.segments.flatMap((segment, index) => {
      const segmentWords = words.filter((word) => word.start < segment.end + 0.25 && word.end > segment.start - 0.25);
      const firstWordIndex = segmentWords[0] ? words.findIndex((word) => word === segmentWords[0]) : -1;
      const segmentText = segment.text.trim();
      const textOffsets = segmentText ? computeWordOffsetsInText(segmentText, segmentWords) : [];
      const computed = computeWordOffsets(segmentWords);
      const useSegmentText = Boolean(segmentText && (!segmentWords.length || textOffsets.length > 0));
      const text = useSegmentText ? segmentText : computed.fullText;
      if (!text) return [];
      return [appendSegment({
        id: segment.id || `segment-${index}`,
        start: segment.start,
        end: segment.end,
        speaker: segment.speaker || segmentWords[0]?.speaker,
        text,
        offsets: useSegmentText ? textOffsets : computed.offsets,
        wordStartIndex: firstWordIndex === -1 ? 0 : firstWordIndex,
      })];
    });
  }
  if (transcript.words.length > 0) return withTranscriptCharOffsets(chunkWordsIntoDisplaySegments(transcript.words), sourceText);
  if (transcript.segments.length === 1 && transcript.segments[0].text.trim()) {
    const segment = transcript.segments[0];
    return [appendSegment({
      id: segment.id || "segment-0",
      start: segment.start,
      end: segment.end,
      speaker: segment.speaker,
      text: segment.text.trim(),
      offsets: [],
      wordStartIndex: 0,
    })];
  }
  const text = transcript.text.trim();
  if (!text) return [];
  return [appendSegment({
    id: "full-transcript",
    start: 0,
    end: 0,
    text,
    offsets: [],
    wordStartIndex: 0,
  })];
}

function chunkWordsIntoDisplaySegments(words: TranscriptWord[]): Omit<TranscriptDisplaySegment, "startChar" | "endChar" | "charAnchorReliable">[] {
  const segments: Omit<TranscriptDisplaySegment, "startChar" | "endChar" | "charAnchorReliable">[] = [];
  let current: TranscriptWord[] = [];
  let currentStartIndex = 0;

  const flush = () => {
    if (!current.length) return;
    const first = current[0];
    const last = current[current.length - 1];
    const computed = computeWordOffsets(current);
    if (computed.fullText) {
      segments.push({
        id: `word-segment-${segments.length}`,
        start: first.start,
        end: last.end,
        speaker: first.speaker,
        text: computed.fullText,
        offsets: computed.offsets,
        wordStartIndex: currentStartIndex,
      });
    }
    currentStartIndex += current.length;
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

function withTranscriptCharOffsets(segments: Omit<TranscriptDisplaySegment, "startChar" | "endChar" | "charAnchorReliable">[], sourceText: string) {
  let nextStartChar = 0;
  let searchFrom = 0;
  return segments.map((segment) => {
    const sourceIndex = sourceText ? sourceText.indexOf(segment.text, searchFrom) : -1;
    if (sourceIndex !== -1) {
      const endChar = sourceIndex + segment.text.length;
      searchFrom = endChar;
      nextStartChar = endChar + 1;
      return { ...segment, startChar: sourceIndex, endChar, charAnchorReliable: true };
    }

    const startChar = nextStartChar;
    const endChar = startChar + segment.text.length;
    nextStartChar = endChar + 1;
    return { ...segment, startChar, endChar, charAnchorReliable: false };
  });
}

function renderTranscriptTextWithHighlights(
  segment: TranscriptDisplaySegment,
  ranges: SegmentHighlightRange[],
  onShowHighlightMenu: (annotationId: string, rect: DOMRect) => void,
  onHideHighlightMenu: () => void
): ReactNode {
  if (!ranges.length) return segment.text;

  const nodes: ReactNode[] = [];
  let cursor = 0;

  ranges.forEach((range, index) => {
    const start = Math.max(0, Math.min(segment.text.length, range.start));
    const end = Math.max(start, Math.min(segment.text.length, range.end));
    if (start > cursor) nodes.push(segment.text.slice(cursor, start));
    if (end > start) {
      const annotationId = range.annotationIds[0];
      nodes.push(
        <mark
          className="scr-transcript-highlight"
          key={`${start}-${end}-${index}`}
          tabIndex={0}
          data-annotation-id={annotationId}
          aria-label="Saved highlight"
          onFocus={(event) => onShowHighlightMenu(annotationId, event.currentTarget.getBoundingClientRect())}
          onMouseEnter={(event) => onShowHighlightMenu(annotationId, event.currentTarget.getBoundingClientRect())}
          onMouseLeave={onHideHighlightMenu}
        >
          {segment.text.slice(start, end)}
        </mark>
      );
    }
    cursor = end;
  });

  if (cursor < segment.text.length) nodes.push(segment.text.slice(cursor));
  return nodes;
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
