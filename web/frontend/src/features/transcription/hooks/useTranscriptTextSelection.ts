import { useCallback, useEffect, useState, type RefObject } from "react";
import type { TranscriptAnnotationAnchor } from "@/features/transcription/api/annotationsApi";
import type { WordOffset } from "@/features/transcription/hooks/useKaraokeHighlight";
import { normalizeHashText } from "@/features/transcription/utils/transcriptHighlighting";

export type TranscriptSelectionSegment = {
  index: number;
  start: number;
  end: number;
  startChar: number;
  endChar: number;
  charAnchorReliable: boolean;
  wordStartIndex: number;
  offsets: WordOffset[];
};

export type TranscriptTextSelection = {
  quote: string;
  anchor: TranscriptAnnotationAnchor;
  rect: DOMRect;
};

export function useTranscriptTextSelection(
  containerRef: RefObject<HTMLElement | null>,
  segments: TranscriptSelectionSegment[],
  enabled: boolean
) {
  const [selection, setSelection] = useState<TranscriptTextSelection | null>(null);

  const clearSelection = useCallback(() => {
    setSelection(null);
    window.getSelection()?.removeAllRanges();
  }, []);

  useEffect(() => {
    if (!enabled) {
      setSelection(null);
      return;
    }

    let selectionToken = 0;
    let readTimer: number | undefined;
    const segmentByIndex = new Map(segments.map((segment) => [segment.index, segment]));

    const readSelection = () => {
      const root = containerRef.current;
      const browserSelection = window.getSelection();
      if (!root || !browserSelection || browserSelection.isCollapsed || browserSelection.rangeCount === 0) {
        setSelection(null);
        return;
      }
      if (!root.contains(browserSelection.anchorNode) || !root.contains(browserSelection.focusNode)) {
        setSelection(null);
        return;
      }

      const range = browserSelection.getRangeAt(0);
      const parsed = parseTranscriptRange(root, range, segmentByIndex);
      const token = ++selectionToken;
      if (!parsed) {
        setSelection(null);
        return;
      }

      void buildTextHash(parsed.quote).then((text_hash) => {
        if (token !== selectionToken) return;
        if (!text_hash) {
          setSelection(null);
          return;
        }
        setSelection({
          quote: parsed.quote,
          anchor: { ...parsed.anchor, text_hash },
          rect: parsed.rect,
        });
      });
    };

    const scheduleReadSelection = () => {
      if (readTimer) window.clearTimeout(readTimer);
      readTimer = window.setTimeout(readSelection, 0);
    };

    const handleSelectionChange = () => {
      const root = containerRef.current;
      const browserSelection = window.getSelection();
      if (!root || !browserSelection || browserSelection.isCollapsed) {
        setSelection(null);
        return;
      }
      if (!root.contains(browserSelection.anchorNode) || !root.contains(browserSelection.focusNode)) {
        setSelection(null);
      }
    };
    const handleScroll = () => setSelection(null);
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") clearSelection();
    };
    const root = containerRef.current;

    root?.addEventListener("mouseup", scheduleReadSelection);
    root?.addEventListener("touchend", scheduleReadSelection);
    document.addEventListener("keyup", scheduleReadSelection);
    document.addEventListener("keydown", handleKeyDown);
    document.addEventListener("selectionchange", handleSelectionChange);
    window.addEventListener("scroll", handleScroll, true);

    return () => {
      selectionToken += 1;
      if (readTimer) window.clearTimeout(readTimer);
      root?.removeEventListener("mouseup", scheduleReadSelection);
      root?.removeEventListener("touchend", scheduleReadSelection);
      document.removeEventListener("keyup", scheduleReadSelection);
      document.removeEventListener("keydown", handleKeyDown);
      document.removeEventListener("selectionchange", handleSelectionChange);
      window.removeEventListener("scroll", handleScroll, true);
    };
  }, [clearSelection, containerRef, enabled, segments]);

  return { selection, clearSelection };
}

type TranscriptTextPiece = {
  node: Text;
  textElement: HTMLElement;
  segment: TranscriptSelectionSegment;
  selectedText: string;
  textNodeStartOffset: number;
  textNodeEndOffset: number;
  startOffset: number;
  endOffset: number;
  globalStartChar: number;
  globalEndChar: number;
};

type ParsedTranscriptRange = {
  quote: string;
  anchor: Omit<TranscriptAnnotationAnchor, "text_hash">;
  rect: DOMRect;
};

function parseTranscriptRange(
  root: HTMLElement,
  range: Range,
  segmentByIndex: Map<number, TranscriptSelectionSegment>
): ParsedTranscriptRange | null {
  if (!rangeStartsAndEndsInTranscriptText(root, range)) return null;

  const pieces = collectSelectedTranscriptText(root, range, segmentByIndex);
  if (!pieces.length) return null;

  const quote = buildQuote(pieces);
  if (!quote) return null;

  const first = pieces[0];
  const last = pieces[pieces.length - 1];
  const start_ms = resolveStartMs(first);
  const end_ms = resolveEndMs(last);
  if (!Number.isFinite(start_ms) || !Number.isFinite(end_ms) || end_ms <= start_ms) return null;

  const rect = transcriptSelectionRect(pieces) || range.getBoundingClientRect();
  if (rect.width <= 0 || rect.height <= 0) return null;

  const anchor: Omit<TranscriptAnnotationAnchor, "text_hash"> = {
    start_ms: Math.round(start_ms * 1000),
    end_ms: Math.round(end_ms * 1000),
  };

  if (pieces.every((piece) => piece.segment.charAnchorReliable)) {
    anchor.start_char = first.globalStartChar;
    anchor.end_char = last.globalEndChar;
  }

  const startWord = resolveStartWord(first);
  const endWord = resolveEndWord(last);
  if (startWord !== undefined && endWord !== undefined && endWord >= startWord) {
    anchor.start_word = startWord;
    anchor.end_word = endWord;
  }

  return { quote, anchor, rect };
}

function rangeStartsAndEndsInTranscriptText(root: HTMLElement, range: Range) {
  return Boolean(
    root.contains(range.startContainer) &&
    root.contains(range.endContainer) &&
    transcriptTextElementForBoundary(range.startContainer) &&
    transcriptTextElementForBoundary(range.endContainer)
  );
}

function collectSelectedTranscriptText(
  root: HTMLElement,
  range: Range,
  segmentByIndex: Map<number, TranscriptSelectionSegment>
) {
  const pieces: TranscriptTextPiece[] = [];
  const scope = rangeTreeScope(root, range);
  const walker = document.createTreeWalker(scope, NodeFilter.SHOW_TEXT, {
    acceptNode: (node) => range.intersectsNode(node) ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_REJECT,
  });

  let node = walker.nextNode();
  while (node) {
    const textNode = node as Text;
    const textElement = transcriptTextElementForNode(textNode);
    const segmentIndex = textElement ? Number(textElement.dataset.transcriptSegmentIndex) : NaN;
    const segment = Number.isInteger(segmentIndex) ? segmentByIndex.get(segmentIndex) : undefined;
    if (!textElement) {
      node = walker.nextNode();
      continue;
    }
    if (!segment) return [];

    const startOffset = textNode === range.startContainer ? range.startOffset : 0;
    const endOffset = textNode === range.endContainer ? range.endOffset : textNode.length;
    const selectedText = textNode.data.slice(startOffset, endOffset);
    if (selectedText.trim()) {
      const nodeStartOffset = textNodeOffsetInside(textElement, textNode);
      const localStartOffset = nodeStartOffset + startOffset;
      const localEndOffset = nodeStartOffset + endOffset;
      pieces.push({
        node: textNode,
        textElement,
        segment,
        selectedText,
        textNodeStartOffset: startOffset,
        textNodeEndOffset: endOffset,
        startOffset: localStartOffset,
        endOffset: localEndOffset,
        globalStartChar: segment.startChar + localStartOffset,
        globalEndChar: segment.startChar + localEndOffset,
      });
    }
    node = walker.nextNode();
  }

  return pieces;
}

function rangeTreeScope(root: HTMLElement, range: Range) {
  const commonAncestor = range.commonAncestorContainer;
  if (commonAncestor.nodeType === Node.TEXT_NODE) return commonAncestor.parentElement || root;
  if (commonAncestor instanceof HTMLElement && root.contains(commonAncestor)) return commonAncestor;
  return root;
}

function transcriptTextElementForNode(node: Node) {
  return node.parentElement?.closest<HTMLElement>("[data-transcript-text]") || null;
}

function transcriptTextElementForBoundary(node: Node) {
  if (node.nodeType === Node.TEXT_NODE) return transcriptTextElementForNode(node);
  if (node instanceof HTMLElement) return node.closest<HTMLElement>("[data-transcript-text]");
  return node.parentElement?.closest<HTMLElement>("[data-transcript-text]") || null;
}

function textNodeOffsetInside(root: HTMLElement, target: Text) {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  let offset = 0;
  let node = walker.nextNode();

  while (node) {
    if (node === target) return offset;
    offset += (node as Text).length;
    node = walker.nextNode();
  }

  return 0;
}

function transcriptSelectionRect(pieces: TranscriptTextPiece[]) {
  const rects: DOMRect[] = [];

  for (const piece of pieces) {
    const range = document.createRange();
    range.setStart(piece.node, piece.textNodeStartOffset);
    range.setEnd(piece.node, piece.textNodeEndOffset);
    rects.push(...Array.from(range.getClientRects()).filter((rect) => rect.width > 0 && rect.height > 0));
    range.detach();
  }

  return unionRects(rects);
}

function unionRects(rects: DOMRect[]) {
  if (!rects.length) return null;

  let left = rects[0].left;
  let top = rects[0].top;
  let right = rects[0].right;
  let bottom = rects[0].bottom;

  for (const rect of rects.slice(1)) {
    left = Math.min(left, rect.left);
    top = Math.min(top, rect.top);
    right = Math.max(right, rect.right);
    bottom = Math.max(bottom, rect.bottom);
  }

  return DOMRect.fromRect({
    x: left,
    y: top,
    width: right - left,
    height: bottom - top,
  });
}

function buildQuote(pieces: TranscriptTextPiece[]) {
  let quote = "";
  let previousTextElement: HTMLElement | null = null;

  for (const piece of pieces) {
    if (previousTextElement && previousTextElement !== piece.textElement && quote && !/\s$/.test(quote)) {
      quote += " ";
    }
    quote += piece.selectedText;
    previousTextElement = piece.textElement;
  }

  return quote.trim();
}

function resolveStartMs(piece: TranscriptTextPiece) {
  const word = wordAtOrAfter(piece.segment.offsets, piece.startOffset);
  return word?.startTime ?? piece.segment.start;
}

function resolveEndMs(piece: TranscriptTextPiece) {
  const word = wordAtOrBefore(piece.segment.offsets, piece.endOffset);
  return word?.endTime ?? piece.segment.end;
}

function resolveStartWord(piece: TranscriptTextPiece) {
  const index = wordIndexAtOrAfter(piece.segment.offsets, piece.startOffset);
  return index === -1 ? undefined : piece.segment.wordStartIndex + index;
}

function resolveEndWord(piece: TranscriptTextPiece) {
  const index = wordIndexAtOrBefore(piece.segment.offsets, piece.endOffset);
  return index === -1 ? undefined : piece.segment.wordStartIndex + index;
}

function wordAtOrAfter(offsets: WordOffset[], charOffset: number) {
  const index = wordIndexAtOrAfter(offsets, charOffset);
  return index === -1 ? undefined : offsets[index];
}

function wordAtOrBefore(offsets: WordOffset[], charOffset: number) {
  const index = wordIndexAtOrBefore(offsets, charOffset);
  return index === -1 ? undefined : offsets[index];
}

function wordIndexAtOrAfter(offsets: WordOffset[], charOffset: number) {
  return offsets.findIndex((offset) => charOffset <= offset.endChar);
}

function wordIndexAtOrBefore(offsets: WordOffset[], charOffset: number) {
  for (let index = offsets.length - 1; index >= 0; index -= 1) {
    if (charOffset >= offsets[index].startChar) return index;
  }
  return -1;
}

async function buildTextHash(text: string) {
  if (!crypto.subtle) return null;
  try {
    const bytes = new TextEncoder().encode(normalizeHashText(text));
    const hashBuffer = await crypto.subtle.digest("SHA-256", bytes);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return `sha256:${hashArray.map((byte) => byte.toString(16).padStart(2, "0")).join("")}`;
  } catch {
    return null;
  }
}
