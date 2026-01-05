import { forwardRef, useRef, useState, useCallback, useEffect, useMemo } from 'react';
import { useKaraokeHighlight, computeWordOffsets, findActiveWordIndex } from '@/features/transcription/hooks/useKaraokeHighlight';
import { cn } from '@/lib/utils';
import { useIsDesktop } from '@/hooks/useIsDesktop';
import type { Note } from '@/types/note';

// Helper for cross-browser caret position
function getCaretOffsetFromPoint(x: number, y: number) {
    if (document.caretRangeFromPoint) {
        const range = document.caretRangeFromPoint(x, y);
        return range ? range.startOffset : null;
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    if ((document as any).caretPositionFromPoint) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const pos = (document as any).caretPositionFromPoint(x, y);
        return pos ? pos.offset : null;
    }
    return null;
}

interface WordSegment {
    start: number;
    end: number;
    word: string;
    score: number;
    speaker?: string;
}

interface Transcript {
    text: string;
    segments?: Array<{
        start: number;
        end: number;
        text: string;
        speaker?: string;
    }>;
    word_segments?: WordSegment[];
}

interface TranscriptViewProps {
    transcript: Transcript | null;
    mode: 'compact' | 'expanded';
    currentWordIndex: number | null;
    currentTime: number;
    isPlaying: boolean;
    notes: Note[];
    highlightedWordRef: React.RefObject<HTMLSpanElement | null>;
    speakerMappings: Record<string, string>;
    autoScrollEnabled: boolean;
    onSeek: (time: number) => void;
    className?: string;
}

export const TranscriptView = forwardRef<HTMLDivElement, TranscriptViewProps>(({
    transcript,
    mode,
    // currentWordIndex,
    currentTime,
    isPlaying,
    // notes,
    // highlightedWordRef,
    speakerMappings,
    autoScrollEnabled,
    onSeek,
    className
}, ref) => {

    const getDisplaySpeakerName = (originalSpeaker: string): string => {
        return speakerMappings[originalSpeaker] || originalSpeaker;
    };

    const containerRef = useRef<HTMLDivElement>(null);
    const [isModifierPressed, setIsModifierPressed] = useState(false);
    const isDesktop = useIsDesktop();

    // Use CSS Highlight API for Compact Mode
    // Note: We only use this hook when in compact mode to save resources
    const words = transcript?.word_segments || [];
    const { fullText, offsets } = useKaraokeHighlight(
        containerRef,
        words,
        currentTime,
        isPlaying
    );

    // Click-to-Seek Handler
    const handleWordClick = useCallback((e: React.MouseEvent) => {
        // Only trigger if Cmd (Mac) or Ctrl (Windows) is held
        if (!e.metaKey && !e.ctrlKey) return;

        const clickOffset = getCaretOffsetFromPoint(e.clientX, e.clientY);
        if (clickOffset === null) return;

        const clickedWord = offsets.find(w =>
            clickOffset >= w.startChar && clickOffset <= w.endChar
        );

        if (clickedWord) {
            onSeek(clickedWord.startTime);
            e.preventDefault();
        }
    }, [offsets, onSeek]);

    // Keyboard listener for modifier key visual cue
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Meta' || e.key === 'Control') setIsModifierPressed(true);
        };
        const handleKeyUp = (e: KeyboardEvent) => {
            if (e.key === 'Meta' || e.key === 'Control') setIsModifierPressed(false);
        };

        window.addEventListener('keydown', handleKeyDown);
        window.addEventListener('keyup', handleKeyUp);
        return () => {
            window.removeEventListener('keydown', handleKeyDown);
            window.removeEventListener('keyup', handleKeyUp);
        };
    }, []);

    // Expanded View Logic
    const segmentRefs = useRef<(HTMLDivElement | null)[]>([]);

    // 1. Precompute per-segment text and offsets
    const expandedData = useMemo(() => {
        if (!transcript?.segments || !transcript.word_segments) return [];

        return transcript.segments.map((segment) => {
            // Filter words belonging to this segment
            const segmentWords = transcript.word_segments!.filter(
                word => word.start >= segment.start - 0.1 && word.end <= segment.end + 0.1
            );

            // Compute local offsets for this segment's text
            const { fullText, offsets } = computeWordOffsets(segmentWords);

            return {
                ...segment,
                fullText, // The text to render
                offsets   // Offsets relative to this segment's text node
            };
        });
    }, [transcript]);

    // Compute which segment is currently active based on playback time
    const activeSegmentIndex = useMemo(() => {
        if (!expandedData.length) return -1;
        // Find the latest segment that has started (search backwards)
        for (let i = expandedData.length - 1; i >= 0; i--) {
            if (expandedData[i].start <= currentTime) return i;
        }
        return 0;
    }, [expandedData, currentTime]);

    // Track previous active segment to only scroll on segment change
    const prevActiveSegmentRef = useRef<number>(-1);

    // Auto-scroll to active segment during playback
    useEffect(() => {
        if (mode !== 'expanded' || !autoScrollEnabled || !isPlaying) return;
        if (activeSegmentIndex < 0) return;
        // Only scroll when the segment actually changes
        if (activeSegmentIndex === prevActiveSegmentRef.current) return;
        prevActiveSegmentRef.current = activeSegmentIndex;

        const el = segmentRefs.current[activeSegmentIndex];
        if (el) {
            el.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    }, [activeSegmentIndex, autoScrollEnabled, isPlaying, mode]);

    // 2. Highlight Effect for Expanded View
    useEffect(() => {
        if (mode !== 'expanded' || !expandedData.length || !isPlaying) return;
        if (typeof CSS === 'undefined' || !CSS.highlights) return;

        // Find the active segment and word
        // Optimization: We could binary search segments, but N is usually small (<1000). Linear is okay or optimize later.
        // Actually for real-time validation, let's just find the active word in the relevant segment.

        let found = false;

        // Search backwards to find the LATEST segment that has started
        // This prevents getting stuck on the first segment (which is always "started" relative to future time)
        for (let i = expandedData.length - 1; i >= 0; i--) {
            const seg = expandedData[i];

            // Optimization: If segment hasn't started yet, skip it
            // (heuristic using segment start time)
            if (seg.start > currentTime) continue;

            const activeIndex = findActiveWordIndex(seg.offsets, currentTime);
            if (activeIndex !== -1) {
                const w = seg.offsets[activeIndex];
                const el = segmentRefs.current[i];

                if (el && el.firstChild) {
                    try {
                        const range = new Range();
                        if (w.endChar <= (el.firstChild as Text).length) {
                            range.setStart(el.firstChild, w.startChar);
                            range.setEnd(el.firstChild, w.endChar);
                            const highlight = new Highlight(range);
                            CSS.highlights.set('karaoke-word', highlight);
                            found = true;
                        }
                    } catch {
                        // Ignore range errors
                    }
                }
                if (found) break;
            }
        }

        if (!found) {
            if (CSS.highlights.has('karaoke-word')) CSS.highlights.delete('karaoke-word');
        }

    }, [currentTime, isPlaying, mode, expandedData]);

    // 3. Click Handler for Expanded View
    const handleExpandedClick = useCallback((e: React.MouseEvent, segmentIndex: number) => {
        if (!e.metaKey && !e.ctrlKey) return;

        const clickOffset = getCaretOffsetFromPoint(e.clientX, e.clientY);
        if (clickOffset === null) return;

        const segData = expandedData[segmentIndex];
        if (!segData) return;

        const clickedWord = segData.offsets.find(w =>
            clickOffset >= w.startChar && clickOffset <= w.endChar
        );

        if (clickedWord) {
            onSeek(clickedWord.startTime);
            e.preventDefault();
        }
    }, [expandedData, onSeek]);

    if (!transcript) {
        return (
            <div className="flex flex-col items-center justify-center h-64 text-carbon-400">
                <p>No transcript available.</p>
            </div>
        );
    }

    // Render transcript with word-level highlighting for compact view
    const renderCompactView = () => {
        if (!transcript.word_segments || transcript.word_segments.length === 0) {
            return <p className="text-lg leading-relaxed text-carbon-700 dark:text-carbon-300 whitespace-pre-wrap">{transcript.text}</p>;
        }

        return (
            <div
                ref={containerRef}
                onClick={isDesktop ? handleWordClick : undefined}
                className={cn(
                    "text-lg leading-relaxed text-carbon-700 dark:text-carbon-300 whitespace-pre-wrap font-reading selection:bg-orange-500/30 transition-colors duration-200 select-text",
                    isDesktop && isModifierPressed ? 'cursor-pointer hover:text-carbon-900 dark:hover:text-carbon-100' : 'cursor-text'
                )}
                style={{
                    // CRITICAL: Enable native text selection on iOS/Android
                    WebkitUserSelect: 'text',
                    userSelect: 'text',
                    // CRITICAL: Remove grey tap highlight on iOS
                    WebkitTapHighlightColor: 'transparent',
                    // CRITICAL: Allow text selection gestures while supporting scroll
                    // 'manipulation' allows pan and pinch-zoom but not double-tap zoom
                    touchAction: 'pan-y pinch-zoom',
                    // Ensure text is the selection target, not the container
                    WebkitTouchCallout: 'default'
                }}
                data-transcript-text
            >
                {/* The hook returns the built text string, so we just render it directly */}
                {fullText}
            </div>

        );
    };

    const renderExpandedView = () => {
        if (!transcript?.segments) {
            return renderCompactView();
        }

        return (
            <div className="space-y-4"> {/* Reduced spacing from space-y-6 */}
                {expandedData.map((segment, i) => (
                    <div
                        key={i}
                        className={cn(
                            "group flex flex-col sm:flex-row items-start gap-4 p-3 rounded-lg transition-colors border",
                            i === activeSegmentIndex && isPlaying
                                ? "bg-carbon-50 dark:bg-carbon-800/40 border-carbon-200 dark:border-carbon-700"
                                : "hover:bg-carbon-50 dark:hover:bg-carbon-800/50 border-transparent hover:border-carbon-100 dark:hover:border-carbon-800"
                        )}
                    >
                        {/* Timestamp & Speaker */}
                        <div className="flex-shrink-0 w-24 sm:w-28 flex flex-col items-start sm:items-end gap-1 text-xs text-carbon-500 dark:text-carbon-400 select-none mt-1">
                            <span className="font-mono bg-carbon-100 dark:bg-carbon-800/80 px-1.5 py-0.5 rounded text-[10px] sm:text-xs">
                                {new Date(segment.start * 1000).toISOString().substr(11, 8)}
                            </span>
                            {segment.speaker && (
                                <span
                                    className="font-medium text-carbon-700 dark:text-carbon-300 truncate max-w-full"
                                    title={getDisplaySpeakerName(segment.speaker)}
                                >
                                    {getDisplaySpeakerName(segment.speaker)}
                                </span>
                            )}
                        </div>

                        {/* Text */}
                        <div
                            ref={(el) => { segmentRefs.current[i] = el; }}
                            onClick={isDesktop ? (e) => handleExpandedClick(e, i) : undefined}
                            className={cn(
                                "flex-grow text-base text-primary leading-relaxed whitespace-pre-wrap font-reading transition-colors duration-200 select-text",
                                isDesktop && isModifierPressed ? 'cursor-pointer hover:text-carbon-900 dark:hover:text-carbon-100' : 'cursor-text'
                            )}
                            style={{
                                // CRITICAL: Enable native text selection on iOS/Android
                                WebkitUserSelect: 'text',
                                userSelect: 'text',
                                // CRITICAL: Remove grey tap highlight on iOS
                                WebkitTapHighlightColor: 'transparent',
                                // CRITICAL: Allow text selection gestures while supporting scroll
                                touchAction: 'pan-y pinch-zoom',
                                WebkitTouchCallout: 'default'
                            }}
                            data-transcript-text
                        >
                            {segment.fullText || segment.text}
                        </div>
                    </div>
                ))}
            </div>
        );
    };

    return (
        <div
            ref={ref}
            className={cn("w-full max-w-none font-literata mt-4", className)}
        >
            {mode === 'compact' ? renderCompactView() : renderExpandedView()}

            {/* CSS for the Highlight API - Global for both views */}
            <style>{`
                ::highlight(karaoke-word) {
                    background-color: transparent;
                    color: var(--brand-solid) !important;
                    font-weight: 600;
                    text-decoration: underline decoration-dotted var(--brand-solid);
                    text-underline-offset: 4px;
                }
            `}</style>
        </div>
    );
});

TranscriptView.displayName = 'TranscriptView';
