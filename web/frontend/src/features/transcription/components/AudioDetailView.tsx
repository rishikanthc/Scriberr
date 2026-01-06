import { useRef, useState, useEffect, useCallback } from "react";
import { createPortal } from "react-dom";
import { useParams, useNavigate } from "react-router-dom";
import { MoreVertical, Edit2, Activity, FileText, Bot, Check, Loader2, List, AlignLeft, ArrowDownCircle, StickyNote, MessageCircle, FileImage, FileJson, Clock, AlertCircle, Users } from "lucide-react";
import { Header } from "@/components/Header";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { EmberPlayer, type EmberPlayerRef } from "@/components/audio/EmberPlayer";
import { cn } from "@/lib/utils";

// Custom Hooks
import { useAudioDetail, useUpdateTitle, useTranscript, type TranscriptSegment } from "@/features/transcription/hooks/useAudioDetail";
import { useSpeakerMappings } from "@/features/transcription/hooks/useTranscriptionSpeakers";
import { useTranscriptDownload } from "@/features/transcription/hooks/useTranscriptDownload";

// Sub-components
import { TranscriptSection } from "./audio-detail/TranscriptSection";
import { ExecutionInfoDialog } from "./audio-detail/ExecutionInfoDialog";
import { LogsDialog } from "./audio-detail/LogsDialog";
import { SummaryDialog } from "./audio-detail/SummaryDialog";
import { ChatSidePanel } from "./ChatSidePanel";
import { useIsMobile } from "@/hooks/use-mobile";

// Types
interface AudioDetailViewProps {
    audioId?: string; // Optional prop if used as a controlled component, though mainly route-based
}

export const AudioDetailView = function AudioDetailView({ audioId: propAudioId }: AudioDetailViewProps) {
    const { audioId: paramAudioId } = useParams<{ audioId: string }>();
    const audioId = propAudioId || paramAudioId;
    const navigate = useNavigate();

    // Refs
    const audioPlayerRef = useRef<EmberPlayerRef>(null);

    // State
    const [currentTime, setCurrentTime] = useState(0);
    const [isPlaying, setIsPlaying] = useState(false);
    const [isEditingTitle, setIsEditingTitle] = useState(false);
    const [newTitle, setNewTitle] = useState("");

    // Lifted Transcript State
    const [transcriptMode, setTranscriptMode] = useState<"compact" | "expanded">("compact");
    const [autoScrollEnabled, setAutoScrollEnabled] = useState(true);
    const [notesOpen, setNotesOpen] = useState(false);
    const [speakerRenameOpen, setSpeakerRenameOpen] = useState(false);
    const [downloadDialogOpen, setDownloadDialogOpen] = useState(false);
    const [downloadFormat, setDownloadFormat] = useState<'txt' | 'json'>('txt');

    // Dialog States
    const [executionDialogOpen, setExecutionDialogOpen] = useState(false);
    const [logsDialogOpen, setLogsDialogOpen] = useState(false);
    const [summaryDialogOpen, setSummaryDialogOpen] = useState(false);

    // Data Fetching
    const { data: audioFile, isLoading, error } = useAudioDetail(audioId || "");
    const { mutate: updateTitle } = useUpdateTitle(audioId || "");
    // Fetch transcript & speakers here to support menu actions
    const { data: transcript } = useTranscript(audioId || "", true);
    const { data: speakerMappings = {} } = useSpeakerMappings(audioId || "", true);

    // Download Logic
    const { downloadSRT } = useTranscriptDownload();

    // State for Split View
    const [chatOpen, setChatOpen] = useState(false);
    const [sidebarWidth, setSidebarWidth] = useState(400);
    const [isResizing, setIsResizing] = useState(false);
    const splitContainerRef = useRef<HTMLDivElement>(null);
    const isMobile = useIsMobile();

    // Resizing Logic
    useEffect(() => {
        const handleMouseMove = (e: MouseEvent) => {
            if (!isResizing) return;
            const containerWidth = splitContainerRef.current?.getBoundingClientRect().width || window.innerWidth;
            const newWidth = containerWidth - e.clientX;
            // Constraints
            if (newWidth > 300 && newWidth < 800) {
                setSidebarWidth(newWidth);
            }
        };

        const handleMouseUp = () => {
            setIsResizing(false);
            document.body.style.cursor = 'default';
        };

        if (isResizing) {
            window.addEventListener('mousemove', handleMouseMove);
            window.addEventListener('mouseup', handleMouseUp);
            document.body.style.cursor = 'col-resize';
        }

        return () => {
            window.removeEventListener('mousemove', handleMouseMove);
            window.removeEventListener('mouseup', handleMouseUp);
        };
    }, [isResizing]);

    // Helpers




    // Effects
    useEffect(() => {
        if (audioFile) {
            setNewTitle(audioFile.title || "");
        }
    }, [audioFile]);

    // Handlers
    const handleTimeUpdate = useCallback((time: number) => {
        setCurrentTime(time);
    }, []);

    const handleTitleSave = () => {
        if (newTitle.trim() !== audioFile?.title) {
            updateTitle(newTitle);
        }
        setIsEditingTitle(false);
    };

    const handleSeek = (time: number) => {
        if (audioPlayerRef.current) {
            audioPlayerRef.current.seekTo(time);
            setCurrentTime(time);
        }
    };

    if (!audioId) return <div>Invalid Audio ID</div>;

    // Handler for notes/chat exclusivity
    const handleSetNotesOpen = (open: boolean) => {
        if (open) setChatOpen(false);
        setNotesOpen(open);
    };

    const handleSetChatOpen = (open: boolean) => {
        if (open) setNotesOpen(false);
        setChatOpen(open);
    };


    if (!audioId) return <div>Invalid Audio ID</div>;

    // Render
    if (isLoading) {
        return (
            <div className="h-full flex items-center justify-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
        );
    }

    if (error || !audioFile) {
        return (
            <div className="h-full flex flex-col items-center justify-center gap-4">
                <p className="text-red-500">Failed to load audio details.</p>
                <Button onClick={() => navigate('/dashboard')}>Go to Dashboard</Button>
            </div>
        );
    }

    // Helper to format date "Premium" style
    const formattedDate = new Date(audioFile.created_at).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric"
    }).toUpperCase();

    return (
        <div className="h-screen flex flex-col bg-[var(--bg-main)] relative selection:bg-[var(--brand-light)] overflow-hidden">
            {/* Split Container */}
            <div ref={splitContainerRef} className="flex-1 flex overflow-hidden relative">
                {/* LEFT PANE (Main) */}
                <main className="flex-1 min-w-0 flex flex-col h-full relative z-0">


                    {/* Scrollable Content */}
                    <div className="flex-1 overflow-y-auto scrollbar-thin">
                        <div className="mx-auto w-full max-w-[960px] px-4 sm:px-6 py-6 pb-32">
                            <div className="mb-6 pb-6">
                                <Header />
                            </div>
                            <div className="space-y-6 sm:space-y-8">
                                {/* Sticky header: Title + Audio Player */}
                                <div className="sticky top-0 z-10">
                                    {/* Title & Metadata */}
                                    <div className="space-y-4 pb-4 bg-[var(--bg-main)]">
                                    <div className="flex items-start justify-between gap-4">
                                        <div className="space-y-3 flex-1 min-w-0">
                                            {/* Title Edit Logic */}
                                            {isEditingTitle ? (
                                                <Input
                                                    value={newTitle}
                                                    onChange={(e) => setNewTitle(e.target.value)}
                                                    onBlur={handleTitleSave}
                                                    onKeyDown={(e) => e.key === 'Enter' && handleTitleSave()}
                                                    className="h-10 text-3xl font-bold tracking-tight bg-transparent border-none focus:ring-0 focus:outline-none p-0 placeholder:text-[var(--text-tertiary)]"
                                                    autoFocus
                                                />
                                            ) : (
                                                <div
                                                    className="group flex items-center gap-3 cursor-text"
                                                    onClick={() => setIsEditingTitle(true)}
                                                >
                                                    <h1 className="text-3xl font-bold tracking-tight text-[var(--text-primary)] truncate font-display">
                                                        {audioFile.title || "Untitled Recording"}
                                                    </h1>
                                                    <Edit2 className="h-4 w-4 text-[var(--text-tertiary)] opacity-0 group-hover:opacity-100 transition-opacity" />
                                                </div>
                                            )}

                                            {/* Badges */}
                                            <div className="flex items-center gap-3 text-xs font-medium uppercase tracking-wider text-[var(--text-tertiary)]">
                                                <span>{formattedDate}</span>
                                                <span className="w-1 h-1 rounded-full bg-[var(--text-tertiary)] opacity-50"></span>

                                                {/* Status Icon */}
                                                <div>
                                                    {audioFile.status === 'completed' && (
                                                        <Tooltip>
                                                            <TooltipTrigger asChild>
                                                                <div className="cursor-help text-emerald-500">
                                                                    <Check className="h-5 w-5" strokeWidth={2.5} />
                                                                </div>
                                                            </TooltipTrigger>
                                                            <TooltipContent>Completed</TooltipContent>
                                                        </Tooltip>
                                                    )}
                                                    {audioFile.status === 'processing' && (
                                                        <Tooltip>
                                                            <TooltipTrigger asChild>
                                                                <div className="cursor-help text-amber-500">
                                                                    <Loader2 className="h-5 w-5 animate-spin" strokeWidth={2.5} />
                                                                </div>
                                                            </TooltipTrigger>
                                                            <TooltipContent>Processing</TooltipContent>
                                                        </Tooltip>
                                                    )}
                                                    {audioFile.status === 'failed' && (
                                                        <Tooltip>
                                                            <TooltipTrigger asChild>
                                                                <div className="cursor-help text-red-500">
                                                                    <AlertCircle className="h-5 w-5" strokeWidth={2.5} />
                                                                </div>
                                                            </TooltipTrigger>
                                                            <TooltipContent>Failed</TooltipContent>
                                                        </Tooltip>
                                                    )}
                                                    {audioFile.status === 'pending' && (
                                                        <Tooltip>
                                                            <TooltipTrigger asChild>
                                                                <div className="cursor-help text-gray-400">
                                                                    <Clock className="h-5 w-5" strokeWidth={2.5} />
                                                                </div>
                                                            </TooltipTrigger>
                                                            <TooltipContent>Queued</TooltipContent>
                                                        </Tooltip>
                                                    )}
                                                </div>
                                            </div>
                                        </div>

                                        {/* Action Menu */}
                                        {/* ... keeping existing Logic but updating Chat action ... */}
                                        <div className="flex items-center gap-2">
                                            {/* Quick Chat Button */}
                                            <Button
                                                variant="outline"
                                                size="sm"
                                                onClick={() => setChatOpen(!chatOpen)}
                                                className={cn(
                                                    "rounded-full border-[var(--border-subtle)] shadow-sm bg-[var(--bg-card)] hover:bg-[var(--bg-main)] transition-all gap-2 px-3",
                                                    chatOpen && "border-[var(--brand-solid)] text-[var(--brand-solid)]"
                                                )}
                                            >
                                                <MessageCircle className="h-4 w-4" />
                                                <span className="hidden sm:inline">Chat</span>
                                            </Button>

                                            <DropdownMenu>
                                                <DropdownMenuTrigger asChild>
                                                    <Button
                                                        variant="outline"
                                                        size="icon"
                                                        className="rounded-full border-[var(--border-subtle)] shadow-sm bg-[var(--bg-card)] hover:bg-[var(--bg-main)] transition-all"
                                                    >
                                                        <MoreVertical className="h-4 w-4 text-[var(--text-secondary)]" />
                                                    </Button>
                                                </DropdownMenuTrigger>
                                                <DropdownMenuContent align="end" className="w-56 glass-card rounded-[var(--radius-card)] shadow-[var(--shadow-float)] border-[var(--border-subtle)] p-1.5">
                                                    {/* ... (Menu Items same as before, update handlers) ... */}
                                                    {/* Only show timeline view toggle if transcript has word-level timestamps */}
                                                    {transcript?.word_segments && transcript.word_segments.length > 0 ? (
                                                        <DropdownMenuItem onClick={() => setTranscriptMode(transcriptMode === 'compact' ? 'expanded' : 'compact')} className="rounded-[8px] cursor-pointer">
                                                            {transcriptMode === 'compact' ? <List className="mr-2 h-4 w-4 opacity-70" /> : <AlignLeft className="mr-2 h-4 w-4 opacity-70" />}
                                                            {transcriptMode === 'compact' ? 'Timeline View' : 'Compact View'}
                                                        </DropdownMenuItem>
                                                    ) : (
                                                        <DropdownMenuItem disabled className="rounded-[8px] opacity-50 cursor-not-allowed">
                                                            <List className="mr-2 h-4 w-4 opacity-70" />
                                                            Timeline View (No timestamps)
                                                        </DropdownMenuItem>
                                                    )}
                                                    <DropdownMenuItem onClick={() => setAutoScrollEnabled(!autoScrollEnabled)} className="rounded-[8px] cursor-pointer">
                                                        <ArrowDownCircle className={cn("mr-2 h-4 w-4 opacity-70", autoScrollEnabled && "text-[var(--brand-solid)]")} />
                                                        Auto Scroll {autoScrollEnabled ? 'On' : 'Off'}
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem onClick={() => handleSetNotesOpen(!notesOpen)} className="rounded-[8px] cursor-pointer">
                                                        <StickyNote className={cn("mr-2 h-4 w-4 opacity-70", notesOpen && "text-[var(--brand-solid)]")} />
                                                        Notes
                                                    </DropdownMenuItem>
                                                    <DropdownMenuSeparator className="bg-[var(--border-subtle)] my-1" />
                                                    <DropdownMenuItem onClick={() => handleSetChatOpen(!chatOpen)} className="rounded-[8px] cursor-pointer">
                                                        <MessageCircle className={cn("mr-2 h-4 w-4 opacity-70", chatOpen && "text-[var(--brand-solid)]")} />
                                                        Chat with Audio
                                                    </DropdownMenuItem>
                                                    {transcript?.segments?.some((s: TranscriptSegment) => s.speaker) && (
                                                        <DropdownMenuItem onClick={() => setSpeakerRenameOpen(true)} className="rounded-[8px] cursor-pointer">
                                                            <Users className="mr-2 h-4 w-4 opacity-70" />
                                                            Rename Speakers
                                                        </DropdownMenuItem>
                                                    )}
                                                    <DropdownMenuItem onClick={() => setSummaryDialogOpen(true)} className="rounded-[8px] cursor-pointer text-[var(--brand-solid)] focus:text-[var(--brand-solid)] focus:bg-[var(--brand-light)]">
                                                        <Bot className="mr-2 h-4 w-4" /> AI Summary
                                                    </DropdownMenuItem>
                                                    <DropdownMenuSeparator className="bg-[var(--border-subtle)] my-1" />
                                                    <DropdownMenuItem onClick={() => transcript && downloadSRT(transcript, audioFile?.title || 'transcript', speakerMappings)} className="rounded-[8px] cursor-pointer">
                                                        <FileImage className="mr-2 h-4 w-4 opacity-70" /> Download SRT
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem onClick={() => { setDownloadFormat('txt'); setDownloadDialogOpen(true); }} className="rounded-[8px] cursor-pointer">
                                                        <AlignLeft className="mr-2 h-4 w-4 opacity-70" /> Download Text
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem onClick={() => { setDownloadFormat('json'); setDownloadDialogOpen(true); }} className="rounded-[8px] cursor-pointer">
                                                        <FileJson className="mr-2 h-4 w-4 opacity-70" /> Download JSON
                                                    </DropdownMenuItem>
                                                    <DropdownMenuSeparator className="bg-[var(--border-subtle)] my-1" />
                                                    <DropdownMenuItem onClick={() => setExecutionDialogOpen(true)} className="rounded-[8px] cursor-pointer">
                                                        <Activity className="mr-2 h-4 w-4 opacity-70" /> Execution Info
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem onClick={() => setLogsDialogOpen(true)} className="rounded-[8px] cursor-pointer">
                                                        <FileText className="mr-2 h-4 w-4 opacity-70" /> View Logs
                                                    </DropdownMenuItem>
                                                </DropdownMenuContent>
                                            </DropdownMenu>
                                        </div>
                                    </div>
                                </div>

                                    {/* Audio Player */}
                                    <div className="glass-card rounded-[var(--radius-card)] border-[var(--border-subtle)] shadow-[var(--shadow-card)] p-4 md:p-6 mb-8 transition-all duration-300 hover:shadow-[var(--shadow-float)]">
                                        <EmberPlayer
                                            ref={audioPlayerRef}
                                            audioId={audioId}
                                            onTimeUpdate={handleTimeUpdate}
                                            onPlayStateChange={setIsPlaying}
                                        />
                                    </div>
                                </div>

                                {/* Transcript */}
                                <TranscriptSectionWrapper
                                    audioId={audioId}
                                    currentTime={currentTime}
                                    onSeek={handleSeek}
                                    transcript={transcript}
                                    speakerMappings={speakerMappings}
                                    transcriptMode={transcriptMode}
                                    autoScrollEnabled={autoScrollEnabled}
                                    notesOpen={notesOpen}
                                    setNotesOpen={handleSetNotesOpen}
                                    speakerRenameOpen={speakerRenameOpen}
                                    setSpeakerRenameOpen={setSpeakerRenameOpen}
                                    downloadDialogOpen={downloadDialogOpen}
                                    setDownloadDialogOpen={setDownloadDialogOpen}
                                    downloadFormat={downloadFormat}
                                    isPlaying={isPlaying}
                                />
                            </div>
                        </div>
                    </div>
                </main>

                {/* Handlers & Right Pane (Desktop Split) */}
                {chatOpen && !isMobile && (
                    <>
                        {/* Resizer Handle */}
                        <div
                            // 1. Container: Wider hit area (w-3 = 12px) for easy grabbing, transparent bg
                            className="w-1 flex justify-center cursor-col-resize z-30 flex-shrink-0 group relative select-none"
                            onMouseDown={() => setIsResizing(true)}
                        >
                            {/* 2. Visual Line: The actual thin line the user sees */}
                            <div className="w-[1px] h-full bg-[var(--border-subtle)] transition-colors group-hover:bg-[var(--brand-solid)] group-active:bg-[var(--brand-solid)]" />
                        </div>
                        {/* Right Pane */}
                        {/* Right Pane */}
                        <div style={{ width: sidebarWidth }} className="flex-shrink-0 h-full  bg-[var(--bg-card)] z-20">
                            <ChatSidePanel
                                transcriptionId={audioId}
                                isOpen={chatOpen}
                                onClose={() => setChatOpen(false)}
                                isMobile={false}
                            />
                        </div>
                    </>
                )}
            </div>

            {/* Mobile / Overlay Chat (If we want overlay behavior even on desktop, we can adjust logic) */}
            {/* Note: User asked for sliding over on mobile. NotesSidebar handles this via portal internally often, or we do it here.
                 Let's do it here for Chat.
             */}
            {/* If we define isMobile properly (using hook), we can conditional rendering.
                 Since I don't have the hook imported in this snippet yet, I will add it.
              */}


            {/* Dialogs */}
            <ExecutionInfoDialog
                audioId={audioId}
                isOpen={executionDialogOpen}
                onClose={setExecutionDialogOpen}
            />
            <LogsDialog
                audioId={audioId}
                isOpen={logsDialogOpen}
                onClose={setLogsDialogOpen}
            />
            <SummaryDialog
                audioId={audioId}
                isOpen={summaryDialogOpen}
                onClose={setSummaryDialogOpen}
                llmReady={true}
            />

            {/* Mobile / Overlay Chat */}
            {chatOpen && isMobile && createPortal(
                <div className="fixed inset-0 z-[50] flex justify-end bg-background/80 backdrop-blur-sm animate-in fade-in duration-200">
                    <div className="w-full h-full bg-[var(--bg-card)] shadow-2xl animate-in slide-in-from-right duration-300">
                        <ChatSidePanel
                            transcriptionId={audioId}
                            isOpen={chatOpen} onClose={() => setChatOpen(false)}
                            isMobile={true}
                        />
                    </div>
                </div>,
                document.body
            )}
        </div>
    );
};

// Wrapper to handle transcript word index calculation without polluting main view
// Wrapper to handle word index calc
function TranscriptSectionWrapper({ audioId, currentTime, transcript, isPlaying, ...props }: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
    // If transcript not passed (loading?), handle it
    let currentWordIndex = null;
    if (transcript?.word_segments) {
        // Simple linear find for now.
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const idx = transcript.word_segments.findIndex((w: any) => w.start <= currentTime && w.end >= currentTime);
        if (idx !== -1) currentWordIndex = idx;
    }

    return (
        <TranscriptSection
            audioId={audioId}
            currentTime={currentTime}
            currentWordIndex={currentWordIndex}
            transcript={transcript}
            isPlaying={isPlaying}
            className="font-transcript"
            {...props}
        />
    );
}
