import { useState, useCallback, useEffect } from "react";
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
import { Upload, X, FileAudio, File, AlertCircle } from "lucide-react";

interface MultiTrackUploadDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onMultiTrackUpload?: (files: File[], aupFile: File, title: string) => Promise<void>;
	prePopulatedFiles?: File[];
	prePopulatedAupFile?: File;
	prePopulatedTitle?: string;
}

interface FileWithPreview {
	file: File;
	id: string;
	isApu: boolean;
}


export function MultiTrackUploadDialog({
	open,
	onOpenChange,
	onMultiTrackUpload,
	prePopulatedFiles,
	prePopulatedAupFile,
	prePopulatedTitle,
}: MultiTrackUploadDialogProps) {
	const [title, setTitle] = useState("");
	const [files, setFiles] = useState<FileWithPreview[]>([]);

	// Effect to populate dialog with pre-populated data from drag-and-drop
	useEffect(() => {
		if (open && prePopulatedFiles && prePopulatedAupFile) {
			// Set title
			if (prePopulatedTitle) {
				setTitle(prePopulatedTitle);
			}

			// Prepare all files (audio + aup)
			const allFiles = [...prePopulatedFiles, prePopulatedAupFile];
			const fileItems: FileWithPreview[] = allFiles.map(file => ({
				file,
				id: Math.random().toString(36),
				isApu: file.name.toLowerCase().endsWith('.aup')
			}));

			setFiles(fileItems);
		} else if (!open) {
			// Reset when dialog closes
			setTitle("");
			setFiles([]);
		}
	}, [open, prePopulatedFiles, prePopulatedAupFile, prePopulatedTitle]);

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

	const canUpload = title.trim() !== "" && hasAupFile && hasAudioFiles;

	const handleUpload = async () => {
		if (!canUpload) return;

		// Extract audio files and aup file
		const trackFiles = audioFiles.map(fileItem => fileItem.file);
		const aupFileToUpload = aupFiles[0].file;

		// Call the callback with the files and title
		await onMultiTrackUpload?.(trackFiles, aupFileToUpload, title.trim());

		// Reset form and close dialog (Note: this may not be called if callback handles closing)
		setTitle("");
		setFiles([]);
		onOpenChange(false);
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
						{prePopulatedFiles && prePopulatedAupFile
							? `Auto-detected multi-track project with ${prePopulatedFiles.length} audio tracks. Review and upload when ready.`
							: "Upload multiple audio tracks with an .aup Audacity project file for multi-speaker transcription."
						}
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
						/>
					</div>

					{/* File Upload Zone */}
					<div className="space-y-4">
						<Label>Files</Label>

						{/* Drop Zone */}
						<div
							className="border-2 border-dashed rounded-[var(--radius-card)] p-8 text-center transition-all duration-200 border-[var(--border-subtle)] hover:border-[var(--brand-solid)] hover:bg-[var(--bg-card)] group cursor-pointer"
							onDrop={handleDrop}
							onDragOver={(e) => e.preventDefault()}
							onClick={() => document.getElementById('file-upload')?.click()}
						>
							<div className="mx-auto w-12 h-12 rounded-full bg-[var(--bg-card)] flex items-center justify-center mb-4 border border-[var(--border-subtle)] group-hover:border-[var(--brand-solid)] transition-colors">
								<Upload className="h-6 w-6 text-[var(--text-tertiary)] group-hover:text-[var(--brand-solid)] transition-colors" />
							</div>
							<div className="space-y-2">
								<p className="text-lg font-medium text-[var(--text-primary)]">Drop files here or click to upload</p>
								<p className="text-sm text-[var(--text-secondary)]">
									Upload multiple audio files and one .aup Audacity project file
								</p>
								<input
									type="file"
									multiple
									accept="audio/*,.aup"
									onChange={handleFileSelect}
									className="hidden"
									id="file-upload"
								/>
								<Button className="mt-4 bg-[var(--brand-solid)] text-white hover:opacity-90 rounded-[var(--radius-btn)]">
									Choose Files
								</Button>
							</div>
						</div>

						{/* File List */}
						{files.length > 0 && (
							<div className="space-y-2">
								<h4 className="font-medium text-sm">Uploaded Files:</h4>

								{/* AUP Files */}
								{aupFiles.map(fileItem => (
									<div key={fileItem.id} className="flex items-center gap-3 p-3 bg-[var(--success-translucent)] border border-[var(--success-solid)]/20 rounded-[var(--radius-input)]">
										<File className="h-4 w-4 text-[var(--success-solid)] flex-shrink-0" />
										<div className="flex-1 min-w-0">
											<p className="font-medium text-[var(--text-primary)] truncate">
												{fileItem.file.name}
											</p>
											<p className="text-xs text-[var(--success-solid)]">Audacity project file</p>
										</div>
										<Button
											variant="ghost"
											size="sm"
											onClick={() => removeFile(fileItem.id)}
											className="hover:bg-[var(--success-solid)]/10 text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
										>
											<X className="h-4 w-4" />
										</Button>
									</div>
								))}

								{/* Audio Files */}
								{audioFiles.map(fileItem => (
									<div key={fileItem.id} className="flex items-center gap-3 p-3 bg-[var(--warning-translucent)] border border-[var(--warning-solid)]/20 rounded-[var(--radius-input)]">
										<FileAudio className="h-4 w-4 text-[var(--warning-solid)] flex-shrink-0" />
										<div className="flex-1 min-w-0">
											<p className="font-medium text-[var(--text-primary)] truncate">
												{fileItem.file.name}
											</p>
											<p className="text-xs text-[var(--warning-solid)]">
												Speaker: {getSpeakerName(fileItem.file.name)}
											</p>
										</div>
										<Button
											variant="ghost"
											size="sm"
											onClick={() => removeFile(fileItem.id)}
											className="hover:bg-[var(--warning-solid)]/10 text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
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
									<div className="flex items-center gap-2 text-[var(--error)]">
										<AlertCircle className="h-4 w-4" />
										<span>An .aup Audacity project file is required</span>
									</div>
								)}
								{!hasAudioFiles && (
									<div className="flex items-center gap-2 text-[var(--error)]">
										<AlertCircle className="h-4 w-4" />
										<span>At least one audio file is required</span>
									</div>
								)}
								{!title.trim() && (
									<div className="flex items-center gap-2 text-[var(--error)]">
										<AlertCircle className="h-4 w-4" />
										<span>Title is required</span>
									</div>
								)}
							</div>
						)}
					</div>

				</div>

				<DialogFooter>
					<Button
						variant="outline"
						onClick={() => onOpenChange(false)}
					>
						Cancel
					</Button>
					<Button
						onClick={handleUpload}
						disabled={!canUpload}
					>
						Upload
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}