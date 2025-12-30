import { useState, useEffect, useMemo, useCallback, memo, useRef } from "react";
import { useInView } from "react-intersection-observer";
import {
	Loader2,
	Trash2,
	StopCircle,
	Music,
	FileAudio,
	Wand2,
	Check,
	AlertCircle,
	Clock,
	X
} from "lucide-react";
import { WandAdvancedIcon } from "@/components/icons/WandAdvancedIcon";
// Checkbox removed

import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "@/components/ui/tooltip";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { TranscriptionConfigDialog, type WhisperXParams } from "@/components/TranscriptionConfigDialog";
import { TranscribeDDialog } from "@/components/TranscribeDDialog";
import { useNavigate } from "react-router-dom";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useAudioListInfinite, type AudioFile } from "@/features/transcription/hooks/useAudioFiles";
import { useTranscriptionEvents } from "@/features/transcription/hooks/useTranscriptionEvents";

const JobStatusMonitor = memo(function JobStatusMonitor({ jobId }: { jobId: string }) {
	useTranscriptionEvents(jobId);
	return null;
});


import { DebouncedSearchInput } from "@/components/DebouncedSearchInput";
import { SwipeableItem } from "@/components/ui/swipeable-item";
import { useSwipeHint } from "@/hooks/use-swipe-hint";



interface AudioFilesTableProps {
	refreshTrigger?: number; // Optional now, kept for compatibility during refactor
	onTranscribe?: (jobId: string) => void;
}

export const AudioFilesTable = memo(function AudioFilesTable({
	onTranscribe,
}: AudioFilesTableProps) {
	const navigate = useNavigate();
	const { getAuthHeaders } = useAuth();
	const { shouldShowHint, markHintShown } = useSwipeHint();

	// Table State
	const sorting = [
		{ id: "created_at", desc: true }
	];
	const [globalFilter, setGlobalFilter] = useState("");

	// Query
	const {
		data: infiniteData,
		fetchNextPage,
		hasNextPage,
		isFetchingNextPage,
		isLoading: queryLoading,
		refetch
	} = useAudioListInfinite({
		limit: 20, // Fetch 20 items per page
		search: globalFilter,
		sortBy: sorting[0]?.id,
		sortOrder: sorting[0]?.desc ? 'desc' : 'asc'
	});

	// Get active jobs for real-time monitoring
	const activeJobs = useMemo(() => {
		if (!infiniteData) return [];
		return infiniteData.pages.flatMap(page => page.jobs).filter(
			job => job.status === 'processing' || job.status === 'pending'
		);
	}, [infiniteData]);

	// Flatten data from pages
	const data = useMemo(() => {
		return infiniteData?.pages.flatMap(page => page.jobs) || [];
	}, [infiniteData]);

	const loading = queryLoading;
	// Pagination state no longer needed in same way

	// Infinite Scroll Trigger
	const { ref: scrollRef, inView } = useInView({
		threshold: 0,
		rootMargin: '400px', // Start fetching 400px before reaching the bottom
	});

	useEffect(() => {
		if (inView && hasNextPage && !isFetchingNextPage) {
			fetchNextPage();
		}
	}, [inView, hasNextPage, isFetchingNextPage, fetchNextPage]);


	// Local state for UI
	// queuePositions state removed


	// Selection and Dialog state
	const [rowSelection, setRowSelection] = useState<Record<string, boolean>>({});
	const longPressTimer = useRef<NodeJS.Timeout | undefined>(undefined);
	const isLongPress = useRef(false);
	const touchStartPos = useRef<{ x: number; y: number } | null>(null);
	const isSwipingRef = useRef(false);
	const suppressClickUntil = useRef(0);

	// Threshold to cancel long-press (in pixels)
	const LONG_PRESS_CANCEL_THRESHOLD = 10;

	const handleAudioClick = useCallback((audioId: string) => {
		navigate(`/audio/${audioId}`);
	}, [navigate]);

	const toggleSelection = useCallback((id: string) => {
		setRowSelection(prev => {
			const next = { ...prev };
			if (next[id]) {
				delete next[id];
			} else {
				next[id] = true;
			}
			return next;
		});
	}, []);

	// Called when swipe starts/ends to coordinate with clicks
	const handleSwipeStateChange = useCallback((isSwiping: boolean) => {
		isSwipingRef.current = isSwiping;
		if (!isSwiping) {
			// After swipe ends, suppress clicks for 150ms to prevent accidental navigation
			suppressClickUntil.current = Date.now() + 150;
		}
		// Cancel any pending long-press when swiping starts
		if (isSwiping && longPressTimer.current) {
			clearTimeout(longPressTimer.current);
		}
	}, []);

	const handleRowClick = useCallback((file: AudioFile, e: React.MouseEvent) => {
		// Suppress click if we just finished a swipe
		if (Date.now() < suppressClickUntil.current || isSwipingRef.current) {
			e.stopPropagation();
			return;
		}

		if (isLongPress.current) {
			isLongPress.current = false;
			return; // Ignore click after long press
		}

		const isSelectionMode = Object.keys(rowSelection).length > 0;

		if (isSelectionMode || e.shiftKey) {
			e.stopPropagation();
			toggleSelection(file.id);
		} else {
			handleAudioClick(file.id);
		}
	}, [rowSelection, handleAudioClick, toggleSelection]);

	const startLongPress = useCallback((id: string, e: React.TouchEvent | React.MouseEvent) => {
		isLongPress.current = false;

		// Record touch start position for movement detection
		if ('touches' in e) {
			const touch = e.touches[0];
			touchStartPos.current = { x: touch.clientX, y: touch.clientY };
		} else {
			touchStartPos.current = { x: e.clientX, y: e.clientY };
		}

		longPressTimer.current = setTimeout(() => {
			isLongPress.current = true;
			toggleSelection(id);
			// Haptic feedback
			if (navigator.vibrate) navigator.vibrate(50);
		}, 600); // 600ms threshold
	}, [toggleSelection]);

	// Cancel long-press if finger moves beyond threshold
	const handleTouchMove = useCallback((e: React.TouchEvent) => {
		if (!touchStartPos.current || !longPressTimer.current) return;

		const touch = e.touches[0];
		const dx = Math.abs(touch.clientX - touchStartPos.current.x);
		const dy = Math.abs(touch.clientY - touchStartPos.current.y);

		// If moved beyond threshold, cancel long-press
		if (dx > LONG_PRESS_CANCEL_THRESHOLD || dy > LONG_PRESS_CANCEL_THRESHOLD) {
			if (longPressTimer.current) {
				clearTimeout(longPressTimer.current);
				longPressTimer.current = undefined;
			}
		}
	}, []);

	const clearLongPress = useCallback(() => {
		if (longPressTimer.current) {
			clearTimeout(longPressTimer.current);
			longPressTimer.current = undefined;
		}
		touchStartPos.current = null;
	}, []);

	const [bulkActionLoading, setBulkActionLoading] = useState(false);
	const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = useState(false);
	// data state removed
	// loading state removed
	// totalItems derived from query
	// pageCount derived from query
	const [configDialogOpen, setConfigDialogOpen] = useState(false);
	const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
	const [transcriptionLoading, setTranscriptionLoading] = useState(false);
	const [killingJobs, setKillingJobs] = useState<Set<string>>(new Set());
	const [transcribeDDialogOpen, setTranscribeDDialogOpen] = useState(false);
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	const [trackProgress, setTrackProgress] = useState<Record<string, any>>({});

	// Dialog state management
	const [stopDialogOpen, setStopDialogOpen] = useState(false);
	const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
	const [selectedFile, setSelectedFile] = useState<AudioFile | null>(null);

	// Calculate queue positions for pending jobs
	const queuePositions = useMemo(() => {
		const pendingJobs = data.filter(job => job.status === "pending");
		// Sort by created_at ascending (FIFO)
		pendingJobs.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime());

		const positions: Record<string, number> = {};
		pendingJobs.forEach((job, index) => {
			positions[job.id] = index + 1;
		});
		return positions;
	}, [data]);

	// Side effects for queue and progress
	useEffect(() => {
		if (data.length > 0) {
			// queuePositions logic derived above

			// Fetch track progress for processing multi-track jobs
			const processingMultiTrackJobs = data.filter(job =>
				job.is_multi_track && (job.status === "processing" || job.status === "pending")
			);

			if (processingMultiTrackJobs.length > 0) {
				const fetchTrackProgress = async (jobs: AudioFile[]) => {
					try {
						const progressPromises = jobs.map(async (job) => {
							const response = await fetch(`/api/v1/transcription/${job.id}/track-progress`, {
								headers: { ...getAuthHeaders() },
							});
							if (response.ok) {
								const progress = await response.json();
								return { jobId: job.id, progress };
							}
							return null;
						});

						const results = await Promise.all(progressPromises);
						const progressData: Record<string, any> = {}; // eslint-disable-line @typescript-eslint/no-explicit-any
						results.forEach(result => {
							if (result) progressData[result.jobId] = result.progress;
						});
						setTrackProgress(prev => ({ ...prev, ...progressData }));
					} catch (error) {
						console.error("Failed to fetch track progress:", error);
					}
				};
				fetchTrackProgress(processingMultiTrackJobs);
			}
		}
	}, [data, getAuthHeaders]);

	// Fetch queue positions removed


	// Handle transcribe action - opens configuration dialog
	const handleTranscribeClick = useCallback((jobId: string) => {
		setSelectedJobId(jobId);
		setConfigDialogOpen(true);
	}, []);

	// Handle transcribe-D action - opens profile selection dialog
	const handleTranscribeDClick = useCallback((jobId: string) => {
		setSelectedJobId(jobId);
		setTranscribeDDialogOpen(true);
	}, []);

	// Handle actual transcription start with parameters
	const handleStartTranscription = useCallback(async (params: WhisperXParams) => {
		if (!selectedJobId) return;

		// Validate multi-track compatibility
		const selectedJob = data.find(job => job.id === selectedJobId);
		if (selectedJob?.is_multi_track && !params.is_multi_track_enabled) {
			alert("Multi-track audio requires a profile with multi-track transcription enabled. Please select or create a profile with multi-track support.");
			return;
		}

		if (!selectedJob?.is_multi_track && params.is_multi_track_enabled) {
			alert("Multi-track transcription cannot be used with single-track audio files.");
			return;
		}

		try {
			setTranscriptionLoading(true);

			const response = await fetch(`/api/v1/transcription/${selectedJobId}/start`, {
				method: "POST",
				headers: {
					...getAuthHeaders(),
					"Content-Type": "application/json",
				},
				body: JSON.stringify(params),
			});

			if (response.ok) {
				// Close dialog and refresh
				setConfigDialogOpen(false);
				setSelectedJobId(null);
				// Refresh the list immediately to show the new processing status
				// This also triggers SSE connection if it wasn't active
				refetch();
				if (onTranscribe) {
					onTranscribe(selectedJobId);
				}
			} else {
				alert("Failed to start transcription");
			}
		} catch {
			alert("Error starting transcription");
		} finally {
			setTranscriptionLoading(false);
		}
	}, [selectedJobId, refetch, onTranscribe, data, getAuthHeaders]);

	// Handle actual transcription start with profile parameters
	const handleStartTranscriptionWithProfile = useCallback(async (params: WhisperXParams) => {
		if (!selectedJobId) return;

		// Validate multi-track compatibility
		const selectedJob = data.find(job => job.id === selectedJobId);
		if (selectedJob?.is_multi_track && !params.is_multi_track_enabled) {
			alert("Multi-track audio requires a profile with multi-track transcription enabled. Please select a different profile with multi-track support.");
			return;
		}

		if (!selectedJob?.is_multi_track && params.is_multi_track_enabled) {
			alert("Multi-track transcription cannot be used with single-track audio files. Please select a different profile.");
			return;
		}

		try {
			setTranscriptionLoading(true);

			const response = await fetch(`/api/v1/transcription/${selectedJobId}/start`, {
				method: "POST",
				headers: {
					...getAuthHeaders(),
					"Content-Type": "application/json",
				},
				body: JSON.stringify(params),
			});

			if (response.ok) {
				// Close dialog and refresh
				setTranscribeDDialogOpen(false);
				setSelectedJobId(null);
				refetch();
				if (onTranscribe) {
					onTranscribe(selectedJobId);
				}
			} else {
				alert("Failed to start transcription");
			}
		} catch {
			alert("Error starting transcription");
		} finally {
			setTranscriptionLoading(false);
		}
	}, [selectedJobId, refetch, onTranscribe, data, getAuthHeaders]);

	// Modified to verify file exists before opening dialog
	const handleDeleteClick = useCallback((file: AudioFile) => {
		setSelectedFile(file);
		setDeleteDialogOpen(true);
	}, []);

	const handleStopClick = useCallback((file: AudioFile) => {
		setSelectedFile(file);
		setStopDialogOpen(true);
	}, []);

	// Handle delete confirmation
	const handleConfirmDelete = useCallback(async () => {
		if (!selectedFile) return;

		try {
			const response = await fetch(`/api/v1/transcription/${selectedFile.id}`, {
				method: "DELETE",
				headers: {
					...getAuthHeaders(),
				},
			});

			if (response.ok) {
				refetch();
				setDeleteDialogOpen(false);
				setSelectedFile(null);
			} else {
				alert("Failed to delete audio file");
			}
		} catch {
			alert("Error deleting audio file");
		}
	}, [selectedFile, getAuthHeaders, refetch]);

	// Handle kill confirmation
	const handleConfirmStop = useCallback(async () => {
		if (!selectedFile) return;
		const jobId = selectedFile.id;

		try {
			setKillingJobs((prev) => new Set(prev).add(jobId));

			const response = await fetch(`/api/v1/transcription/${jobId}/kill`, {
				method: "POST",
				headers: {
					...getAuthHeaders(),
				},
			});

			if (response.ok) {
				refetch();
				setStopDialogOpen(false);
				setSelectedFile(null);
			} else {
				alert("Failed to kill transcription job");
			}
		} catch {
			alert("Error killing transcription job");
		} finally {
			setKillingJobs((prev) => {
				const newSet = new Set(prev);
				newSet.delete(jobId);
				return newSet;
			});
		}
	}, [selectedFile, getAuthHeaders, refetch]);

	// Bulk Actions Handlers
	const handleBulkTranscribe = useCallback(async (params: WhisperXParams) => {
		const selectedIds = Object.keys(rowSelection);
		if (selectedIds.length === 0) return;

		setBulkActionLoading(true);
		try {
			// Process sequentially to avoid overwhelming the server
			for (const id of selectedIds) {
				const job = data.find(j => j.id === id);
				if (!job) continue;

				// Skip if multi-track mismatch
				if (job.is_multi_track && !params.is_multi_track_enabled) continue;
				if (!job.is_multi_track && params.is_multi_track_enabled) continue;

				await fetch(`/api/v1/transcription/${id}/start`, {
					method: "POST",
					headers: {
						...getAuthHeaders(),
						"Content-Type": "application/json",
					},
					body: JSON.stringify(params),
				});
			}

			// Clear selection and refresh
			setRowSelection({});
			setConfigDialogOpen(false);
			setTranscribeDDialogOpen(false);
			setTranscribeDDialogOpen(false);
			refetch();
		} catch (error) {
			console.error("Bulk transcribe error:", error);
			alert("Error processing bulk transcription");
		} finally {
			setBulkActionLoading(false);
		}
	}, [rowSelection, data, getAuthHeaders, refetch]);

	const handleBulkDelete = useCallback(async () => {
		const selectedIds = Object.keys(rowSelection);
		if (selectedIds.length === 0) return;

		setBulkActionLoading(true);
		try {
			// Process sequentially
			for (const id of selectedIds) {
				await fetch(`/api/v1/transcription/${id}`, {
					method: "DELETE",
					headers: {
						...getAuthHeaders(),
					},
				});
			}

			// Clear selection and refresh
			setRowSelection({});
			setBulkDeleteDialogOpen(false);
			setBulkDeleteDialogOpen(false);
			refetch();
		} catch (error) {
			console.error("Bulk delete error:", error);
			alert("Error processing bulk delete");
		} finally {
			setBulkActionLoading(false);
		}
	}, [rowSelection, getAuthHeaders, refetch]);

	// Modified handlers to support bulk actions
	const onStartTranscribe = (params: WhisperXParams) => {
		if (Object.keys(rowSelection).length > 0) {
			handleBulkTranscribe(params);
		} else {
			handleStartTranscription(params);
		}
	};

	const onStartTranscribeWithProfile = (params: WhisperXParams) => {
		if (Object.keys(rowSelection).length > 0) {
			handleBulkTranscribe(params);
		} else {
			handleStartTranscriptionWithProfile(params);
		}
	};

	// Initial load handled by useQuery
	/* useEffect(() => {
		const isInitialLoad = data.length === 0;
		fetchAudioFiles(undefined, undefined, undefined, isInitialLoad);
	}, [refreshTrigger, fetchAudioFiles]); */

	// Data fetching handled by useQuery dependencies
	/* useEffect(() => {
		if (data.length > 0) { // Only fetch if not initial load
			fetchAudioFiles(
				pagination.pageIndex + 1,
				pagination.pageSize,
				globalFilter || undefined
			);
		}
	}, [pagination.pageIndex, pagination.pageSize, globalFilter, sorting, fetchAudioFiles]); */



	// Polling handled by useQuery refetchInterval
	/* const pollingIntervalRef = useRef<NodeJS.Timeout | null>(null);
	
	useEffect(() => {
		const activeJobs = data.filter(
			(job) => job.status === "pending" || job.status === "processing",
		);
	
		// Clear any existing polling interval
		if (pollingIntervalRef.current) {
			clearInterval(pollingIntervalRef.current);
			pollingIntervalRef.current = null;
		}
	
		// Only poll if there are active jobs
		if (activeJobs.length > 0) {
			// Use shorter interval for processing jobs, longer for pending jobs
			const hasProcessingJobs = activeJobs.some(job => job.status === "processing");
			const pollingInterval = hasProcessingJobs ? 2000 : 5000; // 2s for processing, 5s for pending
	
			pollingIntervalRef.current = setInterval(() => {
				// Keep current pagination when polling, but don't show loading indicators
				fetchAudioFiles(undefined, undefined, undefined, false, true);
			}, pollingInterval);
		}
	
		return () => {
			if (pollingIntervalRef.current) {
				clearInterval(pollingIntervalRef.current);
				pollingIntervalRef.current = null;
			}
		};
	}, [data, fetchAudioFiles]); */

	const getStatusIcon = useCallback((file: AudioFile) => {
		const status = file.status;
		const progress = trackProgress[file.id];

		// Multi-track processing
		if (file.is_multi_track && status === "processing" && progress) {
			const { progress: progressInfo } = progress;
			const percentage = Math.round(progressInfo.percentage || 0);
			return (
				<Tooltip>
					<TooltipTrigger asChild>
						<div className="flex items-center gap-1.5 cursor-help text-blue-600">
							<Loader2 className="h-4 w-4 animate-spin" />
							<span className="text-xs font-medium tabular-nums">{percentage}%</span>
						</div>
					</TooltipTrigger>
					<TooltipContent>Processing Multi-Track</TooltipContent>
				</Tooltip>
			);
		}

		switch (status) {
			case "completed":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help text-emerald-500">
								<Check className="h-5 w-5" strokeWidth={2.5} />
							</div>
						</TooltipTrigger>
						<TooltipContent>Completed</TooltipContent>
					</Tooltip>
				);
			case "processing":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help text-amber-500">
								<Loader2 className="h-4 w-4 animate-spin" strokeWidth={2.5} />
							</div>
						</TooltipTrigger>
						<TooltipContent>Processing</TooltipContent>
					</Tooltip>
				);
			case "failed":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help text-red-500">
								<AlertCircle className="h-5 w-5" strokeWidth={2.5} />
							</div>
						</TooltipTrigger>
						<TooltipContent>Failed</TooltipContent>
					</Tooltip>
				);
			case "pending": {
				const position = queuePositions[file.id];
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="flex items-center gap-1.5 cursor-help">
								<div className="flex items-center justify-center min-w-[20px] h-5 px-1.5 rounded-full bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 border border-gray-200 dark:border-gray-700 text-[10px] font-bold shadow-sm whitespace-nowrap">
									#{position || "-"}
								</div>
							</div>
						</TooltipTrigger>
						<TooltipContent>Queue Position: #{position}</TooltipContent>
					</Tooltip>
				);
			}
			case "uploaded":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help text-gray-300">
								<Clock className="h-4 w-4" />
							</div>
						</TooltipTrigger>
						<TooltipContent>Uploaded (Ready to Transcribe)</TooltipContent>
					</Tooltip>
				);
			default:
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help text-gray-300">
								<Clock className="h-4 w-4" />
							</div>
						</TooltipTrigger>
						<TooltipContent>Unknown Status</TooltipContent>
					</Tooltip>
				);
		}
	}, [trackProgress, queuePositions]);

	// formatDuration removed as requested

	const formatDate = useCallback((dateString: string) => {
		return new Date(dateString).toLocaleDateString("en-US", {
			year: "numeric",
			month: "short",
			day: "numeric",
		});
	}, []);




	const getFileName = useCallback((audioPath: string) => {
		const parts = audioPath.split("/");
		return parts[parts.length - 1];
	}, []);


	// Table logic removed (Floating Cards used instead)

	const selectedCount = Object.keys(rowSelection).length;

	// RENDER: Floating Row List (Premium UI)
	return (
		<div className="space-y-6">
			{/* Toolbar */}
			<div className="flex flex-col sm:flex-row gap-4 items-center justify-between">
				<DebouncedSearchInput
					placeholder="Search recordings..."
					value={globalFilter ?? ""}
					onChange={(value) => setGlobalFilter(String(value))}
					className="w-full sm:w-80 shadow-sm border-transparent focus:border-[var(--brand-solid)] bg-white dark:bg-zinc-900"
				/>
			</div>

			{/* List Container */}
			<div className="space-y-3 min-h-[300px] pb-24">
				{loading ? (
					// Skeleton Loaders
					Array.from({ length: 5 }).map((_, i) => (
						<div key={i} className="h-20 w-full bg-[var(--bg-card)] rounded-xl animate-pulse" />
					))
				) : data.length === 0 ? (
					<div className="flex flex-col items-center justify-center p-12 text-center border-2 border-dashed border-[var(--border-subtle)] rounded-xl bg-[var(--bg-card)]/50">
						<div className="p-4 bg-[var(--bg-main)] rounded-full mb-4">
							<Music className="h-8 w-8 text-[var(--text-tertiary)]" />
						</div>
						<h3 className="text-lg font-medium text-[var(--text-primary)]">No recordings found</h3>
						<p className="text-[var(--text-secondary)] max-w-sm mt-2">
							Upload an audio file or start a recording to get started.
						</p>
					</div>
				) : (
					<div className="space-y-3">
						{data.map((file, index) => (
							<SwipeableItem
								key={file.id}
								onTranscribe={() => handleTranscribeDClick(file.id)}
								onTranscribeAdvanced={() => handleTranscribeClick(file.id)}
								onDelete={() => handleDeleteClick(file)}
								onStop={() => handleStopClick(file)}
								isProcessing={file.status === "processing" || file.status === "pending"}
								isSelectionMode={Object.keys(rowSelection).length > 0}
								shouldShowHint={shouldShowHint && index === 0}
								onHintComplete={markHintShown}
								onSwipeStateChange={handleSwipeStateChange}
							>
								<div
									className={cn(
										"group relative flex justify-between items-center p-4",
										"bg-[var(--bg-card)] rounded-xl border border-[var(--border-subtle)]",
										"shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-all duration-200 cursor-pointer select-none",
										rowSelection[file.id as keyof typeof rowSelection] && "border-[var(--brand-solid)] ring-1 ring-[var(--brand-solid)]/10 bg-orange-50 dark:bg-orange-950"
									)}
									onClick={(e) => handleRowClick(file, e)}
									onMouseDown={(e) => startLongPress(file.id, e)}
									onMouseUp={clearLongPress}
									onMouseLeave={clearLongPress}
									onTouchStart={(e) => startLongPress(file.id, e)}
									onTouchMove={handleTouchMove}
									onTouchEnd={clearLongPress}
								>
									<div className="flex items-center gap-4 min-w-0 transition-[padding] duration-200">
										{/* Icon (Tinted Pastel Square) - Lighter Shade */}
										<div className="flex-shrink-0 w-12 h-12 flex items-center justify-center rounded-xl bg-[#FFFAF0] text-[#FF6D20] transition-opacity duration-200">
											<FileAudio className="h-6 w-6" strokeWidth={2} />
										</div>

										{/* Text */}
										<div className="min-w-0">
											<h4 className="font-normal text-gray-900 dark:text-gray-100 truncate text-lg leading-tight group-hover:text-[#FF6D20] transition-colors">
												{file.title || getFileName(file.audio_path)}
											</h4>
											<div className="flex items-center gap-1.5 mt-1 text-sm text-gray-500">
												{formatDate(file.created_at)}
											</div>
										</div>
									</div>

									{/* Right: Cluster (Actions • Status) */}
									<div className="flex items-center gap-6">
										{/* Desktop Actions (Hover) - Hidden on mobile */}
										<div
											className="hidden md:flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity duration-200"
											onClick={(e) => e.stopPropagation()}
										>
											{(file.status !== "processing" && file.status !== "pending") && (
												<>
													<Tooltip>
														<TooltipTrigger asChild>
															<Button
																variant="ghost"
																size="icon"
																onClick={() => handleTranscribeDClick(file.id)}
																className="h-9 w-9 rounded-lg text-gray-400 hover:text-[var(--brand-solid)] hover:bg-orange-50 cursor-pointer transition-colors"
															>
																<Wand2 className="h-5 w-5" strokeWidth={2} />
															</Button>
														</TooltipTrigger>
														<TooltipContent>Transcribe</TooltipContent>
													</Tooltip>

													<Tooltip>
														<TooltipTrigger asChild>
															<Button
																variant="ghost"
																size="icon"
																onClick={() => handleTranscribeClick(file.id)}
																className="h-9 w-9 rounded-lg text-gray-400 hover:text-[var(--brand-solid)] hover:bg-orange-50 cursor-pointer transition-colors"
															>
																<WandAdvancedIcon className="h-5 w-5" strokeWidth={2} />
															</Button>
														</TooltipTrigger>
														<TooltipContent>Transcribe (Advanced)</TooltipContent>
													</Tooltip>
												</>
											)}

											{(file.status === "processing" || file.status === "pending") ? (
												<Tooltip>
													<TooltipTrigger asChild>
														<Button
															variant="ghost"
															size="icon"
															onClick={() => handleStopClick(file)}
															className="h-9 w-9 rounded-lg text-gray-400 hover:text-red-600 hover:bg-red-50 cursor-pointer transition-colors"
														>
															<StopCircle className="h-5 w-5" strokeWidth={2} />
														</Button>
													</TooltipTrigger>
													<TooltipContent>Stop Transcription</TooltipContent>
												</Tooltip>
											) : (
												<Tooltip>
													<TooltipTrigger asChild>
														<Button
															variant="ghost"
															size="icon"
															onClick={() => handleDeleteClick(file)}
															className="h-9 w-9 rounded-lg text-gray-400 hover:text-red-600 hover:bg-red-50 cursor-pointer transition-colors"
														>
															<Trash2 className="h-5 w-5" strokeWidth={2} />
														</Button>
													</TooltipTrigger>
													<TooltipContent>Delete</TooltipContent>
												</Tooltip>
											)}
										</div>

										{/* Status Icon */}
										<div className="flex items-center justify-center w-6">
											{getStatusIcon(file)}
										</div>
									</div>
								</div>
							</SwipeableItem>
						))}
					</div>
				)}
			</div>

			{/* Floating Bulk Actions Pill */}
			{Object.keys(rowSelection).length > 0 && (
				<div className="fixed bottom-8 left-1/2 -translate-x-1/2 z-50 animate-in fade-in slide-in-from-bottom-4 duration-300">
					<div className="glass-card flex items-center gap-1 p-1.5 pl-4 pr-1.5 rounded-full shadow-[var(--shadow-float)] border border-[var(--border-subtle)] bg-[var(--bg-main)]/95 backdrop-blur-xl ring-1 ring-black/5">
						<div className="flex items-center gap-3 mr-2">
							<span className="flex h-5 w-5 items-center justify-center rounded-full bg-[var(--brand-solid)] text-[10px] font-bold text-white shadow-sm">
								{selectedCount}
							</span>
							<span className="text-sm font-medium text-[var(--text-primary)]">Selected</span>
						</div>

						<div className="h-4 w-px bg-[var(--border-subtle)] mx-1" />

						{/* Bulk Transcribe */}
						<Tooltip>
							<TooltipTrigger asChild>
								<Button
									variant="ghost"
									size="icon"
									onClick={() => setTranscribeDDialogOpen(true)}
									disabled={bulkActionLoading}
									className="h-9 w-9 rounded-full hover:bg-[var(--brand-light)] hover:text-[var(--brand-solid)] transition-colors"
								>
									<Wand2 className="h-4 w-4" />
								</Button>
							</TooltipTrigger>
							<TooltipContent>Transcribe Selected</TooltipContent>
						</Tooltip>

						{/* Bulk Advanced Transcribe */}
						<Tooltip>
							<TooltipTrigger asChild>
								<Button
									variant="ghost"
									size="icon"
									onClick={() => setConfigDialogOpen(true)}
									disabled={bulkActionLoading}
									className="h-9 w-9 rounded-full hover:bg-[var(--brand-light)] hover:text-[var(--brand-solid)] transition-colors"
								>
									<WandAdvancedIcon className="h-4 w-4" />
								</Button>
							</TooltipTrigger>
							<TooltipContent>Transcribe (Advanced)</TooltipContent>
						</Tooltip>

						<div className="h-4 w-px bg-[var(--border-subtle)] mx-1" />

						{/* Bulk Delete */}
						<Tooltip>
							<TooltipTrigger asChild>
								<Button
									variant="ghost"
									size="icon"
									onClick={() => setBulkDeleteDialogOpen(true)}
									disabled={bulkActionLoading}
									className="h-9 w-9 rounded-full hover:bg-red-50 hover:text-[var(--error)] transition-colors"
								>
									<Trash2 className="h-4 w-4" />
								</Button>
							</TooltipTrigger>
							<TooltipContent>Delete Selected</TooltipContent>
						</Tooltip>

						<div className="h-4 w-px bg-[var(--border-subtle)] mx-1" />

						{/* Clear Selection */}
						<Tooltip>
							<TooltipTrigger asChild>
								<Button
									variant="ghost"
									size="icon"
									onClick={() => setRowSelection({})}
									className="h-9 w-9 rounded-full hover:bg-[var(--bg-card)] hover:text-[var(--text-secondary)] transition-colors"
								>
									<X className="h-4 w-4" />
								</Button>
							</TooltipTrigger>
							<TooltipContent>Clear Selection</TooltipContent>
						</Tooltip>
					</div>
				</div>
			)}

			{/* Infinite Scroll & Loading State */}
			<div ref={scrollRef} className="pt-2 pb-8">
				{isFetchingNextPage && (
					<div className="space-y-4 animate-in fade-in slide-in-from-bottom-4 duration-500">
						{/* Premium Skeleton Loaders - Shimmering Bars */}
						{Array.from({ length: 3 }).map((_, i) => (
							<div
								key={i}
								className="h-20 w-full rounded-xl border border-[var(--border-subtle)] bg-[var(--bg-card)]/50 p-4 flex items-center gap-4"
							>
								{/* Icon Skeleton */}
								<div className="h-12 w-12 rounded-xl bg-gray-200 dark:bg-zinc-800 animate-pulse" />

								{/* Text Skeleton */}
								<div className="space-y-2 flex-1">
									<div className="h-4 w-1/3 bg-gray-200 dark:bg-zinc-800 rounded animate-pulse" />
									<div className="h-3 w-1/4 bg-gray-100 dark:bg-zinc-900 rounded animate-pulse" />
								</div>

								{/* Action Skeleton */}
								<div className="h-8 w-24 bg-gray-100 dark:bg-zinc-900 rounded-lg animate-pulse opacity-50" />
							</div>
						))}
					</div>
				)}
			</div>
			<AlertDialog open={bulkDeleteDialogOpen} onOpenChange={setBulkDeleteDialogOpen}>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Are you sure?</AlertDialogTitle>
						<AlertDialogDescription>
							This will permanently delete {Object.keys(rowSelection).length} selected recordings.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction onClick={handleBulkDelete} className="bg-red-600 hover:bg-red-700">Delete</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
			{/* Keeping existing dialogs */}
			<TranscriptionConfigDialog
				open={configDialogOpen}
				onOpenChange={setConfigDialogOpen}
				onStartTranscription={onStartTranscribe}
				loading={transcriptionLoading}
			/>
			<TranscribeDDialog
				open={transcribeDDialogOpen}
				onOpenChange={setTranscribeDDialogOpen}
				onStartTranscription={onStartTranscribeWithProfile}
				loading={transcriptionLoading}
			/>

			{/* Stop Transcription Dialog */}
			<AlertDialog open={stopDialogOpen} onOpenChange={setStopDialogOpen}>
				<AlertDialogContent className="glass-card bg-[var(--bg-main)]/90 border-[var(--border-subtle)]">
					<AlertDialogHeader>
						<AlertDialogTitle className="text-[var(--text-primary)]">
							Stop Transcription?
						</AlertDialogTitle>
						<AlertDialogDescription className="text-[var(--text-secondary)]">
							Are you sure you want to stop the transcription process
							for "{selectedFile?.title || (selectedFile ? getFileName(selectedFile.audio_path) : "")}"?
							Partially transcribed data may be saved.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel className="bg-[var(--secondary)] border-[var(--border-subtle)] text-[var(--text-secondary)] hover:bg-[var(--bg-card)]">
							Cancel
						</AlertDialogCancel>
						<AlertDialogAction
							className="bg-[var(--warning)] text-white hover:opacity-90"
							onClick={handleConfirmStop}
						>
							{killingJobs.has(selectedFile?.id || "") ? (
								<>
									<Loader2 className="mr-2 h-4 w-4 animate-spin" />
									Stopping...
								</>
							) : (
								"Stop Transcription"
							)}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>

			{/* Delete Audio File Dialog */}
			<AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
				<AlertDialogContent className="glass-card bg-[var(--bg-main)]/90 border-[var(--border-subtle)]">
					<AlertDialogHeader>
						<AlertDialogTitle className="text-[var(--text-primary)]">
							Delete Audio File
						</AlertDialogTitle>
						<AlertDialogDescription className="text-[var(--text-secondary)]">
							Are you sure you want to delete "
							{selectedFile?.title || (selectedFile ? getFileName(selectedFile.audio_path) : "")}
							"? This action cannot be undone and will
							permanently remove the audio file and any
							transcription data.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel className="bg-[var(--secondary)] border-[var(--border-subtle)] text-[var(--text-secondary)] hover:bg-[var(--bg-card)]">
							Cancel
						</AlertDialogCancel>
						<AlertDialogAction
							className="bg-[var(--error)] text-white hover:opacity-90"
							onClick={handleConfirmDelete}
						>
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>

			{/* Active Job Monitors */}
			{activeJobs.map(job => (
				<JobStatusMonitor key={job.id} jobId={job.id} />
			))}
		</div >
	);
});

AudioFilesTable.displayName = "AudioFilesTable";
