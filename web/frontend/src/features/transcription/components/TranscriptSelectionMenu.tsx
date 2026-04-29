import { Highlighter, MessageSquare } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { TranscriptTextSelection } from "@/features/transcription/hooks/useTranscriptTextSelection";
import { positionSelectionMenu } from "@/features/transcription/utils/transcriptHighlighting";

type TranscriptSelectionMenuProps = {
  selection: TranscriptTextSelection | null;
  isCreatingHighlight: boolean;
  isDuplicateHighlight: boolean;
  onCreateHighlight: () => void;
  onOpenNoteComposer: () => void;
};

const menuWidth = 72;
const menuOffset = 6;
const menuHeight = 34;

export function TranscriptSelectionMenu({
  selection,
  isCreatingHighlight,
  isDuplicateHighlight,
  onCreateHighlight,
  onOpenNoteComposer,
}: TranscriptSelectionMenuProps) {
  if (!selection) return null;

  const { left, top } = positionSelectionMenu(selection.rect, {
    menuWidth,
    menuHeight,
    viewportWidth: window.innerWidth,
    offset: menuOffset,
  });

  return (
    <div
      className="scr-transcript-selection-menu"
      style={{ left, top }}
      role="toolbar"
      aria-label="Transcript selection actions"
      onMouseDown={(event) => event.preventDefault()}
      onTouchStart={(event) => event.preventDefault()}
    >
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            className="scr-transcript-selection-action"
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Highlight selection"
            aria-busy={isCreatingHighlight}
            disabled={isCreatingHighlight || isDuplicateHighlight}
            onClick={onCreateHighlight}
          >
            <Highlighter size={16} aria-hidden="true" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{isDuplicateHighlight ? "Already highlighted" : "Highlight"}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            className="scr-transcript-selection-action"
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Add note"
            onClick={onOpenNoteComposer}
          >
            <MessageSquare size={16} aria-hidden="true" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Note</TooltipContent>
      </Tooltip>
    </div>
  );
}
