import { useState } from "react";
import { Header } from "./Header";
import { AudioFilesTable } from "./AudioFilesTable";
import { useAuth } from "../contexts/AuthContext";
import { Progress } from "./ui/progress";
import { X, CheckCircle, AlertCircle } from "lucide-react";

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

	return (
		<div className="min-h-screen bg-gray-50 dark:bg-gray-900">
			<div className="mx-auto w-full max-w-6xl px-2 sm:px-6 md:px-8 py-3 sm:py-6">
				<Header 
					onFileSelect={handleFileSelect} 
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
		</div>
	);
}
