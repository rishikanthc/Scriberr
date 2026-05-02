import * as assert from "node:assert/strict";
import { buildWordSeekIndex, buildWordSeekTargetsFromOffsets, findWordSeekTarget } from "./wordSeekIndex";

const index = buildWordSeekIndex("Hello, hello world.", [
  { word: "hello", start: 0.1, end: 0.4 },
  { word: "hello", start: 0.5, end: 0.8 },
  { word: "world", start: 0.9, end: 1.2 },
]);

assert.deepEqual(index.targets, [
  { word: "hello", wordIndex: 0, startChar: 0, endChar: 5, startMs: 100, endMs: 400 },
  { word: "hello", wordIndex: 1, startChar: 7, endChar: 12, startMs: 500, endMs: 800 },
  { word: "world", wordIndex: 2, startChar: 13, endChar: 18, startMs: 900, endMs: 1200 },
]);

assert.equal(findWordSeekTarget(index.targets, 0)?.startMs, 100);
assert.equal(findWordSeekTarget(index.targets, 5)?.startMs, 100);
assert.equal(findWordSeekTarget(index.targets, 6), null);
assert.equal(findWordSeekTarget(index.targets, 7)?.startMs, 500);
assert.equal(findWordSeekTarget(index.targets, 18)?.startMs, 900);
assert.equal(findWordSeekTarget(index.targets, 19), null);

const punctuationIndex = buildWordSeekIndex("Well... this works?", [
  { word: "Well", start: 1, end: 1.2 },
  { word: "this", start: 1.3, end: 1.6 },
  { word: "works", start: 1.7, end: 2 },
]);

assert.equal(findWordSeekTarget(punctuationIndex.targets, 1)?.word, "Well");
assert.equal(findWordSeekTarget(punctuationIndex.targets, 8)?.word, "this");
assert.equal(findWordSeekTarget(punctuationIndex.targets, 17)?.word, "works");

const missingWordsIndex = buildWordSeekIndex("Only matched words remain.", [
  { word: "Only", start: 0, end: 0.2 },
  { word: "missing", start: 0.3, end: 0.4 },
  { word: "words", start: 0.5, end: 0.7 },
  { word: "remain", start: Number.NaN, end: 1 },
]);

assert.deepEqual(
  missingWordsIndex.targets.map((target) => target.word),
  ["Only", "words"]
);

const emptyIndex = buildWordSeekIndex("No usable timings.", [
  { word: "", start: 0, end: 1 },
  { word: "No", start: 0, end: Number.NaN },
  { word: "usable", start: 2, end: 2 },
]);

assert.deepEqual(emptyIndex.targets, []);
assert.equal(findWordSeekTarget(emptyIndex.targets, 0), null);

assert.deepEqual(
  buildWordSeekTargetsFromOffsets([
    { word: "alpha", startChar: 0, endChar: 5, startTime: 2, endTime: 2.2 },
    { word: "bad", startChar: 6, endChar: 9, startTime: 3, endTime: 3 },
    { word: "beta", startChar: 10, endChar: 14, startTime: 3.1, endTime: 3.4 },
  ], 12),
  [
    { word: "alpha", wordIndex: 12, startChar: 0, endChar: 5, startMs: 2000, endMs: 2200 },
    { word: "beta", wordIndex: 14, startChar: 10, endChar: 14, startMs: 3100, endMs: 3400 },
  ]
);

console.info("word seek index regression checks passed");
