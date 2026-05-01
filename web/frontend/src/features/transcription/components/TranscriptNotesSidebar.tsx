import { useEffect, useState, type FormEvent, type KeyboardEvent, type PointerEvent } from "react";
import { MessageCircle, Mic2, MoreHorizontal, PanelRightClose, PanelRightOpen, Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { TranscriptChatPanel } from "@/features/transcription/components/TranscriptChatPanel";
import type { TranscriptNoteAnnotation } from "@/features/transcription/api/annotationsApi";

type TranscriptNotesSidebarProps = {
  notes: TranscriptNoteAnnotation[];
  parentTranscriptionId?: string;
  isOpen: boolean;
  isLoading: boolean;
  isError: boolean;
  isCreatingEntry: boolean;
  width: number;
  onWidthChange: (width: number) => void;
  onCreateEntry: (annotationId: string, content: string) => Promise<void>;
  onSeekRequest: (seconds: number) => void;
  onOpenChange: (isOpen: boolean) => void;
};

export function TranscriptNotesSidebar({
  notes,
  parentTranscriptionId,
  isOpen,
  isLoading,
  isError,
  isCreatingEntry,
  width,
  onWidthChange,
  onCreateEntry,
  onSeekRequest,
  onOpenChange,
}: TranscriptNotesSidebarProps) {
  const [activeReplyNoteId, setActiveReplyNoteId] = useState<string | null>(null);
  const [activePanel, setActivePanel] = useState<"chat" | "notes">("chat");
  const [dragState, setDragState] = useState<{ startX: number; startWidth: number } | null>(null);

  useEffect(() => {
    if (!dragState) return;

    const handlePointerMove = (event: globalThis.PointerEvent) => {
      onWidthChange(dragState.startWidth + dragState.startX - event.clientX);
    };
    const handlePointerUp = () => {
      setDragState(null);
    };

    document.body.dataset.notesSidebarResizing = "true";
    window.addEventListener("pointermove", handlePointerMove);
    window.addEventListener("pointerup", handlePointerUp, { once: true });
    window.addEventListener("pointercancel", handlePointerUp, { once: true });

    return () => {
      delete document.body.dataset.notesSidebarResizing;
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerup", handlePointerUp);
      window.removeEventListener("pointercancel", handlePointerUp);
    };
  }, [dragState, onWidthChange]);

  const handleResizePointerDown = (event: PointerEvent<HTMLDivElement>) => {
    if (!isOpen) return;
    event.preventDefault();
    setDragState({ startX: event.clientX, startWidth: width });
  };

  const handleResizeKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!isOpen) return;
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
    event.preventDefault();
    const step = event.shiftKey ? 48 : 16;
    onWidthChange(width + (event.key === "ArrowLeft" ? step : -step));
  };

  return (
    <aside className="scr-transcript-notes-sidebar" data-open={isOpen} aria-label="Transcript notes">
      {isOpen ? (
        <div
          className="scr-transcript-notes-resize-handle"
          role="separator"
          aria-label="Resize notes sidebar"
          aria-orientation="vertical"
          aria-valuenow={Math.round(width)}
          tabIndex={0}
          onPointerDown={handleResizePointerDown}
          onKeyDown={handleResizeKeyDown}
        />
      ) : null}
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            className="scr-transcript-notes-toggle"
            type="button"
            variant="ghost"
            size="icon"
            aria-label={isOpen ? "Collapse notes" : "Open notes"}
            aria-expanded={isOpen}
            onClick={() => onOpenChange(!isOpen)}
          >
            {isOpen ? <PanelRightClose size={17} aria-hidden="true" /> : <PanelRightOpen size={17} aria-hidden="true" />}
          </Button>
        </TooltipTrigger>
        <TooltipContent>{isOpen ? "Collapse notes" : "Notes"}</TooltipContent>
      </Tooltip>

      {isOpen ? (
        <div className="scr-transcript-notes-panel">
          <nav className="scr-transcript-notes-tabs" aria-label="Detail sidebar">
            <button type="button" data-active={activePanel === "chat" ? "true" : undefined} onClick={() => setActivePanel("chat")}>Chat</button>
            <button type="button" data-active={activePanel === "notes" ? "true" : undefined} onClick={() => setActivePanel("notes")}>Notes</button>
            <Button
              className="scr-transcript-notes-close"
              type="button"
              variant="ghost"
              size="icon"
              aria-label="Collapse notes"
              onClick={() => onOpenChange(false)}
            >
              <PanelRightClose size={16} aria-hidden="true" />
            </Button>
          </nav>

          {activePanel === "chat" ? (
            <TranscriptChatPanel parentTranscriptionId={parentTranscriptionId} />
          ) : (
            <div className="scr-transcript-notes-list">
              {isLoading ? <p className="scr-transcript-notes-status">Loading notes.</p> : null}
              {isError ? <p className="scr-transcript-notes-status">Notes could not be loaded.</p> : null}
              {!isLoading && !isError && notes.length === 0 ? (
                <p className="scr-transcript-notes-status">No notes yet.</p>
              ) : null}
              {!isLoading && !isError ? notes.map((note) => (
                <TranscriptNoteItem
                  key={note.id}
                  note={note}
                  isReplyActive={activeReplyNoteId === note.id}
                  isCreatingEntry={isCreatingEntry && activeReplyNoteId === note.id}
                  onActivateReply={() => setActiveReplyNoteId(note.id)}
                  onCancelReply={() => setActiveReplyNoteId(null)}
                  onCreateEntry={async (content) => {
                    setActiveReplyNoteId(note.id);
                    await onCreateEntry(note.id, content);
                    setActiveReplyNoteId(null);
                  }}
                  onSeekRequest={onSeekRequest}
                />
              )) : null}
            </div>
          )}
        </div>
      ) : null}
    </aside>
  );
}

type TranscriptNoteItemProps = {
  note: TranscriptNoteAnnotation;
  isReplyActive: boolean;
  isCreatingEntry: boolean;
  onActivateReply: () => void;
  onCancelReply: () => void;
  onCreateEntry: (content: string) => Promise<void>;
  onSeekRequest: (seconds: number) => void;
};

function TranscriptNoteItem({
  note,
  isReplyActive,
  isCreatingEntry,
  onActivateReply,
  onCancelReply,
  onCreateEntry,
  onSeekRequest,
}: TranscriptNoteItemProps) {
  const timeLabel = formatAnnotationTime(note.anchor.start_ms);
  const seekSeconds = Math.max(0, note.anchor.start_ms / 1000);
  const noteCount = note.entries.length;
  const [replyContent, setReplyContent] = useState("");
  const canSubmitReply = replyContent.trim().length > 0 && !isCreatingEntry;

  const handleReplySubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const content = replyContent.trim();
    if (!content || isCreatingEntry) return;
    await onCreateEntry(content);
    setReplyContent("");
  };

  const handleReplyKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Escape") {
      event.preventDefault();
      setReplyContent("");
      onCancelReply();
      event.currentTarget.blur();
      return;
    }
    if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      event.currentTarget.form?.requestSubmit();
    }
  };

  return (
    <article className="scr-transcript-note-item">
      <h3>{note.quote}</h3>
      <div className="scr-transcript-note-meta">
        <button
          className="scr-transcript-note-time"
          type="button"
          aria-label={`Seek to ${timeLabel}`}
          onClick={() => onSeekRequest(seekSeconds)}
        >
          <Mic2 size={16} aria-hidden="true" />
          {timeLabel}
        </button>
        <span className="scr-transcript-note-count" aria-label={`${noteCount} ${noteCount === 1 ? "note" : "notes"}`}>
          <MessageCircle size={16} aria-hidden="true" />
          {noteCount}
        </span>
      </div>
      {note.entries.map((entry) => (
        <div className="scr-transcript-note-entry" key={entry.id}>
          <p>{entry.content}</p>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                className="scr-transcript-note-delete"
                type="button"
                variant="ghost"
                size="icon"
                aria-label="Note actions"
              >
                <MoreHorizontal size={16} aria-hidden="true" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Note actions</TooltipContent>
          </Tooltip>
        </div>
      ))}
      <form className="scr-transcript-note-reply" onSubmit={handleReplySubmit}>
        <input
          className="scr-transcript-note-reply-input"
          value={replyContent}
          aria-label={`Reply to note at ${timeLabel}`}
          placeholder="Reply..."
          disabled={isCreatingEntry}
          onChange={(event) => setReplyContent(event.currentTarget.value)}
          onFocus={onActivateReply}
          onKeyDown={handleReplyKeyDown}
        />
        <Button
          className="scr-transcript-note-reply-submit"
          type="submit"
          variant="ghost"
          size="icon"
          aria-label="Send reply"
          disabled={!canSubmitReply || !isReplyActive}
        >
          <Send size={16} aria-hidden="true" />
        </Button>
      </form>
    </article>
  );
}

function formatAnnotationTime(milliseconds: number) {
  const totalSeconds = Math.max(0, Math.floor(milliseconds / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
  }
  return `${minutes}:${String(seconds).padStart(2, "0")}`;
}
