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
import { MultiTrackUploadDialog } from "./MultiTrackUploadDialog";
import { useRouter } from "../contexts/RouterContext";
import { useAuth } from "../contexts/AuthContext";

interface FileWithType {
	file: File;
	isVideo: boolean;
}

interface HeaderProps {
	onFileSelect: (files: File | File[] | FileWithType | FileWithType[]) => void;
	onMultiTrackUpload?: (files: File[], aupFile: File, title: string) => void;
	onDownloadComplete?: () => void;
}

export function Header({ onFileSelect, onMultiTrackUpload, onDownloadComplete }: HeaderProps) {
	const { navigate } = useRouter();
	const { logout } = useAuth();
	const fileInputRef = useRef<HTMLInputElement>(null);
	const videoFileInputRef = useRef<HTMLInputElement>(null);
	const [isRecorderOpen, setIsRecorderOpen] = useState(false);
	const [isQuickTranscriptionOpen, setIsQuickTranscriptionOpen] = useState(false);
	const [isYouTubeDialogOpen, setIsYouTubeDialogOpen] = useState(false);
	const [isMultiTrackDialogOpen, setIsMultiTrackDialogOpen] = useState(false);

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
		setIsMultiTrackDialogOpen(true);
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
			// Filter to only audio files
			const audioFiles = Array.from(files).filter(file => file.type.startsWith("audio/"));
			if (audioFiles.length > 0) {
				onFileSelect(audioFiles.length === 1 ? audioFiles[0] : audioFiles);
				// Reset the input so the same files can be selected again
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
		<header className="bg-white dark:bg-gray-800 rounded-xl p-4 sm:p-6 mb-4 sm:mb-6">
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
								className="bg-blue-500 hover:bg-blue-600 text-white h-9 w-9 sm:h-10 sm:w-10 cursor-pointer"
							>
								<Plus className="h-4 w-4" />
								<span className="sr-only">Add audio</span>
							</Button>
						</DropdownMenuTrigger>
						<DropdownMenuContent
							align="end"
							className="w-48 bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700 shadow-lg"
						>
							<DropdownMenuItem
								onClick={handleQuickTranscriptionClick}
								className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
							>
								<Zap className="h-4 w-4 text-yellow-500" />
								<div>
									<div className="font-medium">Quick Transcribe</div>
									<div className="text-xs text-gray-500 dark:text-gray-400">
										Fast transcribe without saving
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleYouTubeClick}
								className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
							>
								<Youtube className="h-4 w-4 text-red-500" />
								<div>
									<div className="font-medium">YouTube URL</div>
									<div className="text-xs text-gray-500 dark:text-gray-400">
										Download audio from YouTube
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleUploadClick}
								className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
							>
								<Upload className="h-4 w-4 text-blue-500" />
								<div>
									<div className="font-medium">Upload Files</div>
									<div className="text-xs text-gray-500 dark:text-gray-400">
										Choose one or more audio files
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleVideoUploadClick}
								className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
							>
								<Video className="h-4 w-4 text-purple-500" />
								<div>
									<div className="font-medium">Upload Videos</div>
									<div className="text-xs text-gray-500 dark:text-gray-400">
										Extract audio from video files
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleRecordClick}
								className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
							>
								<Mic className="h-4 w-4 text-red-500" />
								<div>
									<div className="font-medium">Record Audio</div>
									<div className="text-xs text-gray-500 dark:text-gray-400">
										Record using microphone
									</div>
								</div>
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={handleMultiTrackClick}
								className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
							>
								<Users className="h-4 w-4 text-green-500" />
								<div>
									<div className="font-medium">Multi-Track Audio</div>
									<div className="text-xs text-gray-500 dark:text-gray-400">
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
								className="h-9 w-9 sm:h-10 sm:w-10 cursor-pointer"
							>
								<Grip className="h-5 w-5" />
								<span className="sr-only">Open menu</span>
							</Button>
						</DropdownMenuTrigger>
						<DropdownMenuContent align="end" className="w-44 bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700">
							<DropdownMenuItem onClick={handleHomeClick} className="cursor-pointer">
								<Home className="h-4 w-4" />
								Home
							</DropdownMenuItem>
							<DropdownMenuItem onClick={handleSettingsClick} className="cursor-pointer">
								<Settings className="h-4 w-4" />
								Settings
							</DropdownMenuItem>
							<DropdownMenuItem onClick={handleLogout} className="cursor-pointer" variant="destructive">
								<LogOut className="h-4 w-4" />
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

			{/* Multi-Track Upload Dialog */}
			<MultiTrackUploadDialog
				open={isMultiTrackDialogOpen}
				onOpenChange={setIsMultiTrackDialogOpen}
				onMultiTrackUpload={onMultiTrackUpload}
			/>
		</header>
	);
}
