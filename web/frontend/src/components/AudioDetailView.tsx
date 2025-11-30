import { useState, useEffect, useRef, memo } from "react";
import { createPortal } from "react-dom";
import { ArrowLeft, Play, Pause, List, AlignLeft, MessageCircle, Download, FileText, FileJson, FileImage, Check, StickyNote, Plus, X, Sparkles, Pencil, ChevronUp, ChevronDown, Info, Clock, Settings, Users, Loader2 } from "lucide-react";
import { AudioPlayer, type AudioPlayerRef } from "./audio/AudioPlayer";
import { TranscriptView } from "./transcript/TranscriptView";
import { Button } from "./ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "./ui/dropdown-menu";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "./ui/dialog";
import { Label } from "./ui/label";
import { Switch } from "./ui/switch";
import { useRouter } from "../contexts/RouterContext";
import { ThemeSwitcher } from "./ThemeSwitcher";

import { useAuth } from "../contexts/AuthContext";
import { ChatInterface } from "./ChatInterface";
import type { Note } from "../types/note";
import { NotesSidebar } from "./NotesSidebar";
import SpeakerRenameDialog from "./SpeakerRenameDialog";
import ReactMarkdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import rehypeHighlight from 'rehype-highlight';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import { Dialog as UIDialog, DialogContent as UIDialogContent, DialogHeader as UIDialogHeader, DialogTitle as UIDialogTitle, DialogDescription as UIDialogDescription } from "./ui/dialog";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "./ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";
import { Label as UILabel } from "./ui/label";
import { useToast } from "./ui/toast";
import { MergeStatusBadge } from "./MergeStatusBadge";

interface MultiTrackFile {
    id: number;
    file_name: string;
    file_path: string;
    track_index: number;
}

interface MultiTrackTiming {
    track_name: string;
    start_time: string;
    end_time: string;
    duration: number; // milliseconds
}

interface ExecutionData {
    id: string;
    transcription_job_id: string;
    started_at: string;
    completed_at: string | null;
    processing_duration: number | null; // milliseconds
    actual_parameters?: any;
    status: string;
    error_message?: string | null;
    created_at: string;
    updated_at: string;
    // Multi-track specific fields
    is_multi_track?: boolean;
    multi_track_timings?: MultiTrackTiming[];
    merge_start_time?: string | null;
    merge_end_time?: string | null;
    merge_duration?: number | null; // milliseconds
    multi_track_files?: MultiTrackFile[];
}

interface AudioFile {
    id: string;
    title?: string;
    status: "uploaded" | "pending" | "processing" | "completed" | "failed";
    created_at: string;
    audio_path: string;
    diarization?: boolean;
    is_multi_track?: boolean;
    multi_track_files?: MultiTrackFile[];
    merged_audio_path?: string;
    merge_status?: string;
    merge_error?: string;
    parameters?: {
        diarize?: boolean;
        [key: string]: any;
    };
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

interface AudioDetailViewProps {
    audioId: string;
}

// Helper function to get display name for diarization model
const getDiarizationModelDisplayName = (model: string): string => {
    switch (model) {
        case 'pyannote':
            return 'PyAnnote Speaker Diarization 3.1';
        case 'nvidia_sortformer':
            return 'NVIDIA Sortformer 4-Speaker v2';
        default:
            // Fallback to the raw value if unknown
            return model || 'Unknown';
    }
};

// Helper function to calculate audio duration from transcript segments
const getAudioDurationFromTranscript = (transcript: Transcript | null): number | null => {
    if (!transcript) return null;

    // Try word segments first (most accurate)
    if (transcript.word_segments && transcript.word_segments.length > 0) {
        const lastWord = transcript.word_segments[transcript.word_segments.length - 1];
        return lastWord.end;
    }

    // Fall back to segments
    if (transcript.segments && transcript.segments.length > 0) {
        const lastSegment = transcript.segments[transcript.segments.length - 1];
        return lastSegment.end;
    }

    return null;
};

// Helper function to format duration in seconds to a readable format
const formatDuration = (seconds: number): string => {
    if (seconds < 60) {
        return `${seconds.toFixed(1)}s`;
    }

    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;

    if (minutes < 60) {
        return `${minutes}m ${remainingSeconds.toFixed(0)}s`;
    }

    const hours = Math.floor(minutes / 60);
    const remainingMinutes = minutes % 60;
    return `${hours}h ${remainingMinutes}m ${remainingSeconds.toFixed(0)}s`;
};

const formatElapsedTime = (seconds: number): string => {
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
};

export const AudioDetailView = memo(function AudioDetailView({ audioId }: AudioDetailViewProps) {
    const { navigate } = useRouter();
    const { getAuthHeaders } = useAuth();
    const [audioFile, setAudioFile] = useState<AudioFile | null>(null);
    const [transcript, setTranscript] = useState<Transcript | null>(null);

    // Debug transcript changes
    useEffect(() => {
        console.log("[DEBUG] *** TRANSCRIPT STATE CHANGED ***");
        console.log("[DEBUG] transcript:", transcript);
        console.log("[DEBUG] has word_segments:", !!transcript?.word_segments);
        console.log("[DEBUG] word_segments length:", transcript?.word_segments?.length);
        console.log("[DEBUG] transcript.text length:", transcript?.text?.length);
    }, [transcript]);
    const [loading, setLoading] = useState(true);
    const [isPlaying, setIsPlaying] = useState(false);
    const [transcriptMode, setTranscriptMode] = useState<"compact" | "expanded">(
        "compact",
    );
    const [viewMode, setViewMode] = useState<"transcript" | "chat">("transcript");
    const [currentTime, setCurrentTime] = useState(0);
    const [currentWordIndex, setCurrentWordIndex] = useState<number | null>(null);
    const [downloadDialogOpen, setDownloadDialogOpen] = useState(false);
    const [downloadFormat, setDownloadFormat] = useState<'txt' | 'json'>('txt');
    const [includeSpeakerLabels, setIncludeSpeakerLabels] = useState(true);
    const [includeTimestamps, setIncludeTimestamps] = useState(true);

    // Speaker renaming state
    const [speakerRenameDialogOpen, setSpeakerRenameDialogOpen] = useState(false);
    const [speakerMappings, setSpeakerMappings] = useState<Record<string, string>>({});

    // Polling state
    const [pollingInterval, setPollingInterval] = useState<NodeJS.Timeout | null>(null);
    const [processingStartTime, setProcessingStartTime] = useState<Date | null>(null);
    const [elapsedTime, setElapsedTime] = useState<number>(0);
    const [currentStatus, setCurrentStatus] = useState<string | null>(null);
    const [duration, setDuration] = useState(0);
    const audioPlayerRef = useRef<AudioPlayerRef>(null);
    const transcriptRef = useRef<HTMLDivElement>(null);
    const highlightedWordRef = useRef<HTMLSpanElement>(null);
    const audioSectionRef = useRef<HTMLDivElement>(null);

    // Notes state
    const [notes, setNotes] = useState<Note[]>([]);

    const sortNotes = (list: Note[]) => {
        return [...list].sort((a, b) => {
            if (a.start_time !== b.start_time) return a.start_time - b.start_time;
            if (a.start_word_index !== b.start_word_index) return a.start_word_index - b.start_word_index;
            // Fallback stable tiebreaker by created_at
            return (a.created_at || '').localeCompare(b.created_at || '');
        });
    };
    const [notesOpen, setNotesOpen] = useState(false);
    const [showSelectionMenu, setShowSelectionMenu] = useState(false);
    const [pendingSelection, setPendingSelection] = useState<{ startIdx: number; endIdx: number; startTime: number; endTime: number; quote: string } | null>(null);
    const [newNoteContent, setNewNoteContent] = useState("");
    const [showEditor, setShowEditor] = useState(false);

    // Summarization state
    type SummTpl = { id: string; name: string; model: string; prompt: string; };
    const [summarizeOpen, setSummarizeOpen] = useState(false);
    const [templates, setTemplates] = useState<SummTpl[]>([]);
    const [templatesLoading, setTemplatesLoading] = useState(false);
    const [selectedTemplateId, setSelectedTemplateId] = useState<string>("");
    const selectedTemplate: SummTpl | undefined = templates.find(t => t.id === selectedTemplateId);
    const [tplPopoverOpen, setTplPopoverOpen] = useState(false);
    const [summaryOpen, setSummaryOpen] = useState(false);
    const [summaryStream, setSummaryStream] = useState("");
    const [isSummarizing, setIsSummarizing] = useState(false);
    const [summaryError, setSummaryError] = useState<string | null>(null);
    const [llmReady, setLlmReady] = useState<boolean | null>(null);
    const [selectionViewportPos, setSelectionViewportPos] = useState<{ x: number, y: number }>({ x: 0, y: 0 });
    const { toast } = useToast();
    const [editingTitle, setEditingTitle] = useState(false);
    const [titleInput, setTitleInput] = useState("");
    const [savingTitle, setSavingTitle] = useState(false);
    const [audioCollapsed, setAudioCollapsed] = useState(false);

    // Execution info state
    const [executionInfoOpen, setExecutionInfoOpen] = useState(false);
    const [executionData, setExecutionData] = useState<ExecutionData | null>(null);
    const [executionDataLoading, setExecutionDataLoading] = useState(false);

    // Logs state
    const [logsOpen, setLogsOpen] = useState(false);
    const [logsContent, setLogsContent] = useState<string | null>(null);
    const [logsLoading, setLogsLoading] = useState(false);

    // Persist collapsed state per transcription
    useEffect(() => {
        try {
            const key = `scriberr.audioCollapsed.${audioId}`;
            const saved = localStorage.getItem(key);
            if (saved !== null) setAudioCollapsed(saved === '1');
        } catch { }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [audioId]);

    useEffect(() => {
        try {
            const key = `scriberr.audioCollapsed.${audioId}`;
            localStorage.setItem(key, audioCollapsed ? '1' : '0');
        } catch { }
    }, [audioId, audioCollapsed]);

    useEffect(() => {
        fetchAudioDetails();
        fetchNotes();
        // Check LLM configured status for gating
        (async () => {
            try {
                const res = await fetch('/api/v1/llm/config', { headers: { ...getAuthHeaders() } });
                if (res.ok) {
                    const cfg = await res.json();
                    setLlmReady(!!cfg && cfg.is_active);
                } else {
                    setLlmReady(false);
                }
            } catch { setLlmReady(false); }
        })();
    }, [audioId]);

    // Polling mechanism for status updates
    useEffect(() => {
        // Start polling if job is processing or pending
        const status = currentStatus || audioFile?.status;
        if (audioFile && status && (status === "processing" || status === "pending")) {
            // Set processing start time if not already set
            if (!processingStartTime && status === "processing") {
                setProcessingStartTime(new Date());
            }

            // Clear any existing interval
            if (pollingInterval) {
                clearInterval(pollingInterval);
            }

            // Start polling every 3 seconds
            const interval = setInterval(async () => {
                await fetchStatusOnly();
            }, 3000);

            setPollingInterval(interval);

            // Cleanup interval on unmount or when status changes
            return () => {
                if (interval) {
                    clearInterval(interval);
                }
            };
        } else {
            // Stop polling if status is completed, failed, or uploaded
            if (pollingInterval) {
                clearInterval(pollingInterval);
                setPollingInterval(null);
            }
            // Clear processing start time if completed or failed
            const status = currentStatus || audioFile?.status;
            if (status && (status === "completed" || status === "failed")) {
                setProcessingStartTime(null);
                setElapsedTime(0);
            }
        }
    }, [currentStatus, audioFile?.status, audioId]); // Re-run when status or audioId changes

    // Update elapsed time counter
    useEffect(() => {
        const status = currentStatus || audioFile?.status;
        if (processingStartTime && status === "processing") {
            const timer = setInterval(() => {
                const now = new Date();
                const elapsed = Math.floor((now.getTime() - processingStartTime.getTime()) / 1000);
                setElapsedTime(elapsed);
            }, 1000);

            return () => clearInterval(timer);
        }
    }, [processingStartTime, currentStatus, audioFile?.status]);

    // Cleanup polling on unmount
    useEffect(() => {
        return () => {
            if (pollingInterval) {
                clearInterval(pollingInterval);
            }
        };
    }, [pollingInterval]);

    // Fetch speaker mappings when audio file is loaded and has diarization enabled
    useEffect(() => {
        if (audioFile) {
            // Clear existing mappings first (in case of retranscription)
            setSpeakerMappings({});
            fetchSpeakerMappings();
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [audioFile?.id]);

    // Also clear and refetch speaker mappings when transcript changes (handles retranscription)
    useEffect(() => {
        if (transcript && audioFile) {
            setSpeakerMappings({});
            fetchSpeakerMappings();
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [transcript]);

    // Former floating-controls visibility logic removed: controls are always fixed.

    // WaveSurfer initialization logic removed - handled by AudioPlayer component

    // Update current word index based on audio time
    useEffect(() => {
        if (!transcript?.word_segments) {
            return;
        }

        // Find the word that should be highlighted based on current time
        const currentWordIdx = transcript.word_segments.findIndex(
            (word) => {
                // Use word's end time for more accurate highlighting
                return currentTime >= word.start && currentTime <= word.end;
            }
        );

        // If no exact match, find the closest upcoming word
        const fallbackWordIdx = currentWordIdx === -1
            ? transcript.word_segments.findIndex(word => word.start > currentTime) - 1
            : currentWordIdx;

        const finalWordIdx = Math.max(0, fallbackWordIdx);

        if (finalWordIdx !== currentWordIndex && (isPlaying || currentTime > 0)) {
            setCurrentWordIndex(finalWordIdx);
        }

        // Clear highlighting when audio is stopped and at the beginning
        if (!isPlaying && currentTime === 0) {
            setCurrentWordIndex(null);
        }
    }, [currentTime, transcript?.word_segments, isPlaying]);

    // Auto-scroll to highlighted word
    useEffect(() => {
        if (currentWordIndex !== null && highlightedWordRef.current) {
            const highlightedElement = highlightedWordRef.current;

            // Check if the highlighted word is outside the visible viewport
            const highlightedRect = highlightedElement.getBoundingClientRect();
            const viewportHeight = window.innerHeight;

            // Consider the word out of view if it's too close to the top or bottom edges
            // This provides a buffer so the word isn't right at the edge
            const buffer = viewportHeight * 0.2; // 20% buffer
            const isAboveView = highlightedRect.top < buffer;
            const isBelowView = highlightedRect.bottom > (viewportHeight - buffer);

            if (isAboveView || isBelowView) {
                highlightedElement.scrollIntoView({
                    behavior: 'smooth',
                    block: 'center',
                });
            }
        }
    }, [currentWordIndex]);

    const fetchTranscriptOnly = async () => {
        console.log("[DEBUG] *** fetchTranscriptOnly CALLED ***");
        try {
            const transcriptResponse = await fetch(
                `/api/v1/transcription/${audioId}/transcript`,
                {
                    headers: {
                        ...getAuthHeaders(),
                    },
                },
            );

            if (transcriptResponse.ok) {
                const transcriptData = await transcriptResponse.json();
                console.log("[DEBUG] fetchTranscriptOnly - transcriptData:", transcriptData);

                // The API returns transcript data in a nested structure
                if (transcriptData.transcript) {
                    console.log("[DEBUG] transcript has word_segments:", !!transcriptData.transcript.word_segments);
                    console.log("[DEBUG] word_segments length:", transcriptData.transcript.word_segments?.length);
                    // Check if transcript has segments or text
                    if (typeof transcriptData.transcript === "string") {
                        console.log("[DEBUG] Setting transcript as STRING");
                        setTranscript({ text: transcriptData.transcript });
                    } else if (transcriptData.transcript.text) {
                        console.log("[DEBUG] Setting transcript with TEXT and word_segments");
                        setTranscript({
                            text: transcriptData.transcript.text,
                            segments: transcriptData.transcript.segments,
                            word_segments: transcriptData.transcript.word_segments,
                        });
                    } else if (transcriptData.transcript.segments) {
                        console.log("[DEBUG] Setting transcript with SEGMENTS and word_segments");
                        setTranscript({
                            text: "",
                            segments: transcriptData.transcript.segments,
                            word_segments: transcriptData.transcript.word_segments,
                        });
                    }
                }
            }
        } catch (error) {
            console.error("Error fetching transcript:", error);
        }
    };

    const fetchStatusOnly = async () => {
        try {
            const response = await fetch(`/api/v1/transcription/${audioId}`, {
                headers: {
                    ...getAuthHeaders(),
                },
            });

            if (response.ok) {
                const data = await response.json();
                const previousStatus = currentStatus || audioFile?.status;

                // Only update the status state, not the entire audioFile
                setCurrentStatus(data.status);

                // If status changed to completed, update audioFile status and fetch transcript
                if (data.status === "completed" && previousStatus === "processing") {
                    setAudioFile(prev => prev ? { ...prev, status: "completed" } : null);
                    await fetchTranscriptOnly();
                }
            }
        } catch (error) {
            console.error('Error fetching status:', error);
        }
    };

    const fetchAudioDetails = async () => {
        try {
            // Fetch audio file details
            const audioResponse = await fetch(`/api/v1/transcription/${audioId}`, {
                headers: {
                    ...getAuthHeaders(),
                },
            });


            if (audioResponse.ok) {
                const audioData = await audioResponse.json();
                setAudioFile(audioData);
                setCurrentStatus(audioData.status);

                // Fetch transcript if completed
                if (audioData.status === "completed") {
                    const transcriptResponse = await fetch(
                        `/api/v1/transcription/${audioId}/transcript`,
                        {
                            headers: {
                                ...getAuthHeaders(),
                            },
                        },
                    );

                    if (transcriptResponse.ok) {
                        const transcriptData = await transcriptResponse.json();
                        console.log("[DEBUG] *** fetchAudioDetails TRANSCRIPT LOADING ***");
                        console.log("[DEBUG] initial transcriptData:", transcriptData);

                        // The API returns transcript data in a nested structure
                        if (transcriptData.transcript) {
                            console.log("[DEBUG] initial transcript has word_segments:", !!transcriptData.transcript.word_segments);
                            console.log("[DEBUG] initial word_segments length:", transcriptData.transcript.word_segments?.length);
                            // Check if transcript has segments or text
                            if (typeof transcriptData.transcript === "string") {
                                console.log("[DEBUG] INITIAL: Setting transcript as STRING");
                                setTranscript({ text: transcriptData.transcript });
                            } else if (transcriptData.transcript.text) {
                                console.log("[DEBUG] INITIAL: Setting transcript with TEXT and word_segments");
                                setTranscript({
                                    text: transcriptData.transcript.text,
                                    segments: transcriptData.transcript.segments,
                                    word_segments: transcriptData.transcript.word_segments,
                                });
                            } else if (transcriptData.transcript.segments) {
                                console.log("[DEBUG] INITIAL: Setting transcript with SEGMENTS only");
                                // If only segments, combine them into text
                                const fullText = transcriptData.transcript.segments
                                    .map((segment: any) => segment.text)
                                    .join(" ");
                                setTranscript({
                                    text: fullText,
                                    segments: transcriptData.transcript.segments,
                                    word_segments: transcriptData.transcript.word_segments,
                                });
                            }
                        }
                    } else {
                        console.error(
                            "Failed to fetch transcript:",
                            transcriptResponse.status,
                        );
                    }
                }
            } else {
                console.error("Failed to fetch audio details:", audioResponse.status);
            }
        } catch (error) {
            console.error("Failed to fetch audio details:", error);
        } finally {
            setLoading(false);
        }
    };

    const fetchNotes = async () => {
        try {
            const res = await fetch(`/api/v1/transcription/${audioId}/notes`, { headers: { ...getAuthHeaders() } });
            if (res.ok) {
                const data = await res.json();
                setNotes(sortNotes(data));
            }
        } catch (e) { console.error("Failed to fetch notes", e); }
    };

    const fetchExecutionData = async () => {
        if (executionData) return; // Already loaded
        setExecutionDataLoading(true);
        try {
            const res = await fetch(`/api/v1/transcription/${audioId}/execution`, {
                headers: { ...getAuthHeaders() }
            });
            if (res.ok) {
                const data = await res.json();
                setExecutionData(data);
            } else {
            }
        } catch (e) {
            console.error("Failed to fetch execution data", e);
        } finally {
            setExecutionDataLoading(false);
        }
    };

    const fetchLogs = async () => {
        if (logsContent) return; // Already loaded
        setLogsLoading(true);
        try {
            const res = await fetch(`/api/v1/transcription/${audioId}/logs`, {
                headers: { ...getAuthHeaders() }
            });
            if (res.ok) {
                const text = await res.text();
                setLogsContent(text);
            } else {
                setLogsContent("Failed to load logs.");
            }
        } catch (e) {
            console.error("Failed to fetch logs", e);
            setLogsContent("Error loading logs.");
        } finally {
            setLogsLoading(false);
        }
    };

    const openLogsDialog = () => {
        setLogsOpen(true);
        fetchLogs();
    };

    const openExecutionInfo = () => {
        setExecutionInfoOpen(true);
        fetchExecutionData();
    };

    const togglePlayPause = () => {
        audioPlayerRef.current?.playPause();
    };

    const handleBack = () => {
        navigate({ path: "home" });
    };

    // Selection handling for annotation
    useEffect(() => {
        const el = transcriptRef.current;
        if (!el) return;

        const onMouseUp = () => {
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

            const range = sel.getRangeAt(0);
            const rect = range.getBoundingClientRect();
            // Robust viewport coordinates for the selection UI
            // Clamp X within viewport with 16px gutters
            const centerX = rect.left + rect.width / 2;
            const clampedX = Math.min(window.innerWidth - 16, Math.max(16, centerX));
            // Prefer above selection; if too close to the top, place below
            let bubbleY = rect.top - 10;
            if (bubbleY < 12) {
                bubbleY = rect.bottom + 8;
            }
            setSelectionViewportPos({ x: clampedX, y: bubbleY });
            setPendingSelection({ startIdx, endIdx, startTime, endTime, quote });
            setShowSelectionMenu(true);
        };

        el.addEventListener('mouseup', onMouseUp);
        return () => el.removeEventListener('mouseup', onMouseUp);
    }, [transcript, transcriptMode]);

    // Cmd/Ctrl + click to seek to word start (without breaking selection or follow-along)
    // Attach the handler when the transcript DOM content is present (not on ref.current changes)
    useEffect(() => {
        const el = transcriptRef.current;
        if (!el) return;
        const onClick = (e: MouseEvent) => {
            // Only handle when meta/ctrl is pressed; otherwise let normal selection/add-note flow work
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
            // Use the audioPlayerRef
            const player = audioPlayerRef.current;
            if (player) {
                const dur = player.getDuration() || 1;
                const ratio = Math.min(0.999, Math.max(0, start / dur));
                player.seekTo(ratio);
                setCurrentTime(start);
            }
        };
        el.addEventListener('click', onClick);
        return () => el.removeEventListener('click', onClick);
    }, [transcript, transcriptMode]);

    // Hide selection bubble when selection collapses (and not editing)
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

    const openEditorForSelection = () => {
        setShowEditor(true);
        setShowSelectionMenu(false);
        setNewNoteContent("");
    };

    const saveNewNote = async () => {
        if (!pendingSelection) return;
        try {
            const res = await fetch(`/api/v1/transcription/${audioId}/notes`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
                body: JSON.stringify({
                    start_word_index: pendingSelection.startIdx,
                    end_word_index: pendingSelection.endIdx,
                    start_time: pendingSelection.startTime,
                    end_time: pendingSelection.endTime,
                    quote: pendingSelection.quote,
                    content: newNoteContent.trim() || pendingSelection.quote,
                }),
            });
            if (res.ok) {
                const created = await res.json();
                setNotes(prev => sortNotes([...(prev || []), created]));
                setShowEditor(false);
                setPendingSelection(null);
                const sel = window.getSelection(); sel?.removeAllRanges();
            }
        } catch (e) { console.error('Failed to create note', e); }
    };

    const updateNote = async (id: string, newContent: string) => {
        await fetch(`/api/v1/notes/${id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
            body: JSON.stringify({ content: newContent }),
        });
        setNotes(prev => prev.map(n => n.id === id ? { ...n, content: newContent } : n));
    };

    const deleteNote = async (id: string) => {
        await fetch(`/api/v1/notes/${id}`, { method: 'DELETE', headers: { ...getAuthHeaders() } });
        setNotes(prev => prev.filter(n => n.id !== id));
    };


    // Render transcript with word-level highlighting
    // Logic moved to TranscriptView

    // Segment rendering logic moved to TranscriptView

    const getFileName = (audioPath: string) => {
        const parts = audioPath.split("/");
        return parts[parts.length - 1];
    };

    const getFullTranscriptText = (): string => {
        if (!transcript) return '';
        if (transcript.segments && transcript.segments.length > 0) {
            return transcript.segments.map(s => s.text.trim()).join('\n');
        }
        return transcript.text || '';
    };

    const toggleAudioCollapsed = () => {
        if (!audioCollapsed && audioPlayerRef.current && isPlaying) {
            // Pause when collapsing
            try { audioPlayerRef.current.playPause(); } catch { }
        }
        setAudioCollapsed(v => !v);
    };

    const saveTitle = async () => {
        if (!audioFile) { setEditingTitle(false); return; }
        const currentDisplay = audioFile.title || getFileName(audioFile.audio_path);
        const trimmed = titleInput.trim();
        if (!trimmed || trimmed === currentDisplay) { setEditingTitle(false); return; }
        setSavingTitle(true);
        try {
            const res = await fetch(`/api/v1/transcription/${audioFile.id}/title`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
                body: JSON.stringify({ title: trimmed }),
            });
            if (res.ok) {
                const data = await res.json();
                setAudioFile(prev => prev ? { ...prev, title: data.title } : prev);
                toast({ title: 'Title updated' });
            } else {
                const msg = await res.text();
                toast({ title: 'Failed to update title', description: msg });
            }
        } catch (e) {
            toast({ title: 'Failed to update title' });
        } finally {
            setSavingTitle(false);
            setEditingTitle(false);
        }
    };

    const openSummarizeDialog = async () => {
        if (llmReady === false) return;
        // If a summary already exists for this transcription, show it directly
        try {
            const resExisting = await fetch(`/api/v1/transcription/${audioId}/summary`, { headers: { ...getAuthHeaders() } });
            if (resExisting.ok) {
                const data = await resExisting.json();
                setSummaryStream(data.content || '');
                setSummaryOpen(true);
                return;
            }
        } catch { }
        // Else open template picker
        setSummarizeOpen(true);
        if (templates.length === 0) {
            try {
                setTemplatesLoading(true);
                const res = await fetch('/api/v1/summaries', { headers: { ...getAuthHeaders() } });
                if (res.ok) {
                    const data = await res.json();
                    setTemplates(data || []);
                }
            } finally { setTemplatesLoading(false); }
        }
    };

    const startSummarization = async () => {
        const tpl = templates.find(t => t.id === selectedTemplateId);
        if (!tpl) return;
        const transcriptText = getFullTranscriptText();
        const combined = `Transcript:\n${transcriptText}\n\nInstructions:\n${tpl.prompt}`;
        setSummaryOpen(true);
        setSummaryStream("");
        setSummaryError(null);
        setIsSummarizing(true);
        try {
            const res = await fetch('/api/v1/summarize', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
                body: JSON.stringify({ model: tpl.model, content: combined, transcription_id: audioId, template_id: tpl.id }),
            });
            if (!res.body) {
                setIsSummarizing(false);
                setSummaryError('Failed to start summary stream.');
                toast({ title: 'Summary failed', description: 'Failed to start summary stream.' });
                return;
            }
            const reader = res.body.getReader();
            // Use streaming decode to avoid dropping multi-byte characters across chunks
            const decoder = new TextDecoder();
            let receivedAny = false;
            while (true) {
                const { done, value } = await reader.read();
                if (done) {
                    // Flush any remaining bytes in the decoder
                    const tail = decoder.decode();
                    if (tail) { setSummaryStream(prev => prev + tail); receivedAny = true; }
                    break;
                }
                const chunk = decoder.decode(value, { stream: true });
                if (chunk) { setSummaryStream(prev => prev + chunk); receivedAny = true; }
            }
            if (!receivedAny) {
                setSummaryError('No content returned by the model.');
                toast({ title: 'Summary failed', description: 'No content returned by the model.' });
            }
        } catch (e) {
            setSummaryError('Summary generation failed. Please try again.');
            toast({ title: 'Summary failed', description: 'Summary generation failed. Please try again.' });
        } finally {
            setIsSummarizing(false);
        }
    };

    const formatTimestamp = (seconds: number): string => {
        const minutes = Math.floor(seconds / 60);
        const remainingSeconds = Math.floor(seconds % 60);
        return `${minutes}:${remainingSeconds.toString().padStart(2, "0")}`;
    };

    const formatDate = (dateString: string) => {
        return new Date(dateString).toLocaleDateString("en-US", {
            year: "numeric",
            month: "long",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
        });
    };

    // Download functions
    const downloadSRT = () => {
        if (!transcript) return;

        let srtContent = '';
        let counter = 1;

        if (transcript.segments) {
            // Use segments for SRT format
            transcript.segments.forEach((segment) => {
                const startTime = formatSRTTime(segment.start);
                const endTime = formatSRTTime(segment.end);
                let text = segment.text.trim();

                // Add speaker label if available (common practice in SRT files)
                if (segment.speaker) {
                    text = `${getDisplaySpeakerName(segment.speaker)}: ${text}`;
                }

                srtContent += `${counter}\n${startTime} --> ${endTime}\n${text}\n\n`;
                counter++;
            });
        } else {
            // If no segments, create one entry with the full text
            srtContent = `1\n00:00:00,000 --> 99:59:59,999\n${transcript.text}\n\n`;
        }

        downloadFile(srtContent, `${getFileNameWithoutExt()}.srt`, 'text/plain');
    };

    const downloadTXT = () => {
        if (!transcript) return;

        let content = '';

        if (!includeSpeakerLabels && !includeTimestamps) {
            // Simple paragraph format
            content = transcript.text;
        } else if (transcript.segments) {
            // Segmented format with optional labels/timestamps
            transcript.segments.forEach((segment, index) => {
                if (index > 0) content += '\n\n';

                // Add timestamp if enabled
                if (includeTimestamps) {
                    content += `[${formatTimestamp(segment.start)}] `;
                }

                // Add speaker if enabled and available
                if (includeSpeakerLabels && segment.speaker) {
                    content += `${getDisplaySpeakerName(segment.speaker)}: `;
                }

                content += segment.text.trim();
            });
        } else {
            // No segments, just use the full text
            content = transcript.text;
        }

        downloadFile(content, `${getFileNameWithoutExt()}.txt`, 'text/plain');
    };

    const downloadJSON = () => {
        if (!transcript) return;

        let jsonData: any;

        if (!includeSpeakerLabels && !includeTimestamps) {
            // Simple text format
            jsonData = {
                text: transcript.text,
                format: 'simple'
            };
        } else if (transcript.segments) {
            // Detailed format with segments
            jsonData = {
                text: transcript.text,
                format: 'segmented',
                segments: transcript.segments.map(segment => {
                    const segmentData: any = {
                        text: segment.text.trim()
                    };

                    if (includeTimestamps) {
                        segmentData.start = segment.start;
                        segmentData.end = segment.end;
                        segmentData.timestamp = formatTimestamp(segment.start);
                    }

                    if (includeSpeakerLabels && segment.speaker) {
                        segmentData.speaker = getDisplaySpeakerName(segment.speaker);
                    }

                    return segmentData;
                })
            };
        } else {
            // No segments available
            jsonData = {
                text: transcript.text,
                format: 'simple'
            };
        }

        downloadFile(JSON.stringify(jsonData, null, 2), `${getFileNameWithoutExt()}.json`, 'application/json');
    };

    const formatSRTTime = (seconds: number): string => {
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = Math.floor(seconds % 60);
        const milliseconds = Math.floor((seconds % 1) * 1000);

        return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')},${milliseconds.toString().padStart(3, '0')}`;
    };

    const downloadFile = (content: string, filename: string, contentType: string) => {
        const blob = new Blob([content], { type: contentType });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
    };

    const getFileNameWithoutExt = (): string => {
        const name = audioFile?.title || getFileName(audioFile?.audio_path || '');
        return name.replace(/\.[^/.]+$/, '') || 'transcript';
    };

    const handleDownloadWithDialog = (format: 'txt' | 'json') => {
        setDownloadFormat(format);
        setDownloadDialogOpen(true);
    };

    const handleDownloadConfirm = () => {
        if (downloadFormat === 'txt') {
            downloadTXT();
        } else {
            downloadJSON();
        }
        setDownloadDialogOpen(false);
    };

    // Speaker helper functions
    const getDetectedSpeakers = (): string[] => {
        if (!transcript?.segments) return [];
        const speakers = new Set<string>();
        transcript.segments.forEach(segment => {
            if (segment.speaker) {
                speakers.add(segment.speaker);
            }
        });
        return Array.from(speakers).sort();
    };

    const hasSpeakers = (): boolean => {
        // Return true if diarization is enabled OR if this is a multi-track job (which also has speakers)
        return audioFile?.diarization || audioFile?.parameters?.diarize || audioFile?.is_multi_track || false;
    };

    const handleSpeakerMappingsUpdate = (mappings: { id?: number; original_speaker: string; custom_name: string }[]) => {
        // Convert the array of mappings to a lookup object
        const mappingObj: Record<string, string> = {};
        mappings.forEach(mapping => {
            mappingObj[mapping.original_speaker] = mapping.custom_name;
        });
        setSpeakerMappings(mappingObj);
    };

    const getDisplaySpeakerName = (originalSpeaker: string): string => {
        return speakerMappings[originalSpeaker] || originalSpeaker;
    };

    const fetchSpeakerMappings = async () => {
        // Only fetch if there are speakers (from diarization or multi-track)
        if (!audioFile || !hasSpeakers()) {
            return;
        }

        try {
            const response = await fetch(`/api/v1/transcription/${audioId}/speakers`, {
                headers: { ...getAuthHeaders() },
            });

            if (response.ok) {
                const mappings: { id?: number; original_speaker: string; custom_name: string }[] = await response.json();

                // Convert to lookup object
                const mappingObj: Record<string, string> = {};
                mappings.forEach(mapping => {
                    mappingObj[mapping.original_speaker] = mapping.custom_name;
                });

                setSpeakerMappings(mappingObj);
            }
        } catch (err) {
            console.error('Error fetching speaker mappings:', err);
        }
    };

    if (loading) {
        return (
            <div className="min-h-screen bg-gray-50 dark:bg-black-950">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 sm:py-8">
                    <div className="bg-white dark:bg-black-900 rounded-xl p-3 sm:p-6">
                        <div className="animate-pulse">
                            <div className="h-6 bg-gray-200 dark:bg-black-600 rounded w-1/4 mb-4"></div>
                            <div className="h-4 bg-gray-200 dark:bg-black-600 rounded w-1/2 mb-8"></div>
                            <div className="h-20 bg-gray-200 dark:bg-black-600 rounded mb-8"></div>
                            <div className="space-y-3">
                                {[...Array(5)].map((_, i) => (
                                    <div
                                        key={i}
                                        className="h-4 bg-gray-200 dark:bg-black-600 rounded"
                                    ></div>
                                ))}
                                {/* Selection bubble and editor moved to portal */}
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        );
    }

    if (!audioFile) {
        return (
            <div className="min-h-screen bg-gray-50 dark:bg-black-950">
                <div className="flex items-center justify-center min-h-[50vh]">
                    <div className="bg-white dark:bg-black-900 rounded-xl p-3 sm:p-6 text-center">
                        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-50 mb-4">
                            Audio file not found
                        </h1>
                        <Button onClick={handleBack} variant="outline" className="cursor-pointer">
                            <ArrowLeft className="mr-2 h-4 w-4" />
                            Back to Audio Files
                        </Button>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-background text-foreground transition-colors duration-300">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 sm:py-8">
                {/* Header with back button and theme switcher */}
                <div className="flex items-center justify-between mb-3 sm:mb-6">
                    <Button onClick={handleBack} variant="outline" size="sm" className="cursor-pointer">
                        <ArrowLeft className="mr-2 h-4 w-4" />
                        Back to Audio Files
                    </Button>
                    <ThemeSwitcher />
                </div>

                {/* Audio Player Section */}
                <div ref={audioSectionRef} className={`glass rounded-xl ${audioCollapsed ? 'p-3 sm:p-4' : 'p-3 sm:p-6'} mb-3 sm:mb-6 transition-all duration-300`}>
                    <div className="mb-6">
                        <div className="mb-2 flex items-center gap-2 justify-between">
                            {editingTitle ? (
                                <input
                                    autoFocus
                                    className="w-full max-w-xl text-2xl font-bold bg-transparent border-b border-amber-400 focus:outline-none focus:ring-0 dark:text-gray-50 text-gray-900"
                                    value={titleInput}
                                    disabled={savingTitle}
                                    onChange={(e) => setTitleInput(e.target.value)}
                                    onBlur={async () => { await saveTitle(); }}
                                    onKeyDown={async (e) => {
                                        if (e.key === 'Enter') { e.preventDefault(); await saveTitle(); }
                                        if (e.key === 'Escape') { setEditingTitle(false); }
                                    }}
                                />
                            ) : (
                                <div className="flex items-center gap-2">
                                    <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-50">
                                        {audioFile.title || getFileName(audioFile.audio_path)}
                                    </h1>
                                    {audioFile.is_multi_track && (
                                        <>
                                            <span className="inline-flex items-center gap-1 px-2 py-1 bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300 text-xs font-medium rounded-md">
                                                <Users className="h-3 w-3" />
                                                Multi-Track ({audioFile.multi_track_files?.length || 0} speakers)
                                            </span>
                                            <MergeStatusBadge
                                                jobId={audioFile.id}
                                                mergeStatus={audioFile.merge_status}
                                                mergeError={audioFile.merge_error}
                                            />
                                        </>
                                    )}
                                    <button
                                        className="h-7 w-7 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-500 hover:text-gray-700 hover:bg-gray-200/60 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:bg-gray-700/60 transition-colors"
                                        aria-label="Edit title"
                                        title="Edit title"
                                        onClick={() => {
                                            const currentTitle = audioFile.title || getFileName(audioFile.audio_path);
                                            setTitleInput(currentTitle);
                                            setEditingTitle(true);
                                        }}
                                    >
                                        <Pencil className="h-4 w-4" />
                                    </button>
                                </div>
                            )}
                            <button
                                className="h-7 w-7 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-500 hover:text-gray-700 hover:bg-gray-200/60 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:bg-gray-700/60 transition-colors"
                                aria-label={audioCollapsed ? 'Expand audio' : 'Collapse audio'}
                                title={audioCollapsed ? 'Expand audio' : 'Collapse audio'}
                                onClick={toggleAudioCollapsed}
                            >
                                {audioCollapsed ? (
                                    <ChevronDown className="h-4 w-4" />
                                ) : (
                                    <ChevronUp className="h-4 w-4" />
                                )}
                            </button>
                        </div>
                        <p className="text-gray-600 dark:text-gray-400 text-sm">
                            Added on {formatDate(audioFile.created_at)}
                        </p>
                    </div>

                    {/* Audio Player Controls (hidden when collapsed, but kept mounted) */}
                    {/* Audio Player Controls */}
                    <div className={`mb-6 ${audioCollapsed ? 'hidden' : ''}`}>
                        <AudioPlayer
                            ref={audioPlayerRef}
                            audioId={audioId}
                            collapsed={audioCollapsed}
                            onToggleCollapse={toggleAudioCollapsed}
                            onTimeUpdate={setCurrentTime}
                            onPlayStateChange={setIsPlaying}
                            onDurationChange={setDuration}
                        />
                    </div>
                </div>

                {(currentStatus || audioFile.status) === "completed" && transcript && (
                    <div className="glass rounded-xl p-3 sm:p-6 transition-all duration-300">
                        {/* Header Section */}
                        <div className="mb-10 sm:mb-16">
                            {/* Title Row */}
                            <div className="flex items-center justify-between mb-3 sm:mb-0">
                                <div className="flex items-center gap-3 min-w-0 flex-1 mr-2">
                                    <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100 truncate flex-1" title={audioFile.title || audioFile.audio_path}>
                                        {getFileNameWithoutExt()}
                                    </h1>
                                </div>

                                {/* Desktop: Show toolbar inline, Mobile: Hide here (shown below) */}
                                <div className="hidden sm:flex items-center gap-2">
                                    {/* Sleek toolbar (desktop only) */}
                                    {viewMode === 'transcript' && (
                                        <div className="flex items-center gap-1 sm:gap-1.5 rounded-md sm:rounded-lg bg-gray-100/80 dark:bg-black-800/80 px-1.5 py-0.5 sm:px-2 sm:py-1 border border-gray-200 dark:border-black-700 shadow-sm">
                                            {/* View toggle */}
                                            <button
                                                type="button"
                                                onClick={() => setTranscriptMode(m => m === 'compact' ? 'expanded' : 'compact')}
                                                className={`h-6 w-6 sm:h-7 sm:w-7 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors ${transcriptMode === 'compact' ? 'bg-white dark:bg-black-700 shadow-sm' : ''}`}
                                                title={transcriptMode === 'compact' ? 'Switch to Timeline view' : 'Switch to Compact view'}
                                            >
                                                {transcriptMode === 'compact' ? (
                                                    <List className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
                                                ) : (
                                                    <AlignLeft className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
                                                )}
                                            </button>

                                            <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />

                                            {/* Notes toggle (icon + tiny count) */}
                                            <button
                                                type="button"
                                                onClick={() => setNotesOpen(v => !v)}
                                                className={`relative h-6 w-6 sm:h-7 sm:w-7 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors ${notesOpen ? 'bg-white dark:bg-black-700 shadow-sm' : ''}`}
                                                title="Toggle notes"
                                            >
                                                <StickyNote className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
                                                {notes.length > 0 && (
                                                    <span className="absolute -top-1 -right-0.5 min-w-[15px] h-[15px] px-1 rounded-full bg-gray-900 text-white text-[10px] leading-[15px] text-center">
                                                        {notes.length > 99 ? '99+' : notes.length}
                                                    </span>
                                                )}
                                            </button>

                                            <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />

                                            {/* Execution Info */}
                                            <button
                                                type="button"
                                                onClick={openExecutionInfo}
                                                className="h-6 w-6 sm:h-7 sm:w-7 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                                                title="View execution parameters and timing"
                                            >
                                                <Info className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
                                            </button>

                                            <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />

                                            {/* Logs */}
                                            <button
                                                type="button"
                                                onClick={openLogsDialog}
                                                className="h-6 w-6 sm:h-7 sm:w-7 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                                                title="View transcription logs"
                                            >
                                                <FileText className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
                                            </button>

                                            {/* Speaker Renaming - only show if there are speakers (from diarization or multi-track) */}
                                            {hasSpeakers() && getDetectedSpeakers().length > 0 && (
                                                <>
                                                    <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />
                                                    <button
                                                        type="button"
                                                        onClick={() => setSpeakerRenameDialogOpen(true)}
                                                        className="h-6 w-6 sm:h-7 sm:w-7 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                                                        title="Rename speakers"
                                                    >
                                                        <Users className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
                                                    </button>
                                                </>
                                            )}

                                            <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />

                                            {/* Summarize */}
                                            <button
                                                type="button"
                                                onClick={openSummarizeDialog}
                                                className="h-6 w-6 sm:h-7 sm:w-7 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
                                                title={llmReady === false ? 'Configure LLM in Settings' : 'Summarize transcript'}
                                                disabled={llmReady === false}
                                            >
                                                <Sparkles className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
                                            </button>

                                            <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />

                                            {/* Download dropdown */}
                                            <DropdownMenu>
                                                <DropdownMenuTrigger asChild>
                                                    <button
                                                        type="button"
                                                        className="h-6 w-6 sm:h-7 sm:w-7 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                                                        title="Download transcript"
                                                    >
                                                        <Download className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
                                                    </button>
                                                </DropdownMenuTrigger>
                                                <DropdownMenuContent className="w-44 bg-white dark:bg-black-800 border-gray-200 dark:border-black-700">
                                                    <DropdownMenuItem onClick={downloadSRT} className="flex items-center gap-2 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100">
                                                        <FileImage className="h-4 w-4" />
                                                        Download as SRT
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem onClick={() => handleDownloadWithDialog('txt')} className="flex items-center gap-2 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100">
                                                        <FileText className="h-4 w-4" />
                                                        Download as TXT
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem onClick={() => handleDownloadWithDialog('json')} className="flex items-center gap-2 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100">
                                                        <FileJson className="h-4 w-4" />
                                                        Download as JSON
                                                    </DropdownMenuItem>
                                                </DropdownMenuContent>
                                            </DropdownMenu>

                                            <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />

                                            {/* Open Chat Page */}
                                            <button
                                                type="button"
                                                onClick={() => navigate({ path: 'chat', params: { audioId } })}
                                                className="h-7 w-7 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                                                title="Open chat"
                                            >
                                                <MessageCircle className="h-4 w-4" />
                                            </button>
                                        </div>
                                    )}
                                </div>

                                {/* Mobile Toolbar Placeholder */}
                                {viewMode === 'transcript' && (
                                    <div className="flex sm:hidden justify-center">
                                        <div className="flex items-center gap-1 rounded-md bg-gray-100/80 dark:bg-black-800/80 px-1.5 py-0.5 border border-gray-200 dark:border-black-700 shadow-sm">
                                            {/* View toggle */}
                                            <button
                                                type="button"
                                                onClick={() => setTranscriptMode(m => m === 'compact' ? 'expanded' : 'compact')}
                                                className={`h-6 w-6 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors ${transcriptMode === 'compact' ? 'bg-white dark:bg-black-700 shadow-sm' : ''}`}
                                                title={transcriptMode === 'compact' ? 'Switch to Timeline view' : 'Switch to Compact view'}
                                            >
                                                {transcriptMode === 'compact' ? (
                                                    <List className="h-3.5 w-3.5" />
                                                ) : (
                                                    <AlignLeft className="h-3.5 w-3.5" />
                                                )}
                                            </button>

                                            <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />

                                            {/* Notes toggle */}
                                            <button
                                                type="button"
                                                onClick={() => setNotesOpen(v => !v)}
                                                className={`relative h-6 w-6 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors ${notesOpen ? 'bg-white dark:bg-black-700 shadow-sm' : ''}`}
                                                title="Toggle notes"
                                            >
                                                <StickyNote className="h-3.5 w-3.5" />
                                                {notes.length > 0 && (
                                                    <span className="absolute -top-1 -right-0.5 min-w-[15px] h-[15px] px-1 rounded-full bg-gray-900 text-white text-[10px] leading-[15px] text-center">
                                                        {notes.length > 99 ? '99+' : notes.length}
                                                    </span>
                                                )}
                                            </button>

                                            <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />

                                            {/* Execution Info */}
                                            <button
                                                type="button"
                                                onClick={openExecutionInfo}
                                                className="h-6 w-6 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                                                title="View execution parameters and timing"
                                            >
                                                <Info className="h-3.5 w-3.5" />
                                            </button>

                                            {/* Speaker Renaming - only show if there are speakers (from diarization or multi-track) */}
                                            {hasSpeakers() && getDetectedSpeakers().length > 0 && (
                                                <>
                                                    <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />
                                                    <button
                                                        type="button"
                                                        onClick={() => setSpeakerRenameDialogOpen(true)}
                                                        className="h-6 w-6 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                                                        title="Rename speakers"
                                                    >
                                                        <Users className="h-3.5 w-3.5" />
                                                    </button>
                                                </>
                                            )}

                                            <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />

                                            {/* Summarize */}
                                            <button
                                                type="button"
                                                onClick={openSummarizeDialog}
                                                className="h-6 w-6 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
                                                title={llmReady === false ? 'Configure LLM in Settings' : 'Summarize transcript'}
                                                disabled={llmReady === false}
                                            >
                                                <Sparkles className="h-3.5 w-3.5" />
                                            </button>

                                            <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />

                                            {/* Download dropdown */}
                                            <DropdownMenu>
                                                <DropdownMenuTrigger asChild>
                                                    <button
                                                        type="button"
                                                        className="h-6 w-6 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                                                        title="Download transcript"
                                                    >
                                                        <Download className="h-3.5 w-3.5" />
                                                    </button>
                                                </DropdownMenuTrigger>
                                                <DropdownMenuContent className="w-44 bg-white dark:bg-black-800 border-gray-200 dark:border-black-700">
                                                    <DropdownMenuItem onClick={downloadSRT} className="flex items-center gap-2 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100">
                                                        <FileImage className="h-4 w-4" />
                                                        Download as SRT
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem onClick={() => handleDownloadWithDialog('txt')} className="flex items-center gap-2 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100">
                                                        <FileText className="h-4 w-4" />
                                                        Download as TXT
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem onClick={() => handleDownloadWithDialog('json')} className="flex items-center gap-2 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100">
                                                        <FileJson className="h-4 w-4" />
                                                        Download as JSON
                                                    </DropdownMenuItem>
                                                </DropdownMenuContent>
                                            </DropdownMenu>

                                            <div className="mx-1 h-5 w-px bg-gray-300 dark:bg-black-700" />

                                            {/* Open Chat Page */}
                                            <button
                                                type="button"
                                                onClick={() => navigate({ path: 'chat', params: { audioId } })}
                                                className="h-6 w-6 inline-flex items-center justify-center rounded-md cursor-pointer text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                                                title="Open chat"
                                            >
                                                <MessageCircle className="h-3.5 w-3.5" />
                                            </button>
                                        </div>
                                    </div>
                                )}
                            </div>

                            {/* Content Area - Show transcript or chat based on view mode */}
                            {viewMode === "transcript" ? (
                                <div className="relative overflow-hidden">
                                    {/* Transcript Content */}
                                    <div className="prose prose-gray dark:prose-invert max-w-none font-transcript">
                                        <div ref={transcriptRef} className="relative">
                                            <TranscriptView
                                                transcript={transcript}
                                                mode={transcriptMode}
                                                currentWordIndex={currentWordIndex}
                                                notes={notes}
                                                highlightedWordRef={highlightedWordRef}
                                                speakerMappings={speakerMappings}
                                            />
                                        </div>
                                    </div>
                                </div>
                            ) : (
                                <div style={{ height: "600px" }}>
                                    <ChatInterface
                                        transcriptionId={audioId}
                                        onClose={() => setViewMode("transcript")}
                                    />
                                </div>
                            )}
                        </div>
                    </div>
                )}



                {/* Status Messages */}
                {(currentStatus || audioFile.status) !== "completed" && (
                    <div className="bg-white dark:bg-black-900 rounded-xl p-6">
                        <div className="text-center">
                            {/* Processing Status with Animation */}
                            {(currentStatus || audioFile.status) === "processing" && (
                                <div className="flex flex-col items-center space-y-4">
                                    <div className="flex items-center space-x-3">
                                        <Loader2 className="h-8 w-8 text-gray-500 animate-spin" />
                                        <div>
                                            <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-50">
                                                Transcription in Progress
                                            </h2>
                                            {elapsedTime > 0 && (
                                                <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                                                    Processing for {formatElapsedTime(elapsedTime)}
                                                </p>
                                            )}
                                        </div>
                                    </div>
                                    <div className="w-full max-w-md">
                                        <div className="bg-gray-200 dark:bg-black-600 rounded-full h-2">
                                            <div className="bg-gray-500 h-2 rounded-full transition-all duration-1000 ease-out animate-pulse" style={{ width: '60%' }}></div>
                                        </div>
                                    </div>
                                    <p className="text-gray-600 dark:text-gray-400 text-sm">
                                        Converting your audio to text... This may take a few minutes.
                                    </p>
                                </div>
                            )}

                            {/* Other Status Messages */}
                            {(currentStatus || audioFile.status) !== "processing" && (
                                <>
                                    <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-50 mb-2">
                                        {(currentStatus || audioFile.status) === "pending" && "Transcription Queued"}
                                        {(currentStatus || audioFile.status) === "uploaded" && "Ready for Transcription"}
                                        {(currentStatus || audioFile.status) === "failed" && "Transcription Failed"}
                                    </h2>
                                    <p className="text-gray-600 dark:text-gray-400">
                                        {(currentStatus || audioFile.status) === "pending" &&
                                            "Your audio file is in the transcription queue."}
                                        {(currentStatus || audioFile.status) === "uploaded" &&
                                            "Start transcription from the audio files list."}
                                        {(currentStatus || audioFile.status) === "failed" &&
                                            "There was an error processing your audio file."}
                                    </p>
                                    {(currentStatus || audioFile.status) === "failed" && (
                                        <div className="mt-4">
                                            <button
                                                onClick={openLogsDialog}
                                                className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-gray-900 rounded-md hover:bg-black focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-500 transition-colors"
                                            >
                                                <FileText className="h-4 w-4" />
                                                View Logs
                                            </button>
                                        </div>
                                    )}
                                </>
                            )}
                        </div>
                    </div>
                )}

                {/* Fixed bottom-left compact circular control with progress ring */}
                <div className="fixed bottom-4 left-4 z-[9999]">
                    {(() => {
                        const progress = duration > 0 ? Math.min(1, Math.max(0, currentTime / duration)) : 0
                        const size = 64 // outer box size
                        const strokeWidth = 5
                        const center = size / 2
                        const radius = center - strokeWidth - 2
                        const circumference = 2 * Math.PI * radius
                        const dashOffset = circumference * (1 - progress)
                        return (
                            <div
                                className="relative"
                                style={{ width: size, height: size }}
                                aria-label="Audio playback controls"
                            >
                                <svg
                                    width={size}
                                    height={size}
                                    className="block -rotate-90 drop-shadow-sm"
                                    role="img"
                                    aria-hidden
                                >
                                    <circle
                                        cx={center}
                                        cy={center}
                                        r={radius}
                                        fill="none"
                                        stroke="currentColor"
                                        className="text-white"
                                        strokeWidth={strokeWidth}
                                    />
                                    <circle
                                        cx={center}
                                        cy={center}
                                        r={radius}
                                        fill="none"
                                        stroke="currentColor"
                                        className="text-gray-500 dark:text-gray-400"
                                        strokeWidth={strokeWidth}
                                        strokeLinecap="round"
                                        strokeDasharray={circumference}
                                        strokeDashoffset={dashOffset}
                                    />
                                </svg>
                                <button
                                    onClick={togglePlayPause}
                                    aria-label={isPlaying ? 'Pause audio' : 'Play audio'}
                                    title={isPlaying ? 'Pause' : 'Play'}
                                    style={{ transform: 'translate(-50%, -50%)' }}
                                    className="absolute left-1/2 top-1/2 w-11 h-11 rounded-full bg-gray-900 hover:bg-black text-white shadow-lg transition-colors flex items-center justify-center cursor-pointer"
                                >
                                    {isPlaying ? (
                                        <Pause className="h-5 w-5" />
                                    ) : (
                                        <Play className="h-5 w-5 ml-0.5" />
                                    )}
                                </button>
                            </div>
                        )
                    })()}
                </div>

                {/* Download Options Dialog */}
                <Dialog open={downloadDialogOpen} onOpenChange={setDownloadDialogOpen}>
                    <DialogContent className="sm:max-w-md bg-white dark:bg-black-800 border-gray-200 dark:border-black-700">
                        <DialogHeader>
                            <DialogTitle className="flex items-center gap-2 text-xl">
                                <Sparkles className="h-5 w-5 text-primary" />
                                Transcription Details
                            </DialogTitle>
                            <DialogDescription>
                                Technical execution parameters and performance metrics
                            </DialogDescription>
                        </DialogHeader>

                        <div className="space-y-4 py-4">
                            <div className="flex items-center justify-between">
                                <Label htmlFor="speaker-labels" className="text-gray-700 dark:text-gray-300">
                                    Include Speaker Labels
                                </Label>
                                <Switch
                                    id="speaker-labels"
                                    checked={includeSpeakerLabels}
                                    onCheckedChange={setIncludeSpeakerLabels}
                                    disabled={!transcript?.segments?.some(s => s.speaker)}
                                />
                            </div>

                            <div className="flex items-center justify-between">
                                <Label htmlFor="timestamps" className="text-gray-700 dark:text-gray-300">
                                    Include Timestamps
                                </Label>
                                <Switch
                                    id="timestamps"
                                    checked={includeTimestamps}
                                    onCheckedChange={setIncludeTimestamps}
                                    disabled={!transcript?.segments}
                                />
                            </div>

                            {(!includeSpeakerLabels && !includeTimestamps) && (
                                <div className="text-sm text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-black-800 p-3 rounded-md">
                                    <div className="flex items-center gap-2">
                                        <Check className="h-4 w-4 text-gray-900 dark:text-gray-100" />
                                        Transcript will be formatted as a single paragraph
                                    </div>
                                </div>
                            )}

                            {(includeSpeakerLabels || includeTimestamps) && (
                                <div className="text-sm text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-black-800 p-3 rounded-md">
                                    <div className="flex items-center gap-2">
                                        <Check className="h-4 w-4 text-gray-900 dark:text-gray-100" />
                                        Transcript will be formatted in segments with selected labels
                                    </div>
                                </div>
                            )}
                        </div>

                        <DialogFooter className="gap-2">
                            <Button
                                variant="outline"
                                onClick={() => setDownloadDialogOpen(false)}
                                className="bg-white dark:bg-black-800 border-gray-300 dark:border-black-600 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
                            >
                                Cancel
                            </Button>
                            <Button
                                onClick={handleDownloadConfirm}
                                className="bg-gray-900 dark:bg-gray-700 hover:bg-black dark:hover:bg-gray-600 text-white"
                            >
                                <Download className="mr-2 h-4 w-4" />
                                Download {downloadFormat.toUpperCase()}
                            </Button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>

                {/* Speaker Rename Dialog */}
                <SpeakerRenameDialog
                    open={speakerRenameDialogOpen}
                    onOpenChange={setSpeakerRenameDialogOpen}
                    transcriptionId={audioId}
                    onSpeakerMappingsUpdate={handleSpeakerMappingsUpdate}
                    initialSpeakers={getDetectedSpeakers()}
                />

                {/* Summarization template selector dialog */}
                <UIDialog open={summarizeOpen} onOpenChange={(o) => { setSummarizeOpen(o); if (!o) { setTplPopoverOpen(false); } }}>
                    <UIDialogContent className="sm:max-w-lg bg-white dark:bg-black-800 border-gray-200 dark:border-black-700">
                        <UIDialogHeader>
                            <UIDialogTitle className="text-gray-900 dark:text-gray-100">Summarize Transcript</UIDialogTitle>
                            <UIDialogDescription className="text-gray-600 dark:text-gray-400">Choose a summarization template</UIDialogDescription>
                        </UIDialogHeader>
                        <div className="py-2 space-y-3">
                            <div className="space-y-1">
                                <UILabel className="text-sm text-gray-700 dark:text-gray-300">Template</UILabel>
                                <Popover open={tplPopoverOpen} onOpenChange={setTplPopoverOpen}>
                                    <PopoverTrigger asChild>
                                        <button
                                            className="w-full inline-flex justify-between items-center rounded-md border border-gray-300 dark:border-black-600 bg-white dark:bg-black-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer"
                                            aria-label="Choose template"
                                        >
                                            <span className="truncate text-left">{selectedTemplate ? selectedTemplate.name : (templatesLoading ? 'Loading...' : 'Select a template')}</span>
                                            <span className="text-xs text-gray-500 ml-2 truncate">{selectedTemplate?.model ? `(${selectedTemplate.model})` : ''}</span>
                                        </button>
                                    </PopoverTrigger>
                                    <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0 bg-white dark:bg-black-800 border border-gray-200 dark:border-black-700">
                                        <Command>
                                            <CommandInput placeholder="Search templates..." />
                                            <CommandList className="max-h-64 overflow-auto">
                                                <CommandEmpty>{templatesLoading ? 'Loading...' : 'No templates found'}</CommandEmpty>
                                                <CommandGroup heading="Templates">
                                                    {templates.map(t => (
                                                        <CommandItem
                                                            key={t.id}
                                                            value={t.name}
                                                            onSelect={() => { setSelectedTemplateId(t.id); setTplPopoverOpen(false); }}
                                                        >
                                                            <div className="flex flex-col">
                                                                <span className="text-sm">{t.name}</span>
                                                                <span className="text-xs text-gray-500">Model: {t.model || '—'}</span>
                                                            </div>
                                                        </CommandItem>
                                                    ))}
                                                </CommandGroup>
                                            </CommandList>
                                        </Command>
                                    </PopoverContent>
                                </Popover>
                                {!templatesLoading && templates.length === 0 && (
                                    <p className="text-xs text-gray-500">No templates. Create one in Settings → Summary.</p>
                                )}
                                {selectedTemplate && !selectedTemplate.model && (
                                    <p className="text-xs text-red-600">Selected template has no model configured. Edit it in Settings.</p>
                                )}
                            </div>
                            <div className="mt-1 flex items-center justify-end gap-2">
                                <button className="px-3 py-1.5 rounded-md bg-gray-200 dark:bg-black-700" onClick={() => setSummarizeOpen(false)}>Cancel</button>
                                <button className="px-3 py-1.5 rounded-md bg-gray-900 text-white disabled:opacity-50 hover:bg-black transition-colors" disabled={!selectedTemplateId || !selectedTemplate?.model} onClick={() => { setSummarizeOpen(false); startSummarization(); }}>Summarize</button>
                            </div>
                        </div>
                    </UIDialogContent>
                </UIDialog>

                {/* Summary output dialog */}
                <UIDialog open={summaryOpen} onOpenChange={setSummaryOpen}>
                    <UIDialogContent className="sm:max-w-3xl bg-white dark:bg-black-800 border-gray-200 dark:border-black-700 max-h-[85vh] overflow-y-auto">
                        <UIDialogHeader>
                            <UIDialogTitle className="text-gray-900 dark:text-gray-100">Summary</UIDialogTitle>
                            <UIDialogDescription className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
                                {isSummarizing ? (
                                    <>
                                        <span>Generating summary...</span>
                                        <span className="inline-block h-3.5 w-3.5 border-2 border-gray-600 border-t-transparent rounded-full animate-spin" aria-label="Loading" />
                                    </>
                                ) : (
                                    <span>Summary {summaryError ? 'failed' : 'ready'}</span>
                                )}
                            </UIDialogDescription>
                        </UIDialogHeader>
                        <div className="flex items-center justify-end gap-2 mb-2">
                            <button
                                className="px-2.5 py-1.5 rounded-md bg-gray-900 text-white text-sm hover:bg-black transition-colors"
                                onClick={async () => {
                                    // Keep summary dialog open; open template picker on top
                                    setSummarizeOpen(true);
                                    // Reset selection so user explicitly chooses a template
                                    setSelectedTemplateId('');
                                    if (templates.length === 0) {
                                        try {
                                            setTemplatesLoading(true);
                                            const res = await fetch('/api/v1/summaries', { headers: { ...getAuthHeaders() } });
                                            if (res.ok) {
                                                const data = await res.json();
                                                setTemplates(data || []);
                                            }
                                        } finally { setTemplatesLoading(false); }
                                    }
                                }}
                                disabled={isSummarizing}
                            >
                                Regenerate
                            </button>
                            <button
                                className="px-2.5 py-1.5 rounded-md bg-gray-200 dark:bg-black-700 text-sm"
                                onClick={async () => {
                                    try {
                                        await navigator.clipboard.writeText(summaryStream || '');
                                        toast({ title: 'Copied to clipboard' });
                                    } catch { }
                                }}
                                disabled={!summaryStream}
                            >
                                Copy Text
                            </button>
                            <button
                                className="px-2.5 py-1.5 rounded-md bg-gray-200 dark:bg-black-700 text-sm"
                                onClick={() => {
                                    if (!summaryStream) return;
                                    const base = getFileNameWithoutExt();
                                    downloadFile(summaryStream, `${base}-summary.md`, 'text/markdown');
                                }}
                                disabled={!summaryStream}
                            >
                                Download .md
                            </button>
                        </div>
                        <div className="prose prose-gray dark:prose-invert max-w-none min-h-[200px]">
                            {summaryError ? (
                                <p className="text-sm text-red-600 dark:text-red-400">{summaryError}</p>
                            ) : summaryStream ? (
                                <ReactMarkdown remarkPlugins={[remarkMath]} rehypePlugins={[rehypeRaw as any, rehypeKatex as any, rehypeHighlight as any]}>
                                    {summaryStream}
                                </ReactMarkdown>
                            ) : (
                                <p className="text-sm text-gray-500">{isSummarizing ? 'Generating summary...' : 'No content'}</p>
                            )}
                        </div>
                    </UIDialogContent>
                </UIDialog>

                {/* Execution info dialog */}
                <UIDialog open={executionInfoOpen} onOpenChange={setExecutionInfoOpen}>
                    <UIDialogContent className="sm:max-w-4xl w-[95vw] bg-white dark:bg-gray-950 border-gray-200 dark:border-gray-800 max-h-[90vh] overflow-y-auto">
                        <UIDialogHeader>
                            <UIDialogTitle className="text-gray-900 dark:text-gray-100 flex items-center gap-2">
                                <Info className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                                Transcription Execution Details
                            </UIDialogTitle>
                            <UIDialogDescription className="text-gray-600 dark:text-gray-400">
                                Parameters used and processing time for this transcription
                            </UIDialogDescription>
                        </UIDialogHeader>

                        {executionDataLoading ? (
                            <div className="py-8 text-center">
                                <div className="animate-pulse">
                                    <div className="h-4 bg-gray-200 dark:bg-black-600 rounded w-3/4 mx-auto mb-4"></div>
                                    <div className="h-4 bg-gray-200 dark:bg-black-600 rounded w-1/2 mx-auto"></div>
                                </div>
                            </div>
                        ) : executionData ? (
                            <div className="space-y-4 sm:space-y-6 py-2 sm:py-4">
                                {/* Processing Time - Conditional for Multi-track */}
                                {executionData.is_multi_track ? (
                                    // Multi-track timing display
                                    <div className="space-y-4">
                                        {/* Overall Processing Time */}
                                        <div className="bg-gray-100/50 dark:bg-gray-900/50 backdrop-blur-md border border-gray-200/50 dark:border-gray-800/50 rounded-lg p-4 sm:p-6 shadow-sm">
                                            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-3 sm:mb-4 flex items-center gap-2">
                                                <Clock className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                                                Overall Processing Time
                                            </h3>
                                            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4 text-sm">
                                                <div className="bg-white/60 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
                                                    <span className="text-gray-600 dark:text-gray-400 font-medium">Started:</span>
                                                    <p className="font-mono text-gray-900 dark:text-gray-100 mt-1 text-xs sm:text-sm">
                                                        {new Date(executionData.started_at).toLocaleString()}
                                                    </p>
                                                </div>
                                                <div className="bg-white/60 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
                                                    <span className="text-gray-600 dark:text-gray-400 font-medium">Completed:</span>
                                                    <p className="font-mono text-gray-900 dark:text-gray-100 mt-1 text-xs sm:text-sm">
                                                        {executionData.completed_at
                                                            ? new Date(executionData.completed_at).toLocaleString()
                                                            : 'N/A'
                                                        }
                                                    </p>
                                                </div>
                                                <div className="bg-white/60 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
                                                    <span className="text-gray-600 dark:text-gray-400 font-medium">Total Duration:</span>
                                                    <p className="font-mono text-xl sm:text-2xl font-bold text-gray-900 dark:text-gray-100 mt-1">
                                                        {executionData.processing_duration
                                                            ? `${(executionData.processing_duration / 1000).toFixed(1)}s`
                                                            : 'N/A'
                                                        }
                                                    </p>
                                                </div>
                                            </div>
                                        </div>

                                        {/* Individual Track Processing */}
                                        {executionData.multi_track_timings && executionData.multi_track_timings.length > 0 && (
                                            <div className="bg-gray-100/50 dark:bg-gray-900/50 backdrop-blur-md border border-gray-200/50 dark:border-gray-800/50 rounded-lg p-4 sm:p-6 shadow-sm">
                                                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-3 sm:mb-4 flex items-center gap-2">
                                                    <svg className="h-5 w-5 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                                                    </svg>
                                                    Individual Track Processing
                                                </h3>
                                                <div className="space-y-3">
                                                    {executionData.multi_track_timings.map((timing, index) => (
                                                        <div key={index} className="flex items-center gap-2 bg-gray-200/50 dark:bg-gray-800/50 backdrop-blur-sm p-1 rounded-lg border border-gray-200/50 dark:border-gray-700/50">
                                                            <div className="bg-white/60 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-700/50 flex-grow">
                                                                <div className="flex justify-between items-center mb-2">
                                                                    <span className="font-medium text-gray-800 dark:text-gray-200">
                                                                        {timing.track_name}
                                                                    </span>
                                                                    <span className="font-mono text-lg font-bold text-gray-600 dark:text-gray-400">
                                                                        {(timing.duration / 1000).toFixed(1)}s
                                                                    </span>
                                                                </div>
                                                                <div className="grid grid-cols-2 gap-2 text-xs text-gray-600 dark:text-gray-400">
                                                                    <div>
                                                                        <span className="font-medium">Started:</span>
                                                                        <p className="font-mono">{new Date(timing.start_time).toLocaleTimeString()}</p>
                                                                    </div>
                                                                    <div>
                                                                        <span className="font-medium">Completed:</span>
                                                                        <p className="font-mono">{new Date(timing.end_time).toLocaleTimeString()}</p>
                                                                    </div>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    ))}
                                                </div>
                                            </div>
                                        )}

                                        {/* Audio Merge Phase */}
                                        {executionData.merge_duration && (
                                            <div className="bg-gray-100/50 dark:bg-gray-900/50 backdrop-blur-md border border-gray-200/50 dark:border-gray-800/50 rounded-lg p-4 sm:p-6 shadow-sm">
                                                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-3 sm:mb-4 flex items-center gap-2">
                                                    <svg className="h-5 w-5 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                                                    </svg>
                                                    Transcript Merge Phase
                                                </h3>
                                                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 text-sm">
                                                    <div className="bg-white/60 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
                                                        <span className="text-gray-600 dark:text-gray-400 font-medium">Started:</span>
                                                        <p className="font-mono text-gray-900 dark:text-gray-100 mt-1 text-xs">
                                                            {executionData.merge_start_time
                                                                ? new Date(executionData.merge_start_time).toLocaleTimeString()
                                                                : 'N/A'
                                                            }
                                                        </p>
                                                    </div>
                                                    <div className="bg-white/60 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
                                                        <span className="text-gray-600 dark:text-gray-400 font-medium">Completed:</span>
                                                        <p className="font-mono text-gray-900 dark:text-gray-100 mt-1 text-xs">
                                                            {executionData.merge_end_time
                                                                ? new Date(executionData.merge_end_time).toLocaleTimeString()
                                                                : 'N/A'
                                                            }
                                                        </p>
                                                    </div>
                                                    <div className="bg-white/60 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
                                                        <span className="text-gray-600 dark:text-gray-400 font-medium">Duration:</span>
                                                        <p className="font-mono text-lg font-bold text-gray-900 dark:text-gray-100 mt-1">
                                                            {(executionData.merge_duration / 1000).toFixed(1)}s
                                                        </p>
                                                    </div>
                                                    <div className="bg-white/60 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
                                                        <span className="text-gray-600 dark:text-gray-400 font-medium">Audio Length:</span>
                                                        <p className="font-mono text-lg font-bold text-gray-900 dark:text-gray-100 mt-1">
                                                            {(() => {
                                                                const duration = getAudioDurationFromTranscript(transcript);
                                                                return duration ? formatDuration(duration) : 'N/A';
                                                            })()}
                                                        </p>
                                                    </div>
                                                </div>
                                            </div>
                                        )}
                                        {/* Model Parameters for Multi-track */}
                                        <div>
                                            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-3 sm:mb-4 flex items-center gap-2">
                                                <Settings className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                                                {(() => {
                                                    const modelFamily = executionData.actual_parameters?.model_family;
                                                    if (modelFamily === 'nvidia_parakeet') return 'NVIDIA Parakeet Parameters';
                                                    if (modelFamily === 'nvidia_canary') return 'NVIDIA Canary Parameters';
                                                    if (modelFamily === 'whisper') return 'WhisperX Parameters';
                                                    return 'Model Parameters';
                                                })()}
                                            </h3>
                                            <div className="glass-card rounded-lg p-3 sm:p-4">
                                                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4 text-sm">
                                                    {/* Model Settings */}
                                                    <div className="bg-white/40 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-800/50 shadow-sm">
                                                        <h4 className="font-semibold text-gray-800 dark:text-gray-200 mb-2 text-sm sm:text-base">Model & Configuration</h4>
                                                        <div className="space-y-2">
                                                            <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Model Family:</span>
                                                                <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">
                                                                    {(() => {
                                                                        const family = executionData.actual_parameters?.model_family;
                                                                        if (family === 'nvidia_parakeet') return 'NVIDIA Parakeet';
                                                                        if (family === 'nvidia_canary') return 'NVIDIA Canary';
                                                                        if (family === 'whisper') return 'WhisperX';
                                                                        return family || 'N/A';
                                                                    })()}
                                                                </span>
                                                            </div>
                                                            <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Model:</span>
                                                                <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm break-all">{executionData.actual_parameters?.model || 'N/A'}</span>
                                                            </div>
                                                            {executionData.actual_parameters?.model_family === 'whisper' && (
                                                                <>
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Device:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.device || 'N/A'}</span>
                                                                    </div>
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Compute Type:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.compute_type || 'N/A'}</span>
                                                                    </div>
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Batch Size:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.batch_size || 'N/A'}</span>
                                                                    </div>
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Threads:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.threads || 0}</span>
                                                                    </div>
                                                                </>
                                                            )}
                                                            {executionData.actual_parameters?.model_family === 'nvidia_parakeet' && (
                                                                <>
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Context Left:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.attention_context_left || 256}</span>
                                                                    </div>
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Context Right:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.attention_context_right || 256}</span>
                                                                    </div>
                                                                </>
                                                            )}
                                                            {executionData.actual_parameters?.model_family === 'nvidia_canary' && (
                                                                <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                    <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Source Language:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.language || 'auto'}</span>
                                                                </div>
                                                            )}
                                                        </div>
                                                    </div>

                                                    {/* Processing Settings - Only for WhisperX */}
                                                    {executionData.actual_parameters?.model_family === 'whisper' && (
                                                        <div className="bg-white/40 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-800/50 shadow-sm">
                                                            <h4 className="font-semibold text-gray-800 dark:text-gray-200 mb-2 text-sm sm:text-base">Processing</h4>
                                                            <div className="space-y-1">
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Task:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.task || 'transcribe'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Language:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.language || 'auto'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Diarization:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.diarize ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">VAD Method:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.vad_method || 'pyannote'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">VAD Onset:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.vad_onset ?? 0.5}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">VAD Offset:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.vad_offset ?? 0.363}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Chunk Size:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.chunk_size || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Align Chunks:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.align_chunks ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Align Method:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.align_method || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Return Scores:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.return_scores ? 'Yes' : 'No'}</span>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    )}

                                                    {/* Advanced Settings - Only for WhisperX */}
                                                    {executionData.actual_parameters?.model_family === 'whisper' && (
                                                        <div className="bg-white/40 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-800/50 shadow-sm">
                                                            <h4 className="font-semibold text-gray-800 dark:text-gray-200 mb-2 text-sm sm:text-base">Advanced</h4>
                                                            <div className="space-y-1">
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Beam Size:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.beam_size || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Best Of:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.best_of || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Temperature:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.temperature || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">No Speech Threshold:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.no_speech_threshold || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Log Prob Threshold:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.logprob_threshold || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Compression Ratio Threshold:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.compression_ratio_threshold || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Condition On Previous Text:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.condition_on_previous_text ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Initial Prompt:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.initial_prompt || 'N/A'}</span>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    )}

                                                    {/* Advanced Settings - Only for NVIDIA Parakeet/Canary */}
                                                    {(executionData.actual_parameters?.model_family === 'nvidia_parakeet' || executionData.actual_parameters?.model_family === 'nvidia_canary') && (
                                                        <div className="bg-white/40 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-800/50 shadow-sm">
                                                            <h4 className="font-semibold text-gray-800 dark:text-gray-200 mb-2 text-sm sm:text-base">Advanced</h4>
                                                            <div className="space-y-1">
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Word Boost:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.word_boost ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Punctuation:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.enable_punctuation ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Automatic Capitalization:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.enable_automatic_capitalization ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">ITN:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.enable_inverse_text_normalization ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Speaker Diarization:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.enable_speaker_diarization ? 'Yes' : 'No'}</span>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    )}
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                ) : (
                                    // Single track execution info
                                    <div className="space-y-4">
                                        <div className="bg-gray-100/50 dark:bg-gray-900/50 backdrop-blur-md border border-gray-200/50 dark:border-gray-800/50 rounded-lg p-4 sm:p-6 shadow-sm">
                                            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-3 sm:mb-4 flex items-center gap-2">
                                                <Clock className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                                                Processing Time
                                            </h3>
                                            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4 text-sm">
                                                <div className="bg-white/60 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
                                                    <span className="text-gray-600 dark:text-gray-400 font-medium">Started:</span>
                                                    <p className="font-mono text-gray-900 dark:text-gray-100 mt-1 text-xs sm:text-sm">
                                                        {new Date(executionData.started_at).toLocaleString()}
                                                    </p>
                                                </div>
                                                <div className="bg-white/60 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
                                                    <span className="text-gray-600 dark:text-gray-400 font-medium">Completed:</span>
                                                    <p className="font-mono text-gray-900 dark:text-gray-100 mt-1 text-xs sm:text-sm">
                                                        {executionData.completed_at
                                                            ? new Date(executionData.completed_at).toLocaleString()
                                                            : 'N/A'
                                                        }
                                                    </p>
                                                </div>
                                                <div className="bg-white/60 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
                                                    <span className="text-gray-600 dark:text-gray-400 font-medium">Duration:</span>
                                                    <p className="font-mono text-xl sm:text-2xl font-bold text-gray-900 dark:text-gray-100 mt-1">
                                                        {executionData.processing_duration
                                                            ? `${(executionData.processing_duration / 1000).toFixed(1)}s`
                                                            : 'N/A'
                                                        }
                                                    </p>
                                                </div>
                                                <div className="bg-white/60 dark:bg-black/40 backdrop-blur-sm rounded-md p-3 border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
                                                    <span className="text-gray-600 dark:text-gray-400 font-medium">Audio Length:</span>
                                                    <p className="font-mono text-xl sm:text-2xl font-bold text-gray-900 dark:text-gray-100 mt-1">
                                                        {(() => {
                                                            const duration = getAudioDurationFromTranscript(transcript);
                                                            return duration ? formatDuration(duration) : 'N/A';
                                                        })()}
                                                    </p>
                                                </div>
                                            </div>
                                        </div>
                                        {/* Model Parameters for Single-track */}
                                        <div>
                                            <h3 className="text-lg font-semibold text-foreground mb-3 sm:mb-4 flex items-center gap-2">
                                                <Settings className="h-5 w-5 text-muted-foreground" />
                                                {(() => {
                                                    const modelFamily = executionData.actual_parameters?.model_family;
                                                    if (modelFamily === 'nvidia_parakeet') return 'NVIDIA Parakeet Parameters';
                                                    if (modelFamily === 'nvidia_canary') return 'NVIDIA Canary Parameters';
                                                    if (modelFamily === 'whisper') return 'WhisperX Parameters';
                                                    return 'Model Parameters';
                                                })()}
                                            </h3>
                                            <div className="glass-card rounded-lg p-3 sm:p-4">
                                                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4 text-sm">
                                                    {/* Model Settings */}
                                                    <div className="bg-white/40 dark:bg-black/40 backdrop-blur-sm rounded-lg p-4 border border-gray-200/50 dark:border-gray-800/50 shadow-sm">
                                                        <h4 className="font-semibold text-gray-800 dark:text-gray-200 mb-2 text-sm sm:text-base">Model & Configuration</h4>
                                                        <div className="space-y-2">
                                                            <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                <span className="text-muted-foreground text-xs sm:text-sm font-medium">Model Family:</span>
                                                                <span className="font-mono text-foreground text-xs sm:text-sm">
                                                                    {(() => {
                                                                        const family = executionData.actual_parameters?.model_family;
                                                                        if (family === 'nvidia_parakeet') return 'NVIDIA Parakeet';
                                                                        if (family === 'nvidia_canary') return 'NVIDIA Canary';
                                                                        if (family === 'whisper') return 'WhisperX';
                                                                        return family || 'N/A';
                                                                    })()}
                                                                </span>
                                                            </div>
                                                            <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                <span className="text-muted-foreground text-xs sm:text-sm font-medium">Model:</span>
                                                                <span className="font-mono text-foreground text-xs sm:text-sm break-all">{executionData.actual_parameters?.model || 'N/A'}</span>
                                                            </div>
                                                            {executionData.actual_parameters?.model_family === 'whisper' && (
                                                                <>
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Device:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.device || 'N/A'}</span>
                                                                    </div>
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Compute Type:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.compute_type || 'N/A'}</span>
                                                                    </div>
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Batch Size:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.batch_size || 'N/A'}</span>
                                                                    </div>
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Threads:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.threads || 0}</span>
                                                                    </div>
                                                                </>
                                                            )}
                                                            {executionData.actual_parameters?.model_family === 'nvidia_parakeet' && (
                                                                <>
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Context Left:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.attention_context_left || 256}</span>
                                                                    </div>
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Context Right:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.attention_context_right || 256}</span>
                                                                    </div>
                                                                </>
                                                            )}
                                                            {executionData.actual_parameters?.model_family === 'nvidia_canary' && (
                                                                <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                    <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Source Language:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters?.language || 'auto'}</span>
                                                                </div>
                                                            )}
                                                        </div>
                                                    </div>

                                                    {/* Processing Settings - Only for WhisperX */}
                                                    {executionData.actual_parameters?.model_family === 'whisper' && (
                                                        <div className="bg-white/40 dark:bg-black/40 backdrop-blur-sm rounded-lg p-4 border border-gray-200/50 dark:border-gray-800/50 shadow-sm">
                                                            <h4 className="font-semibold text-gray-800 dark:text-gray-200 mb-2 text-sm sm:text-base">Processing</h4>
                                                            <div className="space-y-1">
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Task:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.task || 'transcribe'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Language:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.language || 'auto'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Diarization:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.diarize ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">VAD Method:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.vad_method || 'pyannote'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">VAD Onset:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.vad_onset ?? 0.5}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">VAD Offset:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.vad_offset ?? 0.363}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Chunk Size:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.chunk_size || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Align Chunks:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.align_chunks ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Align Method:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.align_method || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Return Scores:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.return_scores ? 'Yes' : 'No'}</span>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    )}

                                                    {/* Speaker Diarization Settings - For all model families and multi-track jobs */}
                                                    {(executionData.actual_parameters?.diarize || executionData.is_multi_track) && (
                                                        <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4 border border-gray-200 dark:border-gray-800">
                                                            <h4 className="font-semibold text-gray-800 dark:text-gray-200 mb-2 text-sm sm:text-base flex items-center gap-2">
                                                                <Users className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                                                                Speaker Diarization
                                                            </h4>
                                                            <div className="space-y-1">
                                                                <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                    <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Enabled:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">
                                                                        {executionData.is_multi_track ? 'Yes (Multi-Track)' : (executionData.actual_parameters?.diarize ? 'Yes' : 'No')}
                                                                    </span>
                                                                </div>
                                                                {executionData.is_multi_track && (
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Number of Tracks:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.multi_track_files?.length || 0}</span>
                                                                    </div>
                                                                )}
                                                                {!executionData.is_multi_track && executionData.actual_parameters?.min_speakers && (
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Min Speakers:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters.min_speakers}</span>
                                                                    </div>
                                                                )}
                                                                {!executionData.is_multi_track && executionData.actual_parameters?.max_speakers && (
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Max Speakers:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters.max_speakers}</span>
                                                                    </div>
                                                                )}
                                                                {executionData.actual_parameters?.diarize_model && (
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Diarization Model:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{getDiarizationModelDisplayName(executionData.actual_parameters.diarize_model)}</span>
                                                                    </div>
                                                                )}
                                                                {executionData.actual_parameters?.speaker_embeddings !== undefined && (
                                                                    <div className="flex flex-col sm:flex-row sm:justify-between gap-1">
                                                                        <span className="text-gray-600 dark:text-gray-400 text-xs sm:text-sm font-medium">Speaker Embeddings:</span>
                                                                        <span className="font-mono text-gray-900 dark:text-gray-100 text-xs sm:text-sm">{executionData.actual_parameters.speaker_embeddings ? 'Yes' : 'No'}</span>
                                                                    </div>
                                                                )}
                                                            </div>
                                                        </div>
                                                    )}

                                                    {/* Advanced Settings - Only for WhisperX */}
                                                    {executionData.actual_parameters?.model_family === 'whisper' && (
                                                        <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4 border border-gray-200 dark:border-gray-800">
                                                            <h4 className="font-semibold text-gray-800 dark:text-gray-200 mb-2 text-sm sm:text-base">Advanced</h4>
                                                            <div className="space-y-1">
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Beam Size:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.beam_size || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Best Of:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.best_of || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Temperature:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.temperature || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">No Speech Threshold:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.no_speech_threshold || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Log Prob Threshold:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.logprob_threshold || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Compression Ratio Threshold:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.compression_ratio_threshold || 'N/A'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Condition On Previous Text:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.condition_on_previous_text ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Initial Prompt:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.initial_prompt || 'N/A'}</span>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    )}

                                                    {/* Advanced Settings - Only for NVIDIA Parakeet/Canary */}
                                                    {(executionData.actual_parameters?.model_family === 'nvidia_parakeet' || executionData.actual_parameters?.model_family === 'nvidia_canary') && (
                                                        <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4 border border-gray-200 dark:border-gray-800">
                                                            <h4 className="font-semibold text-gray-800 dark:text-gray-200 mb-2 text-sm sm:text-base">Advanced</h4>
                                                            <div className="space-y-1">
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Word Boost:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.word_boost ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Punctuation:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.enable_punctuation ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Automatic Capitalization:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.enable_automatic_capitalization ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">ITN:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.enable_inverse_text_normalization ? 'Yes' : 'No'}</span>
                                                                </div>
                                                                <div className="flex justify-between">
                                                                    <span className="text-gray-600 dark:text-gray-400">Speaker Diarization:</span>
                                                                    <span className="font-mono text-gray-900 dark:text-gray-100">{executionData.actual_parameters?.enable_speaker_diarization ? 'Yes' : 'No'}</span>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    )}
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                )}

                                {/* Parameters */}
                                {executionData.actual_parameters && (
                                    <div className="bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg p-4 sm:p-6">
                                        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-3 sm:mb-4 flex items-center gap-2">
                                            <Settings className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                                            Parameters Used
                                        </h3>
                                        <div className="bg-gray-900 rounded-md p-4 overflow-x-auto">
                                            <pre className="text-xs sm:text-sm font-mono text-gray-300">
                                                {JSON.stringify(executionData.actual_parameters, null, 2)}
                                            </pre>
                                        </div>
                                    </div>
                                )}
                            </div>
                        ) : (
                            <div className="py-8 text-center text-gray-500 dark:text-gray-400">
                                No execution data available
                            </div>
                        )}
                    </UIDialogContent>
                </UIDialog>

                {/* Logs dialog */}
                <UIDialog open={logsOpen} onOpenChange={setLogsOpen}>
                    <UIDialogContent className="sm:max-w-4xl w-[95vw] bg-white dark:bg-black-700 border-gray-200 dark:border-black-600 max-h-[90vh] overflow-y-auto">
                        <UIDialogHeader>
                            <UIDialogTitle className="text-gray-900 dark:text-gray-100 flex items-center gap-2">
                                <FileText className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                                Transcription Logs
                            </UIDialogTitle>
                            <UIDialogDescription className="text-gray-600 dark:text-gray-400">
                                Raw output from the transcription process
                            </UIDialogDescription>
                        </UIDialogHeader>

                        {logsLoading ? (
                            <div className="py-8 text-center">
                                <div className="animate-pulse">
                                    <div className="h-4 bg-gray-200 dark:bg-black-600 rounded w-3/4 mx-auto mb-4"></div>
                                    <div className="h-4 bg-gray-200 dark:bg-black-600 rounded w-1/2 mx-auto"></div>
                                </div>
                            </div>
                        ) : (
                            <div className="bg-gray-900 rounded-md p-4 overflow-x-auto">
                                <pre className="text-xs sm:text-sm font-mono text-gray-300 whitespace-pre-wrap">
                                    {logsContent || "No logs available."}
                                </pre>
                            </div>
                        )}
                    </UIDialogContent>
                </UIDialog>
                {/* Logs dialog */}
                < UIDialog open={logsOpen} onOpenChange={setLogsOpen} >
                    <UIDialogContent className="sm:max-w-4xl w-[95vw] bg-white dark:bg-black-700 border-gray-200 dark:border-black-600 max-h-[90vh] overflow-y-auto">
                        <UIDialogHeader>
                            <UIDialogTitle className="text-gray-900 dark:text-gray-100 flex items-center gap-2">
                                <FileText className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                                Transcription Logs
                            </UIDialogTitle>
                            <UIDialogDescription className="text-gray-600 dark:text-gray-400">
                                Raw output from the transcription process
                            </UIDialogDescription>
                        </UIDialogHeader>

                        {logsLoading ? (
                            <div className="py-8 text-center">
                                <div className="animate-pulse">
                                    <div className="h-4 bg-gray-200 dark:bg-black-600 rounded w-3/4 mx-auto mb-4"></div>
                                    <div className="h-4 bg-gray-200 dark:bg-black-600 rounded w-1/2 mx-auto"></div>
                                </div>
                            </div>
                        ) : (
                            <div className="bg-gray-900 rounded-md p-4 overflow-x-auto">
                                <pre className="text-xs sm:text-sm font-mono text-gray-300 whitespace-pre-wrap">
                                    {logsContent || "No logs available."}
                                </pre>
                            </div>
                        )}
                    </UIDialogContent>
                </UIDialog >

                {/* Portal: add-note bubble + editor */}
                {
                    ((showSelectionMenu || showEditor) && pendingSelection) ? (
                        createPortal(
                            <div>
                                {/* Backdrop to intercept clicks below the portal UI */}
                                <div
                                    style={{ position: 'fixed', inset: 0, zIndex: 9995, background: 'transparent' }}
                                    onMouseDown={() => {
                                        // Clicking backdrop cancels the add-note bubble
                                        if (showSelectionMenu && !showEditor) {
                                            setShowSelectionMenu(false);
                                            setPendingSelection(null);
                                            const sel = window.getSelection();
                                            sel?.removeAllRanges();
                                        }
                                    }}
                                />

                                {showSelectionMenu && (
                                    <div style={{ position: 'fixed', left: selectionViewportPos.x, top: selectionViewportPos.y, transform: 'translate(-50%, -100%)', zIndex: 10000 }} onMouseDown={(e) => e.stopPropagation()}>
                                        <div className="bg-gray-900 text-white text-base font-medium rounded-xl shadow-2xl px-6 py-3 flex items-center gap-3 pointer-events-auto hover:bg-black transition-colors ring-2 ring-white/20 transform hover:scale-105 duration-200">
                                            <button type="button" className="flex items-center gap-2" onClick={openEditorForSelection}>
                                                <Plus className="h-5 w-5" /> <span className="font-semibold">Add note</span>
                                            </button>
                                        </div>
                                    </div>
                                )}

                                {showEditor && (
                                    <div style={{ position: 'fixed', left: selectionViewportPos.x, top: selectionViewportPos.y + 18, transform: 'translate(-50%, 0)', zIndex: 10001 }} className="w-[min(90vw,520px)]" onMouseDown={(e) => e.stopPropagation()}>
                                        <div className="bg-white dark:bg-black-900 rounded-lg shadow-2xl p-3 pointer-events-auto">
                                            <div className="text-xs text-gray-500 dark:text-gray-400 border-l-2 border-gray-300 dark:border-black-600 pl-2 italic mb-2 max-h-32 overflow-auto">
                                                {pendingSelection.quote}
                                            </div>
                                            <textarea className="w-full text-sm bg-transparent border rounded-md p-2 border-gray-300 dark:border-black-700 text-gray-900 dark:text-gray-100" placeholder="Add a note..." value={newNoteContent} onChange={e => setNewNoteContent(e.target.value)} rows={4} />
                                            <div className="mt-2 flex items-center justify-end gap-2">
                                                <button type="button" className="px-2 py-1 text-sm rounded-md bg-gray-200 dark:bg-black-700" onClick={() => { setShowEditor(false); setPendingSelection(null); }}>{"Cancel"}</button>
                                                <button type="button" className="px-2 py-1 text-sm rounded-md bg-gray-900 text-white hover:bg-black transition-colors" onClick={saveNewNote}>{"Save"}</button>
                                            </div>
                                        </div>
                                    </div>
                                )}
                            </div>,
                            document.body
                        )
                    ) : null
                }

                {/* Notes sidebar (right, full height) */}
                {
                    notesOpen ? (
                        createPortal(
                            <div className="fixed inset-y-0 right-0 w-[88vw] max-w-[380px] md:max-w-[420px] bg-white dark:bg-black-900 shadow-2xl z-[9990]">
                                <div className="h-full flex flex-col">
                                    <div className="px-3 md:px-4 py-3">
                                        <div className="flex items-center justify-between">
                                            <h3 className="font-semibold text-gray-900 dark:text-gray-100 flex items-center gap-2">
                                                <StickyNote className="h-4 w-4" /> Notes
                                                <span className="ml-1 text-xs rounded-full px-1.5 py-0.5 bg-gray-200 dark:bg-black-700">{notes.length}</span>
                                            </h3>
                                            <button
                                                type="button"
                                                onClick={() => setNotesOpen(false)}
                                                className="h-8 w-8 inline-flex items-center justify-center rounded-md text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer"
                                                aria-label="Close notes"
                                            >
                                                <X className="h-4 w-4" />
                                            </button>
                                        </div>
                                    </div>
                                    <div className="flex-1 overflow-y-auto px-3 md:px-4 pb-4">
                                        <NotesSidebar
                                            notes={notes}
                                            onEdit={updateNote}
                                            onDelete={deleteNote}
                                            onJumpTo={(t) => {
                                                const player = audioPlayerRef.current;
                                                if (player) {
                                                    const dur = player.getDuration();
                                                    player.seekTo(Math.min(0.999, Math.max(0, t / dur)));
                                                    setCurrentTime(t);
                                                }
                                            }}
                                        />
                                    </div>
                                </div>
                            </div>,
                            document.body
                        )
                    ) : null
                }

            </div >
        </div >
    );
});
