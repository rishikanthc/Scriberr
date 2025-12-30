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

    words.forEach((w) => {
        const startChar = textBuilder.length;
        textBuilder += w.word;
        const endChar = textBuilder.length;

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

// O(log N) binary search to find the latest word that has started (startTime <= currentTime)
// Returns the index of the word, or -1 if no word has started yet.
export function findActiveWordIndex(
    offsets: { startTime: number; endTime: number; }[],
    currentTime: number
): number {
    let low = 0;
    let high = offsets.length - 1;
    let result = -1;

    while (low <= high) {
        const mid = Math.floor((low + high) / 2);
        if (offsets[mid].startTime <= currentTime) {
            result = mid; // Candidate found, look effectively later for a tighter match? 
            // Actually, since sorted by startTime, we want the LARGEST startTime <= currentTime.
            low = mid + 1;
        } else {
            high = mid - 1;
        }
    }
    return result;
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

        const activeIndex = findActiveWordIndex(offsets, currentTime);
        const activeWord = activeIndex !== -1 ? offsets[activeIndex] : null;

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
