import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "../contexts/AuthContext";
import { Upload, X, FileAudio, File, AlertCircle, Check } from "lucide-react";

interface MultiTrackUploadDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onUploadComplete?: () => void;
}

interface FileWithPreview {
	file: File;
	id: string;
	isApu: boolean;
}

interface UploadProgress {
	status: 'idle' | 'uploading' | 'success' | 'error';
	error?: string;
}

export function MultiTrackUploadDialog({
	open,
	onOpenChange,
	onUploadComplete,
}: MultiTrackUploadDialogProps) {
	const { getAuthHeaders } = useAuth();
	const [title, setTitle] = useState("");
	const [files, setFiles] = useState<FileWithPreview[]>([]);
	const [uploadProgress, setUploadProgress] = useState<UploadProgress>({ status: 'idle' });

	const handleDrop = useCallback((e: React.DragEvent<HTMLDivElement>) => {
		e.preventDefault();
		const droppedFiles = Array.from(e.dataTransfer.files);
		addFiles(droppedFiles);
	}, []);

	const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
		if (e.target.files) {
			const selectedFiles = Array.from(e.target.files);
			addFiles(selectedFiles);
		}
	}, []);

	const addFiles = (newFiles: File[]) => {
		const fileItems: FileWithPreview[] = newFiles.map(file => ({
			file,
			id: Math.random().toString(36),
			isApu: file.name.toLowerCase().endsWith('.aup')
		}));
		
		setFiles(prev => [...prev, ...fileItems]);
	};

	const removeFile = (fileId: string) => {
		setFiles(prev => prev.filter(f => f.id !== fileId));
	};

	const audioFiles = files.filter(f => !f.isApu);
	const aupFiles = files.filter(f => f.isApu);
	const hasAupFile = aupFiles.length > 0;
	const hasAudioFiles = audioFiles.length > 0;

	const canUpload = title.trim() !== "" && hasAupFile && hasAudioFiles && uploadProgress.status !== 'uploading';

	const handleUpload = async () => {
		if (!canUpload) return;

		setUploadProgress({ status: 'uploading' });

		try {
			const formData = new FormData();
			formData.append('title', title.trim());
			formData.append('aup', aupFiles[0].file);
			
			audioFiles.forEach(fileItem => {
				formData.append('tracks', fileItem.file);
			});

			const response = await fetch("/api/v1/transcription/upload-multitrack", {
				method: "POST",
				headers: {
					...getAuthHeaders(),
				},
				body: formData,
			});

			if (response.ok) {
				setUploadProgress({ status: 'success' });
				
				// Reset form
				setTitle("");
				setFiles([]);
				
				// Close dialog and notify parent
				setTimeout(() => {
					onOpenChange(false);
					onUploadComplete?.();
					setUploadProgress({ status: 'idle' });
				}, 1500);
			} else {
				const errorData = await response.json();
				setUploadProgress({
					status: 'error',
					error: errorData.error || 'Upload failed'
				});
			}
		} catch (error) {
			setUploadProgress({
				status: 'error',
				error: 'Upload failed: Network error'
			});
		}
	};

	const getSpeakerName = (fileName: string) => {
		// Remove file extension to get speaker name
		return fileName.replace(/\.[^/.]+$/, "");
	};

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
				<DialogHeader>
					<DialogTitle>Upload Multi-Track Audio</DialogTitle>
					<DialogDescription>
						Upload multiple audio tracks with an .aup Audacity project file for multi-speaker transcription.
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-6">
					{/* Title Input */}
					<div className="space-y-2">
						<Label htmlFor="title">Title *</Label>
						<Input
							id="title"
							placeholder="Enter a title for this recording..."
							value={title}
							onChange={(e) => setTitle(e.target.value)}
							disabled={uploadProgress.status === 'uploading'}
						/>
					</div>

					{/* File Upload Zone */}
					<div className="space-y-4">
						<Label>Files</Label>
						
						{/* Drop Zone */}
						<div
							className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
								uploadProgress.status === 'uploading' 
									? 'border-gray-200 bg-gray-50' 
									: 'border-gray-300 hover:border-gray-400 hover:bg-gray-50'
							}`}
							onDrop={handleDrop}
							onDragOver={(e) => e.preventDefault()}
						>
							<Upload className="mx-auto h-12 w-12 text-gray-400 mb-4" />
							<div className="space-y-2">
								<p className="text-lg font-medium">Drop files here or click to upload</p>
								<p className="text-sm text-gray-500">
									Upload multiple audio files and one .aup Audacity project file
								</p>
								<input
									type="file"
									multiple
									accept="audio/*,.aup"
									onChange={handleFileSelect}
									className="hidden"
									id="file-upload"
									disabled={uploadProgress.status === 'uploading'}
								/>
								<label
									htmlFor="file-upload"
									className={`inline-block px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600 cursor-pointer transition-colors ${
										uploadProgress.status === 'uploading' ? 'opacity-50 cursor-not-allowed' : ''
									}`}
								>
									Choose Files
								</label>
							</div>
						</div>

						{/* File List */}
						{files.length > 0 && (
							<div className="space-y-2">
								<h4 className="font-medium text-sm">Uploaded Files:</h4>
								
								{/* AUP Files */}
								{aupFiles.map(fileItem => (
									<div key={fileItem.id} className="flex items-center gap-3 p-3 bg-green-50 border border-green-200 rounded-lg">
										<File className="h-4 w-4 text-green-600 flex-shrink-0" />
										<div className="flex-1 min-w-0">
											<p className="font-medium text-green-800 truncate">
												{fileItem.file.name}
											</p>
											<p className="text-xs text-green-600">Audacity project file</p>
										</div>
										<Button
											variant="ghost"
											size="sm"
											onClick={() => removeFile(fileItem.id)}
											disabled={uploadProgress.status === 'uploading'}
										>
											<X className="h-4 w-4" />
										</Button>
									</div>
								))}

								{/* Audio Files */}
								{audioFiles.map(fileItem => (
									<div key={fileItem.id} className="flex items-center gap-3 p-3 bg-blue-50 border border-blue-200 rounded-lg">
										<FileAudio className="h-4 w-4 text-blue-600 flex-shrink-0" />
										<div className="flex-1 min-w-0">
											<p className="font-medium text-blue-800 truncate">
												{fileItem.file.name}
											</p>
											<p className="text-xs text-blue-600">
												Speaker: {getSpeakerName(fileItem.file.name)}
											</p>
										</div>
										<Button
											variant="ghost"
											size="sm"
											onClick={() => removeFile(fileItem.id)}
											disabled={uploadProgress.status === 'uploading'}
										>
											<X className="h-4 w-4" />
										</Button>
									</div>
								))}
							</div>
						)}

						{/* Validation Messages */}
						{files.length > 0 && (
							<div className="space-y-1 text-sm">
								{!hasAupFile && (
									<div className="flex items-center gap-2 text-red-600">
										<AlertCircle className="h-4 w-4" />
										<span>An .aup Audacity project file is required</span>
									</div>
								)}
								{!hasAudioFiles && (
									<div className="flex items-center gap-2 text-red-600">
										<AlertCircle className="h-4 w-4" />
										<span>At least one audio file is required</span>
									</div>
								)}
								{!title.trim() && (
									<div className="flex items-center gap-2 text-red-600">
										<AlertCircle className="h-4 w-4" />
										<span>Title is required</span>
									</div>
								)}
							</div>
						)}
					</div>

					{/* Upload Progress */}
					{uploadProgress.status !== 'idle' && (
						<div className="space-y-2">
							{uploadProgress.status === 'uploading' && (
								<div className="flex items-center gap-2 text-blue-600">
									<div className="animate-spin h-4 w-4 border-2 border-blue-600 border-t-transparent rounded-full"></div>
									<span>Uploading files...</span>
								</div>
							)}
							{uploadProgress.status === 'success' && (
								<div className="flex items-center gap-2 text-green-600">
									<Check className="h-4 w-4" />
									<span>Upload successful!</span>
								</div>
							)}
							{uploadProgress.status === 'error' && (
								<div className="flex items-center gap-2 text-red-600">
									<AlertCircle className="h-4 w-4" />
									<span>{uploadProgress.error}</span>
								</div>
							)}
						</div>
					)}
				</div>

				<DialogFooter>
					<Button 
						variant="outline" 
						onClick={() => onOpenChange(false)}
						disabled={uploadProgress.status === 'uploading'}
					>
						Cancel
					</Button>
					<Button 
						onClick={handleUpload}
						disabled={!canUpload}
						className="min-w-[100px]"
					>
						{uploadProgress.status === 'uploading' ? (
							<div className="flex items-center gap-2">
								<div className="animate-spin h-4 w-4 border-2 border-white border-t-transparent rounded-full"></div>
								Uploading...
							</div>
						) : (
							'Upload'
						)}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}