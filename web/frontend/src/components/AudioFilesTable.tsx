import { useState, useEffect, useMemo, useCallback } from "react";
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
	AlertDialogTrigger,
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
import { TranscriptionConfigDialog, type WhisperXParams } from "./TranscriptionConfigDialog";
import { TranscribeDDialog } from "./TranscribeDDialog";
import { useRouter } from "../contexts/RouterContext";
import { useAuth } from "../contexts/AuthContext";
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
			className={className}
		/>
	);
}

interface AudioFile {
	id: string;
	title?: string;
	status: "uploaded" | "pending" | "processing" | "completed" | "failed";
	created_at: string;
	audio_path: string;
}

interface AudioFilesTableProps {
	refreshTrigger: number;
	onTranscribe?: (jobId: string) => void;
}

interface PaginationResponse {
	jobs: AudioFile[];
	pagination: {
		page: number;
		limit: number;
		total: number;
		pages: number;
	};
}


export function AudioFilesTable({
	refreshTrigger,
	onTranscribe,
}: AudioFilesTableProps) {
	const { navigate } = useRouter();
	const { getAuthHeaders } = useAuth();
	const [data, setData] = useState<AudioFile[]>([]);
	const [loading, setLoading] = useState(true);
	const [isPageChanging, setIsPageChanging] = useState(false);
	const [pagination, setPagination] = useState({
		pageIndex: 0,
		pageSize: 10,
	});
	const [sorting, setSorting] = useState<SortingState>([]);
	const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
	const [globalFilter, setGlobalFilter] = useState("");
	const [queuePositions, setQueuePositions] = useState<Record<string, number>>({});
	const [openPopovers, setOpenPopovers] = useState<Record<string, boolean>>({});
	const [totalItems, setTotalItems] = useState(0);
	const [pageCount, setPageCount] = useState(0);
	const [configDialogOpen, setConfigDialogOpen] = useState(false);
	const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
	const [transcriptionLoading, setTranscriptionLoading] = useState(false);
	const [killingJobs, setKillingJobs] = useState<Set<string>>(new Set());
	const [transcribeDDialogOpen, setTranscribeDDialogOpen] = useState(false);

	const fetchAudioFiles = useCallback(async (page?: number, limit?: number, searchQuery?: string, isInitialLoad = false) => {
		try {
			// Only show loading skeleton on initial load, use page changing indicator for pagination
			if (isInitialLoad) {
				setLoading(true);
			} else {
				setIsPageChanging(true);
			}
			
			const currentPage = page || pagination.pageIndex + 1;
			const currentLimit = limit || pagination.pageSize;
			const currentSearch = searchQuery !== undefined ? searchQuery : globalFilter;
			
			const params = new URLSearchParams({
				page: currentPage.toString(),
				limit: currentLimit.toString(),
			});
			
			// Add search parameter if provided
			if (currentSearch) {
				params.set('q', currentSearch);
			}
			
			const response = await fetch(`/api/v1/transcription/list?${params}`, {
				headers: {
					...getAuthHeaders(),
				},
			});

			if (response.ok) {
				const result: PaginationResponse = await response.json();
				console.log("Fetched data:", result.jobs?.length, "items");
				console.log("Total from API:", result.pagination.total);
				console.log("Current page:", result.pagination.page);
				console.log("Total pages:", result.pagination.pages);
				console.log("Search query:", currentSearch);
				
				setData(result.jobs || []);
				setTotalItems(result.pagination.total);
				setPageCount(result.pagination.pages);
				// Fetch queue positions for pending jobs
				fetchQueuePositions(result.jobs || []);
			}
		} catch (error) {
			console.error("Failed to fetch audio files:", error);
		} finally {
			setLoading(false);
			setIsPageChanging(false);
		}
	}, [pagination.pageIndex, pagination.pageSize, globalFilter]);

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
				fetchAudioFiles();
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
	}, [selectedJobId, fetchAudioFiles, onTranscribe]);

	// Handle actual transcription start with profile parameters
	const handleStartTranscriptionWithProfile = useCallback(async (params: WhisperXParams, _profileId?: string) => {
		if (!selectedJobId) return;

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
				fetchAudioFiles();
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
	}, [selectedJobId, fetchAudioFiles, onTranscribe]);

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
				fetchAudioFiles();
			} else {
				alert("Failed to delete audio file");
			}
		} catch {
			alert("Error deleting audio file");
		}
	}, [fetchAudioFiles]);

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
				fetchAudioFiles();
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
	}, [fetchAudioFiles]);

	// Initial load
	useEffect(() => {
		const isInitialLoad = data.length === 0;
		fetchAudioFiles(undefined, undefined, undefined, isInitialLoad);
	}, [refreshTrigger, fetchAudioFiles]);

	// Handle pagination and search changes
	useEffect(() => {
		if (data.length > 0) { // Only fetch if not initial load
			fetchAudioFiles(
				pagination.pageIndex + 1, 
				pagination.pageSize, 
				globalFilter || undefined
			);
		}
	}, [pagination.pageIndex, pagination.pageSize, globalFilter, fetchAudioFiles]);

	// Reset to first page when search changes
	useEffect(() => {
		if (globalFilter !== undefined) {
			setPagination(prev => ({ ...prev, pageIndex: 0 }));
		}
	}, [globalFilter]);

	// Poll for status updates every 3 seconds for active jobs
	useEffect(() => {
		const activeJobs = data.filter(
			(job) => job.status === "pending" || job.status === "processing",
		);

		if (activeJobs.length === 0) return;

		const interval = setInterval(() => {
			// Keep current pagination when polling
			fetchAudioFiles();
		}, 3000);

		return () => clearInterval(interval);
	}, [data, fetchAudioFiles]);

	const getStatusIcon = useCallback((file: AudioFile) => {
		const iconSize = 16;
		const status = file.status;
		const queuePosition = queuePositions[file.id];

		switch (status) {
			case "completed":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help inline-block">
								<CheckCircle size={iconSize} className="text-green-500" />
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-gray-900 border-gray-700 text-white">
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
									className="text-blue-400 animate-spin"
								/>
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-gray-900 border-gray-700 text-white">
							<p>Processing</p>
						</TooltipContent>
					</Tooltip>
				);
			case "failed":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help inline-block">
								<XCircle size={iconSize} className="text-magenta-400" />
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-gray-900 border-gray-700 text-white">
							<p>Failed</p>
						</TooltipContent>
					</Tooltip>
				);
			case "pending":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="flex items-center gap-1 cursor-help inline-flex">
								<Hash size={12} className="text-purple-400" />
								<span className="text-xs text-purple-400 font-medium">
									{queuePosition || "?"}
								</span>
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-gray-900 border-gray-700 text-white">
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
								<Clock size={iconSize} className="text-blue-500" />
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-gray-900 border-gray-700 text-white">
							<p>Uploaded</p>
						</TooltipContent>
					</Tooltip>
				);
		}
	}, [queuePositions]);

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
		navigate({ path: 'audio-detail', params: { id: audioId } });
	}, [navigate]);

	// Define table columns
	const columns = useMemo<ColumnDef<AudioFile>[]>(
		() => [
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
						<button
							onClick={() => handleAudioClick(file.id)}
							className="text-gray-900 dark:text-gray-50 font-medium hover:text-blue-600 dark:hover:text-blue-400 transition-colors cursor-pointer text-left"
						>
							{file.title || getFileName(file.audio_path)}
						</button>
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
					<span className="text-gray-600 dark:text-gray-300 text-sm">
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
											className="h-8 w-8 sm:h-9 sm:w-9 p-0"
										>
											<MoreVertical className="h-5 w-5" />
										</Button>
									</PopoverTrigger>
								<PopoverContent className="w-40 bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-600 p-1">
									<div className="space-y-1">
										{file.status === "completed" && (
											<Button
												variant="ghost"
												size="sm"
												className="w-full justify-start h-8 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer"
												onClick={() => {
													setOpenPopovers((prev) => ({ ...prev, [file.id]: false }));
													navigate({ path: 'chat', params: { audioId: file.id } });
												}}
											>
												<MessageCircle className="mr-2 h-4 w-4" />
												Open Chat
											</Button>
										)}
										<Button
											variant="ghost"
											size="sm"
											className="w-full justify-start h-8 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer disabled:cursor-not-allowed"
											disabled={!canTranscribe(file)}
											onClick={() => handleTranscribeD(file.id)}
										>
											<QuickTranscribeIcon className="mr-2 h-4 w-4" />
											Transcribe
										</Button>
										
										<Button
											variant="ghost"
											size="sm"
											className="w-full justify-start h-8 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer disabled:cursor-not-allowed"
											disabled={!canTranscribe(file)}
											onClick={() => handleTranscribe(file.id)}
										>
											<AdvancedTranscribeIcon className="mr-2 h-4 w-4" />
											Transcribe+
										</Button>
										
										{file.status === "processing" && (
											<AlertDialog>
												<AlertDialogTrigger asChild>
													<Button
														variant="ghost"
														size="sm"
														className="w-full justify-start h-8 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 text-orange-500 dark:text-orange-400 hover:text-orange-600 dark:hover:text-orange-300 cursor-pointer"
														disabled={killingJobs.has(file.id)}
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
												</AlertDialogTrigger>
												<AlertDialogContent className="bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700">
													<AlertDialogHeader>
														<AlertDialogTitle className="text-gray-900 dark:text-gray-100">
															Stop Transcription
														</AlertDialogTitle>
														<AlertDialogDescription className="text-gray-600 dark:text-gray-400">
															Are you sure you want to stop the transcription of "
															{file.title || getFileName(file.audio_path)}
															"? This will cancel the current transcription process.
														</AlertDialogDescription>
													</AlertDialogHeader>
													<AlertDialogFooter>
														<AlertDialogCancel className="bg-gray-100 dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700">
															Cancel
														</AlertDialogCancel>
														<AlertDialogAction
															className="bg-orange-600 text-white hover:bg-orange-700"
															onClick={() => handleKillJob(file.id)}
														>
															Stop Transcription
														</AlertDialogAction>
													</AlertDialogFooter>
												</AlertDialogContent>
											</AlertDialog>
										)}
										
										<AlertDialog>
											<AlertDialogTrigger asChild>
												<Button
													variant="ghost"
													size="sm"
													className="w-full justify-start h-8 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 text-red-500 dark:text-red-400 hover:text-red-600 dark:hover:text-red-300 cursor-pointer"
												>
													<Trash2 className="mr-2 h-4 w-4" />
													Delete
												</Button>
											</AlertDialogTrigger>
											<AlertDialogContent className="bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700">
												<AlertDialogHeader>
													<AlertDialogTitle className="text-gray-900 dark:text-gray-100">
														Delete Audio File
													</AlertDialogTitle>
													<AlertDialogDescription className="text-gray-600 dark:text-gray-400">
														Are you sure you want to delete "
														{file.title || getFileName(file.audio_path)}
														"? This action cannot be undone and will
														permanently remove the audio file and any
														transcription data.
													</AlertDialogDescription>
												</AlertDialogHeader>
												<AlertDialogFooter>
													<AlertDialogCancel className="bg-gray-100 dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700">
														Cancel
													</AlertDialogCancel>
													<AlertDialogAction
														className="bg-red-600 text-white hover:bg-red-700"
														onClick={() => handleDelete(file.id)}
													>
														Delete
													</AlertDialogAction>
												</AlertDialogFooter>
											</AlertDialogContent>
										</AlertDialog>
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
		[openPopovers, queuePositions, getStatusIcon, handleAudioClick, handleTranscribe, handleTranscribeD, handleDelete, canTranscribe, getFileName, killingJobs, handleKillJob]
	);

	// Create the table instance with server-side pagination and search
	const table = useReactTable({
		data,
		columns,
		state: {
			sorting,
			columnFilters,
			globalFilter,
			pagination,
		},
		onSortingChange: setSorting,
		onColumnFiltersChange: setColumnFilters,
		onGlobalFilterChange: setGlobalFilter,
		onPaginationChange: setPagination,
		getCoreRowModel: getCoreRowModel(),
		getSortedRowModel: getSortedRowModel(),
		// Server-side pagination and search
		manualPagination: true,
		pageCount: pageCount,
	});


	if (loading) {
		return (
			<div className="bg-white dark:bg-gray-700 rounded-xl p-6">
				<div className="animate-pulse">
					<div className="h-4 bg-gray-200 dark:bg-gray-600 rounded w-1/4 mb-6"></div>
					<div className="space-y-3">
						{[...Array(5)].map((_, i) => (
							<div
								key={i}
								className="h-12 bg-gray-100 dark:bg-gray-600/50 rounded-lg"
							></div>
						))}
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="bg-white dark:bg-gray-800 rounded-xl overflow-hidden">
			<div className="p-3 sm:p-6">
				<div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 sm:gap-0 mb-4">
					<div>
						<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-50 mb-2">
							Audio Files
						</h2>
						<p className="text-gray-600 dark:text-gray-400 text-sm">
							{globalFilter 
								? `${totalItems} file${totalItems !== 1 ? "s" : ""} found`
								: `${totalItems} file${totalItems !== 1 ? "s" : ""} total`
							}
						</p>
					</div>
					
					{/* Global Search */}
					<div className="relative w-full sm:w-72">
						<Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4 z-10" />
						<DebouncedSearchInput
							placeholder="Search audio files..."
							value={globalFilter ?? ""}
							onChange={setGlobalFilter}
							className="pl-10 bg-white dark:bg-gray-700 border-gray-300 dark:border-gray-600"
						/>
					</div>
				</div>

				{data.length === 0 && !loading ? (
					<div className="p-12 text-center">
						<div className="text-5xl mb-4 opacity-50">🎵</div>
						<h3 className="text-lg font-medium text-gray-600 dark:text-gray-300 mb-2">
							{globalFilter ? "No matching audio files" : "No audio files yet"}
						</h3>
						<p className="text-gray-500 dark:text-gray-400">
							{globalFilter 
								? "Try adjusting your search terms" 
								: "Upload your first audio file to get started"
							}
						</p>
					</div>
				) : (
					<>
						{/* Table */}
					<div className={`border border-gray-100 dark:border-gray-900 rounded-lg overflow-hidden relative transition-opacity duration-200 ${isPageChanging ? 'opacity-75' : ''}`}>
							{isPageChanging && (
								<div className="absolute inset-0 bg-white/20 dark:bg-gray-800/20 flex items-center justify-center z-10">
									<div className="flex items-center space-x-2 text-gray-600 dark:text-gray-400 bg-white dark:bg-gray-800 px-3 py-1 rounded-md shadow-sm">
										<Loader2 className="h-4 w-4 animate-spin" />
										<span className="text-sm">Loading...</span>
									</div>
								</div>
							)}
							<Table>
								<TableHeader className="hidden sm:table-header-group">
									{table.getHeaderGroups().map((headerGroup) => (
										<TableRow 
											key={headerGroup.id}
											className="bg-gray-50 dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 border-b border-gray-100 dark:border-gray-900"
										>
											{headerGroup.headers.map((header) => (
												<TableHead 
													key={header.id}
													className={`text-gray-700 dark:text-gray-300 ${header.column.id === 'created_at' ? 'hidden sm:table-cell' : ''} ${header.column.id === 'title' ? 'w-full' : ''} ${header.column.id === 'status' ? 'w-10 text-center' : ''} ${header.column.id === 'actions' ? 'w-10 text-center' : ''}`}
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
												className="hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors duration-200 border-b border-gray-100 dark:border-gray-700 last:border-b-0"
											>
												{row.getVisibleCells().map((cell) => (
													<TableCell
														key={cell.id}
														className={`
															${cell.column.id === 'created_at' ? 'hidden sm:table-cell' : ''}
															${cell.column.id === 'title' ? 'whitespace-normal break-words pr-1 sm:pr-2' : ''}
															${cell.column.id === 'status' ? 'w-[36px] px-1 text-center' : ''}
															${cell.column.id === 'actions' ? 'w-[36px] px-1 text-center' : ''}
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
												className="h-24 text-center"
											>
												No results.
											</TableCell>
										</TableRow>
									)}
								</TableBody>
							</Table>
						</div>

						{/* Pagination */}
						<div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 sm:gap-0 px-2 py-4">
							<div className="flex items-center space-x-2">
								<p className="text-sm text-gray-600 dark:text-gray-400">
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
									className="text-gray-600 dark:text-gray-400"
								>
									<ChevronsLeft className="h-4 w-4" />
								</Button>
								<Button
									variant="outline"
									size="sm"
									onClick={() => table.previousPage()}
									disabled={!table.getCanPreviousPage()}
									className="text-gray-600 dark:text-gray-400"
								>
									<ChevronLeft className="h-4 w-4" />
								</Button>
								<span className="text-sm text-gray-600 dark:text-gray-400">
									Page {table.getState().pagination.pageIndex + 1} of{" "}
									{pageCount}
								</span>
								<Button
									variant="outline"
									size="sm"
									onClick={() => table.nextPage()}
									disabled={!table.getCanNextPage()}
									className="text-gray-600 dark:text-gray-400"
								>
									<ChevronRight className="h-4 w-4" />
								</Button>
								<Button
									variant="outline"
									size="sm"
									onClick={() => table.setPageIndex(pageCount - 1)}
									disabled={!table.getCanNextPage()}
									className="text-gray-600 dark:text-gray-400"
								>
									<ChevronsRight className="h-4 w-4" />
								</Button>
							</div>
						</div>
					</>
				)}
			</div>

			{/* Transcription Configuration Dialog */}
			<TranscriptionConfigDialog
				open={configDialogOpen}
				onOpenChange={setConfigDialogOpen}
				onStartTranscription={handleStartTranscription}
				loading={transcriptionLoading}
			/>

			{/* Transcribe-D Dialog */}
			<TranscribeDDialog
				open={transcribeDDialogOpen}
				onOpenChange={setTranscribeDDialogOpen}
				onStartTranscription={handleStartTranscriptionWithProfile}
				loading={transcriptionLoading}
			/>
		</div>
	);
}
