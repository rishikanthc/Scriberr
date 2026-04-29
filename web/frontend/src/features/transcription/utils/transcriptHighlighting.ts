import type { TranscriptAnnotation, TranscriptAnnotationAnchor } from "../api/annotationsApi";

export type TranscriptHighlightSegment = {
  startChar: number;
  endChar: number;
  text: string;
  wordStartIndex?: number;
  offsets?: Array<{
    startChar: number;
    endChar: number;
  }>;
};

export type SegmentHighlightRange = {
  start: number;
  end: number;
  annotationIds: string[];
};

export type SelectionMenuRect = {
  left: number;
  top: number;
  bottom: number;
  width: number;
};

export type SelectionMenuPositionOptions = {
  menuWidth: number;
  menuHeight: number;
  viewportWidth: number;
  offset: number;
};

export function normalizeHashText(text: string) {
  return text.trim().split(/\s+/).filter(Boolean).join(" ");
}

export function positionSelectionMenu(rect: SelectionMenuRect, options: SelectionMenuPositionOptions) {
  const centerX = rect.left + rect.width / 2;
  const minLeft = options.menuWidth / 2 + options.offset;
  const maxLeft = options.viewportWidth - options.menuWidth / 2 - options.offset;
  const left = Math.min(maxLeft, Math.max(minLeft, centerX));
  const fitsAbove = rect.top >= options.menuHeight + options.offset * 2;
  const top = fitsAbove ? rect.top - options.menuHeight - options.offset : rect.bottom + options.offset;

  const placement = fitsAbove ? "top" : "bottom";
  return { left, top, placement };
}

export function buildHighlightRangesBySegment(
  segments: TranscriptHighlightSegment[],
  annotations: TranscriptAnnotation[]
) {
  const rangesBySegment = new Map<number, SegmentHighlightRange[]>();
  const activeHighlights = annotations
    .filter((annotation) => annotation.kind === "highlight" && annotation.status === "active");

  for (const annotation of activeHighlights) {
    const anchorRange = annotationRangeForSegments(annotation, segments);
    if (!anchorRange) continue;
    const { start: anchorStart, end: anchorEnd } = anchorRange;

    segments.forEach((segment, index) => {
      const start = Math.max(anchorStart, segment.startChar);
      const end = Math.min(anchorEnd, segment.endChar);
      if (end <= start) return;
      const existing = rangesBySegment.get(index) || [];
      existing.push({ start: start - segment.startChar, end: end - segment.startChar, annotationIds: [annotation.id] });
      rangesBySegment.set(index, existing);
    });
  }

  rangesBySegment.forEach((ranges, index) => {
    rangesBySegment.set(index, mergeHighlightRanges(ranges));
  });

  return rangesBySegment;
}

export function hasDuplicateActiveHighlight(
  annotations: TranscriptAnnotation[],
  anchor: TranscriptAnnotationAnchor,
  quote: string
) {
  return annotations.some((annotation) => (
    annotation.kind === "highlight" &&
    annotation.status === "active" &&
    sameHighlightAnchor(annotation, anchor, quote)
  ));
}

function sameHighlightAnchor(annotation: TranscriptAnnotation, anchor: TranscriptAnnotationAnchor, quote: string) {
  if (
    annotation.anchor.start_char !== undefined &&
    annotation.anchor.end_char !== undefined &&
    anchor.start_char !== undefined &&
    anchor.end_char !== undefined
  ) {
    return annotation.anchor.start_char === anchor.start_char && annotation.anchor.end_char === anchor.end_char;
  }
  if (
    annotation.anchor.start_word !== undefined &&
    annotation.anchor.end_word !== undefined &&
    anchor.start_word !== undefined &&
    anchor.end_word !== undefined
  ) {
    return annotation.anchor.start_word === anchor.start_word && annotation.anchor.end_word === anchor.end_word;
  }
  return annotation.anchor.start_ms === anchor.start_ms &&
    annotation.anchor.end_ms === anchor.end_ms &&
    normalizeHashText(annotation.quote) === normalizeHashText(quote);
}

function annotationRangeForSegments(annotation: TranscriptAnnotation, segments: TranscriptHighlightSegment[]) {
  const anchorStart = annotation.anchor.start_char;
  const anchorEnd = annotation.anchor.end_char;
  if (anchorStart !== undefined && anchorEnd !== undefined && anchorEnd > anchorStart) {
    return { start: anchorStart, end: anchorEnd };
  }

  const startWord = annotation.anchor.start_word;
  const endWord = annotation.anchor.end_word;
  if (startWord === undefined || endWord === undefined || endWord < startWord) return null;

  let start: number | undefined;
  let end: number | undefined;

  for (const segment of segments) {
    if (segment.wordStartIndex === undefined || !segment.offsets?.length) continue;
    const localStart = startWord - segment.wordStartIndex;
    const localEnd = endWord - segment.wordStartIndex;

    if (start === undefined && localStart >= 0 && localStart < segment.offsets.length) {
      start = segment.startChar + segment.offsets[localStart].startChar;
    }
    if (localEnd >= 0 && localEnd < segment.offsets.length) {
      end = segment.startChar + segment.offsets[localEnd].endChar;
    }
  }

  if (start === undefined || end === undefined || end <= start) return null;
  return { start, end };
}

export function mergeHighlightRanges(ranges: SegmentHighlightRange[]) {
  const sorted = [...ranges].sort((a, b) => a.start - b.start || b.end - a.end);
  const merged: SegmentHighlightRange[] = [];

  for (const range of sorted) {
    const current = merged[merged.length - 1];
    if (!current || range.start > current.end) {
      merged.push({ ...range, annotationIds: [...range.annotationIds] });
      continue;
    }
    current.end = Math.max(current.end, range.end);
    current.annotationIds = Array.from(new Set([...current.annotationIds, ...range.annotationIds]));
  }

  return merged;
}
