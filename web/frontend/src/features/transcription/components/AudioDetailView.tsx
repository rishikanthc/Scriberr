import { useRef, useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { MoreVertical, Edit2, Activity, FileText, Bot, Check, Loader2 } from "lucide-react";
import { Header } from "@/components/Header";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { AudioPlayer } from "@/components/audio/AudioPlayer";
import type { AudioPlayerRef } from "@/components/audio/AudioPlayer";
import { cn } from "@/lib/utils";

// Custom Hooks
import { useAudioDetail, useUpdateTitle, useTranscript } from "@/features/transcription/hooks/useAudioDetail";

// Sub-components
import { TranscriptSection } from "./audio-detail/TranscriptSection";
import { ExecutionInfoDialog } from "./audio-detail/ExecutionInfoDialog";
import { LogsDialog } from "./audio-detail/LogsDialog";
import { SummaryDialog } from "./audio-detail/SummaryDialog";

// Types
interface AudioDetailViewProps {
    audioId?: string; // Optional prop if used as a controlled component, though mainly route-based
}

export const AudioDetailView = function AudioDetailView({ audioId: propAudioId }: AudioDetailViewProps) {
    const { audioId: paramAudioId } = useParams<{ audioId: string }>();
    const audioId = propAudioId || paramAudioId;
    const navigate = useNavigate();

    // Refs
    const audioPlayerRef = useRef<AudioPlayerRef>(null);

    // State
    const [currentTime, setCurrentTime] = useState(0);
    const [, setIsPlaying] = useState(false);
    const [isEditingTitle, setIsEditingTitle] = useState(false);
    const [newTitle, setNewTitle] = useState("");

    // Dialog States
    const [executionDialogOpen, setExecutionDialogOpen] = useState(false);
    const [logsDialogOpen, setLogsDialogOpen] = useState(false);
    const [summaryDialogOpen, setSummaryDialogOpen] = useState(false);

    // Data Fetching
    const { data: audioFile, isLoading, error } = useAudioDetail(audioId || "");
    const { mutate: updateTitle } = useUpdateTitle(audioId || "");

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

    // Helper to format date "Premium" style: "OCT 12, 2023"
    const formattedDate = new Date(audioFile.created_at).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric"
    }).toUpperCase();

    return (
        <div className="h-full flex flex-col bg-[var(--bg-main)] relative selection:bg-[var(--brand-light)]">
            {/* 
              1. Header Redesign:
              - Constrained width (max-w-4xl) to match content
              - Only Logo + Theme Switcher
              - "Invisible" feel (glass background, subtle border)
            */}
            {/* 
              1. Header Redesign:
              - Using shared Header component for consistency
            */}
            <div className="max-w-6xl mx-auto w-full px-6 py-6">
                <Header
                    onFileSelect={() => { }} // No file upload in detail view
                />
            </div>

            {/* Main Content */}
            <main className="flex-1 overflow-y-auto overflow-x-hidden p-6 pb-32">
                <div className="max-w-6xl mx-auto space-y-8">

                    {/* 
                      2. Metadata Section (New):
                      - Back Navigation
                      - Title (Large, Bold)
                      - Metadata Row (Date, Status, Actions)
                      - Generous whitespace
                    */}
                    <div className="space-y-6 pt-4">
                        {/* Top Row: Back Button REMOVED (Redundant) */}


                        {/* Title & Actions Row */}
                        <div className="flex items-start justify-between gap-4">
                            <div className="space-y-3 flex-1 min-w-0">
                                {/* Title with Edit Hover */}
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

                                {/* Metadata Badges */}
                                <div className="flex items-center gap-3 text-xs font-medium uppercase tracking-wider text-[var(--text-tertiary)]">
                                    <span>{formattedDate}</span>
                                    <span className="w-1 h-1 rounded-full bg-[var(--text-tertiary)] opacity-50"></span>
                                    <div className={cn(
                                        "flex items-center gap-1.5 px-2 py-0.5 rounded-full border",
                                        audioFile.status === 'completed' && "border-[var(--success-solid)]/20 text-[var(--success-solid)] bg-[var(--success-translucent)]",
                                        audioFile.status === 'processing' && "border-[var(--brand-solid)]/20 text-[var(--brand-solid)] bg-[var(--brand-light)]",
                                        audioFile.status === 'failed' && "border-[var(--error)]/20 text-[var(--error)] bg-[var(--error)]/10",
                                    )}>
                                        {audioFile.status === 'completed' && <Check className="h-3 w-3" />}
                                        {audioFile.status === 'processing' && <Loader2 className="h-3 w-3 animate-spin" />}
                                        {audioFile.status === 'failed' && <Activity className="h-3 w-3" />}
                                        <span>{audioFile.status}</span>
                                    </div>
                                </div>
                            </div>

                            {/* Action Menu (Floating) */}
                            <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                    <Button
                                        variant="outline"
                                        size="icon"
                                        className="rounded-full border-[var(--border-subtle)] shadow-sm bg-[var(--bg-card)] hover:bg-[var(--bg-main)] hover:border-[var(--border-focus)] transition-all"
                                    >
                                        <MoreVertical className="h-4 w-4 text-[var(--text-secondary)]" />
                                    </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="end" className="w-56 glass-card rounded-[var(--radius-card)] shadow-[var(--shadow-float)] border-[var(--border-subtle)] p-1.5">
                                    <DropdownMenuLabel className="text-xs font-semibold text-[var(--text-tertiary)] uppercase tracking-wider px-2 py-1.5">
                                        Recording Options
                                    </DropdownMenuLabel>
                                    <DropdownMenuItem onClick={() => setExecutionDialogOpen(true)} className="rounded-[8px] cursor-pointer text-[var(--text-secondary)] focus:text-[var(--text-primary)] focus:bg-[var(--bg-main)]/80">
                                        <Activity className="mr-2 h-4 w-4 opacity-70" /> Execution Info
                                    </DropdownMenuItem>
                                    <DropdownMenuItem onClick={() => setLogsDialogOpen(true)} className="rounded-[8px] cursor-pointer text-[var(--text-secondary)] focus:text-[var(--text-primary)] focus:bg-[var(--bg-main)]/80">
                                        <FileText className="mr-2 h-4 w-4 opacity-70" /> View Logs
                                    </DropdownMenuItem>
                                    <DropdownMenuSeparator className="bg-[var(--border-subtle)] my-1" />
                                    <DropdownMenuItem onClick={() => setSummaryDialogOpen(true)} className="rounded-[8px] cursor-pointer text-[var(--brand-solid)] focus:text-[var(--brand-solid)] focus:bg-[var(--brand-light)]">
                                        <Bot className="mr-2 h-4 w-4" /> AI Summary
                                    </DropdownMenuItem>
                                </DropdownMenuContent>
                            </DropdownMenu>
                        </div>
                    </div>

                    {/* 
                      3. Audio Player:
                      - Floating Card style
                      - 1px hairline border
                      - Soft dual-shadow
                    */}
                    <div className="bg-[var(--bg-card)] rounded-[var(--radius-card)] border border-[var(--border-subtle)] shadow-[var(--shadow-card)] p-1 overflow-hidden transition-shadow hover:shadow-[var(--shadow-float)]">
                        <div className="bg-[var(--bg-main)]/50 rounded-[calc(var(--radius-card)-4px)] p-4">
                            <AudioPlayer
                                ref={audioPlayerRef}
                                audioId={audioId}
                                onTimeUpdate={handleTimeUpdate}
                                onPlayStateChange={setIsPlaying}
                            />
                        </div>
                    </div>

                    {/* 
                      4. Transcript Section:
                      - Clean container
                      - Typography handled by component (ensure it uses font-transcript)
                    */}
                    {/* 
                      4. Transcript Section:
                      - Clean container, removed nested card
                    */}
                    <TranscriptSectionWrapper
                        audioId={audioId}
                        currentTime={currentTime}
                        onSeek={handleSeek}
                        onOpenExecutionInfo={() => setExecutionDialogOpen(true)}
                        onOpenLogs={() => setLogsDialogOpen(true)}
                        onOpenSummarize={() => setSummaryDialogOpen(true)}
                        llmReady={true}
                    />
                </div>
            </main>

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
        </div>
    );
};

// Wrapper to handle transcript word index calculation without polluting main view
function TranscriptSectionWrapper({ audioId, currentTime, ...props }: any) {
    const { data: transcript } = useTranscript(audioId, true);

    let currentWordIndex = null;
    if (transcript?.word_segments) {
        // Simple linear find for now.
        const idx = transcript.word_segments.findIndex((w: any) => w.start <= currentTime && w.end >= currentTime);
        if (idx !== -1) currentWordIndex = idx;
    }

    return (
        <TranscriptSection
            audioId={audioId}
            currentTime={currentTime}
            currentWordIndex={currentWordIndex}
            {...props}
        />
    );
}
