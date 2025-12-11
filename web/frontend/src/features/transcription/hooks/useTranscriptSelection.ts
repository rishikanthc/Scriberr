import { useState, useEffect } from "react";
import type { RefObject } from "react";
import { useIsMobile } from "@/hooks/use-mobile";

interface Transcript {
    word_segments?: Array<{
        start: number;
        end: number;
        word: string;
    }>;
}

export function useTranscriptSelection(transcript: Transcript | null | undefined, transcriptRef: RefObject<HTMLElement>) {
    const isMobile = useIsMobile();
    const [showSelectionMenu, setShowSelectionMenu] = useState(false);
    const [pendingSelection, setPendingSelection] = useState<{ startIdx: number; endIdx: number; startTime: number; endTime: number; quote: string } | null>(null);
    const [selectionViewportPos, setSelectionViewportPos] = useState<{ x: number, y: number }>({ x: 0, y: 0 });
    const [showEditor, setShowEditor] = useState(false);

    useEffect(() => {
        const el = transcriptRef.current;
        if (!el) return;

        const handleSelection = () => {
            const sel = window.getSelection();
            if (!sel || sel.isCollapsed) { setShowSelectionMenu(false); setShowEditor(false); return; }

            const anchor = sel.anchorNode as HTMLElement | null;
            const focus = sel.focusNode as HTMLElement | null;
            if (!anchor || !focus) return;

            const aSpan = (anchor.nodeType === 3 ? anchor.parentElement : anchor) as HTMLElement;
            const fSpan = (focus.nodeType === 3 ? focus.parentElement : focus) as HTMLElement;
            if (!aSpan || !fSpan) return;

            const aIdx = aSpan.closest('span[data-word-index]') as HTMLElement | null;
            const fIdx = fSpan.closest('span[data-word-index]') as HTMLElement | null;
            if (!aIdx || !fIdx) { setShowSelectionMenu(false); return; }

            const startIdx = Math.min(Number(aIdx.dataset.wordIndex), Number(fIdx.dataset.wordIndex));
            const endIdx = Math.max(Number(aIdx.dataset.wordIndex), Number(fIdx.dataset.wordIndex));

            if (!transcript?.word_segments || endIdx < startIdx) { setShowSelectionMenu(false); return; }

            const startTime = transcript.word_segments[startIdx]?.start ?? 0;
            const endTime = transcript.word_segments[endIdx]?.end ?? startTime;
            const quote = transcript.word_segments.slice(startIdx, endIdx + 1).map(w => w.word).join(" ");

            setPendingSelection({ startIdx, endIdx, startTime, endTime, quote });
            setShowSelectionMenu(true);

            if (!isMobile) {
                const range = sel.getRangeAt(0);
                const rect = range.getBoundingClientRect();
                const centerX = rect.left + rect.width / 2;
                const clampedX = Math.min(window.innerWidth - 16, Math.max(16, centerX));
                let bubbleY = rect.top - 10;
                if (bubbleY < 12) {
                    bubbleY = rect.bottom + 8;
                }
                setSelectionViewportPos({ x: clampedX, y: bubbleY });
            }
        };

        const onMouseUp = () => {
            if (!isMobile) handleSelection();
        };

        const onTouchEnd = () => {
            if (isMobile) setTimeout(handleSelection, 100);
        };

        el.addEventListener('mouseup', onMouseUp);
        el.addEventListener('touchend', onTouchEnd);

        const onSelectionChange = () => {
            if (isMobile && !showEditor) handleSelection();
        };

        if (isMobile) {
            document.addEventListener('selectionchange', onSelectionChange);
        }

        return () => {
            el.removeEventListener('mouseup', onMouseUp);
            el.removeEventListener('touchend', onTouchEnd);
            if (isMobile) document.removeEventListener('selectionchange', onSelectionChange);
        };
    }, [transcript, isMobile, showEditor, transcriptRef]);

    // Hide selection bubble when selection collapses
    useEffect(() => {
        const onSelectionChange = () => {
            if (showEditor) return;
            const sel = window.getSelection();
            if (!sel || sel.isCollapsed) {
                setShowSelectionMenu(false);
                setPendingSelection(null);
            }
        };
        document.addEventListener('selectionchange', onSelectionChange);
        return () => document.removeEventListener('selectionchange', onSelectionChange);
    }, [showEditor]);

    const openEditor = () => {
        setShowEditor(true);
        setShowSelectionMenu(false);
    };

    const closeEditor = () => {
        setShowEditor(false);
        setPendingSelection(null);
        window.getSelection()?.removeAllRanges();
    };

    return {
        showSelectionMenu,
        pendingSelection,
        selectionViewportPos,
        showEditor,
        openEditor,
        closeEditor
    };
}
