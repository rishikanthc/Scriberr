import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Upload, Mic, Settings, LogOut, Home, Plus, Grip, Zap } from "lucide-react";
import { ScriberrLogo } from "./ScriberrLogo";
import { ThemeSwitcher } from "./ThemeSwitcher";
import { AudioRecorder } from "./AudioRecorder";
import { QuickTranscriptionDialog } from "./QuickTranscriptionDialog";
import { useRouter } from "../contexts/RouterContext";
import { useAuth } from "../contexts/AuthContext";

interface HeaderProps {
	onFileSelect: (file: File) => void;
}

export function Header({ onFileSelect }: HeaderProps) {
	const { navigate } = useRouter();
	const { logout } = useAuth();
	const fileInputRef = useRef<HTMLInputElement>(null);
	const [isRecorderOpen, setIsRecorderOpen] = useState(false);
	const [isQuickTranscriptionOpen, setIsQuickTranscriptionOpen] = useState(false);

	const handleUploadClick = () => {
		fileInputRef.current?.click();
	};

	const handleRecordClick = () => {
		setIsRecorderOpen(true);
	};

	const handleQuickTranscriptionClick = () => {
		setIsQuickTranscriptionOpen(true);
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
		const file = event.target.files?.[0];
		if (file && file.type.startsWith("audio/")) {
			onFileSelect(file);
			// Reset the input so the same file can be selected again
			event.target.value = "";
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
								onClick={handleUploadClick}
								className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
							>
								<Upload className="h-4 w-4 text-blue-500" />
								<div>
									<div className="font-medium">Upload File</div>
									<div className="text-xs text-gray-500 dark:text-gray-400">
										Choose audio from device
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
						onChange={handleFileChange}
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
		</header>
	);
}
