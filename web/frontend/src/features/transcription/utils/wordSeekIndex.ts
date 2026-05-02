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

export type WordSeekIndex = {
  text: string;
  targets: WordSeekTarget[];
};

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

function earliestMatch(first: number, second: number) {
  if (first === -1) return second;
  if (second === -1) return first;
  return Math.min(first, second);
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
