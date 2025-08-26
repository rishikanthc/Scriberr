import { useState, useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { ArrowLeft, Play, Pause, List, AlignLeft, MessageCircle, Download, FileText, FileJson, FileImage, Check, StickyNote, Plus, X } from "lucide-react";
import WaveSurfer from "wavesurfer.js";
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
import { useTheme } from "../contexts/ThemeContext";
import { ThemeSwitcher } from "./ThemeSwitcher";
import { useAuth } from "../contexts/AuthContext";
import { ChatInterface } from "./ChatInterface";
import type { Note } from "../types/note";
import { NotesSidebar } from "./NotesSidebar";

interface AudioFile {
	id: string;
	title?: string;
	status: "uploaded" | "pending" | "processing" | "completed" | "failed";
	created_at: string;
	audio_path: string;
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

export function AudioDetailView({ audioId }: AudioDetailViewProps) {
	const { navigate } = useRouter();
	const { theme } = useTheme();
	const { getAuthHeaders } = useAuth();
	const [audioFile, setAudioFile] = useState<AudioFile | null>(null);
	const [transcript, setTranscript] = useState<Transcript | null>(null);
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
	const waveformRef = useRef<HTMLDivElement>(null);
	const wavesurferRef = useRef<WaveSurfer | null>(null);
	const transcriptRef = useRef<HTMLDivElement>(null);
	const highlightedWordRef = useRef<HTMLSpanElement>(null);

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
    const [pendingSelection, setPendingSelection] = useState<{startIdx:number; endIdx:number; startTime:number; endTime:number; quote:string} | null>(null);
    const [newNoteContent, setNewNoteContent] = useState("");
    const [showEditor, setShowEditor] = useState(false);
    const [selectionViewportPos, setSelectionViewportPos] = useState<{x:number,y:number}>({x:0,y:0});

useEffect(() => {
    console.log("AudioDetailView mounted, audioId:", audioId);
    fetchAudioDetails();
    fetchNotes();
}, [audioId]);

	// Initialize WaveSurfer when audioFile is available - with proper DOM timing
	useEffect(() => {
		if (!audioFile) {
			console.log("No audioFile yet, skipping WaveSurfer initialization");
			return;
		}

		console.log("AudioFile available, checking DOM readiness...");

		// Use a longer timeout and multiple checks to ensure DOM is ready
		const checkAndInitialize = () => {
			console.log("Checking DOM and WaveSurfer state:", {
				audioFile: !!audioFile,
				waveformRefElement: !!waveformRef.current,
				wavesurferInstance: !!wavesurferRef.current,
				waveformRefType: waveformRef.current?.tagName,
			});

			if (waveformRef.current && !wavesurferRef.current) {
				console.log("✅ All conditions met! Initializing WaveSurfer...");
				initializeWaveSurfer();
			} else if (!waveformRef.current) {
				console.log("❌ Waveform DOM element not ready yet, will retry...");
				// Retry after a bit more time
				setTimeout(checkAndInitialize, 200);
			} else if (wavesurferRef.current) {
				console.log("WaveSurfer already initialized");
			}
		};

		// Initial check with a small delay
		const timer = setTimeout(checkAndInitialize, 50);

		return () => {
			clearTimeout(timer);
			if (wavesurferRef.current) {
				console.log("Cleaning up WaveSurfer");
				wavesurferRef.current.destroy();
				wavesurferRef.current = null;
			}
		};
	}, [audioFile]);

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
	}, [currentTime, transcript?.word_segments, isPlaying, currentWordIndex]);

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

	const fetchAudioDetails = async () => {
		console.log("Fetching audio details for ID:", audioId);
		try {
			// Fetch audio file details
			const audioResponse = await fetch(`/api/v1/transcription/${audioId}`, {
				headers: {
					...getAuthHeaders(),
				},
			});

			console.log("Audio response status:", audioResponse.status);

			if (audioResponse.ok) {
				const audioData = await audioResponse.json();
				console.log("Audio data received:", audioData);
				setAudioFile(audioData);

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
						// The API returns transcript data in a nested structure
						if (transcriptData.transcript) {
							// Check if transcript has segments or text
							if (typeof transcriptData.transcript === "string") {
								setTranscript({ text: transcriptData.transcript });
							} else if (transcriptData.transcript.text) {
								setTranscript({
									text: transcriptData.transcript.text,
									segments: transcriptData.transcript.segments,
								});
							} else if (transcriptData.transcript.segments) {
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
            const res = await fetch(`/api/v1/transcription/${audioId}/notes`, { headers: { ...getAuthHeaders() }});
            if (res.ok) {
                const data = await res.json();
                setNotes(sortNotes(data));
            }
        } catch (e) { console.error("Failed to fetch notes", e); }
    };

	const initializeWaveSurfer = async () => {
		if (!waveformRef.current || !audioFile) return;

		try {
			// First, try to load the audio file manually to check if it's accessible
			const audioUrl = `/api/v1/transcription/${audioId}/audio`;
			console.log("Testing audio URL:", audioUrl);

			const response = await fetch(audioUrl, {
				headers: {
					...getAuthHeaders(),
				},
			});

			if (!response.ok) {
				console.error(
					"Audio file request failed:",
					response.status,
					response.statusText,
				);
				const errorText = await response.text();
				console.error("Error response body:", errorText);
				return;
			}

			console.log("Audio file accessible, creating WaveSurfer...");

			// Theme-aware colors
			const isDark = theme === "dark";
			const waveColor = isDark ? "#4b5563" : "#d1d5db"; // dark: gray-600, light: gray-300
			const progressColor = "#3b82f6"; // blue-500 for both themes

			// Create WaveSurfer instance
			wavesurferRef.current = WaveSurfer.create({
				container: waveformRef.current,
				waveColor,
				progressColor,
				barWidth: 2,
				barGap: 1,
				barRadius: 2,
				height: 80,
				normalize: true,
				backend: "WebAudio",
			});

			// Load the audio blob
			const audioBlob = await response.blob();
			const audioObjectURL = URL.createObjectURL(audioBlob);

			console.log("Loading audio blob into WaveSurfer...");
			await wavesurferRef.current.load(audioObjectURL);

			wavesurferRef.current.on("ready", () => {
				console.log("WaveSurfer is ready");
			});

			wavesurferRef.current.on("load", (url) => {
				console.log("WaveSurfer loading:", url);
			});

			wavesurferRef.current.on("error", (error) => {
				console.error("WaveSurfer error:", error);
			});

			wavesurferRef.current.on("play", () => {
				setIsPlaying(true);
			});

			wavesurferRef.current.on("pause", () => {
				setIsPlaying(false);
			});

			wavesurferRef.current.on("finish", () => {
				setIsPlaying(false);
				setCurrentWordIndex(null);
			});

			// Add time update listener for word highlighting
			wavesurferRef.current.on("audioprocess", (time) => {
				setCurrentTime(time);
			});
		} catch (error) {
			console.error("Failed to initialize WaveSurfer:", error);
		}
	};

    const togglePlayPause = () => {
        if (wavesurferRef.current) {
            wavesurferRef.current.playPause();
        }
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
        await fetch(`/api/v1/notes/${id}`, { method: 'DELETE', headers: { ...getAuthHeaders() }});
        setNotes(prev => prev.filter(n => n.id !== id));
    };

	// Render transcript with word-level highlighting
	const renderHighlightedTranscript = () => {
		if (!transcript?.word_segments) {
			return transcript?.text || '';
		}

		return transcript.word_segments.map((word, index) => {
			const isHighlighted = index === currentWordIndex;
			const isAnnotated = notes.some(n => index >= n.start_word_index && index <= n.end_word_index);
            return (
                <span
                    key={index}
                    ref={isHighlighted ? highlightedWordRef : undefined}
                    data-word-index={index}
                    data-word={word.word}
                    data-start={word.start}
                    data-end={word.end}
                    className={`cursor-text transition-colors duration-150 hover:bg-blue-100 dark:hover:bg-blue-800 inline ${
                        isHighlighted
                            ? 'bg-yellow-300 dark:bg-yellow-500 dark:text-black px-1 rounded'
                            : isAnnotated ? 'bg-amber-100/70 dark:bg-amber-800/40 px-0.5 rounded' : 'px-0.5'
                    }`}
                >
                    {word.word}{" "}
                </span>
            );
        });
    };

	// Render segment with word-level highlighting for expanded view
	const renderSegmentWithHighlighting = (segment: any) => {
		if (!transcript?.word_segments) {
			return segment.text.trim();
		}

		// Find words that belong to this segment
		const segmentWords = transcript.word_segments.filter(
			word => word.start >= segment.start && word.end <= segment.end
		);

		if (segmentWords.length === 0) {
			return segment.text.trim();
		}

		return segmentWords.map((word, index) => {
			const globalIndex = transcript.word_segments?.findIndex(w => w === word) ?? -1;
			const isHighlighted = globalIndex === currentWordIndex;
			const isAnnotated = notes.some(n => globalIndex >= n.start_word_index && globalIndex <= n.end_word_index);
            return (
                <span
                    key={index}
                    ref={isHighlighted ? highlightedWordRef : undefined}
                    data-word-index={globalIndex}
                    data-word={word.word}
                    data-start={word.start}
                    data-end={word.end}
                    className={`cursor-text transition-colors duration-150 hover:bg-blue-100 dark:hover:bg-blue-800 inline ${
                        isHighlighted
                            ? 'bg-yellow-300 dark:bg-yellow-500 dark:text-black px-1 rounded'
                            : isAnnotated ? 'bg-amber-100/70 dark:bg-amber-800/40 px-0.5 rounded' : 'px-0.5'
                    }`}
                >
                    {word.word}{" "}
                </span>
            );
        });
    };

	const getFileName = (audioPath: string) => {
		const parts = audioPath.split("/");
		return parts[parts.length - 1];
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
				srtContent += `${counter}\n${startTime} --> ${endTime}\n${segment.text.trim()}\n\n`;
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
					content += `${segment.speaker}: `;
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
						segmentData.speaker = segment.speaker;
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

	if (loading) {
		return (
			<div className="min-h-screen bg-gray-50 dark:bg-gray-900">
				<div className="mx-auto w-full max-w-6xl px-2 sm:px-6 md:px-8 py-3 sm:py-6">
					<div className="bg-white dark:bg-gray-800 rounded-xl p-3 sm:p-6">
						<div className="animate-pulse">
							<div className="h-6 bg-gray-200 dark:bg-gray-600 rounded w-1/4 mb-4"></div>
							<div className="h-4 bg-gray-200 dark:bg-gray-600 rounded w-1/2 mb-8"></div>
							<div className="h-20 bg-gray-200 dark:bg-gray-600 rounded mb-8"></div>
							<div className="space-y-3">
								{[...Array(5)].map((_, i) => (
									<div
										key={i}
										className="h-4 bg-gray-200 dark:bg-gray-600 rounded"
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
			<div className="min-h-screen bg-gray-50 dark:bg-gray-900">
				<div className="mx-auto w-full max-w-6xl px-2 sm:px-6 md:px-8 py-3 sm:py-6">
					<div className="bg-white dark:bg-gray-800 rounded-xl p-3 sm:p-6 text-center">
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
		<div className="min-h-screen bg-gray-50 dark:bg-gray-900">
			<div className="mx-auto w-full max-w-6xl px-2 sm:px-6 md:px-8 py-3 sm:py-6">
				{/* Header with back button and theme switcher */}
				<div className="flex items-center justify-between mb-3 sm:mb-6">
					<Button onClick={handleBack} variant="outline" size="sm" className="cursor-pointer">
						<ArrowLeft className="mr-2 h-4 w-4" />
						Back to Audio Files
					</Button>
					<ThemeSwitcher />
				</div>

				{/* Audio Player Section */}
				<div className="bg-white dark:bg-gray-800 rounded-xl p-3 sm:p-6 mb-3 sm:mb-6">
					<div className="mb-6">
						<h1 className="text-2xl font-bold text-gray-900 dark:text-gray-50 mb-2">
							{audioFile.title || getFileName(audioFile.audio_path)}
						</h1>
						<p className="text-gray-600 dark:text-gray-400 text-sm">
							Added on {formatDate(audioFile.created_at)}
						</p>
					</div>

					{/* Audio Player Controls */}
					<div className="mb-6">
						<div className="flex items-center gap-4">
							{/* Circular Play/Pause Button */}
							<button
								onClick={togglePlayPause}
								className="w-12 h-12 sm:w-16 sm:h-16 rounded-full bg-blue-500 hover:bg-blue-600 text-white shadow-lg hover:shadow-xl transition-all duration-200 hover:scale-105 flex items-center justify-center group cursor-pointer"
							>
								{isPlaying ? (
									<Pause className="h-5 w-5 sm:h-6 sm:w-6 group-hover:scale-110 transition-transform" />
								) : (
									<Play className="h-5 w-5 sm:h-6 sm:w-6 ml-0.5 group-hover:scale-110 transition-transform" />
								)}
							</button>

							{/* WaveSurfer Container */}
							<div className="flex-1">
								<div
									ref={waveformRef}
									className="w-full bg-gray-50 dark:bg-gray-700 rounded-lg p-2 sm:p-4"
									style={{ minHeight: "80px" }}
								/>
							</div>
						</div>
					</div>
				</div>

				{audioFile.status === "completed" && transcript && (
					<div className="bg-white dark:bg-gray-800 rounded-xl p-3 sm:p-6">
						<div className="flex items-center justify-between mb-3 sm:mb-6">
							<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-50">
								{viewMode === "transcript" ? "Transcript" : "Chat with Transcript"}
							</h2>

							<div className="flex items-center gap-2">
								{/* Download Transcript */}
								<DropdownMenu>
									<DropdownMenuTrigger asChild>
										<Button
											variant="outline"
											size="sm"
											className="h-8 w-8 p-0 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer"
											title="Download Transcript"
										>
											<Download className="h-4 w-4" />
										</Button>
									</DropdownMenuTrigger>
                            <DropdownMenuContent align="end" className="w-48 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 shadow-md">
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

								{/* Open Chat Page */}
								<Button
									onClick={() => navigate({ path: 'chat', params: { audioId } })}
									variant="outline"
									size="sm"
									className="flex items-center gap-2 cursor-pointer"
								>
									<MessageCircle className="h-4 w-4" />
									<span className="hidden sm:inline">Open Chat</span>
								</Button>

                                {/* Transcript view mode toggle (icon-only) */}
                                {viewMode === "transcript" && (
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={() => setTranscriptMode(transcriptMode === "compact" ? "expanded" : "compact")}
                                        className="h-8 w-8 p-0 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer"
                                        title={transcriptMode === "compact" ? "Switch to Timeline view" : "Switch to Compact view"}
                                    >
                                        {transcriptMode === "compact" ? (
                                            <List className="h-4 w-4" />
                                        ) : (
                                            <AlignLeft className="h-4 w-4" />
                                        )}
                                    </Button>
                                )}

								{/* Notes toggle */}
								<Button
									variant="outline"
									size="sm"
									onClick={() => setNotesOpen(v => !v)}
									className="flex items-center gap-2 cursor-pointer"
									title="Toggle notes"
								>
									<StickyNote className="h-4 w-4" />
									<span className="hidden sm:inline">Notes</span>
									<span className="ml-1 text-xs rounded-full px-1.5 py-0.5 bg-gray-200 dark:bg-gray-700">{notes.length}</span>
								</Button>
							</div>
						</div>

						{/* Content Area - Show transcript or chat based on view mode */}
						{viewMode === "transcript" ? (
							<div className="relative overflow-hidden">
								<div
                                className={`transition-all duration-300 ease-in-out ${
                                    transcriptMode === "compact"
                                        ? "opacity-100 translate-y-0"
                                        : "opacity-0 -translate-y-4 absolute inset-0 pointer-events-none"
                                }`}
								>
									{transcriptMode === "compact" && (
                                    <div 
                                        ref={transcriptRef}
                                        className="prose prose-gray dark:prose-invert max-w-none relative select-text cursor-text"
                                    >
                                    <p className="text-gray-700 dark:text-gray-300 leading-relaxed break-words select-text">
                                        {renderHighlightedTranscript()}
                                    </p>

                                    {/* Selection bubble and editor moved to portal */}
										</div>
									)}
								</div>

								<div
                                className={`transition-all duration-300 ease-in-out ${
                                    transcriptMode === "expanded"
                                        ? "opacity-100 translate-y-0"
                                        : "opacity-0 translate-y-4 absolute inset-0 pointer-events-none"
                                }`}
								>
									{transcriptMode === "expanded" && transcript.segments && (
                                    <div 
                                        ref={transcriptRef}
                                        className="space-y-4 relative select-text cursor-text"
                                    >
											{transcript.segments.map((segment, index) => (
												<div
													key={index}
													className="flex gap-4 p-3 rounded-lg bg-gray-50 dark:bg-gray-700 hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors duration-150"
												>
													<div className="flex-shrink-0 flex flex-col gap-2">
														<span className="inline-block px-2 py-1 text-xs font-mono bg-blue-100 dark:bg-blue-700 text-blue-800 dark:text-blue-200 rounded">
															{formatTimestamp(segment.start)}
														</span>
														{segment.speaker && (
															<span className="inline-block px-2 py-1 text-xs font-medium bg-green-100 dark:bg-green-700 text-green-800 dark:text-green-200 rounded">
																{segment.speaker}
															</span>
														)}
													</div>
                                            <div className="flex-1 min-w-0">
                                                <p className="text-gray-700 dark:text-gray-200 leading-relaxed break-words select-text">
                                                    {renderSegmentWithHighlighting(segment)}
                                                </p>
                                            </div>
												</div>
											))}
										</div>
									)}
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

					)}



				{/* Status Messages */}
				{audioFile.status !== "completed" && (
					<div className="bg-white dark:bg-gray-700 rounded-xl p-6">
						<div className="text-center">
							<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-50 mb-2">
								{audioFile.status === "processing" &&
									"Transcription in Progress"}
								{audioFile.status === "pending" && "Transcription Queued"}
								{audioFile.status === "uploaded" && "Ready for Transcription"}
								{audioFile.status === "failed" && "Transcription Failed"}
							</h2>
							<p className="text-gray-600 dark:text-gray-400">
								{audioFile.status === "processing" &&
									"Please wait while we process your audio file..."}
								{audioFile.status === "pending" &&
									"Your audio file is in the transcription queue."}
								{audioFile.status === "uploaded" &&
									"Start transcription from the audio files list."}
								{audioFile.status === "failed" &&
									"There was an error processing your audio file."}
							</p>
						</div>
					</div>
				)}

			{/* Download Options Dialog */}
			<Dialog open={downloadDialogOpen} onOpenChange={setDownloadDialogOpen}>
				<DialogContent className="sm:max-w-md bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700">
					<DialogHeader>
						<DialogTitle className="text-gray-900 dark:text-gray-100">
							Download as {downloadFormat.toUpperCase()}
						</DialogTitle>
						<DialogDescription className="text-gray-600 dark:text-gray-400">
							Choose your download options for the transcript.
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
							<div className="text-sm text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-800 p-3 rounded-md">
								<div className="flex items-center gap-2">
									<Check className="h-4 w-4 text-green-500" />
									Transcript will be formatted as a single paragraph
								</div>
							</div>
						)}

						{(includeSpeakerLabels || includeTimestamps) && (
							<div className="text-sm text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-800 p-3 rounded-md">
								<div className="flex items-center gap-2">
									<Check className="h-4 w-4 text-green-500" />
									Transcript will be formatted in segments with selected labels
								</div>
							</div>
						)}
					</div>

					<DialogFooter className="gap-2">
						<Button 
							variant="outline" 
							onClick={() => setDownloadDialogOpen(false)}
							className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
						>
							Cancel
						</Button>
						<Button 
							onClick={handleDownloadConfirm}
							className="bg-blue-600 dark:bg-blue-700 hover:bg-blue-700 dark:hover:bg-blue-800 text-white"
						>
							<Download className="mr-2 h-4 w-4" />
							Download {downloadFormat.toUpperCase()}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			{/* Portal: add-note bubble + editor */}
				{((showSelectionMenu || showEditor) && pendingSelection) ? (
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
									<div className="bg-gray-900 text-white text-xs rounded-md shadow-2xl px-2 py-1 flex items-center gap-1 pointer-events-auto">
										<button type="button" className="flex items-center gap-1 hover:opacity-90" onClick={openEditorForSelection}>
											<Plus className="h-3 w-3" /> Add note
										</button>
									</div>
								</div>
							)}

							{showEditor && (
								<div style={{ position: 'fixed', left: selectionViewportPos.x, top: selectionViewportPos.y + 18, transform: 'translate(-50%, 0)', zIndex: 10001 }} className="w-[min(90vw,520px)]" onMouseDown={(e) => e.stopPropagation()}>
									<div className="bg-white dark:bg-gray-900 rounded-lg shadow-2xl p-3 pointer-events-auto">
										<div className="text-xs text-gray-500 dark:text-gray-400 border-l-2 border-gray-300 dark:border-gray-600 pl-2 italic mb-2 max-h-32 overflow-auto">
											{pendingSelection.quote}
										</div>
										<textarea className="w-full text-sm bg-transparent border rounded-md p-2 border-gray-300 dark:border-gray-700 text-gray-900 dark:text-gray-100" placeholder="Add a note..." value={newNoteContent} onChange={e => setNewNoteContent(e.target.value)} rows={4} />
										<div className="mt-2 flex items-center justify-end gap-2">
											<button type="button" className="px-2 py-1 text-sm rounded-md bg-gray-200 dark:bg-gray-700" onClick={() => { setShowEditor(false); setPendingSelection(null); }}>{"Cancel"}</button>
											<button type="button" className="px-2 py-1 text-sm rounded-md bg-blue-600 text-white" onClick={saveNewNote}>{"Save"}</button>
										</div>
									</div>
								</div>
							)}
						</div>,
						document.body
					)
				) : null}

				{/* Notes sidebar (right, full height) */}
				{notesOpen ? (
					createPortal(
						<div className="fixed inset-y-0 right-0 w-[88vw] max-w-[380px] md:max-w-[420px] bg-white dark:bg-gray-900 shadow-2xl z-[9990]">
							<div className="h-full flex flex-col">
								<div className="px-3 md:px-4 py-3">
									<div className="flex items-center justify-between">
										<h3 className="font-semibold text-gray-900 dark:text-gray-100 flex items-center gap-2">
											<StickyNote className="h-4 w-4" /> Notes
											<span className="ml-1 text-xs rounded-full px-1.5 py-0.5 bg-gray-200 dark:bg-gray-700">{notes.length}</span>
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
										onJumpTo={(t) => { if (wavesurferRef.current) { const dur = wavesurferRef.current.getDuration(); wavesurferRef.current.seekTo(Math.min(0.999, Math.max(0, t / dur))); setCurrentTime(t); }}}
									/>
								</div>
							</div>
						</div>,
						document.body
					)
				) : null}

				</div>
		</div>
	);
}
