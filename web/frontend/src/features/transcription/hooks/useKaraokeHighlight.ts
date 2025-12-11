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

        const activeWord = offsets.find(
            (w) => currentTime >= w.startTime && currentTime <= w.endTime
        );

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
