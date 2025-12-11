import { useEffect, useMemo } from 'react';


// Extend the existing CSS interface for the experimental Highlight API
declare global {
    // Use interface augmentation for the CSS namespace
    // This merges with the existing CSS namespace/interface
    interface CSS {
        highlights?: {
            set(name: string, highlight: Highlight): void;
            clear(): void;
            delete(name: string): void;
            has(name: string): boolean;
        }
    }
}

// Helper to calculate offsets (exported for external use)
export function computeWordOffsets(words: { word: string; start: number; end: number }[]) {
    let textBuilder = '';
    const computedOffsets: {
        startChar: number;
        endChar: number;
        startTime: number;
        endTime: number;
        word: string;
    }[] = [];

    if (!words) return { fullText: '', offsets: [] };

    words.forEach((w, index) => {
        const startChar = textBuilder.length;
        textBuilder += w.word;
        const endChar = textBuilder.length;

        // Gap Filling Logic:
        // Extend the previous word's endTime to meet this word's startTime
        // if the gap is small (e.g. natural pauses).
        // This prevents flickering/skipping when the playback update rate is lower than the gap size.
        if (index > 0) {
            const prev = computedOffsets[index - 1];
            const gap = w.start - prev.endTime;
            // Fill gaps smaller than 0.7 seconds
            if (gap > 0 && gap < 0.7) {
                prev.endTime = w.start;
            }
        }

        computedOffsets.push({
            startChar,
            endChar,
            startTime: w.start,
            endTime: w.end,
            word: w.word
        });

        textBuilder += ' ';
    });

    return { fullText: textBuilder, offsets: computedOffsets };
}

export function useKaraokeHighlight(
    containerRef: React.RefObject<HTMLDivElement | null>,
    words: { word: string; start: number; end: number }[],
    currentTime: number,
    isPlaying: boolean
) {
    // 1. Pre-calculate the full text string and character offsets
    const { fullText, offsets } = useMemo(() => {
        return computeWordOffsets(words);
    }, [words]);

    // 2. The Sync Logic
    useEffect(() => {
        if (!containerRef.current || typeof CSS === 'undefined' || !CSS.highlights) return;

        // Binary search for performance (O(logN) vs O(N))
        // particularly important for long transcripts
        let activeWord = null;
        let low = 0;
        let high = offsets.length - 1;

        while (low <= high) {
            const mid = Math.floor((low + high) / 2);
            if (offsets[mid].startTime <= currentTime) {
                // This word started before or at currentTime.
                // It's a candidate, but there might be a later one that also started before currentTime.
                // (Though with non-overlapping words, this is usually unique, but let's be safe).
                // Actually, if we find one, we check if it contains currentTime.
                // If offsets are sorted and non-overlapping (mostly), we can optimize.
                const w = offsets[mid];
                if (currentTime <= w.endTime) {
                    activeWord = w;
                    break;
                }
                // If we are here, w started before currentTime but ended before currentTime.
                // So we need to look later.
                low = mid + 1;
            } else {
                // w started after currentTime. Look earlier.
                high = mid - 1;
            }
        }

        if (!activeWord) {
            if (CSS.highlights.has('karaoke-word')) {
                CSS.highlights.delete('karaoke-word');
            }
            return;
        }

        const textNode = containerRef.current.firstChild;

        if (!textNode || textNode.nodeType !== Node.TEXT_NODE) return;

        try {
            const range = new Range();
            if (activeWord.endChar <= (textNode as Text).length) {
                range.setStart(textNode, activeWord.startChar);
                range.setEnd(textNode, activeWord.endChar);

                const highlight = new Highlight(range);
                CSS.highlights.set('karaoke-word', highlight);
            }
        } catch (e) {
            console.warn("Highlight range error:", e);
        }

    }, [currentTime, isPlaying, offsets, containerRef]);

    return { fullText, offsets };
}
