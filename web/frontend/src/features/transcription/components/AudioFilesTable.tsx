import { useState, useEffect, useMemo, useCallback, memo } from "react";
import {
	CheckCircle,
	Clock,
	XCircle,
	Loader2,
	MoreVertical,
	Hash,
	Trash2,
	StopCircle,
	ChevronUp,
	ChevronDown,
	ChevronsUpDown,
	Search,
	ChevronLeft,
	ChevronRight,
	ChevronsLeft,
	ChevronsRight,
	MessageCircle,
} from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";

// Custom SVG icons for transcription actions
const QuickTranscribeIcon = ({ className }: { className?: string }) => (
	<svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
		<path d="M12 2L15.09 8.26L22 9L17 14L18.18 21L12 17.77L5.82 21L7 14L2 9L8.91 8.26L12 2Z" />
		<path d="M8 12h8" strokeWidth="1.5" />
		<path d="M8 16h6" strokeWidth="1.5" />
	</svg>
);

const AdvancedTranscribeIcon = ({ className }: { className?: string }) => (
	<svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
		<circle cx="12" cy="12" r="3" />
		<path d="M12 1v6m0 6v6" />
		<path d="m21 12-6 0m-6 0-6 0" />
		<path d="m16.24 7.76-4.24 4.24m-4.24 4.24-1.41-1.41" />
		<path d="M16.24 16.24 12 12m-4.24-4.24L6.34 6.34" />
	</svg>
);
import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover";
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
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { TranscriptionConfigDialog, type WhisperXParams } from "@/components/TranscriptionConfigDialog";
import { TranscribeDDialog } from "@/components/TranscribeDDialog";
import { useNavigate } from "react-router-dom";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useAudioList, type AudioFile } from "@/features/transcription/hooks/useAudioFiles";
import {
	useReactTable,
	getCoreRowModel,
	getSortedRowModel,
	flexRender,
	type ColumnDef,
	type SortingState,
	type ColumnFiltersState,
} from "@tanstack/react-table";


// Debounced search input component to prevent focus loss
function DebouncedSearchInput({
	value,
	onChange,
	placeholder,
	className
}: {
	value: string;
	onChange: (value: string) => void;
	placeholder: string;
	className: string;
}) {
	const [searchValue, setSearchValue] = useState(value);

	// Update internal state when external value changes
	useEffect(() => {
		setSearchValue(value);
	}, [value]);

	// Debounce the onChange callback
	useEffect(() => {
		const timeoutId = setTimeout(() => {
			onChange(searchValue);
		}, 300);

		return () => clearTimeout(timeoutId);
	}, [searchValue, onChange]);

	return (
		<Input
			placeholder={placeholder}
			value={searchValue}
			onChange={(e) => setSearchValue(e.target.value)}
			className={`h-10 rounded-[var(--radius-btn)] border-[var(--border-subtle)] bg-[var(--bg-main)] focus:ring-[var(--brand-light)] focus:border-[var(--brand-solid)] transition-all duration-200 ${className}`}
		/>
	);
}



interface AudioFilesTableProps {
	refreshTrigger?: number; // Optional now, kept for compatibility during refactor
	onTranscribe?: (jobId: string) => void;
}

export const AudioFilesTable = memo(function AudioFilesTable({
	onTranscribe,
}: AudioFilesTableProps) {
	const navigate = useNavigate();
	const { getAuthHeaders } = useAuth();

	// Table State
	const [pagination, setPagination] = useState({
		pageIndex: 0,
		pageSize: 10,
	});
	const [sorting, setSorting] = useState<SortingState>([
		{ id: "created_at", desc: true }
	]);
	const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
	const [globalFilter, setGlobalFilter] = useState("");

	// Query
	const {
		data: queryData,
		isLoading: queryLoading,
		refetch,
		isPlaceholderData
	} = useAudioList({
		page: pagination.pageIndex + 1,
		limit: pagination.pageSize,
		search: globalFilter,
		sortBy: sorting[0]?.id,
		sortOrder: sorting[0]?.desc ? 'desc' : 'asc'
	});

	const data = queryData?.jobs || [];
	const totalItems = queryData?.pagination.total || 0;
	const pageCount = queryData?.pagination.pages || 0;
	const loading = queryLoading;
	const isPageChanging = isPlaceholderData;

	// Local state for UI
	const [queuePositions, setQueuePositions] = useState<Record<string, number>>({});

	const [openPopovers, setOpenPopovers] = useState<Record<string, boolean>>({});
	const [rowSelection, setRowSelection] = useState({});
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
	const [trackProgress, setTrackProgress] = useState<Record<string, any>>({});

	// Dialog state management (moved outside table to prevent re-renders)
	const [stopDialogOpen, setStopDialogOpen] = useState(false);
	const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
	const [selectedFile, setSelectedFile] = useState<AudioFile | null>(null);

	// Side effects for queue and progress
	useEffect(() => {
		if (data.length > 0) {
			fetchQueuePositions(data);
			// Fetch track progress for processing multi-track jobs
			const processingMultiTrackJobs = data.filter(job =>
				job.is_multi_track && (job.status === "processing" || job.status === "pending")
			);

			if (processingMultiTrackJobs.length > 0) {
				fetchTrackProgress(processingMultiTrackJobs);
			}
		}
	}, [data]);

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
			const progressData: Record<string, any> = {};
			results.forEach(result => {
				if (result) progressData[result.jobId] = result.progress;
			});
			setTrackProgress(prev => ({ ...prev, ...progressData }));
		} catch (error) {
			console.error("Failed to fetch track progress:", error);
		}
	};

	// Fetch queue positions for pending jobs
	const fetchQueuePositions = async (jobs: AudioFile[]) => {
		const pendingJobs = jobs.filter((job) => job.status === "pending");
		if (pendingJobs.length === 0) return;

		try {
			const response = await fetch("/api/v1/admin/queue/stats", {
				headers: {
					...getAuthHeaders(),
				},
			});

			if (response.ok) {
				// For now, use simple position calculation
				const positions: Record<string, number> = {};
				pendingJobs.forEach((job, index) => {
					positions[job.id] = index + 1; // Simple position calculation
				});
				setQueuePositions(positions);
			}
		} catch (error) {
			console.error("Failed to fetch queue positions:", error);
		}
	};

	// Handle transcribe action - opens configuration dialog
	const handleTranscribe = useCallback((jobId: string) => {
		const job = data.find((f) => f.id === jobId);
		if (!job) return;

		// Close the popover
		setOpenPopovers((prev) => ({ ...prev, [jobId]: false }));

		// Open configuration dialog
		setSelectedJobId(jobId);
		setConfigDialogOpen(true);
	}, [data]);

	// Handle transcribe-D action - opens profile selection dialog
	const handleTranscribeD = useCallback((jobId: string) => {
		const job = data.find((f) => f.id === jobId);
		if (!job) return;

		// Close the popover
		setOpenPopovers((prev) => ({ ...prev, [jobId]: false }));

		// Open Transcribe-D dialog
		setSelectedJobId(jobId);
		setTranscribeDDialogOpen(true);
	}, [data]);

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
	}, [selectedJobId, refetch, onTranscribe]);

	// Handle actual transcription start with profile parameters
	const handleStartTranscriptionWithProfile = useCallback(async (params: WhisperXParams, _profileId?: string) => {
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
	}, [selectedJobId, refetch, onTranscribe]);

	// Check if job can be transcribed (not currently processing or pending)
	const canTranscribe = useCallback((file: AudioFile) => {
		return file.status !== "processing" && file.status !== "pending";
	}, []);

	// Handle delete action
	const handleDelete = useCallback(async (jobId: string) => {
		// Close the popover
		setOpenPopovers((prev) => ({ ...prev, [jobId]: false }));

		try {
			const response = await fetch(`/api/v1/transcription/${jobId}`, {
				method: "DELETE",
				headers: {
					...getAuthHeaders(),
				},
			});

			if (response.ok) {
				// Refresh to show updated list
				// Refresh to show updated list
				refetch();
			} else {
				alert("Failed to delete audio file");
			}
		} catch {
			alert("Error deleting audio file");
		}
	}, [refetch]);

	// Handle kill action
	const handleKillJob = useCallback(async (jobId: string) => {
		// Close the popover
		setOpenPopovers((prev) => ({ ...prev, [jobId]: false }));

		try {
			setKillingJobs((prev) => new Set(prev).add(jobId));

			const response = await fetch(`/api/v1/transcription/${jobId}/kill`, {
				method: "POST",
				headers: {
					...getAuthHeaders(),
				},
			});

			if (response.ok) {
				// Refresh to show updated status
				// Refresh to show updated status
				refetch();
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
	}, [refetch]);

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

	// Reset to first page when search changes
	useEffect(() => {
		if (globalFilter !== undefined) {
			setPagination(prev => ({ ...prev, pageIndex: 0 }));
		}
	}, [globalFilter]);

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
		const iconSize = 16;
		const status = file.status;
		const queuePosition = queuePositions[file.id];
		const progress = trackProgress[file.id];

		// Special handling for multi-track jobs that are processing
		if (file.is_multi_track && status === "processing" && progress) {
			const { progress: progressInfo, tracks } = progress;
			const percentage = Math.round(progressInfo.percentage || 0);
			const completedTracks = progressInfo.completed_tracks || 0;
			const totalTracks = progressInfo.total_tracks || 0;

			return (
				<Tooltip>
					<TooltipTrigger asChild>
						<div className="cursor-help inline-flex items-center gap-1">
							<div className="relative">
								<Loader2 size={iconSize} className="text-[var(--brand-solid)] animate-spin" />
							</div>
							<span className="text-sm text-[var(--brand-solid)] font-medium">
								{completedTracks}/{totalTracks}
							</span>
						</div>
					</TooltipTrigger>
					<TooltipContent className="bg-[var(--bg-card)] border-[var(--border-subtle)] text-[var(--text-primary)]">
						<div className="space-y-1">
							<p>Multi-Track Processing ({percentage}%)</p>
							<div className="space-y-1">
								{tracks && tracks.slice(0, 5).map((track: any, index: number) => (
									<div key={index} className="flex items-center gap-2 text-sm">
										<span className={`w-2 h-2 rounded-full ${track.status === 'completed' ? 'bg-[var(--success)]' :
											track.status === 'processing' ? 'bg-[var(--warning)]' :
												'bg-[var(--text-disabled)]'
											}`}></span>
										<span>{track.track_name}</span>
									</div>
								))}
								{tracks && tracks.length > 5 && (
									<p className="text-sm text-[var(--text-tertiary)]">...and {tracks.length - 5} more</p>
								)}
							</div>
						</div>
					</TooltipContent>
				</Tooltip>
			);
		}

		switch (status) {
			case "completed":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help inline-block">
								<CheckCircle size={iconSize} className="text-[var(--success)]" />
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-[var(--bg-card)] border-[var(--border-subtle)] text-[var(--text-primary)]">
							<p>Completed</p>
						</TooltipContent>
					</Tooltip>
				);
			case "processing":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help inline-block">
								<Loader2
									size={iconSize}
									className="text-[var(--brand-solid)] animate-spin"
								/>
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-[var(--bg-card)] border-[var(--border-subtle)] text-[var(--text-primary)]">
							<p>Processing</p>
						</TooltipContent>
					</Tooltip>
				);
			case "failed":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help inline-block">
								<XCircle size={iconSize} className="text-[var(--error)]" />
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-[var(--bg-card)] border-[var(--border-subtle)] text-[var(--text-primary)]">
							<p>Failed</p>
						</TooltipContent>
					</Tooltip>
				);
			case "pending":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="flex items-center gap-1 cursor-help inline-flex">
								<Hash size={12} className="text-[var(--text-tertiary)]" />
								<span className="text-sm text-[var(--text-tertiary)] font-medium">
									{queuePosition || "?"}
								</span>
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-[var(--bg-card)] border-[var(--border-subtle)] text-[var(--text-primary)]">
							<p>Queued (Position {queuePosition || "?"})</p>
						</TooltipContent>
					</Tooltip>
				);
			case "uploaded":
			default:
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help inline-block">
								<Clock size={iconSize} className="text-[var(--text-tertiary)]" />
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-[var(--bg-card)] border-[var(--border-subtle)] text-[var(--text-primary)]">
							<p>Uploaded</p>
						</TooltipContent>
					</Tooltip>
				);
		}
	}, [queuePositions, trackProgress]);

	const formatDate = useCallback((dateString: string) => {
		return new Date(dateString).toLocaleDateString("en-US", {
			year: "numeric",
			month: "short",
			day: "numeric",
			hour: "2-digit",
			minute: "2-digit",
		});
	}, []);

	const getFileName = useCallback((audioPath: string) => {
		const parts = audioPath.split("/");
		return parts[parts.length - 1];
	}, []);

	const handleAudioClick = useCallback((audioId: string) => {
		navigate(`/audio/${audioId}`);
	}, [navigate]);

	// Memoize column definitions to prevent recreation on every render
	const columns = useMemo<ColumnDef<AudioFile>[]>(
		() => [
			{
				id: "select",
				header: ({ table }) => (
					<div className={`pr-4 transition-opacity duration-200 ${table.getIsSomeRowsSelected() || table.getIsAllRowsSelected() ? 'opacity-100' : 'opacity-0 group-hover/header:opacity-100'}`}>
						<Checkbox
							checked={table.getIsAllPageRowsSelected() || (table.getIsSomePageRowsSelected() && "indeterminate")}
							onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
							aria-label="Select all"
							className="translate-y-[2px] border-[var(--border-focus)] data-[state=checked]:bg-[var(--brand-solid)] data-[state=checked]:border-[var(--brand-solid)]"
						/>
					</div>
				),
				cell: ({ row }) => (
					<div className={`pr-4 transition-opacity duration-200 ${row.getIsSelected() ? 'opacity-100' : 'opacity-0 group-hover/row:opacity-100'}`}>
						<Checkbox
							checked={row.getIsSelected()}
							onCheckedChange={(value) => row.toggleSelected(!!value)}
							aria-label="Select row"
							className="translate-y-[2px] border-[var(--border-focus)] data-[state=checked]:bg-[var(--brand-solid)] data-[state=checked]:border-[var(--brand-solid)]"
						/>
					</div>
				),
				enableSorting: false,
				enableHiding: false,
			},
			{
				accessorFn: (row) => row.title || getFileName(row.audio_path),
				id: "title",
				header: ({ column }) => {
					return (
						<Button
							variant="ghost"
							onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
							className="h-auto p-0 font-medium"
						>
							Title
							{column.getIsSorted() === "asc" ? (
								<ChevronUp className="ml-2 h-4 w-4" />
							) : column.getIsSorted() === "desc" ? (
								<ChevronDown className="ml-2 h-4 w-4" />
							) : (
								<ChevronsUpDown className="ml-2 h-4 w-4" />
							)}
						</Button>
					);
				},
				cell: ({ row }) => {
					const file = row.original;
					return (
						<div className="relative flex items-center justify-between w-full group/title py-2">
							<button
								onClick={() => handleAudioClick(file.id)}
								className="text-[var(--text-primary)] font-medium hover:text-[var(--brand-solid)] transition-colors cursor-pointer text-left truncate pr-8 text-base tracking-tight"
							>
								{file.title || getFileName(file.audio_path)}
							</button>

							{/* Desktop Hover Actions Toolbar */}
							<div className="hidden lg:flex absolute right-0 top-1/2 -translate-y-1/2 opacity-0 group-hover/row:opacity-100 transition-opacity duration-200 items-center gap-1 bg-[var(--bg-main)]/90 backdrop-blur-sm shadow-[var(--shadow-card)] border border-[var(--border-subtle)] rounded-[var(--radius-btn)] px-1 py-0.5 z-10">
								{file.status === "completed" && (
									<Tooltip>
										<TooltipTrigger asChild>
											<Button
												variant="ghost"
												size="icon"
												className="h-7 w-7 text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-card)]"
												onClick={(e) => {
													e.stopPropagation();
													navigate(`/audio/${file.id}/chat`);
												}}
											>
												<MessageCircle className="h-4 w-4" />
											</Button>
										</TooltipTrigger>
										<TooltipContent>Open Chat</TooltipContent>
									</Tooltip>
								)}

								<Tooltip>
									<TooltipTrigger asChild>
										<Button
											variant="ghost"
											size="icon"
											className="h-7 w-7 text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-card)] disabled:opacity-50"
											disabled={!canTranscribe(file)}
											onClick={(e) => {
												e.stopPropagation();
												handleTranscribeD(file.id);
											}}
										>
											<QuickTranscribeIcon className="h-4 w-4" />
										</Button>
									</TooltipTrigger>
									<TooltipContent>Transcribe</TooltipContent>
								</Tooltip>

								<Tooltip>
									<TooltipTrigger asChild>
										<Button
											variant="ghost"
											size="icon"
											className="h-7 w-7 text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-card)] disabled:opacity-50"
											disabled={!canTranscribe(file)}
											onClick={(e) => {
												e.stopPropagation();
												handleTranscribe(file.id);
											}}
										>
											<AdvancedTranscribeIcon className="h-4 w-4" />
										</Button>
									</TooltipTrigger>
									<TooltipContent>Transcribe+</TooltipContent>
								</Tooltip>

								{file.status === "processing" && (
									<Tooltip>
										<TooltipTrigger asChild>
											<Button
												variant="ghost"
												size="icon"
												className="h-7 w-7 text-orange-500 dark:text-orange-400 hover:text-orange-600 dark:hover:text-orange-300 hover:bg-orange-50 dark:hover:bg-orange-900/20"
												disabled={killingJobs.has(file.id)}
												onClick={(e) => {
													e.stopPropagation();
													setSelectedFile(file);
													setStopDialogOpen(true);
												}}
											>
												{killingJobs.has(file.id) ? (
													<Loader2 className="h-4 w-4 animate-spin" />
												) : (
													<StopCircle className="h-4 w-4" />
												)}
											</Button>
										</TooltipTrigger>
										<TooltipContent>Stop Transcription</TooltipContent>
									</Tooltip>
								)}

								<Tooltip>
									<TooltipTrigger asChild>
										<Button
											variant="ghost"
											size="icon"
											className="h-7 w-7 text-red-500 dark:text-red-400 hover:text-red-600 dark:hover:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/20"
											onClick={(e) => {
												e.stopPropagation();
												setSelectedFile(file);
												setDeleteDialogOpen(true);
											}}
										>
											<Trash2 className="h-4 w-4" />
										</Button>
									</TooltipTrigger>
									<TooltipContent>Delete</TooltipContent>
								</Tooltip>
							</div>
						</div>
					);
				},
				enableGlobalFilter: false,
			},
			{
				accessorKey: "created_at",
				header: ({ column }) => {
					return (
						<Button
							variant="ghost"
							onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
							className="h-auto p-0 font-medium"
						>
							Date Added
							{column.getIsSorted() === "asc" ? (
								<ChevronUp className="ml-2 h-4 w-4" />
							) : column.getIsSorted() === "desc" ? (
								<ChevronDown className="ml-2 h-4 w-4" />
							) : (
								<ChevronsUpDown className="ml-2 h-4 w-4" />
							)}
						</Button>
					);
				},
				cell: ({ getValue }) => (
					<span className="text-[var(--text-secondary)] text-sm sm:text-base">
						{formatDate(getValue() as string)}
					</span>
				),
				enableGlobalFilter: false,
			},
			{
				accessorKey: "status",
				header: ({ column }) => {
					return (
						<div className="text-center w-full">
							<Button
								variant="ghost"
								onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
								className="h-auto p-0 font-medium"
							>
								Status
								{column.getIsSorted() === "asc" ? (
									<ChevronUp className="ml-2 h-4 w-4" />
								) : column.getIsSorted() === "desc" ? (
									<ChevronDown className="ml-2 h-4 w-4" />
								) : (
									<ChevronsUpDown className="ml-2 h-4 w-4" />
								)}
							</Button>
						</div>
					);
				},
				cell: ({ row }) => (
					<div className="text-center">
						{getStatusIcon(row.original)}
					</div>
				),
				enableGlobalFilter: false,
			},
			{
				id: "actions",
				header: () => (
					<div className="text-center w-full">
						Actions
					</div>
				),
				cell: ({ row }) => {
					const file = row.original;
					return (
						<div className="text-center">
							<Popover
								open={openPopovers[file.id] || false}
								onOpenChange={(open) =>
									setOpenPopovers((prev) => ({
										...prev,
										[file.id]: open,
									}))
								}
							>
								<PopoverTrigger asChild>
									<Button
										variant="ghost"
										size="sm"
										className="h-8 w-8 sm:h-9 sm:w-9 p-0 cursor-pointer"
									>
										<MoreVertical className="h-5 w-5" />
									</Button>
								</PopoverTrigger>
								<PopoverContent className="w-40 bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700 p-1">
									<div className="space-y-1">

										<Button
											variant="ghost"
											size="sm"
											className="w-full justify-start h-8 text-sm hover:bg-carbon-100 dark:hover:bg-carbon-700 cursor-pointer disabled:cursor-not-allowed"
											disabled={!canTranscribe(file)}
											onClick={() => handleTranscribeD(file.id)}
										>
											<QuickTranscribeIcon className="mr-2 h-4 w-4" />
											Transcribe
										</Button>

										<Button
											variant="ghost"
											size="sm"
											className="w-full justify-start h-8 text-sm hover:bg-carbon-100 dark:hover:bg-carbon-700 cursor-pointer disabled:cursor-not-allowed"
											disabled={!canTranscribe(file)}
											onClick={() => handleTranscribe(file.id)}
										>
											<AdvancedTranscribeIcon className="mr-2 h-4 w-4" />
											Transcribe+
										</Button>

										{file.status === "processing" && (
											<Button
												variant="ghost"
												size="sm"
												className="w-full justify-start h-8 text-sm hover:bg-carbon-100 dark:hover:bg-carbon-700 text-orange-500 dark:text-orange-400 hover:text-orange-600 dark:hover:text-orange-300 cursor-pointer"
												disabled={killingJobs.has(file.id)}
												onClick={() => {
													setSelectedFile(file);
													setStopDialogOpen(true);
												}}
											>
												{killingJobs.has(file.id) ? (
													<>
														<Loader2 className="mr-2 h-4 w-4 animate-spin" />
														Stopping...
													</>
												) : (
													<>
														<StopCircle className="mr-2 h-4 w-4" />
														Stop
													</>
												)}
											</Button>
										)}

										<Button
											variant="ghost"
											size="sm"
											className="w-full justify-start h-8 text-sm hover:bg-carbon-100 dark:hover:bg-carbon-700 text-red-500 dark:text-red-400 hover:text-red-600 dark:hover:text-red-300 cursor-pointer"
											onClick={() => {
												setSelectedFile(file);
												setDeleteDialogOpen(true);
											}}
										>
											<Trash2 className="mr-2 h-4 w-4" />
											Delete
										</Button>
									</div>
								</PopoverContent>
							</Popover>
						</div>
					);
				},
				enableSorting: false,
				enableGlobalFilter: false,
			},
		],
		[openPopovers, queuePositions, trackProgress, getStatusIcon, handleAudioClick, handleTranscribe, handleTranscribeD, canTranscribe, getFileName, killingJobs, setSelectedFile, setStopDialogOpen, setDeleteDialogOpen]
	);

	// Create the table instance with server-side pagination and search
	const table = useReactTable({
		data,
		columns,
		getCoreRowModel: getCoreRowModel(),
		getSortedRowModel: getSortedRowModel(),
		onSortingChange: setSorting,
		onColumnFiltersChange: setColumnFilters,
		onGlobalFilterChange: setGlobalFilter,
		onRowSelectionChange: setRowSelection,
		onPaginationChange: setPagination,
		manualPagination: true,
		manualSorting: true,
		manualFiltering: true,
		pageCount: pageCount,
		state: {
			sorting,
			columnFilters,
			globalFilter,
			rowSelection,
			pagination,
		},
		getRowId: row => row.id, // Use ID for selection
	});

	const selectedCount = Object.keys(rowSelection).length;

	// Create the table instance with server-side pagination and search
	if (loading) {
		return (
			<div className="space-y-4">
				{/* Bulk Actions Toolbar */}
				{selectedCount > 0 && (
					<div className="fixed bottom-6 left-1/2 transform -translate-x-1/2 z-50 bg-[var(--bg-main)] border border-[var(--border-subtle)] shadow-[var(--shadow-float)] rounded-full px-6 py-3 flex items-center gap-4 animate-in slide-in-from-bottom-5 fade-in duration-300">
						<span className="text-sm font-medium text-[var(--text-secondary)] border-r border-[var(--border-subtle)] pr-4">
							{selectedCount} selected
						</span>

						<div className="flex items-center gap-2">
							<Button
								variant="ghost"
								size="sm"
								onClick={() => setConfigDialogOpen(true)}
								disabled={bulkActionLoading}
								className="h-8 rounded-full hover:bg-[var(--secondary)]"
							>
								<QuickTranscribeIcon className="mr-2 h-4 w-4" />
								Transcribe
							</Button>

							<Button
								variant="ghost"
								size="sm"
								onClick={() => setTranscribeDDialogOpen(true)}
								disabled={bulkActionLoading}
								className="h-8 rounded-full hover:bg-[var(--secondary)]"
							>
								<AdvancedTranscribeIcon className="mr-2 h-4 w-4" />
								Transcribe+
							</Button>

							<div className="w-px h-4 bg-[var(--border-subtle)] mx-1" />

							<Button
								variant="ghost"
								size="sm"
								onClick={() => setBulkDeleteDialogOpen(true)}
								disabled={bulkActionLoading}
								className="h-8 rounded-full text-[var(--error)] hover:text-[var(--error)] hover:bg-[var(--error)]/10"
							>
								<Trash2 className="mr-2 h-4 w-4" />
								Delete
							</Button>
						</div>

						<Button
							variant="ghost"
							size="icon"
							onClick={() => setRowSelection({})}
							className="h-6 w-6 rounded-full ml-2 hover:bg-[var(--secondary)]"
						>
							<XCircle className="h-4 w-4 text-[var(--text-tertiary)]" />
						</Button>
					</div>
				)}
				<div className="glass-card rounded-xl p-6">
					<div className="animate-pulse">
						<div className="h-4 bg-muted rounded w-1/4 mb-6"></div>
						<div className="space-y-3">
							{[...Array(5)].map((_, i) => (
								<div
									key={i}
									className="h-12 bg-muted/50 rounded-lg"
								></div>
							))}
						</div>
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="space-y-4">
			{/* Bulk Actions Toolbar */}
			{selectedCount > 0 && (
				<div className="fixed bottom-8 left-1/2 transform -translate-x-1/2 z-50 glass rounded-full px-4 sm:px-6 py-2 sm:py-3 flex items-center gap-2 sm:gap-4 animate-in slide-in-from-bottom-5 fade-in duration-300 shadow-[var(--shadow-float)] border border-[var(--border-subtle)] w-[90%] sm:w-auto justify-between sm:justify-center">
					<span className="text-xs sm:text-sm font-medium text-[var(--text-secondary)] border-r border-[var(--border-subtle)] pr-2 sm:pr-4 whitespace-nowrap">
						{selectedCount} selected
					</span>

					<div className="flex items-center gap-1 sm:gap-2">
						<Button
							variant="ghost"
							size="sm"
							onClick={() => setTranscribeDDialogOpen(true)}
							disabled={bulkActionLoading}
							className="h-8 sm:h-9 rounded-full hover:bg-[var(--brand-light)] text-[var(--text-primary)] hover:text-[var(--brand-solid)] transition-colors px-2 sm:px-4"
						>
							<QuickTranscribeIcon className="sm:mr-2 h-4 w-4" />
							<span className="hidden sm:inline">Transcribe</span>
						</Button>

						<Button
							variant="ghost"
							size="sm"
							onClick={() => setConfigDialogOpen(true)}
							disabled={bulkActionLoading}
							className="h-8 sm:h-9 rounded-full hover:bg-[var(--brand-light)] text-[var(--text-primary)] hover:text-[var(--brand-solid)] transition-colors px-2 sm:px-4"
						>
							<AdvancedTranscribeIcon className="sm:mr-2 h-4 w-4" />
							<span className="hidden sm:inline">Transcribe+</span>
						</Button>

						<div className="w-px h-4 bg-[var(--border-subtle)] mx-1" />

						<Button
							variant="ghost"
							size="sm"
							onClick={() => setBulkDeleteDialogOpen(true)}
							disabled={bulkActionLoading}
							className="h-8 sm:h-9 rounded-full text-[var(--error)] hover:bg-[var(--error)]/10 transition-colors px-2 sm:px-4"
						>
							<Trash2 className="sm:mr-2 h-4 w-4" />
							<span className="hidden sm:inline">Delete</span>
						</Button>
					</div>

					<Button
						variant="ghost"
						size="icon"
						onClick={() => setRowSelection({})}
						className="h-6 w-6 rounded-full ml-1 sm:ml-2 hover:bg-[var(--secondary)] text-[var(--text-tertiary)]"
					>
						<XCircle className="h-4 w-4" />
					</Button>
				</div>
			)}
			<div className="glass-card rounded-[var(--radius-card)] overflow-hidden shadow-[var(--shadow-card)] border border-[var(--border-subtle)]">
				<div className="p-4 sm:p-8">
					<div className="flex flex-col sm:flex-row items-start sm:items-end justify-between gap-4 sm:gap-0 mb-8">
						<div>
							<h2 className="text-2xl font-bold tracking-tight text-[var(--text-primary)] mb-1">
								Audio Files
							</h2>
							<p className="text-[var(--text-secondary)] text-base">
								{globalFilter
									? `${totalItems} file${totalItems !== 1 ? "s" : ""} found`
									: `${totalItems} file${totalItems !== 1 ? "s" : ""}`
								}
							</p>
						</div>

						{/* Global Search */}
						<div className="relative w-full sm:w-80">
							<Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-[var(--text-tertiary)] h-4 w-4 z-10" />
							<DebouncedSearchInput
								placeholder="Search..."
								value={globalFilter ?? ""}
								onChange={setGlobalFilter}
								className="pl-10"
							/>
						</div>
					</div>

					{data.length === 0 && !loading ? (
						<div className="p-16 text-center border-2 border-dashed border-[var(--border-subtle)] rounded-[var(--radius-card)]">
							<div className="text-6xl mb-6 opacity-30 grayscale">🎵</div>
							<h3 className="text-lg font-bold text-[var(--text-primary)] mb-2">
								{globalFilter ? "No matching files" : "Your library is empty"}
							</h3>
							<p className="text-[var(--text-secondary)] max-w-sm mx-auto">
								{globalFilter
									? "Try adjusting your search terms"
									: "Upload your first audio file to get started with transcription"
								}
							</p>
						</div>
					) : (
						<>
							{/* Table */}
							<div className={`rounded-lg overflow-hidden relative transition-opacity duration-300 ${isPageChanging ? 'opacity-60' : ''}`}>
								{isPageChanging && (
									<div className="absolute inset-0 bg-white/50 dark:bg-black/50 backdrop-blur-[1px] flex items-center justify-center z-10">
										<div className="flex items-center space-x-2 text-[var(--text-secondary)] bg-[var(--bg-card)] px-4 py-2 rounded-full shadow-[var(--shadow-float)]">
											<Loader2 className="h-4 w-4 animate-spin text-[var(--brand-solid)]" />
											<span className="text-sm font-medium">Updating...</span>
										</div>
									</div>
								)}
								<Table>
									<TableHeader className="hidden sm:table-header-group">
										{table.getHeaderGroups().map((headerGroup) => (
											<TableRow
												key={headerGroup.id}
												className="!border-b border-[var(--border-subtle)] hover:bg-transparent"
											>
												{headerGroup.headers.map((header) => (
													<TableHead
														key={header.id}
														className={`h-12 text-sm font-bold uppercase tracking-wider text-[var(--text-tertiary)] ${header.column.id === 'created_at' ? 'hidden sm:table-cell' : ''} ${header.column.id === 'title' ? 'w-full pl-0' : ''} ${header.column.id === 'status' ? 'w-10 text-center' : ''} ${header.column.id === 'actions' ? 'w-10 text-center lg:hidden' : ''} ${header.column.id === 'select' ? 'w-[40px] px-2' : ''}`}
													>
														{header.isPlaceholder
															? null
															: flexRender(
																header.column.columnDef.header,
																header.getContext()
															)}
													</TableHead>
												))}
											</TableRow>
										))}
									</TableHeader>
									<TableBody>
										{table.getRowModel().rows?.length ? (
											table.getRowModel().rows.map((row) => (
												<TableRow
													key={row.id}
													className="hover:bg-[var(--brand-light)]/30 transition-all duration-200 border-b border-[var(--border-subtle)] last:border-b-0 group/row h-16"
													data-state={row.getIsSelected() && "selected"}
												>
													{row.getVisibleCells().map((cell) => (
														<TableCell
															key={cell.id}
															className={`
																py-4
																${cell.column.id === 'created_at' ? 'hidden sm:table-cell' : ''}
																${cell.column.id === 'title' ? 'whitespace-normal break-words pr-1 sm:pr-4 pl-0' : ''}
																${cell.column.id === 'status' ? 'w-[36px] px-1 text-center' : ''}
																${cell.column.id === 'actions' ? 'w-[36px] px-1 text-center lg:hidden' : ''}
																${cell.column.id === 'select' ? 'w-[40px] px-2' : ''}
															`}
														>
															{flexRender(
																cell.column.columnDef.cell,
																cell.getContext()
															)}
														</TableCell>
													))}
												</TableRow>
											))
										) : (
											<TableRow>
												<TableCell
													colSpan={columns.length}
													className="h-32 text-center text-[var(--text-secondary)]"
												>
													No results found.
												</TableCell>
											</TableRow>
										)}
									</TableBody>
								</Table>
							</div>

							{/* Pagination */}
							<div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 sm:gap-0 px-2 py-4">
								<div className="flex items-center space-x-2">
									<p className="text-sm text-carbon-600 dark:text-carbon-400">
										{globalFilter ? (
											`Showing ${pagination.pageIndex * pagination.pageSize + 1} to ${Math.min(
												(pagination.pageIndex + 1) * pagination.pageSize,
												totalItems
											)} of ${totalItems} entries (filtered)`
										) : (
											`Showing ${pagination.pageIndex * pagination.pageSize + 1} to ${Math.min(
												(pagination.pageIndex + 1) * pagination.pageSize,
												totalItems
											)} of ${totalItems} entries`
										)}
									</p>
								</div>
								<div className="flex items-center space-x-2">
									<Button
										variant="outline"
										size="sm"
										onClick={() => table.setPageIndex(0)}
										disabled={!table.getCanPreviousPage()}
										className="text-carbon-600 dark:text-carbon-400"
									>
										<ChevronsLeft className="h-4 w-4" />
									</Button>
									<Button
										variant="outline"
										size="sm"
										onClick={() => table.previousPage()}
										disabled={!table.getCanPreviousPage()}
										className="text-carbon-600 dark:text-carbon-400"
									>
										<ChevronLeft className="h-4 w-4" />
									</Button>
									<span className="text-sm text-carbon-600 dark:text-carbon-400">
										Page {table.getState().pagination.pageIndex + 1} of{" "}
										{pageCount}
									</span>
									<Button
										variant="outline"
										size="sm"
										onClick={() => table.nextPage()}
										disabled={!table.getCanNextPage()}
										className="text-carbon-600 dark:text-carbon-400"
									>
										<ChevronRight className="h-4 w-4" />
									</Button>
									<Button
										variant="outline"
										size="sm"
										onClick={() => table.setPageIndex(pageCount - 1)}
										disabled={!table.getCanNextPage()}
										className="text-carbon-600 dark:text-carbon-400"
									>
										<ChevronsRight className="h-4 w-4" />
									</Button>
								</div>
							</div>
						</>
					)}
				</div>
			</div>

			{/* Transcription Config Dialog (reused for bulk) */}
			<TranscriptionConfigDialog
				open={configDialogOpen}
				onOpenChange={setConfigDialogOpen}
				onStartTranscription={onStartTranscribe}
				loading={transcriptionLoading || bulkActionLoading}
				title={selectedCount > 0 ? `Transcribe ${selectedCount} Files` : undefined}
			/>

			{/* Transcribe-D Dialog (reused for bulk) */}
			<TranscribeDDialog
				open={transcribeDDialogOpen}
				onOpenChange={setTranscribeDDialogOpen}
				onStartTranscription={onStartTranscribeWithProfile}
				loading={transcriptionLoading || bulkActionLoading}
				title={selectedCount > 0 ? `Transcribe+ ${selectedCount} Files` : undefined}
			/>

			{/* Bulk Delete Confirmation */}
			<AlertDialog open={bulkDeleteDialogOpen} onOpenChange={setBulkDeleteDialogOpen}>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Delete {selectedCount} Files?</AlertDialogTitle>
						<AlertDialogDescription>
							This action cannot be undone. This will permanently delete the selected audio files and their transcripts.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={handleBulkDelete}
							className="bg-red-600 hover:bg-red-700 text-white"
						>
							{bulkActionLoading ? (
								<>
									<Loader2 className="mr-2 h-4 w-4 animate-spin" />
									Deleting...
								</>
							) : (
								"Delete Files"
							)}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
			{/* Stop Transcription Dialog */}
			<AlertDialog open={stopDialogOpen} onOpenChange={setStopDialogOpen}>
				<AlertDialogContent className="glass-card bg-[var(--bg-main)]/90 border-[var(--border-subtle)]">
					<AlertDialogHeader>
						<AlertDialogTitle className="text-[var(--text-primary)]">
							Stop Transcription
						</AlertDialogTitle>
						<AlertDialogDescription className="text-[var(--text-secondary)]">
							Are you sure you want to stop the transcription of "
							{selectedFile?.title || (selectedFile ? getFileName(selectedFile.audio_path) : "")}
							"? This will cancel the current transcription process.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel className="bg-[var(--secondary)] border-[var(--border-subtle)] text-[var(--text-secondary)] hover:bg-[var(--bg-card)]">
							Cancel
						</AlertDialogCancel>
						<AlertDialogAction
							className="bg-[var(--warning)] text-white hover:opacity-90"
							onClick={() => {
								if (selectedFile) {
									handleKillJob(selectedFile.id);
								}
								setStopDialogOpen(false);
							}}
						>
							Stop Transcription
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
							onClick={() => {
								if (selectedFile) {
									handleDelete(selectedFile.id);
								}
								setDeleteDialogOpen(false);
							}}
						>
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
});
