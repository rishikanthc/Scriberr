import { useRef, useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeft, MoreVertical, Edit2, Activity, FileText, Bot } from "lucide-react";

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

    return (
        <div className="h-full flex flex-col bg-background relative selection:bg-primary/20">
            {/* Header */}
            <header className="flex-none bg-background/80 backdrop-blur-md border-b border-border p-4 z-20">
                <div className="max-w-7xl mx-auto flex items-center justify-between gap-4">
                    <div className="flex items-center gap-4 min-w-0">
                        <Button variant="ghost" size="icon" onClick={() => navigate('/dashboard')} className="shrink-0 rounded-full hover:bg-carbon-100 dark:hover:bg-carbon-800">
                            <ArrowLeft className="h-5 w-5" />
                        </Button>

                        <div className="min-w-0 flex flex-col">
                            {isEditingTitle ? (
                                <Input
                                    value={newTitle}
                                    onChange={(e) => setNewTitle(e.target.value)}
                                    onBlur={handleTitleSave}
                                    onKeyDown={(e) => e.key === 'Enter' && handleTitleSave()}
                                    className="h-8 py-1 text-lg font-semibold"
                                    autoFocus
                                />
                            ) : (
                                <div className="flex items-center gap-2 group cursor-pointer" onClick={() => setIsEditingTitle(true)}>
                                    <h1 className="text-lg font-semibold truncate text-foreground">{audioFile.title || "Untitled"}</h1>
                                    <Edit2 className="h-3.5 w-3.5 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
                                </div>
                            )}
                            <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                <span>{new Date(audioFile.created_at).toLocaleDateString()}</span>
                                <span>•</span>
                                <span className={cn(
                                    "px-1.5 py-0.5 rounded-full font-medium",
                                    audioFile.status === 'completed' && "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
                                    audioFile.status === 'processing' && "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
                                    audioFile.status === 'failed' && "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
                                )}>
                                    {audioFile.status.charAt(0).toUpperCase() + audioFile.status.slice(1)}
                                </span>
                            </div>
                        </div>
                    </div>

                    <div className="flex items-center gap-2">
                        <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="icon" className="rounded-full">
                                    <MoreVertical className="h-5 w-5" />
                                </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                                <DropdownMenuLabel>Actions</DropdownMenuLabel>
                                <DropdownMenuItem onClick={() => setExecutionDialogOpen(true)}>
                                    <Activity className="mr-2 h-4 w-4" /> View Execution Info
                                </DropdownMenuItem>
                                <DropdownMenuItem onClick={() => setLogsDialogOpen(true)}>
                                    <FileText className="mr-2 h-4 w-4" /> View Logs
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem onClick={() => setSummaryDialogOpen(true)}>
                                    <Bot className="mr-2 h-4 w-4" /> Generate Summary
                                </DropdownMenuItem>
                            </DropdownMenuContent>
                        </DropdownMenu>
                    </div>
                </div>
            </header>

            {/* Main Content */}
            <main className="flex-1 overflow-y-auto overflow-x-hidden p-4 md:p-6 pb-32">
                <div className="max-w-4xl mx-auto space-y-6">
                    {/* Audio Player */}
                    <div className="glass-panel p-4 rounded-xl sticky top-2 z-30 shadow-md">
                        <AudioPlayer
                            ref={audioPlayerRef}
                            audioId={audioId}
                            onTimeUpdate={handleTimeUpdate}
                            onPlayStateChange={setIsPlaying}
                        />
                    </div>

                    {/* Transcript Section */}
                    <div className="min-h-[500px]">
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
