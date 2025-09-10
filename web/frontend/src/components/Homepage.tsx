import { useState, useCallback, useRef } from "react";
import { Header } from "./Header";
import { AudioFilesTable } from "./AudioFilesTable";
import { DragDropOverlay } from "./DragDropOverlay";
import { MultiTrackUploadDialog } from "./MultiTrackUploadDialog";
import { useAuth } from "../contexts/AuthContext";
import { Progress } from "./ui/progress";
import { X, CheckCircle, AlertCircle } from "lucide-react";
import { 
	groupFiles, 
	convertToFileWithType, 
	prepareMultiTrackFiles, 
	hasValidFiles, 
	getFileDescription, 
	validateMultiTrackFiles 
} from "../utils/fileProcessor";

interface FileWithType {
	file: File;
	isVideo: boolean;
}

interface UploadProgress {
	fileName: string;
	status: 'uploading' | 'success' | 'error';
	error?: string;
}

export function Homepage() {
	const { getAuthHeaders } = useAuth();
	const [refreshTrigger, setRefreshTrigger] = useState(0);
	const [uploadProgress, setUploadProgress] = useState<UploadProgress[]>([]);
	const [isUploading, setIsUploading] = useState(false);
	
	// Drag and drop state
	const [isDragging, setIsDragging] = useState(false);
	const [dragCount, setDragCount] = useState(0);
	const [draggedFileGroup, setDraggedFileGroup] = useState<ReturnType<typeof groupFiles> | null>(null);
	const [isMultiTrackDialogOpen, setIsMultiTrackDialogOpen] = useState(false);
	const [multiTrackPreview, setMultiTrackPreview] = useState<{audioFiles: File[], aupFile: File, title: string} | null>(null);
	const dragCounter = useRef(0);

	const uploadSingleFile = async (file: File): Promise<boolean> => {
		const formData = new FormData();
		formData.append("audio", file);
		formData.append("title", file.name.replace(/\.[^/.]+$/, ""));

		try {
			const response = await fetch("/api/v1/transcription/upload", {
				method: "POST",
				headers: {
					...getAuthHeaders(),
				},
				body: formData,
			});

			return response.ok;
		} catch {
			return false;
		}
	};

	const uploadSingleVideo = async (file: File): Promise<boolean> => {
		const formData = new FormData();
		formData.append("video", file);
		formData.append("title", file.name.replace(/\.[^/.]+$/, ""));

		try {
			const response = await fetch("/api/v1/transcription/upload-video", {
				method: "POST",
				headers: {
					...getAuthHeaders(),
				},
				body: formData,
			});

			return response.ok;
		} catch {
			return false;
		}
	};

	const handleFileSelect = async (files: File | File[] | FileWithType | FileWithType[]) => {
		// Normalize input to an array of FileWithType objects
		const fileArray = Array.isArray(files) ? files : [files];
		const processedFiles = fileArray.map(item => {
			if ('file' in item && 'isVideo' in item) {
				// It's already a FileWithType
				return item;
			} else {
				// It's a regular File, treat as audio
				return { file: item as File, isVideo: false };
			}
		});
		
		if (processedFiles.length === 0) return;
		
		setIsUploading(true);
		setUploadProgress(processedFiles.map(item => ({
			fileName: item.file.name,
			status: 'uploading'
		})));

		let successCount = 0;
		
		// Upload files sequentially to avoid overwhelming the server
		for (let i = 0; i < processedFiles.length; i++) {
			const fileItem = processedFiles[i];
			const file = fileItem.file;
			const isVideo = fileItem.isVideo;
			
			try {
				const success = isVideo ? await uploadSingleVideo(file) : await uploadSingleFile(file);
				
				setUploadProgress(prev => prev.map((item, index) => 
					index === i ? {
						...item,
						status: success ? 'success' : 'error',
						error: success ? undefined : (isVideo ? 'Video upload failed' : 'Upload failed')
					} : item
				));
				
				if (success) {
					successCount++;
				}
			} catch (error) {
				setUploadProgress(prev => prev.map((item, index) => 
					index === i ? {
						...item,
						status: 'error',
						error: 'Network error'
					} : item
				));
			}
		}
		
		setIsUploading(false);
		
		// Refresh table if any uploads succeeded
		if (successCount > 0) {
			setRefreshTrigger((prev) => prev + 1);
		}
		
		// Auto-hide progress after 3 seconds if all succeeded
		if (successCount === fileArray.length) {
			setTimeout(() => setUploadProgress([]), 3000);
		}
	};

	const handleTranscribe = () => {
		// Refresh table when transcription starts
		setRefreshTrigger((prev) => prev + 1);
	};

	const dismissProgress = () => {
		setUploadProgress([]);
	};

	const handleMultiTrackUpload = async (files: File[], aupFile: File, title: string) => {
		setIsUploading(true);
		
		// Create progress entry for multi-track upload
		const multiTrackProgress = {
			fileName: `${title} (${files.length} tracks)`,
			status: 'uploading' as const
		};
		
		setUploadProgress([multiTrackProgress]);

		try {
			const formData = new FormData();
			formData.append('title', title);
			formData.append('aup', aupFile);
			
			files.forEach(file => {
				formData.append('tracks', file);
			});

			const response = await fetch("/api/v1/transcription/upload-multitrack", {
				method: "POST",
				headers: {
					...getAuthHeaders(),
				},
				body: formData,
			});

			if (response.ok) {
				setUploadProgress([{
					...multiTrackProgress,
					status: 'success'
				}]);
				
				// Refresh table
				setRefreshTrigger((prev) => prev + 1);
				
				// Auto-hide progress after 3 seconds
				setTimeout(() => setUploadProgress([]), 3000);
			} else {
				const errorData = await response.json();
				setUploadProgress([{
					...multiTrackProgress,
					status: 'error',
					error: errorData.error || 'Upload failed'
				}]);
			}
		} catch (error) {
			setUploadProgress([{
				...multiTrackProgress,
				status: 'error',
				error: 'Upload failed: Network error'
			}]);
		} finally {
			setIsUploading(false);
		}
	};

	// Drag and drop handlers
	const handleDragEnter = useCallback((e: React.DragEvent) => {
		e.preventDefault();
		e.stopPropagation();
		
		dragCounter.current++;
		
		if (e.dataTransfer.items && e.dataTransfer.items.length > 0) {
			setIsDragging(true);
			setDragCount(dragCounter.current);
			
			// Preview files being dragged
			const files = Array.from(e.dataTransfer.items)
				.filter(item => item.kind === 'file')
				.map(item => item.getAsFile())
				.filter((file): file is File => file !== null);
			
			if (files.length > 0) {
				const fileGroup = groupFiles(files);
				setDraggedFileGroup(fileGroup);
			}
		}
	}, []);

	const handleDragLeave = useCallback((e: React.DragEvent) => {
		e.preventDefault();
		e.stopPropagation();
		
		dragCounter.current--;
		
		if (dragCounter.current === 0) {
			setIsDragging(false);
			setDragCount(0);
			setDraggedFileGroup(null);
		}
	}, []);

	const handleDragOver = useCallback((e: React.DragEvent) => {
		e.preventDefault();
		e.stopPropagation();
	}, []);

	const handleDrop = useCallback(async (e: React.DragEvent) => {
		e.preventDefault();
		e.stopPropagation();
		
		// Reset drag state
		dragCounter.current = 0;
		setIsDragging(false);
		setDragCount(0);
		setDraggedFileGroup(null);
		
		const files = Array.from(e.dataTransfer.files);
		if (files.length === 0) return;
		
		const fileGroup = groupFiles(files);
		
		// Validate files
		if (!hasValidFiles(fileGroup)) {
			// Show error - could add toast notification here
			console.error('Invalid files dropped');
			return;
		}
		
		// Handle different file types
		if (fileGroup.type === 'multitrack') {
			const multiTrackFiles = prepareMultiTrackFiles(fileGroup);
			if (multiTrackFiles) {
				// Open multi-track dialog with pre-populated data
				setMultiTrackPreview(multiTrackFiles);
				setIsMultiTrackDialogOpen(true);
			}
		} else if (fileGroup.type === 'video') {
			// Handle video files
			const filesWithType = convertToFileWithType(fileGroup.files, true);
			await handleFileSelect(filesWithType);
		} else {
			// Handle regular audio files
			await handleFileSelect(fileGroup.files);
		}
	}, []);

	const handleMultiTrackDialogClose = useCallback(() => {
		setIsMultiTrackDialogOpen(false);
		setMultiTrackPreview(null);
	}, []);

	const handleMultiTrackConfirm = useCallback(async (files: File[], aupFile: File, title: string) => {
		await handleMultiTrackUpload(files, aupFile, title);
		handleMultiTrackDialogClose();
	}, []);

	return (
		<div 
			className="min-h-screen bg-gray-50 dark:bg-gray-900"
			onDragEnter={handleDragEnter}
			onDragLeave={handleDragLeave}
			onDragOver={handleDragOver}
			onDrop={handleDrop}
		>
			<div className="mx-auto w-full max-w-6xl px-2 sm:px-6 md:px-8 py-3 sm:py-6">
				<Header 
					onFileSelect={handleFileSelect} 
					onMultiTrackUpload={handleMultiTrackUpload}
					onDownloadComplete={() => setRefreshTrigger((prev) => prev + 1)}
				/>
				
				{/* Upload Progress */}
				{uploadProgress.length > 0 && (
					<div className="mb-4 sm:mb-6 bg-white dark:bg-gray-800 rounded-xl p-4 sm:p-6">
						<div className="flex items-center justify-between mb-4">
							<h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
								Uploading Files ({uploadProgress.filter(p => p.status === 'success').length}/{uploadProgress.length})
							</h3>
							{!isUploading && (
								<button
									onClick={dismissProgress}
									className="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors cursor-pointer"
								>
									<X className="h-4 w-4 text-gray-500 dark:text-gray-400" />
								</button>
							)}
						</div>
						
						{/* Overall progress */}
						<div className="mb-4">
							<Progress 
								value={(uploadProgress.filter(p => p.status !== 'uploading').length / uploadProgress.length) * 100} 
								className="h-2"
							/>
						</div>
						
						{/* Individual file progress */}
						<div className="space-y-2 max-h-32 overflow-y-auto">
							{uploadProgress.map((progress, index) => (
								<div key={index} className="flex items-center gap-3 text-sm">
									<div className="flex-shrink-0">
										{progress.status === 'uploading' && (
											<div className="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
										)}
										{progress.status === 'success' && (
											<CheckCircle className="w-4 h-4 text-green-500" />
										)}
										{progress.status === 'error' && (
											<AlertCircle className="w-4 h-4 text-red-500" />
										)}
									</div>
									<div className="flex-1 min-w-0">
										<div className="truncate text-gray-900 dark:text-gray-100">
											{progress.fileName}
										</div>
										{progress.error && (
											<div className="text-red-500 dark:text-red-400 text-xs">
												{progress.error}
											</div>
										)}
									</div>
									<div className="flex-shrink-0 text-xs text-gray-500 dark:text-gray-400">
										{progress.status === 'uploading' && 'Uploading...'}
										{progress.status === 'success' && 'Completed'}
										{progress.status === 'error' && 'Failed'}
									</div>
								</div>
							))}
						</div>
					</div>
				)}
				
				<AudioFilesTable
					refreshTrigger={refreshTrigger}
					onTranscribe={handleTranscribe}
				/>
			</div>
			
			{/* Drag and Drop Overlay */}
			<DragDropOverlay
				isDragging={isDragging}
				dragCount={dragCount}
				fileType={draggedFileGroup?.type}
				fileDescription={draggedFileGroup ? getFileDescription(draggedFileGroup) : undefined}
				errorMessage={draggedFileGroup && !hasValidFiles(draggedFileGroup) 
					? (draggedFileGroup.type === 'multitrack' 
						? validateMultiTrackFiles([...draggedFileGroup.files, draggedFileGroup.aupFile!]).error
						: "No supported files found")
					: undefined}
			/>
			
			{/* Multi-track Upload Dialog with pre-populated data */}
			<MultiTrackUploadDialog
				open={isMultiTrackDialogOpen}
				onOpenChange={handleMultiTrackDialogClose}
				onMultiTrackUpload={handleMultiTrackConfirm}
				prePopulatedFiles={multiTrackPreview?.audioFiles}
				prePopulatedAupFile={multiTrackPreview?.aupFile}
				prePopulatedTitle={multiTrackPreview?.title}
			/>
		</div>
	);
}
