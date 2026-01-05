import { useState, useEffect, useCallback } from 'react';

// Data structure expected for the word map
type WordOffset = {
    startChar: number;
    endChar: number;
    startTime: number;
    endTime: number;
};

export interface SelectionMenuState {
    visible: boolean;
    x: number;
    y: number;
    startTime: number;
    endTime: number;
    startIdx: number;
    endIdx: number;
    selectedText: string;
}

export function useSelectionMenu(
    containerRef: React.RefObject<HTMLElement | null>,
    wordMap: WordOffset[]
) {
    const [menuState, setMenuState] = useState<SelectionMenuState | null>(null);
    const [showEditor, setShowEditor] = useState(false);

    // Dismiss the menu programmatically
    const dismissMenu = useCallback(() => {
        setMenuState(null);
    }, []);

    // Open the note editor
    const openEditor = useCallback(() => {
        setShowEditor(true);
        // Keep menu state for quote/time data but hide the menu visually
    }, []);

    // Close the note editor
    const closeEditor = useCallback(() => {
        setShowEditor(false);
        setMenuState(null);
        window.getSelection()?.removeAllRanges();
    }, []);

    useEffect(() => {
        const handleSelectionChange = () => {
            // Don't update menu state while editor is open
            if (showEditor) return;

            const selection = window.getSelection();

            // 1. Validation: Ensure selection exists and is inside our container
            if (
                !selection ||
                selection.isCollapsed ||
                !containerRef.current ||
                !containerRef.current.contains(selection.anchorNode)
            ) {
                setMenuState(null);
                return;
            }

            // Ensure wordMap is available
            if (!wordMap || wordMap.length === 0) {
                setMenuState(null);
                return;
            }

            // 2. Geometry: Get screen coordinates
            const range = selection.getRangeAt(0);
            const rect = range.getBoundingClientRect();

            // 3. Data Lookup: Map character index to Timestamp
            let startNode: Node = range.startContainer;
            const startOffset = range.startOffset;

            // If the selection starts in an ELEMENT node (not text node), use its first child or itself
            if (startNode.nodeType !== Node.TEXT_NODE) {
                if (startOffset < startNode.childNodes.length) {
                    startNode = startNode.childNodes[startOffset];
                } else {
                    startNode = startNode.lastChild || startNode;
                }
            }

            // Calculate absolute character index using TreeWalker
            // Only count text nodes inside elements marked with data-transcript-text
            // This prevents timestamps and speaker names from throwing off the character index
            const walker = document.createTreeWalker(
                containerRef.current,
                NodeFilter.SHOW_TEXT,
                {
                    acceptNode: (node) => {
                        const parent = node.parentElement?.closest('[data-transcript-text]');
                        return parent ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_SKIP;
                    }
                }
            );

            let charIndex = 0;
            let node = walker.nextNode();
            let found = false;

            while (node) {
                if (node === startNode) {
                    found = true;
                    break;
                }
                charIndex += node.textContent?.length || 0;
                node = walker.nextNode();
            }

            // If we found the node, add the local offset
            if (found && startNode.nodeType === Node.TEXT_NODE) {
                charIndex += startOffset;
            } else if (!found) {
                // Selection started outside our container
                return;
            }

            // Find the word corresponding to this character index
            const matchedWord = wordMap.find(w =>
                charIndex >= w.startChar && charIndex <= w.endChar
            );

            // For end index, calculate selection length
            const selectionLen = selection.toString().length;
            const endCharIndex = charIndex + selectionLen;

            const matchedEndWord = wordMap.find(w =>
                endCharIndex >= w.startChar && endCharIndex <= w.endChar
            ) || wordMap.find(w => w.startChar >= endCharIndex);

            if (matchedWord) {
                // Clamp X position to stay within viewport bounds
                const centerX = rect.left + (rect.width / 2);
                const clampedX = Math.min(window.innerWidth - 16, Math.max(16, centerX));

                // Position above selection, but flip below if too close to top
                let posY = rect.top - 12;
                if (posY < 60) {
                    posY = rect.bottom + 12;
                }

                setMenuState({
                    visible: true,
                    x: clampedX,
                    y: posY,
                    startTime: matchedWord.startTime,
                    endTime: matchedEndWord ? matchedEndWord.endTime : matchedWord.endTime,
                    startIdx: wordMap.indexOf(matchedWord),
                    endIdx: matchedEndWord ? wordMap.indexOf(matchedEndWord) : wordMap.indexOf(matchedWord),
                    selectedText: selection.toString().trim()
                });
            }
        };

        // Debounce: Wait 150ms for the user to stop dragging handles
        let timeout: number;
        const onSelectionChange = () => {
            clearTimeout(timeout);
            timeout = window.setTimeout(handleSelectionChange, 150);
        };

        document.addEventListener('selectionchange', onSelectionChange);

        // UX: Hide menu immediately on scroll to mimic native behavior
        const onScroll = () => {
            if (!showEditor) {
                setMenuState(null);
            }
        };
        window.addEventListener('scroll', onScroll, true);

        return () => {
            document.removeEventListener('selectionchange', onSelectionChange);
            window.removeEventListener('scroll', onScroll, true);
            clearTimeout(timeout);
        };
    }, [containerRef, wordMap, showEditor]);

    return {
        menuState,
        showEditor,
        openEditor,
        closeEditor,
        dismissMenu
    };
}
