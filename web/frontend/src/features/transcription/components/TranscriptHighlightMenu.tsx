import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { SelectionMenuRect } from "@/features/transcription/utils/transcriptHighlighting";
import { positionSelectionMenu } from "@/features/transcription/utils/transcriptHighlighting";

type ActiveHighlightMenu = {
  annotationId: string;
  rect: SelectionMenuRect;
};

type TranscriptHighlightMenuProps = {
  activeHighlight: ActiveHighlightMenu | null;
  isDeleting: boolean;
  onDeleteHighlight: (annotationId: string) => void;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
};

const menuWidth = 44;
const menuHeight = 40;
const menuOffset = 8;

export function TranscriptHighlightMenu({
  activeHighlight,
  isDeleting,
  onDeleteHighlight,
  onMouseEnter,
  onMouseLeave,
}: TranscriptHighlightMenuProps) {
  if (!activeHighlight) return null;

  const { left, top } = positionSelectionMenu(activeHighlight.rect, {
    menuWidth,
    menuHeight,
    viewportWidth: window.innerWidth,
    offset: menuOffset,
  });

  return (
    <div
      className="scr-transcript-highlight-menu"
      style={{ left, top }}
      role="toolbar"
      aria-label="Highlight actions"
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
      onMouseDown={(event) => event.preventDefault()}
    >
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            className="scr-transcript-highlight-action"
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Remove highlight"
            aria-busy={isDeleting}
            disabled={isDeleting}
            onClick={() => onDeleteHighlight(activeHighlight.annotationId)}
          >
            <Trash2 size={16} aria-hidden="true" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Remove highlight</TooltipContent>
      </Tooltip>
    </div>
  );
}
