import { useState, useEffect } from "react";
import { CheckCircle, Clock, XCircle, Loader2, MoreVertical, Play, Hash, Trash2 } from "lucide-react";
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
import { Button } from "@/components/ui/button";

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

export function AudioFilesTable({ refreshTrigger, onTranscribe }: AudioFilesTableProps) {
	const [audioFiles, setAudioFiles] = useState<AudioFile[]>([]);
	const [loading, setLoading] = useState(true);
	const [queuePositions, setQueuePositions] = useState<Record<string, number>>({});
	const [openPopovers, setOpenPopovers] = useState<Record<string, boolean>>({});

	const fetchAudioFiles = async () => {
		try {
			const response = await fetch("/api/v1/transcription/list", {
				headers: {
					"X-API-Key": "dev-api-key-123",
				},
			});

			if (response.ok) {
				const data = await response.json();
				setAudioFiles(data.jobs || []);
				// Fetch queue positions for pending jobs
				fetchQueuePositions(data.jobs || []);
			}
		} catch (error) {
			console.error("Failed to fetch audio files:", error);
		} finally {
			setLoading(false);
		}
	};

	// Fetch queue positions for pending jobs
	const fetchQueuePositions = async (jobs: AudioFile[]) => {
		const pendingJobs = jobs.filter(job => job.status === 'pending');
		if (pendingJobs.length === 0) return;

		try {
			const response = await fetch('/api/v1/admin/queue/stats', {
				headers: {
					'X-API-Key': 'dev-api-key-123'
				}
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
			console.error('Failed to fetch queue positions:', error);
		}
	};

	// Handle transcribe action
	const handleTranscribe = async (jobId: string) => {
		try {
			const job = audioFiles.find(f => f.id === jobId);
			if (!job) return;

			// Close the popover
			setOpenPopovers(prev => ({ ...prev, [jobId]: false }));

			const response = await fetch(`/api/v1/transcription/${jobId}/start`, {
				method: 'POST',
				headers: {
					'X-API-Key': 'dev-api-key-123',
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					model: 'base',
					diarization: false
				})
			});

			if (response.ok) {
				// Refresh to show updated status
				fetchAudioFiles();
				if (onTranscribe) {
					onTranscribe(jobId);
				}
			} else {
				alert('Failed to start transcription');
			}
		} catch (error) {
			alert('Error starting transcription');
		}
	};

	// Check if job can be transcribed (not currently processing or pending)
	const canTranscribe = (file: AudioFile) => {
		return file.status !== 'processing' && file.status !== 'pending';
	};

	// Handle delete action
	const handleDelete = async (jobId: string) => {
		// Close the popover
		setOpenPopovers(prev => ({ ...prev, [jobId]: false }));
		
		try {
			const response = await fetch(`/api/v1/transcription/${jobId}`, {
				method: 'DELETE',
				headers: {
					'X-API-Key': 'dev-api-key-123'
				}
			});

			if (response.ok) {
				// Refresh to show updated list
				fetchAudioFiles();
			} else {
				alert('Failed to delete audio file');
			}
		} catch (error) {
			alert('Error deleting audio file');
		}
	};

	useEffect(() => {
		fetchAudioFiles();
	}, [refreshTrigger]);

	// Poll for status updates every 3 seconds for active jobs
	useEffect(() => {
		const activeJobs = audioFiles.filter(job => 
			job.status === 'pending' || job.status === 'processing'
		);
		
		if (activeJobs.length === 0) return;

		const interval = setInterval(() => {
			fetchAudioFiles();
		}, 3000);

		return () => clearInterval(interval);
	}, [audioFiles]);

	const getStatusIcon = (file: AudioFile) => {
		const iconSize = 16;
		const status = file.status;
		const queuePosition = queuePositions[file.id];

		switch (status) {
			case "completed":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help inline-block">
								<CheckCircle size={iconSize} className="text-magnum-400" />
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-gray-900 border-gray-700">
							<p>Completed</p>
						</TooltipContent>
					</Tooltip>
				);
			case "processing":
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help inline-block">
								<Loader2 size={iconSize} className="text-blue-400 animate-spin" />
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-gray-900 border-gray-700">
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
						<TooltipContent className="bg-gray-900 border-gray-700">
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
								<span className="text-xs text-purple-400 font-medium">{queuePosition || '?'}</span>
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-gray-900 border-gray-700">
							<p>Queued (Position {queuePosition || '?'})</p>
						</TooltipContent>
					</Tooltip>
				);
			case "uploaded":
			default:
				return (
					<Tooltip>
						<TooltipTrigger asChild>
							<div className="cursor-help inline-block">
								<Clock size={iconSize} className="text-neon-100" />
							</div>
						</TooltipTrigger>
						<TooltipContent className="bg-gray-900 border-gray-700">
							<p>Uploaded</p>
						</TooltipContent>
					</Tooltip>
				);
		}
	};

	const formatDate = (dateString: string) => {
		return new Date(dateString).toLocaleDateString("en-US", {
			year: "numeric",
			month: "short",
			day: "numeric",
			hour: "2-digit",
			minute: "2-digit",
		});
	};

	const getFileName = (audioPath: string) => {
		const parts = audioPath.split("/");
		return parts[parts.length - 1];
	};

	if (loading) {
		return (
			<div className="bg-gray-800 rounded-xl p-6">
				<div className="animate-pulse">
					<div className="h-4 bg-gray-700 rounded w-1/4 mb-6"></div>
					<div className="space-y-3">
						{[...Array(5)].map((_, i) => (
							<div key={i} className="h-12 bg-gray-700/50 rounded-lg"></div>
						))}
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="bg-gray-800 rounded-xl overflow-hidden">
				<div className="p-6">
					<h2 className="text-xl font-semibold text-gray-50 mb-2">Audio Files</h2>
					<p className="text-gray-400 text-sm">
						{audioFiles.length} file{audioFiles.length !== 1 ? "s" : ""} total
					</p>
				</div>

			{audioFiles.length === 0 ? (
				<div className="p-12 text-center">
					<div className="text-5xl mb-4 opacity-50">🎵</div>
					<h3 className="text-lg font-medium text-gray-300 mb-2">
						No audio files yet
					</h3>
					<p className="text-gray-500">
						Upload your first audio file to get started
					</p>
				</div>
			) : (
				<div className="overflow-x-auto">
					<table className="w-full">
						<thead className="bg-gray-700">
							<tr>
								<th className="text-left px-6 py-3 text-sm font-medium text-gray-300">
									Title
								</th>
								<th className="text-left px-6 py-3 text-sm font-medium text-gray-300">
									Date Added
								</th>
								<th className="text-center px-6 py-3 text-sm font-medium text-gray-300">
									Status
								</th>
								<th className="text-center px-6 py-3 text-sm font-medium text-gray-300">
									Actions
								</th>
							</tr>
						</thead>
						<tbody>
							{audioFiles.map((file) => (
								<tr
									key={file.id}
									className="hover:bg-gray-700 transition-colors duration-200 border-b border-gray-850 last:border-b-0"
								>
									<td className="px-6 py-3">
										<span className="text-gray-50 font-medium">
											{file.title || getFileName(file.audio_path)}
										</span>
									</td>
									<td className="px-6 py-3 text-gray-300 text-sm">
										{formatDate(file.created_at)}
									</td>
									<td className="px-6 py-3 text-center">{getStatusIcon(file)}</td>
									<td className="px-6 py-3 text-center">
										<Popover 
											open={openPopovers[file.id] || false}
											onOpenChange={(open) => setOpenPopovers(prev => ({ ...prev, [file.id]: open }))}
										>
											<PopoverTrigger asChild>
												<Button variant="ghost" size="sm" className="h-9 w-9 p-0">
													<MoreVertical className="h-5 w-5" />
												</Button>
											</PopoverTrigger>
											<PopoverContent className="w-40 bg-gray-900 border-gray-600 p-1">
												<div className="space-y-1">
													<Button
														variant="ghost"
														size="sm"
														className="w-full justify-start h-8 text-sm hover:bg-gray-700"
														disabled={!canTranscribe(file)}
														onClick={() => handleTranscribe(file.id)}
													>
														<Play className="mr-2 h-4 w-4" />
														Transcribe
													</Button>
													<AlertDialog>
														<AlertDialogTrigger asChild>
															<Button
																variant="ghost"
																size="sm"
																className="w-full justify-start h-8 text-sm hover:bg-gray-700 text-red-400 hover:text-red-300"
															>
																<Trash2 className="mr-2 h-4 w-4" />
																Delete
															</Button>
														</AlertDialogTrigger>
														<AlertDialogContent className="bg-gray-900 border-gray-700">
															<AlertDialogHeader>
																<AlertDialogTitle className="text-gray-100">Delete Audio File</AlertDialogTitle>
																<AlertDialogDescription className="text-gray-400">
																	Are you sure you want to delete "{file.title || getFileName(file.audio_path)}"? 
																	This action cannot be undone and will permanently remove the audio file and any transcription data.
																</AlertDialogDescription>
															</AlertDialogHeader>
															<AlertDialogFooter>
																<AlertDialogCancel className="bg-gray-800 border-gray-600 text-gray-200 hover:bg-gray-700">Cancel</AlertDialogCancel>
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
									</td>
								</tr>
							))}
						</tbody>
					</table>
				</div>
			)}
		</div>
	);
}
