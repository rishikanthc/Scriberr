import { useEffect, useMemo, type RefObject } from "react";

export type PlaybackSnapshot = {
    currentTime: number;
    isPlaying: boolean;
};

export type PlaybackSync = {
    getSnapshot: () => PlaybackSnapshot;
    publish: (next: Partial<PlaybackSnapshot>) => void;
    subscribe: (listener: (snapshot: PlaybackSnapshot) => void) => () => void;
};

export type WordOffset = {
    startChar: number;
    endChar: number;
    startTime: number;
    endTime: number;
    word: string;
};

export type KaraokeHighlightSegment = {
    text: string;
    offsets: WordOffset[];
};

const transcriptHighlightName = "scriberr-karaoke-word";
const wordEndGraceSeconds = 0.12;

declare global {
    interface CSS {
        highlights?: {
            set(name: string, highlight: Highlight): void;
            clear(): void;
            delete(name: string): void;
            has(name: string): boolean;
        };
    }
}

export function createPlaybackSync(): PlaybackSync {
    let snapshot: PlaybackSnapshot = { currentTime: 0, isPlaying: false };
    const listeners = new Set<(next: PlaybackSnapshot) => void>();

    return {
        getSnapshot: () => snapshot,
        publish: (next) => {
            snapshot = { ...snapshot, ...next };
            listeners.forEach((listener) => listener(snapshot));
        },
        subscribe: (listener) => {
            listeners.add(listener);
            return () => listeners.delete(listener);
        },
    };
}

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
    const lowerText = text.toLocaleLowerCase();
    let searchFrom = 0;

    words.forEach((w) => {
        const word = w.word.trim();
        if (!word || !Number.isFinite(w.start) || !Number.isFinite(w.end)) return;

        let startChar = text.indexOf(word, searchFrom);
        if (startChar === -1) {
            startChar = lowerText.indexOf(word.toLocaleLowerCase(), searchFrom);
        }
        if (startChar === -1) return;

        const endChar = startChar + word.length;
        offsets.push({
            startChar,
            endChar,
            startTime: w.start,
            endTime: w.end,
            word,
        });
        searchFrom = endChar;
    });

    return offsets;
}

function findActiveWordIndex(
    offsets: { startTime: number; endTime: number }[],
    currentTime: number
): number {
    let low = 0;
    let high = offsets.length - 1;
    let result = -1;

    while (low <= high) {
        const mid = Math.floor((low + high) / 2);
        if (offsets[mid].startTime <= currentTime) {
            result = mid;
            low = mid + 1;
        } else {
            high = mid - 1;
        }
    }
    return result;
}

function findCurrentWordIndex(offsets: WordOffset[], currentTime: number) {
    const activeIndex = findActiveWordIndex(offsets, currentTime);
    if (activeIndex === -1) return -1;
    const activeWord = offsets[activeIndex];
    if (currentTime > activeWord.endTime + wordEndGraceSeconds) return -1;
    return activeIndex;
}

function clearHighlight(name: string) {
    if (typeof CSS !== "undefined" && CSS.highlights?.has(name)) {
        CSS.highlights.delete(name);
    }
}

export function useTranscriptKaraokeHighlight(
    playbackSync: PlaybackSync,
    segments: KaraokeHighlightSegment[],
    textElementsRef: RefObject<(HTMLElement | null)[]>,
    enabled: boolean
) {
    const activeWords = useMemo(() => {
        return segments.flatMap((segment, segmentIndex) => (
            segment.offsets.map((offset) => ({ ...offset, segmentIndex }))
        ));
    }, [segments]);

    useEffect(() => {
        if (!enabled || !activeWords.length || typeof CSS === "undefined" || !CSS.highlights) {
            clearHighlight(transcriptHighlightName);
            return;
        }

        let lastActiveIndex: number | null = null;

        const paintHighlight = (snapshot: PlaybackSnapshot) => {
            if (!snapshot.isPlaying) {
                if (lastActiveIndex !== null) clearHighlight(transcriptHighlightName);
                lastActiveIndex = null;
                return;
            }

            const activeIndex = findCurrentWordIndex(activeWords, snapshot.currentTime);
            if (activeIndex === -1) {
                if (lastActiveIndex !== null) clearHighlight(transcriptHighlightName);
                lastActiveIndex = null;
                return;
            }
            if (activeIndex === lastActiveIndex) return;

            const activeWord = activeWords[activeIndex];
            const element = textElementsRef.current?.[activeWord.segmentIndex];
            const textNode = element?.firstChild;
            if (!textNode || textNode.nodeType !== Node.TEXT_NODE) {
                clearHighlight(transcriptHighlightName);
                lastActiveIndex = null;
                return;
            }

            try {
                const textLength = (textNode as Text).length;
                if (activeWord.endChar > textLength) {
                    clearHighlight(transcriptHighlightName);
                    lastActiveIndex = null;
                    return;
                }
                const range = new Range();
                range.setStart(textNode, activeWord.startChar);
                range.setEnd(textNode, activeWord.endChar);
                CSS.highlights.set(transcriptHighlightName, new Highlight(range));
                lastActiveIndex = activeIndex;
            } catch {
                clearHighlight(transcriptHighlightName);
                lastActiveIndex = null;
            }
        };

        paintHighlight(playbackSync.getSnapshot());
        return playbackSync.subscribe(paintHighlight);
    }, [activeWords, enabled, playbackSync, textElementsRef]);

    useEffect(() => () => clearHighlight(transcriptHighlightName), []);
}
