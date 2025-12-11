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

export function useKaraokeHighlight(
    containerRef: React.RefObject<HTMLDivElement | null>,
    words: { word: string; start: number; end: number }[],
    currentTime: number,
    isPlaying: boolean
) {
    // 1. Pre-calculate the full text string and character offsets
    // This runs ONCE when words change, not on every tick.
    const { fullText, offsets } = useMemo(() => {
        let textBuilder = '';
        const computedOffsets: {
            startChar: number;
            endChar: number;
            startTime: number;
            endTime: number;
        }[] = [];

        if (!words) return { fullText: '', offsets: [] };

        words.forEach((w) => {
            const startChar = textBuilder.length;
            textBuilder += w.word; // Add the word
            const endChar = textBuilder.length;

            computedOffsets.push({
                startChar,
                endChar,
                startTime: w.start,
                endTime: w.end
            });

            textBuilder += ' '; // Add space for readability
        });

        return { fullText: textBuilder, offsets: computedOffsets };
    }, [words]);

    // 2. The Sync Logic (Runs on every time update)
    useEffect(() => {
        if (!containerRef.current || typeof CSS === 'undefined' || !CSS.highlights) return;

        // A. Find the active word
        // Optimization: For very long transcripts, we could store the last index in a Ref
        // and start searching from there. For now, simple find is usually fast enough given 
        // the array size < 1 hour of speech (~10k words).
        const activeWord = offsets.find(
            (w) => currentTime >= w.startTime && currentTime <= w.endTime
        );

        // B. If no word is spoken right now or not playing, clear highlights
        if (!activeWord) {
            // Only delete if it exists to avoid unnecessary work (though delete is cheap)
            if (CSS.highlights.has('karaoke-word')) {
                CSS.highlights.delete('karaoke-word');
            }
            return;
        }

        // C. Locate the DOM Text Node
        // Note: We assume the container has ONE text node child for Compact view.
        // If the structure is more complex, a TreeWalker would be needed.
        const textNode = containerRef.current.firstChild;

        if (!textNode || textNode.nodeType !== Node.TEXT_NODE) return;

        try {
            // D. Create the Range
            const range = new Range();
            // We verify offsets to prevent "IndexSizeError" if DOM text content doesn't match
            if (activeWord.endChar <= (textNode as Text).length) {
                range.setStart(textNode, activeWord.startChar);
                range.setEnd(textNode, activeWord.endChar);

                // E. Register the Highlight
                // We create a new Highlight object each time.
                // The Highlight API is designed for this high-frequency update.
                const highlight = new Highlight(range);
                CSS.highlights.set('karaoke-word', highlight);
            }
        } catch (e) {
            console.warn("Highlight range error:", e);
        }

    }, [currentTime, isPlaying, offsets, containerRef]);

    return fullText;
}
