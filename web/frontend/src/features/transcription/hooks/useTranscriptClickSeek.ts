import { useEffect, useMemo, useRef, type RefObject } from "react";
import { resolveTranscriptCaretHit } from "@/features/transcription/utils/caretHitTesting";
import { findWordSeekTarget, type WordSeekTarget } from "@/features/transcription/utils/wordSeekIndex";

export type TranscriptClickSeekSegment = {
  index: number;
  targets: WordSeekTarget[];
};

type PointerStart = {
  clientX: number;
  clientY: number;
};

const maxClickDriftPx = 6;

export function useTranscriptClickSeek(
  containerRef: RefObject<HTMLElement | null>,
  segments: TranscriptClickSeekSegment[],
  enabled: boolean,
  onSeekRequest: (seconds: number) => void
) {
  const pointerStartRef = useRef<PointerStart | null>(null);
  const segmentByIndex = useMemo(() => new Map(segments.map((segment) => [segment.index, segment])), [segments]);

  useEffect(() => {
    const root = containerRef.current;
    if (!root || !enabled) return;

    const handlePointerDown = (event: PointerEvent) => {
      if (event.button !== 0) {
        pointerStartRef.current = null;
        return;
      }
      pointerStartRef.current = {
        clientX: event.clientX,
        clientY: event.clientY,
      };
    };

    const handleClick = (event: MouseEvent) => {
      if (event.defaultPrevented || event.button !== 0 || hasActiveTranscriptSelection(root)) return;
      if (movedTooFar(pointerStartRef.current, event)) return;

      const hit = resolveTranscriptCaretHit(root, {
        clientX: event.clientX,
        clientY: event.clientY,
        target: event.target,
      });
      if (!hit) return;

      const segment = segmentByIndex.get(hit.segmentIndex);
      if (!segment) return;

      const target = findWordSeekTarget(segment.targets, hit.textOffset);
      if (!target) return;

      onSeekRequest(target.startMs / 1000);
    };

    root.addEventListener("pointerdown", handlePointerDown);
    root.addEventListener("click", handleClick);

    return () => {
      root.removeEventListener("pointerdown", handlePointerDown);
      root.removeEventListener("click", handleClick);
    };
  }, [containerRef, enabled, onSeekRequest, segmentByIndex]);
}

function movedTooFar(start: PointerStart | null, event: MouseEvent) {
  if (!start) return false;
  return Math.abs(start.clientX - event.clientX) > maxClickDriftPx || Math.abs(start.clientY - event.clientY) > maxClickDriftPx;
}

function hasActiveTranscriptSelection(root: HTMLElement) {
  const selection = root.ownerDocument.getSelection();
  if (!selection || selection.isCollapsed) return false;
  return root.contains(selection.anchorNode) || root.contains(selection.focusNode);
}
