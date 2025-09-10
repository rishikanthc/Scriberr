import { FileAudio, Video, Users, Upload, AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";

interface DragDropOverlayProps {
	isDragging: boolean;
	dragCount: number;
	fileType?: 'single' | 'video' | 'multitrack' | 'invalid';
	fileDescription?: string;
	errorMessage?: string;
}

export function DragDropOverlay({
	isDragging,
	dragCount,
	fileType = 'single',
	fileDescription,
	errorMessage
}: DragDropOverlayProps) {
	if (!isDragging || dragCount === 0) {
		return null;
	}

	const getIcon = () => {
		if (errorMessage) {
			return <AlertCircle className="h-16 w-16 text-red-400" />;
		}
		
		switch (fileType) {
			case 'multitrack':
				return <Users className="h-16 w-16 text-blue-400" />;
			case 'video':
				return <Video className="h-16 w-16 text-purple-400" />;
			case 'single':
				return <FileAudio className="h-16 w-16 text-green-400" />;
			default:
				return <Upload className="h-16 w-16 text-gray-400" />;
		}
	};

	const getTitle = () => {
		if (errorMessage) {
			return "Invalid Files";
		}
		
		switch (fileType) {
			case 'multitrack':
				return "Multi-Track Audio Project";
			case 'video':
				return "Video Files";
			case 'single':
				return "Audio Files";
			default:
				return "Drop Files Here";
		}
	};

	const getDescription = () => {
		if (errorMessage) {
			return errorMessage;
		}
		
		return fileDescription || "Release to upload";
	};

	const getBorderColor = () => {
		if (errorMessage) {
			return "border-red-400";
		}
		
		switch (fileType) {
			case 'multitrack':
				return "border-blue-400";
			case 'video':
				return "border-purple-400";
			case 'single':
				return "border-green-400";
			default:
				return "border-gray-400";
		}
	};

	const getBackgroundColor = () => {
		if (errorMessage) {
			return "bg-red-50 dark:bg-red-950/20";
		}
		
		switch (fileType) {
			case 'multitrack':
				return "bg-blue-50 dark:bg-blue-950/20";
			case 'video':
				return "bg-purple-50 dark:bg-purple-950/20";
			case 'single':
				return "bg-green-50 dark:bg-green-950/20";
			default:
				return "bg-gray-50 dark:bg-gray-950/20";
		}
	};

	return (
		<div className="fixed inset-0 z-50 flex items-center justify-center pointer-events-none">
			{/* Backdrop */}
			<div className="absolute inset-0 bg-black/20 dark:bg-black/40 backdrop-blur-sm" />
			
			{/* Drop Zone */}
			<div className={cn(
				"relative flex flex-col items-center justify-center",
				"w-96 h-64 mx-4",
				"border-4 border-dashed rounded-2xl",
				"transition-all duration-300 ease-out",
				"animate-pulse",
				getBorderColor(),
				getBackgroundColor()
			)}>
				{/* Icon */}
				<div className="mb-4 transform transition-transform duration-300 scale-110">
					{getIcon()}
				</div>
				
				{/* Title */}
				<h2 className={cn(
					"text-2xl font-bold mb-2 text-center",
					errorMessage ? "text-red-700 dark:text-red-300" : "text-gray-900 dark:text-gray-100"
				)}>
					{getTitle()}
				</h2>
				
				{/* Description */}
				<p className={cn(
					"text-center px-4 leading-relaxed",
					errorMessage ? "text-red-600 dark:text-red-400" : "text-gray-600 dark:text-gray-400"
				)}>
					{getDescription()}
				</p>
				
				{/* Multi-track badge */}
				{fileType === 'multitrack' && !errorMessage && (
					<div className="mt-3 px-3 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded-full text-sm font-medium">
						Auto-detected multi-track project
					</div>
				)}
			</div>
			
			{/* Animated border effect */}
			<div className={cn(
				"absolute inset-8",
				"border-2 border-dashed rounded-3xl",
				"opacity-30",
				"animate-ping",
				getBorderColor()
			)} />
		</div>
	);
}