import { useState, useRef, useEffect } from "react";
import { createPortal } from "react-dom";
import { TranscriptView } from "@/components/transcript/TranscriptView";
import { TranscriptToolbar } from "@/components/transcript/TranscriptToolbar";
import { useNotes, useCreateNote, useUpdateNote, useDeleteNote } from "@/features/transcription/hooks/useTranscriptionNotes";
import { useSpeakerMappings } from "@/features/transcription/hooks/useTranscriptionSpeakers";
import { useTranscript, useAudioDetail } from "@/features/transcription/hooks/useAudioDetail";
import { useTranscriptSelection } from "@/features/transcription/hooks/useTranscriptSelection";
import { useTranscriptDownload } from "@/features/transcription/hooks/useTranscriptDownload";
import { useNavigate } from "react-router-dom";
import { DownloadDialog } from "./DownloadDialog";
import SpeakerRenameDialog from "./SpeakerRenameDialog";
import { NotesSidebar } from "./NotesSidebar";
import { TranscriptSelectionMenu } from "./TranscriptSelectionMenu";
import { NoteEditorDialog } from "./NoteEditorDialog";
import { useIsMobile } from "@/hooks/use-mobile";
import { X, StickyNote } from "lucide-react";

interface TranscriptSectionProps {
    audioId: string;
    currentWordIndex: number | null;
    onSeek: (time: number) => void;
    onOpenExecutionInfo: () => void;
    onOpenLogs: () => void;
    onOpenSummarize: () => void;
    llmReady: boolean | null;
}

export function TranscriptSection({
    audioId,
    currentWordIndex,
    onSeek,
    onOpenExecutionInfo,
    onOpenLogs,
    onOpenSummarize,
    llmReady
}: TranscriptSectionProps) {
    const navigate = useNavigate();
    const isMobile = useIsMobile();

    // Data hooks
    const { data: transcript } = useTranscript(audioId, true);
    const { data: audioFile } = useAudioDetail(audioId);
    const { data: notes = [] } = useNotes(audioId);
    const { mutate: createNote } = useCreateNote(audioId);
    const { mutateAsync: updateNote } = useUpdateNote(audioId);
    const { mutateAsync: deleteNote } = useDeleteNote(audioId);
    const { data: speakerMappings = {} } = useSpeakerMappings(audioId, true);

    // Download Logic
    const { downloadSRT } = useTranscriptDownload();

    // Local state
    const [transcriptMode, setTranscriptMode] = useState<"compact" | "expanded">("compact");
    const [autoScrollEnabled, setAutoScrollEnabled] = useState(true);
    const [notesOpen, setNotesOpen] = useState(false);
    const [speakerRenameOpen, setSpeakerRenameOpen] = useState(false);

    // Download Dialog State
    const [downloadDialogOpen, setDownloadDialogOpen] = useState(false);
    const [downloadFormat, setDownloadFormat] = useState<'txt' | 'json'>('txt');

    // Refs
    const transcriptRef = useRef<HTMLDivElement>(null);
    const highlightedWordRef = useRef<HTMLSpanElement>(null);

    // Selection Logic
    const {
        showSelectionMenu,
        pendingSelection,
        selectionViewportPos,
        showEditor,
        openEditor,
        closeEditor
    } = useTranscriptSelection(transcript, transcriptRef as React.RefObject<HTMLElement>);

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
        if (!transcript?.segments) return [];
        const speakers = new Set<string>();
        transcript.segments.forEach(segment => {
            if (segment.speaker) speakers.add(segment.speaker);
        });
        return Array.from(speakers).sort();
    };

    const hasSpeakers = () => {
        return audioFile?.diarization || audioFile?.parameters?.diarize || audioFile?.is_multi_track || false;
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
        <div className="bg-[var(--bg-main)] rounded-[var(--radius-card)] border border-[var(--border-subtle)] p-4 md:p-6 transition-all duration-300">
            {/* Sticky Toolbar */}
            <div className="mb-6 sticky top-4 z-10 flex justify-center pointer-events-none">
                <div className="pointer-events-auto bg-[var(--bg-card)]/80 backdrop-blur-xl border border-[var(--border-subtle)] shadow-lg rounded-[var(--radius-card)] p-1">
                    <TranscriptToolbar
                        transcriptMode={transcriptMode}
                        setTranscriptMode={setTranscriptMode}
                        autoScrollEnabled={autoScrollEnabled}
                        setAutoScrollEnabled={setAutoScrollEnabled}
                        notesOpen={notesOpen}
                        setNotesOpen={setNotesOpen}
                        notes={notes}
                        onOpenExecutionInfo={onOpenExecutionInfo}
                        onOpenLogs={onOpenLogs}
                        hasSpeakers={hasSpeakers()}
                        detectedSpeakersCount={getDetectedSpeakers().length}
                        onOpenSpeakerRename={() => setSpeakerRenameOpen(true)}
                        onOpenSummarize={onOpenSummarize}
                        llmReady={llmReady}
                        onDownloadSRT={() => downloadSRT(transcript, audioFile?.title || 'transcript', speakerMappings)}
                        onDownloadTXT={() => { setDownloadFormat('txt'); setDownloadDialogOpen(true); }}
                        onDownloadJSON={() => { setDownloadFormat('json'); setDownloadDialogOpen(true); }}
                        onOpenChat={() => navigate(`/audio/${audioId}/chat`)}
                    />
                </div>
            </div>

            {/* Transcript Content - Systematic Typography */}
            <div className="relative overflow-hidden font-sans">
                <div className="w-full text-[var(--text-secondary)] leading-relaxed">
                    <div ref={transcriptRef} className="relative">
                        <TranscriptView
                            transcript={transcript}
                            mode={transcriptMode}
                            currentWordIndex={currentWordIndex}
                            notes={notes}
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
