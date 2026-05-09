import { useEffect, useMemo, type RefObject } from "react";
import type { WordOffset } from "@/features/transcription/utils/wordSeekIndex";

export type PlaybackSnapshot = {
    currentTime: number;
    isPlaying: boolean;
};

export type PlaybackSync = {
    getSnapshot: () => PlaybackSnapshot;
    publish: (next: Partial<PlaybackSnapshot>) => void;
    subscribe: (listener: (snapshot: PlaybackSnapshot) => void) => () => void;
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
            clear(): void;
        };
    }
}

type CSSHighlightRegistry = {
    set(name: string, highlight: Highlight): void;
    delete(name: string): void;
    has(name: string): boolean;
};

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
    const highlights = cssHighlights();
    if (highlights?.has(name)) {
        highlights.delete(name);
    }
}

function cssHighlights() {
    return typeof CSS !== "undefined" ? CSS.highlights as unknown as CSSHighlightRegistry | undefined : undefined;
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
        const highlights = cssHighlights();
        if (!enabled || !activeWords.length || !highlights) {
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
            if (!element) {
                clearHighlight(transcriptHighlightName);
                lastActiveIndex = null;
                return;
            }

            try {
                const range = resolveTextRange(element, activeWord.startChar, activeWord.endChar);
                if (!range) {
                    clearHighlight(transcriptHighlightName);
                    lastActiveIndex = null;
                    return;
                }
                highlights.set(transcriptHighlightName, new Highlight(range));
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

function resolveTextRange(root: HTMLElement, startChar: number, endChar: number) {
    if (startChar < 0 || endChar < startChar || endChar > (root.textContent?.length || 0)) return null;

    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    let currentStart = 0;
    let startNode: Text | null = null;
    let startOffset = 0;
    let endNode: Text | null = null;
    let endOffset = 0;
    let node = walker.nextNode();

    while (node) {
        const textNode = node as Text;
        const currentEnd = currentStart + textNode.length;

        if (!startNode && startChar >= currentStart && startChar <= currentEnd) {
            startNode = textNode;
            startOffset = startChar - currentStart;
        }
        if (endChar >= currentStart && endChar <= currentEnd) {
            endNode = textNode;
            endOffset = endChar - currentStart;
            break;
        }

        currentStart = currentEnd;
        node = walker.nextNode();
    }

    if (!startNode || !endNode) return null;
    const range = new Range();
    range.setStart(startNode, startOffset);
    range.setEnd(endNode, endOffset);
    return range;
}
