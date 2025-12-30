import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Upload, Mic, Settings, LogOut, Home, Plus, Grip, Zap, Youtube, Video, Users, MonitorSpeaker } from "lucide-react";
import { ScriberrLogo } from "./ScriberrLogo";
import { ThemeSwitcher } from "./ThemeSwitcher";
import { AudioRecorder } from "./AudioRecorder";
import { SystemAudioRecorder } from "./SystemAudioRecorder";
import { QuickTranscriptionDialog } from "@/features/transcription/components/QuickTranscriptionDialog";
import { YouTubeDownloadDialog } from "@/features/transcription/components/YouTubeDownloadDialog";
import { useNavigate } from "react-router-dom";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { isVideoFile, isAudioFile } from "../utils/fileProcessor";
import { useGlobalUpload } from "@/contexts/GlobalUploadContext";

interface FileWithType {
	file: File;
	isVideo: boolean;
}

interface HeaderProps {
	onFileSelect?: (files: File | File[] | FileWithType | FileWithType[]) => void;
	onMultiTrackClick?: () => void;
	onDownloadComplete?: () => void;
}

export function Header({ onFileSelect, onMultiTrackClick, onDownloadComplete }: HeaderProps) {
	const navigate = useNavigate();
	const { logout } = useAuth();
	const fileInputRef = useRef<HTMLInputElement>(null);
	const videoFileInputRef = useRef<HTMLInputElement>(null);
	const [isRecorderOpen, setIsRecorderOpen] = useState(false);
	const [isSystemRecorderOpen, setIsSystemRecorderOpen] = useState(false);
	const [isQuickTranscriptionOpen, setIsQuickTranscriptionOpen] = useState(false);
	const [isYouTubeDialogOpen, setIsYouTubeDialogOpen] = useState(false);

	// Use global upload context as fallback when props are not provided
	const globalUpload = useGlobalUpload();

	// Determine which handlers to use (prop or global context)
	const effectiveFileSelect = onFileSelect ?? globalUpload.handleFileSelect;
	const effectiveMultiTrackClick = onMultiTrackClick ?? globalUpload.openMultiTrackDialog;
	const effectiveRecordingComplete = globalUpload.handleRecordingComplete;

	const handleUploadClick = () => {
		fileInputRef.current?.click();
	};

	const handleVideoUploadClick = () => {
		videoFileInputRef.current?.click();
	};

	const handleRecordClick = () => {
		setIsRecorderOpen(true);
	};

	const handleSystemRecordClick = () => {
		setIsSystemRecorderOpen(true);
	};

	const handleQuickTranscriptionClick = () => {
		setIsQuickTranscriptionOpen(true);
	};

	const handleYouTubeClick = () => {
		setIsYouTubeDialogOpen(true);
	};

	const handleMultiTrackClick = () => {
		effectiveMultiTrackClick();
	};

	const handleSettingsClick = () => {
		navigate("/settings");
	};

	const handleLogout = () => {
		logout();
	};

	const handleHomeClick = () => {
		navigate("/");
	};

	const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
		const files = event.target.files;
		if (files && files.length > 0) {
			const fileArray = Array.from(files);

			// Check for video files that were incorrectly uploaded via audio upload
			const videoFiles = fileArray.filter(file => isVideoFile(file));
			if (videoFiles.length > 0) {
				alert(`Video files detected. Please use "Upload Videos" instead of "Upload Files" to upload ${videoFiles.map(f => f.name).join(', ')}`);
				event.target.value = "";
				return;
			}

			// Filter to only audio files
			const audioFiles = fileArray.filter(file => isAudioFile(file));
			if (audioFiles.length > 0) {
				effectiveFileSelect(audioFiles.length === 1 ? audioFiles[0] : audioFiles);
				// Reset the input so the same files can be selected again
				event.target.value = "";
			} else {
				// No valid audio files found
				alert("No valid audio files found. Please select audio files (.mp3, .wav, .flac, .m4a, .aac, .ogg)");
				event.target.value = "";
			}
		}
	};

	const handleVideoFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
		const files = event.target.files;
		if (files && files.length > 0) {
			// Filter to only video files
			const videoFiles = Array.from(files).filter(file => file.type.startsWith("video/"));
			if (videoFiles.length > 0) {
				// Pass video files with type marker
				const filesWithType: FileWithType[] = videoFiles.map(file => ({ file, isVideo: true }));
				effectiveFileSelect(filesWithType.length === 1 ? filesWithType[0] : filesWithType);
				// Reset the input so the same files can be selected again
				event.target.value = "";
			}
		}
	};

	const handleRecordingComplete = async (blob: Blob, title: string) => {
		// Use global recording complete handler
		await effectiveRecordingComplete(blob, title);
	};


	return (
		<header className="sticky top-4 sm:top-6 z-50 glass rounded-[var(--radius-card)] px-4 py-3 sm:px-6 sm:py-4 transition-all duration-500 shadow-[var(--shadow-float)] border border-[var(--border-subtle)]">
			<div className="flex items-center justify-between">
				{/* Left side - Logo navigates home */}
				<ScriberrLogo onClick={handleHomeClick} />

				{/* Right side - Plus (Add Audio), Grip Menu, Theme Switcher */}
				<div className="flex items-center gap-2 sm:gap-3">
					{/* Add Audio (icon-only) */}
					<DropdownMenu>
						<DropdownMenuTrigger asChild>
							<Button
								variant="default"
								size="icon"
								className="bg-gradient-to-br from-[#FFAB40] to-[#FF3D00] text-white shadow-[0_4px_12px_rgba(255,61,0,0.4)] hover:shadow-[0_6px_16px_rgba(255,61,0,0.5)] border-none h-8 w-8 sm:h-10 sm:w-10 rounded-lg transition-all hover:scale-105 active:scale-95 cursor-pointer"
							>
								<Plus className="h-5 w-5 sm:h-6 sm:w-6" />
								<span className="sr-only">Add audio</span>
							</Button>
						</DropdownMenuTrigger>
						<DropdownMenuContent
							align="end"
							className="w-64 glass-card p-2 rounded-[var(--radius-card)] shadow-[var(--shadow-float)] border-[var(--border-subtle)]"
						>
							<DropdownMenuItem
								onClick={handleQuickTranscriptionClick}
								className="group flex items-center gap-3 px-3 py-3 cursor-pointer rounded-[var(--radius-btn)] focus:bg-[var(--brand-light)] focus:text-[var(--brand-solid)] transition-colors"
							>
								<div className="p-2 bg-amber-500/10 rounded-[var(--radius-btn)] text-amber-600 group-focus:text-[var(--brand-solid)]">
									<Zap className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">Quick Transcribe</div>
									<div className="text-xs text-[var(--text-secondary)]">
										Fast transcribe without saving
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleYouTubeClick}
								className="group flex items-center gap-3 px-3 py-3 cursor-pointer rounded-[var(--radius-btn)] focus:bg-[var(--brand-light)] focus:text-[var(--brand-solid)] transition-colors"
							>
								<div className="p-2 bg-rose-500/10 rounded-[var(--radius-btn)] text-rose-600 group-focus:text-[var(--brand-solid)]">
									<Youtube className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">YouTube URL</div>
									<div className="text-xs text-[var(--text-secondary)]">
										Download audio from YouTube
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleUploadClick}
								className="group flex items-center gap-3 px-3 py-3 cursor-pointer rounded-[var(--radius-btn)] focus:bg-[var(--brand-light)] focus:text-[var(--brand-solid)] transition-colors"
							>
								<div className="p-2 bg-[var(--brand-light)] rounded-[var(--radius-btn)] text-[var(--brand-solid)] group-focus:text-[var(--brand-solid)]">
									<Upload className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">Upload Files</div>
									<div className="text-xs text-[var(--text-secondary)]">
										Choose one or more audio files
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleVideoUploadClick}
								className="group flex items-center gap-3 px-3 py-3 cursor-pointer rounded-[var(--radius-btn)] focus:bg-[var(--brand-light)] focus:text-[var(--brand-solid)] transition-colors"
							>
								<div className="p-2 bg-purple-500/10 rounded-[var(--radius-btn)] text-purple-600 group-focus:text-[var(--brand-solid)]">
									<Video className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">Upload Videos</div>
									<div className="text-xs text-[var(--text-secondary)]">
										Extract audio from video files
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleRecordClick}
								className="group flex items-center gap-3 px-3 py-3 cursor-pointer rounded-[var(--radius-btn)] focus:bg-[var(--brand-light)] focus:text-[var(--brand-solid)] transition-colors"
							>
								<div className="p-2 bg-emerald-500/10 rounded-[var(--radius-btn)] text-emerald-600 group-focus:text-[var(--brand-solid)]">
									<Mic className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">Record Audio</div>
									<div className="text-xs text-[var(--text-secondary)]">
										Record using microphone
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleSystemRecordClick}
								className="group flex items-center gap-3 px-3 py-3 cursor-pointer rounded-[var(--radius-btn)] focus:bg-[var(--brand-light)] focus:text-[var(--brand-solid)] transition-colors"
							>
								<div className="p-2 bg-blue-500/10 rounded-[var(--radius-btn)] text-blue-600 group-focus:text-[var(--brand-solid)]">
									<MonitorSpeaker className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">Record System Audio</div>
									<div className="text-xs text-[var(--text-secondary)]">
										Capture screen + microphone
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleMultiTrackClick}
								className="group flex items-center gap-3 px-3 py-3 cursor-pointer rounded-[var(--radius-btn)] focus:bg-[var(--brand-light)] focus:text-[var(--brand-solid)] transition-colors"
							>
								<div className="p-2 bg-indigo-500/10 rounded-[var(--radius-btn)] text-indigo-600 group-focus:text-[var(--brand-solid)]">
									<Users className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">Multi-Track Audio</div>
									<div className="text-xs text-[var(--text-secondary)]">
										Upload multiple speaker tracks
									</div>
								</div>
							</DropdownMenuItem>
						</DropdownMenuContent>
					</DropdownMenu>

					{/* Main Menu (Grip) */}
					<DropdownMenu>
						<DropdownMenuTrigger asChild>
							<Button
								variant="ghost"
								size="icon"
								className="h-8 w-8 sm:h-10 sm:w-10 hover:bg-[var(--secondary)] rounded-[var(--radius-btn)] cursor-pointer text-[var(--text-secondary)]"
							>
								<Grip className="h-4 w-4 sm:h-5 sm:w-5" />
								<span className="sr-only">Open menu</span>
							</Button>
						</DropdownMenuTrigger>
						<DropdownMenuContent align="end" className="w-48 glass-card border-[var(--border-subtle)] p-2 rounded-[var(--radius-card)] shadow-[var(--shadow-float)]">
							<DropdownMenuItem onClick={handleHomeClick} className="cursor-pointer rounded-[var(--radius-btn)] focus:bg-[var(--secondary)] py-2.5">
								<Home className="h-4 w-4 mr-2" />
								Home
							</DropdownMenuItem>
							<DropdownMenuItem onClick={handleSettingsClick} className="cursor-pointer rounded-[var(--radius-btn)] focus:bg-[var(--secondary)] py-2.5">
								<Settings className="h-4 w-4 mr-2" />
								Settings
							</DropdownMenuItem>
							<DropdownMenuItem onClick={handleLogout} className="cursor-pointer rounded-[var(--radius-btn)] focus:bg-[var(--error)]/10 text-[var(--error)] py-2.5">
								<LogOut className="h-4 w-4 mr-2" />
								Logout
							</DropdownMenuItem>
						</DropdownMenuContent>
					</DropdownMenu>

					{/* Theme Switcher (icon-only) */}
					<ThemeSwitcher />

					{/* Hidden file input */}
					<input
						ref={fileInputRef}
						type="file"
						accept="audio/*"
						multiple
						onChange={handleFileChange}
						className="hidden"
					/>

					{/* Hidden video file input */}
					<input
						ref={videoFileInputRef}
						type="file"
						accept="video/*"
						multiple
						onChange={handleVideoFileChange}
						className="hidden"
					/>
				</div>
			</div>

			{/* Audio Recorder Dialog */}
			<AudioRecorder
				isOpen={isRecorderOpen}
				onClose={() => setIsRecorderOpen(false)}
				onRecordingComplete={handleRecordingComplete}
			/>

			{/* System Audio Recorder Dialog */}
			<SystemAudioRecorder
				isOpen={isSystemRecorderOpen}
				onClose={() => setIsSystemRecorderOpen(false)}
				onRecordingComplete={effectiveRecordingComplete}
			/>

			{/* Quick Transcription Dialog */}
			<QuickTranscriptionDialog
				isOpen={isQuickTranscriptionOpen}
				onClose={() => setIsQuickTranscriptionOpen(false)}
			/>

			{/* YouTube Download Dialog */}
			<YouTubeDownloadDialog
				isOpen={isYouTubeDialogOpen}
				onClose={() => setIsYouTubeDialogOpen(false)}
				onDownloadComplete={onDownloadComplete}
			/>

		</header>
	);
}
