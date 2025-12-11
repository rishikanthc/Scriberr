import { useRef, useEffect, useMemo } from "react";
import { createPortal } from "react-dom";
import { TranscriptView } from "@/components/transcript/TranscriptView";
import { useNotes, useCreateNote, useUpdateNote, useDeleteNote } from "@/features/transcription/hooks/useTranscriptionNotes";
import { useTranscriptSelection } from "@/features/transcription/hooks/useTranscriptSelection";
import { DownloadDialog } from "./DownloadDialog";
import SpeakerRenameDialog from "./SpeakerRenameDialog";
import { NotesSidebar } from "./NotesSidebar";
import { TranscriptSelectionMenu } from "./TranscriptSelectionMenu";
import { NoteEditorDialog } from "./NoteEditorDialog";
import { useIsMobile } from "@/hooks/use-mobile";
import { X, StickyNote } from "lucide-react";
import { computeWordOffsets } from "@/features/transcription/hooks/useKaraokeHighlight";
import type { Transcript } from "@/features/transcription/hooks/useAudioDetail";

interface TranscriptSectionProps {
    audioId: string;
    currentWordIndex: number | null;
    currentTime: number;
    isPlaying: boolean;
    onSeek: (time: number) => void;
    // Lifted State Props
    transcript: Transcript | undefined;
    speakerMappings: Record<string, string>;
    transcriptMode: 'compact' | 'expanded';
    autoScrollEnabled: boolean;
    notesOpen: boolean;
    setNotesOpen: (open: boolean) => void;
    speakerRenameOpen: boolean;
    setSpeakerRenameOpen: (open: boolean) => void;
    downloadDialogOpen: boolean;
    setDownloadDialogOpen: (open: boolean) => void;
    downloadFormat: 'txt' | 'json';
}

export function TranscriptSection({
    audioId,
    currentWordIndex,
    currentTime,
    isPlaying,
    onSeek,
    transcript,
    speakerMappings,
    transcriptMode,
    autoScrollEnabled,
    notesOpen,
    setNotesOpen,
    speakerRenameOpen,
    setSpeakerRenameOpen,
    downloadDialogOpen,
    setDownloadDialogOpen,
    downloadFormat
}: TranscriptSectionProps) {
    const isMobile = useIsMobile();

    // Data hooks
    // const { data: audioFile } = useAudioDetail(audioId); // Unused
    const { data: notes = [] } = useNotes(audioId);
    const { mutate: createNote } = useCreateNote(audioId);
    const { mutateAsync: updateNote } = useUpdateNote(audioId);
    const { mutateAsync: deleteNote } = useDeleteNote(audioId);

    // Refs
    const transcriptRef = useRef<HTMLDivElement>(null);
    const highlightedWordRef = useRef<HTMLSpanElement>(null);

    // Compute offsets for selection logic
    const words = transcript?.word_segments || [];
    const { offsets } = useMemo(() => computeWordOffsets(words), [words]);

    // Selection Logic
    const {
        showSelectionMenu,
        pendingSelection,
        selectionViewportPos,
        showEditor,
        openEditor,
        closeEditor
    } = useTranscriptSelection(transcriptRef as React.RefObject<HTMLElement>, offsets);

    // Auto-scroll logic
    useEffect(() => {
        if (currentWordIndex !== null && highlightedWordRef.current && autoScrollEnabled) {
            const highlightedElement = highlightedWordRef.current;
            const highlightedRect = highlightedElement.getBoundingClientRect();
            const viewportHeight = window.innerHeight;
            const buffer = viewportHeight * 0.2; // 20%
            const isAboveView = highlightedRect.top < buffer;
            const isBelowView = highlightedRect.bottom > (viewportHeight - buffer);

            if (isAboveView || isBelowView) {
                highlightedElement.scrollIntoView({
                    behavior: 'smooth',
                    block: 'center',
                });
            }
        }
    }, [currentWordIndex, autoScrollEnabled]);

    // Click to seek handler
    useEffect(() => {
        const el = transcriptRef.current;
        if (!el) return;
        const onClick = (e: MouseEvent) => {
            if (!(e.metaKey || e.ctrlKey)) return;
            const target = e.target as HTMLElement | null;
            if (!target) return;
            const wordEl = target.closest('span[data-word-index]') as HTMLElement | null;
            if (!wordEl) return;
            const startAttr = wordEl.getAttribute('data-start');
            const start = startAttr ? parseFloat(startAttr) : NaN;
            if (isNaN(start)) return;

            e.preventDefault();
            e.stopPropagation();
            onSeek(start);
        };
        el.addEventListener('click', onClick);
        return () => el.removeEventListener('click', onClick);
    }, [onSeek]);

    // Helpers
    const getDetectedSpeakers = () => {
        // Safe check for segments array
        if (!transcript?.segments) return [];
        const speakers = new Set<string>();
        // Using any for segment here if strict typing is an issue, or typed via Transcript
        transcript.segments.forEach((segment: any) => {
            if (segment.speaker) speakers.add(segment.speaker);
        });
        return Array.from(speakers).sort();
    };

    const handleSaveNote = (content: string) => {
        if (pendingSelection) {
            createNote({
                start_time: pendingSelection.startTime,
                end_time: pendingSelection.endTime,
                content: content,
                quote: pendingSelection.quote,
                start_word_index: pendingSelection.startIdx,
                end_word_index: pendingSelection.endIdx
            });
            closeEditor();
            setNotesOpen(true);
        }
    };

    const handleListenFromHere = () => {
        if (pendingSelection) {
            onSeek(pendingSelection.startTime);
            closeEditor();
        }
    };

    if (!transcript) return null;

    return (
        <div className="glass-card rounded-[var(--radius-card)] p-4 md:p-6 min-h-[500px] border-[var(--border-subtle)] shadow-[var(--shadow-card)] transition-shadow hover:shadow-[var(--shadow-float)]">
            {/* 
                  TOOLBAR REMOVED -> Moved to Context Menu 
                */}

            {/* Transcript Content - Systematic Typography */}
            <div className="relative overflow-hidden font-sans">
                <div className="w-full text-[var(--text-secondary)] leading-relaxed">
                    <div ref={transcriptRef} className="relative">
                        <TranscriptView
                            transcript={transcript}
                            mode={transcriptMode}
                            currentWordIndex={currentWordIndex}
                            currentTime={currentTime}
                            isPlaying={isPlaying}
                            notes={notes}
                            onSeek={onSeek}
                            highlightedWordRef={highlightedWordRef}
                            speakerMappings={speakerMappings}
                            autoScrollEnabled={autoScrollEnabled}
                        />
                    </div>
                </div>
            </div>

            {/* Download Dialog */}
            <DownloadDialog
                audioId={audioId}
                isOpen={downloadDialogOpen}
                onClose={setDownloadDialogOpen}
                initialFormat={downloadFormat}
            />

            {/* Speaker Rename Dialog */}
            <SpeakerRenameDialog
                open={speakerRenameOpen}
                onOpenChange={setSpeakerRenameOpen}
                transcriptionId={audioId}
                initialSpeakers={getDetectedSpeakers()}
                onSpeakerMappingsUpdate={() => { }}
            />
            {/* Portals */}
            {createPortal(
                <>
                    {/* Selection Menu Bubble (Glass) */}
                    <div className="z-[9999]">
                        <TranscriptSelectionMenu
                            isOpen={showSelectionMenu}
                            isMobile={isMobile}
                            position={selectionViewportPos}
                            onAddNote={openEditor}
                            onListenFromHere={handleListenFromHere}
                        />
                    </div>

                    {/* Note Editor Dialog */}
                    <NoteEditorDialog
                        isOpen={showEditor}
                        quote={pendingSelection?.quote || ""}
                        position={selectionViewportPos}
                        onSave={handleSaveNote}
                        onCancel={closeEditor}
                    />

                    {/* Backdrop for menu/editor */}
                    {(showSelectionMenu || showEditor) && (
                        <div
                            style={{ position: 'fixed', inset: 0, zIndex: 9995, background: 'transparent' }}
                            onMouseDown={() => {
                                if (showSelectionMenu && !showEditor) {
                                    closeEditor();
                                }
                            }}
                        />
                    )}

                    {/* Notes Sidebar - Premium Drawer */}
                    {notesOpen && (
                        <div className="fixed inset-y-0 right-0 w-[90vw] max-w-[400px] bg-[var(--bg-card)] border-l border-[var(--border-subtle)] shadow-[var(--shadow-float)] z-[9990] transition-transform duration-300 transform-gpu">
                            <div className="h-full flex flex-col">
                                <div className="px-6 py-5 border-b border-[var(--border-subtle)] flex items-center justify-between">
                                    <h3 className="font-bold text-[var(--text-primary)] flex items-center gap-2 text-lg">
                                        <StickyNote className="h-5 w-5 text-[var(--brand-solid)]" />
                                        Notes
                                        <span className="ml-1 text-xs font-semibold rounded-full px-2 py-0.5 bg-[var(--bg-main)] text-[var(--text-secondary)] border border-[var(--border-subtle)]">
                                            {notes.length}
                                        </span>
                                    </h3>
                                    <button
                                        type="button"
                                        onClick={() => setNotesOpen(false)}
                                        className="h-8 w-8 inline-flex items-center justify-center rounded-[var(--radius-btn)] text-[var(--text-tertiary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-main)] transition-colors"
                                        aria-label="Close notes"
                                    >
                                        <X className="h-5 w-5" />
                                    </button>
                                </div>
                                <div className="flex-1 overflow-y-auto px-6 py-4">
                                    <NotesSidebar
                                        notes={notes}
                                        onEdit={(id, content) => updateNote({ id, content })}
                                        onDelete={(id) => deleteNote(id)}
                                        onJumpTo={(t) => {
                                            onSeek(t);
                                            if (isMobile) setNotesOpen(false);
                                        }}
                                    />
                                </div>
                            </div>
                        </div>
                    )}
                </>,
                document.body
            )}
        </div>
    );
}
