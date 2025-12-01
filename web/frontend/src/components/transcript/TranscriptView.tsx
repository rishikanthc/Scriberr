import { forwardRef } from 'react';
import { cn } from '@/lib/utils';
import type { Note } from '@/types/note';

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
    notes: Note[];
    highlightedWordRef: React.RefObject<HTMLSpanElement | null>;
    speakerMappings: Record<string, string>;
    autoScrollEnabled: boolean;
    className?: string;
}

export const TranscriptView = forwardRef<HTMLDivElement, TranscriptViewProps>(({
    transcript,
    mode,
    currentWordIndex,
    notes,
    highlightedWordRef,
    speakerMappings,
    autoScrollEnabled,
    className
}, ref) => {

    if (!transcript) {
        return (
            <div className="flex flex-col items-center justify-center h-64 text-carbon-400">
                <p>No transcript available.</p>
            </div>
        );
    }

    const getDisplaySpeakerName = (originalSpeaker: string): string => {
        return speakerMappings[originalSpeaker] || originalSpeaker;
    };

    // Render transcript with word-level highlighting for compact view
    const renderCompactView = () => {
        if (!transcript.word_segments || transcript.word_segments.length === 0) {
            return <p className="text-lg leading-relaxed text-carbon-700 dark:text-carbon-300 whitespace-pre-wrap">{transcript.text}</p>;
        }

        return (
            <div className="text-lg leading-relaxed text-carbon-700 dark:text-carbon-300">
                {transcript.word_segments.map((word, index) => {
                    const isHighlighted = index === currentWordIndex;
                    const isAnnotated = notes.some(n => index >= n.start_word_index && index <= n.end_word_index);

                    return (
                        <span
                            key={index}
                            ref={isHighlighted && autoScrollEnabled ? highlightedWordRef : undefined}
                            data-word-index={index}
                            data-word={word.word}
                            data-start={word.start}
                            data-end={word.end}
                            className={cn(
                                "cursor-text transition-colors duration-150 rounded px-0.5 inline-block",
                                "hover:bg-blue-100 dark:hover:bg-blue-900/30",
                                isHighlighted && "bg-amber-200 dark:bg-amber-700/50 text-carbon-900 dark:text-carbon-50 font-medium shadow-sm",
                                !isHighlighted && isAnnotated && "bg-carbon-200 dark:bg-carbon-700/50 border-b-2 border-amber-400 dark:border-amber-600"
                            )}
                        >
                            {word.word}{" "}
                        </span>
                    );
                })}
            </div>
        );
    };

    // Render segment with word-level highlighting for expanded view
    const renderSegmentWords = (segment: any) => {
        if (!transcript.word_segments) {
            return segment.text.trim();
        }

        // Find words that belong to this segment
        // We use a slightly loose matching to ensure we catch words that might slightly overlap boundaries
        const segmentWords = transcript.word_segments.filter(
            word => word.start >= segment.start - 0.1 && word.end <= segment.end + 0.1
        );

        if (segmentWords.length === 0) {
            return segment.text.trim();
        }

        return segmentWords.map((word, index) => {
            // We need to find the global index for correct highlighting
            // This might be slow for very long transcripts, but correct
            const globalIndex = transcript.word_segments?.findIndex(w => w === word) ?? -1;

            const isHighlighted = globalIndex === currentWordIndex;
            const isAnnotated = notes.some(n => globalIndex >= n.start_word_index && globalIndex <= n.end_word_index);

            return (
                <span
                    key={`${segment.start}-${index}`}
                    ref={isHighlighted && autoScrollEnabled ? highlightedWordRef : undefined}
                    data-word-index={globalIndex}
                    data-word={word.word}
                    data-start={word.start}
                    data-end={word.end}
                    className={cn(
                        "cursor-text transition-colors duration-150 rounded px-0.5 inline-block",
                        "hover:bg-blue-100 dark:hover:bg-blue-900/30",
                        isHighlighted && "bg-amber-200 dark:bg-amber-700/50 text-carbon-900 dark:text-carbon-50 font-medium shadow-sm",
                        !isHighlighted && isAnnotated && "bg-carbon-200 dark:bg-carbon-700/50 border-b-2 border-amber-400 dark:border-amber-600"
                    )}
                >
                    {word.word}{" "}
                </span>
            );
        });
    };

    const renderExpandedView = () => {
        if (!transcript.segments) {
            return renderCompactView();
        }

        return (
            <div className="space-y-6">
                {transcript.segments.map((segment, i) => (
                    <div key={i} className="group flex flex-col sm:flex-row items-start gap-2 sm:gap-4 p-3 rounded-lg hover:bg-carbon-50 dark:hover:bg-carbon-800/50 transition-colors">
                        {/* Timestamp & Speaker */}
                        <div className="flex-shrink-0 w-full sm:w-32 flex sm:flex-col items-center sm:items-end gap-2 sm:gap-1 text-xs sm:text-sm text-carbon-500 dark:text-carbon-400 select-none mt-1">
                            <span className="font-mono bg-carbon-100 dark:bg-carbon-800 px-1.5 py-0.5 rounded">
                                {new Date(segment.start * 1000).toISOString().substr(11, 8)}
                            </span>
                            {segment.speaker && (
                                <span
                                    className="font-medium text-carbon-700 dark:text-carbon-300 truncate max-w-[120px]"
                                    title={getDisplaySpeakerName(segment.speaker)}
                                >
                                    {getDisplaySpeakerName(segment.speaker)}
                                </span>
                            )}
                        </div>

                        {/* Text */}
                        <div className="flex-grow text-base sm:text-lg leading-relaxed text-carbon-700 dark:text-carbon-300">
                            {renderSegmentWords(segment)}
                        </div>
                    </div>
                ))}
            </div>
        );
    };

    return (
        <div
            ref={ref}
            className={cn("w-full max-w-none font-inter", className)}
        >
            {mode === 'compact' ? renderCompactView() : renderExpandedView()}
        </div>
    );
});

TranscriptView.displayName = 'TranscriptView';
