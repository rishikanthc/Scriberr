import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { MessageSquare, Send, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import type { TranscriptAnnotationAnchor } from "@/features/transcription/api/annotationsApi";
import { positionSelectionMenu, type SelectionMenuRect } from "@/features/transcription/utils/transcriptHighlighting";

export type TranscriptNoteComposerSelection = {
  quote: string;
  anchor: TranscriptAnnotationAnchor;
  rect: SelectionMenuRect;
};

type TranscriptNoteComposerProps = {
  selection: TranscriptNoteComposerSelection | null;
  isSaving: boolean;
  onCancel: () => void;
  onSave: (content: string) => void;
};

const composerWidth = 340;
const composerHeight = 142;
const composerOffset = 8;

export function TranscriptNoteComposer({ selection, isSaving, onCancel, onSave }: TranscriptNoteComposerProps) {
  const [content, setContent] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  useEffect(() => {
    if (!selection) return;
    setContent("");
    window.setTimeout(() => textareaRef.current?.focus(), 0);
  }, [selection]);

  useEffect(() => {
    if (!selection) return;
    window.addEventListener("resize", onCancel);
    window.addEventListener("scroll", onCancel, true);
    return () => {
      window.removeEventListener("resize", onCancel);
      window.removeEventListener("scroll", onCancel, true);
    };
  }, [onCancel, selection]);

  const position = useMemo(() => {
    if (!selection) return null;
    return positionSelectionMenu(selection.rect, {
      menuWidth: composerWidth,
      menuHeight: composerHeight,
      viewportWidth: window.innerWidth,
      offset: composerOffset,
    });
  }, [selection]);

  if (!selection || !position) return null;

  const trimmedContent = content.trim();
  const canSave = Boolean(trimmedContent) && !isSaving;

  const handleSubmit = () => {
    if (!canSave) return;
    onSave(trimmedContent);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Escape") {
      event.preventDefault();
      onCancel();
      return;
    }
    if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      handleSubmit();
    }
  };

  return (
    <div
      className="scr-transcript-note-composer"
      data-placement={position.placement}
      role="dialog"
      aria-label="Add note"
      style={{ left: position.left, top: position.top }}
    >
      <div className="scr-transcript-note-composer-heading">
        <MessageSquare size={15} aria-hidden="true" />
        <span>Note</span>
      </div>
      <Textarea
        ref={textareaRef}
        className="scr-transcript-note-composer-input"
        value={content}
        onChange={(event) => setContent(event.currentTarget.value)}
        onKeyDown={handleKeyDown}
        placeholder="Add note..."
        aria-label="Note text"
        disabled={isSaving}
      />
      <div className="scr-transcript-note-composer-actions">
        <Button
          className="scr-transcript-note-composer-action"
          type="button"
          variant="ghost"
          size="icon"
          aria-label="Cancel note"
          disabled={isSaving}
          onClick={onCancel}
        >
          <X size={15} aria-hidden="true" />
        </Button>
        <Button
          className="scr-transcript-note-composer-action scr-transcript-note-composer-submit"
          type="button"
          variant="ghost"
          size="icon"
          aria-label="Save note"
          aria-busy={isSaving}
          disabled={!canSave}
          onClick={handleSubmit}
        >
          <Send size={16} aria-hidden="true" />
        </Button>
      </div>
    </div>
  );
}
