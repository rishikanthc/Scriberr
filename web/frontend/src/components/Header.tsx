import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Upload, Mic, Settings, LogOut, Home, Plus, Grip, Zap, Youtube, Video, Users } from "lucide-react";
import { ScriberrLogo } from "./ScriberrLogo";
import { ThemeSwitcher } from "./ThemeSwitcher";
import { AudioRecorder } from "./AudioRecorder";
import { QuickTranscriptionDialog } from "./QuickTranscriptionDialog";
import { YouTubeDownloadDialog } from "./YouTubeDownloadDialog";
import { useRouter } from "../contexts/RouterContext";
import { useAuth } from "../contexts/AuthContext";
import { isVideoFile, isAudioFile } from "../utils/fileProcessor";

interface FileWithType {
	file: File;
	isVideo: boolean;
}

interface HeaderProps {
	onFileSelect: (files: File | File[] | FileWithType | FileWithType[]) => void;
	onMultiTrackClick?: () => void;
	onDownloadComplete?: () => void;
}

export function Header({ onFileSelect, onMultiTrackClick, onDownloadComplete }: HeaderProps) {
	const { navigate } = useRouter();
	const { logout } = useAuth();
	const fileInputRef = useRef<HTMLInputElement>(null);
	const videoFileInputRef = useRef<HTMLInputElement>(null);
	const [isRecorderOpen, setIsRecorderOpen] = useState(false);
	const [isQuickTranscriptionOpen, setIsQuickTranscriptionOpen] = useState(false);
	const [isYouTubeDialogOpen, setIsYouTubeDialogOpen] = useState(false);

	const handleUploadClick = () => {
		fileInputRef.current?.click();
	};

	const handleVideoUploadClick = () => {
		videoFileInputRef.current?.click();
	};

	const handleRecordClick = () => {
		setIsRecorderOpen(true);
	};

	const handleQuickTranscriptionClick = () => {
		setIsQuickTranscriptionOpen(true);
	};

	const handleYouTubeClick = () => {
		setIsYouTubeDialogOpen(true);
	};

	const handleMultiTrackClick = () => {
		onMultiTrackClick?.();
	};

	const handleSettingsClick = () => {
		navigate({ path: "settings" });
	};

	const handleLogout = () => {
		logout();
	};

	const handleHomeClick = () => {
		navigate({ path: "home" });
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
				onFileSelect(audioFiles.length === 1 ? audioFiles[0] : audioFiles);
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
				onFileSelect(filesWithType.length === 1 ? filesWithType[0] : filesWithType);
				// Reset the input so the same files can be selected again
				event.target.value = "";
			}
		}
	};

	const handleRecordingComplete = async (blob: Blob, title: string) => {
		// Convert blob to file and use existing upload logic
		const file = new File([blob], `${title}.webm`, { type: blob.type });
		onFileSelect(file);
	};

	return (
		<header className="sticky top-4 z-50 glass rounded-2xl p-4 mb-6 transition-all duration-300 shadow-sm hover:shadow-md">
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
								className="bg-carbon-900 hover:bg-carbon-800 dark:bg-carbon-100 dark:hover:bg-carbon-200 text-white dark:text-carbon-900 h-10 w-10 rounded-xl shadow-sm transition-all hover:scale-105 cursor-pointer"
							>
								<Plus className="h-5 w-5" />
								<span className="sr-only">Add audio</span>
							</Button>
						</DropdownMenuTrigger>
						<DropdownMenuContent
							align="end"
							className="w-56 glass-card border-carbon-200 dark:border-carbon-800 p-2"
						>
							<DropdownMenuItem
								onClick={handleQuickTranscriptionClick}
								className="flex items-center gap-3 px-3 py-2.5 cursor-pointer rounded-lg focus:bg-carbon-100 dark:focus:bg-carbon-900"
							>
								<div className="p-2 bg-amber-100 dark:bg-amber-900/30 rounded-lg text-amber-700 dark:text-amber-400">
									<Zap className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">Quick Transcribe</div>
									<div className="text-xs text-muted-foreground">
										Fast transcribe without saving
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleYouTubeClick}
								className="flex items-center gap-3 px-3 py-2.5 cursor-pointer rounded-lg focus:bg-carbon-100 dark:focus:bg-carbon-900"
							>
								<div className="p-2 bg-rose-100 dark:bg-rose-900/30 rounded-lg text-rose-600 dark:text-rose-400">
									<Youtube className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">YouTube URL</div>
									<div className="text-xs text-muted-foreground">
										Download audio from YouTube
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleUploadClick}
								className="flex items-center gap-3 px-3 py-2.5 cursor-pointer rounded-lg focus:bg-carbon-100 dark:focus:bg-carbon-900"
							>
								<div className="p-2 bg-carbon-100 dark:bg-carbon-900/30 rounded-lg text-carbon-600 dark:text-carbon-400">
									<Upload className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">Upload Files</div>
									<div className="text-xs text-muted-foreground">
										Choose one or more audio files
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleVideoUploadClick}
								className="flex items-center gap-3 px-3 py-2.5 cursor-pointer rounded-lg focus:bg-carbon-100 dark:focus:bg-carbon-900"
							>
								<div className="p-2 bg-carbon-100 dark:bg-carbon-800 rounded-lg text-carbon-600 dark:text-carbon-400">
									<Video className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">Upload Videos</div>
									<div className="text-xs text-muted-foreground">
										Extract audio from video files
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleRecordClick}
								className="flex items-center gap-3 px-3 py-2.5 cursor-pointer rounded-lg focus:bg-carbon-100 dark:focus:bg-carbon-900"
							>
								<div className="p-2 bg-orange-100 dark:bg-orange-900/30 rounded-lg text-orange-600 dark:text-orange-400">
									<Mic className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">Record Audio</div>
									<div className="text-xs text-muted-foreground">
										Record using microphone
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleMultiTrackClick}
								className="flex items-center gap-3 px-3 py-2.5 cursor-pointer rounded-lg focus:bg-carbon-100 dark:focus:bg-carbon-900"
							>
								<div className="p-2 bg-teal-100 dark:bg-teal-900/30 rounded-lg text-teal-600 dark:text-teal-400">
									<Users className="h-4 w-4" />
								</div>
								<div>
									<div className="font-medium text-sm">Multi-Track Audio</div>
									<div className="text-xs text-muted-foreground">
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
								variant="outline"
								size="icon"
								className="h-10 w-10 hover:bg-carbon-100 dark:hover:bg-carbon-800 cursor-pointer"
							>
								<Grip className="h-5 w-5 text-muted-foreground" />
								<span className="sr-only">Open menu</span>
							</Button>
						</DropdownMenuTrigger>
						<DropdownMenuContent align="end" className="w-48 glass-card border-carbon-200 dark:border-carbon-800 p-2">
							<DropdownMenuItem onClick={handleHomeClick} className="cursor-pointer rounded-lg focus:bg-carbon-100 dark:focus:bg-carbon-900">
								<Home className="h-4 w-4 mr-2" />
								Home
							</DropdownMenuItem>
							<DropdownMenuItem onClick={handleSettingsClick} className="cursor-pointer rounded-lg focus:bg-carbon-100 dark:focus:bg-carbon-900">
								<Settings className="h-4 w-4 mr-2" />
								Settings
							</DropdownMenuItem>
							<DropdownMenuItem onClick={handleLogout} className="cursor-pointer rounded-lg focus:bg-rose-50 dark:focus:bg-rose-600/20 text-rose-600 dark:text-rose-400">
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
