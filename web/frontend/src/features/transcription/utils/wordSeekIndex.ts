export type WordSeekInput = {
  word: string;
  start: number;
  end: number;
};

export type WordSeekTarget = {
  word: string;
  wordIndex: number;
  startChar: number;
  endChar: number;
  startMs: number;
  endMs: number;
};

export type WordSeekOffsetInput = {
  word: string;
  startChar: number;
  endChar: number;
  startTime: number;
  endTime: number;
};

export type WordOffset = WordSeekOffsetInput;

export type TimedWord = {
  start: number;
  end: number;
};

export type SegmentTimeRange = {
  start: number;
  end: number;
};

export type SegmentWordAssignment<TWord extends TimedWord> = {
  words: TWord[];
  firstWordIndex: number;
};

export type WordSeekIndex = {
  text: string;
  targets: WordSeekTarget[];
};

export function computeWordOffsets(words: { word: string; start: number; end: number }[]) {
  let textBuilder = "";
  const computedOffsets: WordOffset[] = [];

  if (!words) return { fullText: "", offsets: [] };

  words.forEach((w) => {
    const word = w.word.trim();
    if (!word || !Number.isFinite(w.start) || !Number.isFinite(w.end)) return;
    if (textBuilder.length > 0) textBuilder += " ";
    const startChar = textBuilder.length;
    textBuilder += word;
    const endChar = textBuilder.length;

    computedOffsets.push({
      startChar,
      endChar,
      startTime: w.start,
      endTime: w.end,
      word,
    });
  });

  return { fullText: textBuilder, offsets: computedOffsets };
}

export function computeWordOffsetsInText(text: string, words: { word: string; start: number; end: number }[]) {
  const offsets: WordOffset[] = [];
  let searchFrom = 0;

  words.forEach((w) => {
    const word = w.word.trim();
    if (!word || !Number.isFinite(w.start) || !Number.isFinite(w.end)) return;

    const match = findDisplayWordMatch(text, word, searchFrom);
    if (!match) return;

    offsets.push({
      startChar: match.startChar,
      endChar: match.endChar,
      startTime: w.start,
      endTime: w.end,
      word,
    });
    searchFrom = match.endChar;
  });

  return offsets;
}

export function buildWordSeekIndex(text: string, words: WordSeekInput[] = []): WordSeekIndex {
  const targets: WordSeekTarget[] = [];
  const sourceText = text || "";
  const lowerText = sourceText.toLocaleLowerCase();
  let searchFrom = 0;

  words.forEach((candidate, wordIndex) => {
    const word = candidate.word.trim();
    if (!word || !Number.isFinite(candidate.start) || !Number.isFinite(candidate.end) || candidate.end <= candidate.start) return;

    const exactStartChar = sourceText.indexOf(word, searchFrom);
    const foldedStartChar = lowerText.indexOf(word.toLocaleLowerCase(), searchFrom);
    const startChar = earliestMatch(exactStartChar, foldedStartChar);
    if (startChar === -1) return;

    const endChar = startChar + word.length;
    targets.push({
      word,
      wordIndex,
      startChar,
      endChar,
      startMs: Math.round(candidate.start * 1000),
      endMs: Math.round(candidate.end * 1000),
    });
    searchFrom = endChar;
  });

  return { text: sourceText, targets };
}

export function buildWordSeekTargetsFromOffsets(offsets: WordSeekOffsetInput[] = [], wordIndexOffset = 0): WordSeekTarget[] {
  return offsets.flatMap((offset, index) => {
    if (
      !offset.word.trim() ||
      !Number.isFinite(offset.startChar) ||
      !Number.isFinite(offset.endChar) ||
      !Number.isFinite(offset.startTime) ||
      !Number.isFinite(offset.endTime) ||
      offset.endChar <= offset.startChar ||
      offset.endTime <= offset.startTime
    ) {
      return [];
    }

    return [{
      word: offset.word,
      wordIndex: wordIndexOffset + index,
      startChar: offset.startChar,
      endChar: offset.endChar,
      startMs: Math.round(offset.startTime * 1000),
      endMs: Math.round(offset.endTime * 1000),
    }];
  });
}

export function wordBelongsToSegment(word: TimedWord, segmentStart: number, segmentEnd: number) {
  if (!Number.isFinite(word.start) || !Number.isFinite(word.end) || word.end <= word.start) return false;
  return word.start >= segmentStart - 0.1 && word.start < segmentEnd + 0.25 && word.end > segmentStart - 0.05;
}

export function assignWordsToSegments<TWord extends TimedWord>(
  segments: SegmentTimeRange[],
  words: TWord[]
): SegmentWordAssignment<TWord>[] {
  const assignments = segments.map<SegmentWordAssignment<TWord>>(() => ({ words: [], firstWordIndex: -1 }));
  let wordIndex = 0;

  segments.forEach((segment, segmentIndex) => {
    while (wordIndex < words.length && wordEndsBeforeSegment(words[wordIndex], segment.start)) {
      wordIndex += 1;
    }

    while (wordIndex < words.length && wordBelongsToSegment(words[wordIndex], segment.start, segment.end)) {
      if (assignments[segmentIndex].firstWordIndex === -1) {
        assignments[segmentIndex].firstWordIndex = wordIndex;
      }
      assignments[segmentIndex].words.push(words[wordIndex]);
      wordIndex += 1;
    }
  });

  return assignments;
}

export function shouldUseAlignedSegmentText(segmentText: string, wordCount: number, offsetCount: number) {
  if (!segmentText) return false;
  if (wordCount === 0) return true;
  if (offsetCount === 0) return false;
  return offsetCount >= Math.ceil(wordCount * 0.65);
}

function earliestMatch(first: number, second: number) {
  if (first === -1) return second;
  if (second === -1) return first;
  return Math.min(first, second);
}

function wordEndsBeforeSegment(word: TimedWord, segmentStart: number) {
  return Number.isFinite(word.end) && word.end <= segmentStart - 0.05;
}

function findDisplayWordMatch(text: string, word: string, searchFrom: number) {
  const lowerText = text.toLocaleLowerCase();
  const lowerWord = word.toLocaleLowerCase();

  let startChar = text.indexOf(word, searchFrom);
  if (startChar !== -1) return { startChar, endChar: startChar + word.length };

  startChar = lowerText.indexOf(lowerWord, searchFrom);
  if (startChar !== -1) return { startChar, endChar: startChar + word.length };

  return findWhitespaceTolerantWordMatch(text, lowerText, lowerWord, searchFrom);
}

function findWhitespaceTolerantWordMatch(text: string, lowerText: string, lowerWord: string, searchFrom: number) {
  for (let start = Math.max(0, searchFrom); start < text.length; start += 1) {
    let textIndex = start;
    let wordIndex = 0;

    while (textIndex < text.length && wordIndex < lowerWord.length) {
      if (lowerText[textIndex] === lowerWord[wordIndex]) {
        textIndex += 1;
        wordIndex += 1;
        continue;
      }
      if (wordIndex > 0 && isDisplayWhitespace(text[textIndex])) {
        textIndex += 1;
        continue;
      }
      break;
    }

    if (wordIndex === lowerWord.length) {
      return { startChar: start, endChar: textIndex };
    }
  }
  return null;
}

function isDisplayWhitespace(value: string) {
  return /\s/.test(value);
}

export function findWordSeekTarget(targets: WordSeekTarget[], charOffset: number) {
  if (!Number.isFinite(charOffset) || !targets.length) return null;

  let low = 0;
  let high = targets.length - 1;

  while (low <= high) {
    const mid = Math.floor((low + high) / 2);
    const target = targets[mid];

    if (charOffset < target.startChar) {
      high = mid - 1;
      continue;
    }
    if (charOffset > target.endChar) {
      low = mid + 1;
      continue;
    }
    return target;
  }

  return null;
}
