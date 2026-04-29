import * as assert from "node:assert/strict";
import {
  buildHighlightRangesBySegment,
  hasDuplicateActiveHighlight,
  mergeHighlightRanges,
  normalizeHashText,
  positionSelectionMenu,
} from "./transcriptHighlighting";
import type { TranscriptAnnotation } from "../api/annotationsApi";

function annotation(start_char: number, end_char: number, status: TranscriptAnnotation["status"] = "active"): TranscriptAnnotation {
  return {
    id: `ann_${start_char}_${end_char}`,
    transcription_id: "tr_test",
    kind: "highlight",
    content: null,
    color: null,
    quote: "quote",
    anchor: {
      start_ms: 1000,
      end_ms: 2000,
      start_char,
      end_char,
    },
    status,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  };
}

function wordAnnotation(start_word: number, end_word: number): TranscriptAnnotation {
  return {
    ...annotation(0, 0),
    id: `ann_word_${start_word}_${end_word}`,
    anchor: {
      start_ms: 1000,
      end_ms: 2000,
      start_word,
      end_word,
    },
  };
}

assert.equal(normalizeHashText("  hello\n\nworld\tagain  "), "hello world again");

assert.deepEqual(
  mergeHighlightRanges([
    { start: 6, end: 12, annotationIds: ["b"] },
    { start: 0, end: 4, annotationIds: ["a"] },
    { start: 3, end: 8, annotationIds: ["c"] },
  ]),
  [{ start: 0, end: 12, annotationIds: ["a", "c", "b"] }]
);

const ranges = buildHighlightRangesBySegment(
  [
    { startChar: 0, endChar: 12, text: "hello world." },
    { startChar: 13, endChar: 28, text: "second segment" },
  ],
  [
    annotation(6, 20),
    annotation(1, 4, "stale"),
    { ...annotation(0, 5), kind: "note" },
  ]
);

assert.deepEqual(ranges.get(0), [{ start: 6, end: 12, annotationIds: ["ann_6_20"] }]);
assert.deepEqual(ranges.get(1), [{ start: 0, end: 7, annotationIds: ["ann_6_20"] }]);
assert.equal(ranges.has(2), false);

const wordRanges = buildHighlightRangesBySegment(
  [
    {
      startChar: 0,
      endChar: 11,
      text: "hello world",
      wordStartIndex: 0,
      offsets: [
        { startChar: 0, endChar: 5 },
        { startChar: 6, endChar: 11 },
      ],
    },
    {
      startChar: 12,
      endChar: 26,
      text: "second segment",
      wordStartIndex: 2,
      offsets: [
        { startChar: 0, endChar: 6 },
        { startChar: 7, endChar: 14 },
      ],
    },
  ],
  [wordAnnotation(1, 2)]
);

assert.deepEqual(wordRanges.get(0), [{ start: 6, end: 11, annotationIds: ["ann_word_1_2"] }]);
assert.deepEqual(wordRanges.get(1), [{ start: 0, end: 6, annotationIds: ["ann_word_1_2"] }]);

assert.equal(
  hasDuplicateActiveHighlight([annotation(6, 20)], {
    start_ms: 1000,
    end_ms: 2000,
    start_char: 6,
    end_char: 20,
  }, "quote"),
  true
);

assert.equal(
  hasDuplicateActiveHighlight([wordAnnotation(1, 2)], {
    start_ms: 1000,
    end_ms: 2000,
    start_word: 1,
    end_word: 2,
  }, "quote"),
  true
);

assert.equal(
  hasDuplicateActiveHighlight([annotation(6, 20, "stale")], {
    start_ms: 1000,
    end_ms: 2000,
    start_char: 6,
    end_char: 20,
  }, "quote"),
  false
);

assert.deepEqual(
  positionSelectionMenu(
    { left: 20, top: 120, bottom: 144, width: 20 },
    { menuWidth: 88, menuHeight: 40, viewportWidth: 320, offset: 8 }
  ),
  { left: 52, top: 72, placement: "top" }
);

assert.deepEqual(
  positionSelectionMenu(
    { left: 260, top: 20, bottom: 44, width: 120 },
    { menuWidth: 88, menuHeight: 40, viewportWidth: 320, offset: 8 }
  ),
  { left: 268, top: 52, placement: "bottom" }
);

console.info("transcript highlighting regression checks passed");
