import { useState, useEffect, useRef } from "react";
import WaveSurfer from "wavesurfer.js";
import RecordPlugin from "wavesurfer.js/dist/plugins/record.js";
import {
	Mic,
	Square,
	Play,
	Pause,
	Upload,
	Loader2,
	ChevronDown,
	Settings,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface AudioRecorderProps {
	isOpen: boolean;
	onClose: () => void;
	onRecordingComplete: (blob: Blob, title: string) => void;
}

export function AudioRecorder({
	isOpen,
	onClose,
	onRecordingComplete,
}: AudioRecorderProps) {
	const [wavesurfer, setWavesurfer] = useState<WaveSurfer | null>(null);
	const [record, setRecord] = useState<RecordPlugin | null>(null);
	const [isRecording, setIsRecording] = useState(false);
	const [isPaused, setIsPaused] = useState(false);
	const [recordingTime, setRecordingTime] = useState(0);
	const [title, setTitle] = useState("");
	const [availableDevices, setAvailableDevices] = useState<MediaDeviceInfo[]>(
		[],
	);
	const [selectedDevice, setSelectedDevice] = useState("");
	const [recordedBlob, setRecordedBlob] = useState<Blob | null>(null);
	const [isUploading, setIsUploading] = useState(false);

	const micContainerRef = useRef<HTMLDivElement>(null);

	// Initialize WaveSurfer and RecordPlugin when dialog opens
	useEffect(() => {
		if (!isOpen) return;

		const initializeWaveSurfer = () => {
			if (!micContainerRef.current) {
				return;
			}

			// Destroy previous instance
			if (wavesurfer) {
				wavesurfer.destroy();
				setWavesurfer(null);
				setRecord(null);
			}

			try {

				// Create new WaveSurfer instance
				const ws = WaveSurfer.create({
					container: micContainerRef.current,
					waveColor: "rgb(168, 85, 247)", // purple-500
					progressColor: "rgb(147, 51, 234)", // purple-600
					height: 80,
					normalize: true,
					interact: false,
				});


				// Set WaveSurfer first
				setWavesurfer(ws);


				// Initialize Record plugin in a separate try-catch
				try {
					const recordPlugin = ws.registerPlugin(
						RecordPlugin.create({
							renderRecordedAudio: false,
							scrollingWaveform: true,
							continuousWaveform: true,
							continuousWaveformDuration: 30,
						}),
					);


					// Handle recording end and progress events
					recordPlugin.on("record-end", (blob: Blob) => {
						setRecordedBlob(blob);
						setIsRecording(false);
						setIsPaused(false);
					});

					// Handle recording progress
					recordPlugin.on("record-progress", (time: number) => {
						setRecordingTime(time);
					});

					setRecord(recordPlugin);
				} catch (recordError) {
					console.error("Failed to create RecordPlugin:", recordError);
					// At least WaveSurfer is working, so we can show that
				}
			} catch (error) {
				console.error("Failed to initialize WaveSurfer:", error);
				if (error instanceof Error) {
					console.error("Error details:", {
						name: error.name,
						message: error.message,
						stack: error.stack,
					});
				}
			}
		};

		// Use setTimeout to ensure the DOM element is ready
		const timeoutId = setTimeout(initializeWaveSurfer, 100);

		// Get available audio devices
		RecordPlugin.getAvailableAudioDevices()
			.then((devices) => {
				setAvailableDevices(devices);
				if (devices.length > 0 && !selectedDevice) {
					setSelectedDevice(devices[0].deviceId);
				}
			})
			.catch((error) => {
				console.error("Failed to get audio devices:", error);
			});

		return () => {
			clearTimeout(timeoutId);
			if (wavesurfer) {
				wavesurfer.destroy();
			}
		};
	}, [isOpen]); // Remove wavesurfer and selectedDevice dependencies to avoid recreation loop

	// Handle background recording - prevent page unload warnings during recording
	useEffect(() => {
		const originalTitle = document.title;

		const handleBeforeUnload = (e: BeforeUnloadEvent) => {
			if (isRecording) {
				e.preventDefault();
				e.returnValue =
					"Recording in progress. Are you sure you want to leave?";
				return e.returnValue;
			}
		};

		const handleVisibilityChange = () => {
			// Continue recording even when tab is not visible
		};

		if (isRecording) {
			// Update page title to show recording status
			document.title = `🔴 Recording... - ${originalTitle}`;
			window.addEventListener("beforeunload", handleBeforeUnload);
			document.addEventListener("visibilitychange", handleVisibilityChange);
		} else {
			// Restore original title
			document.title = originalTitle;
		}

		return () => {
			document.title = originalTitle;
			window.removeEventListener("beforeunload", handleBeforeUnload);
			document.removeEventListener("visibilitychange", handleVisibilityChange);
		};
	}, [isRecording]);

	// Start recording
	const startRecording = async () => {
		if (!record) {
			alert("Recorder not initialized. Please close and reopen the dialog.");
			return;
		}

		try {
			await record.startRecording({ deviceId: selectedDevice });
			setIsRecording(true);
			setIsPaused(false);
			setRecordingTime(0);
			setRecordedBlob(null);
		} catch (error) {
			console.error("Failed to start recording:", error);
			alert(
				"Failed to start recording. Please check microphone permissions and try again.",
			);
		}
	};

	// Stop recording
	const stopRecording = () => {
		if (!record) return;
		record.stopRecording();
	};

	// Pause/Resume recording
	const togglePauseRecording = () => {
		if (!record) return;

		if (isPaused) {
			record.resumeRecording();
			setIsPaused(false);
		} else {
			record.pauseRecording();
			setIsPaused(true);
		}
	};

	// Format time in mm:ss
	const formatTime = (timeMs: number) => {
		const minutes = Math.floor(timeMs / 60000);
		const seconds = Math.floor((timeMs % 60000) / 1000);
		return `${minutes.toString().padStart(2, "0")}:${seconds.toString().padStart(2, "0")}`;
	};

	// Handle upload
	const handleUpload = async () => {
		if (!recordedBlob) return;

		setIsUploading(true);
		try {
			await onRecordingComplete(
				recordedBlob,
				title || `Recording ${new Date().toISOString()}`,
			);
			// Reset state
			setRecordedBlob(null);
			setTitle("");
			setRecordingTime(0);
			onClose();
		} catch (error) {
			console.error("Failed to upload recording:", error);
			alert("Failed to upload recording");
		} finally {
			setIsUploading(false);
		}
	};

	// Handle dialog close
	const handleClose = () => {
		if (isRecording) {
			stopRecording();
		}
		setRecordedBlob(null);
		setTitle("");
		setRecordingTime(0);
		setIsRecording(false);
		setIsPaused(false);
		onClose();
	};

	return (
		<Dialog open={isOpen} onOpenChange={handleClose}>
			<DialogContent className="sm:max-w-[600px] bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
				<DialogHeader>
					<DialogTitle className="text-gray-900 dark:text-gray-100 text-xl font-semibold">
						Record Audio
					</DialogTitle>
					<DialogDescription className="text-gray-600 dark:text-gray-400">
						Record audio directly from your microphone and upload it for
						transcription.
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-6 py-4">
					{/* Title Input */}
					<div className="space-y-2">
						<label className="text-sm font-medium text-gray-700 dark:text-gray-300">
							Recording Title (Optional)
						</label>
						<Input
							value={title}
							onChange={(e) => setTitle(e.target.value)}
							placeholder="Enter a title for your recording..."
							className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
							disabled={isRecording}
						/>
					</div>

					{/* Microphone Selection */}
					{availableDevices.length > 1 && (
						<div className="space-y-2">
							<label className="text-sm font-medium text-gray-700 dark:text-gray-300">
								Microphone
							</label>
							<DropdownMenu>
								<DropdownMenuTrigger asChild disabled={isRecording}>
									<Button
										variant="outline"
										className="w-full justify-between bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700"
									>
										<div className="flex items-center gap-2">
											<Settings className="h-4 w-4" />
											<span className="truncate">
												{availableDevices.find(
													(d) => d.deviceId === selectedDevice,
												)?.label || `Microphone ${selectedDevice.slice(0, 8)}`}
											</span>
										</div>
										<ChevronDown className="h-4 w-4 opacity-50" />
									</Button>
								</DropdownMenuTrigger>
								<DropdownMenuContent className="w-full min-w-[400px] bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700">
									{availableDevices.map((device) => (
										<DropdownMenuItem
											key={device.deviceId}
											onClick={() => setSelectedDevice(device.deviceId)}
											className="flex items-center gap-3 px-3 py-2 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700"
										>
											<Mic className="h-4 w-4 text-gray-500" />
											<div className="flex-1">
												<div className="text-sm font-medium text-gray-900 dark:text-gray-100">
													{device.label ||
														`Microphone ${device.deviceId.slice(0, 8)}`}
												</div>
												<div className="text-xs text-gray-500 dark:text-gray-400">
													Device ID: {device.deviceId.slice(0, 20)}...
												</div>
											</div>
											{selectedDevice === device.deviceId && (
												<div className="h-2 w-2 bg-blue-500 rounded-full"></div>
											)}
										</DropdownMenuItem>
									))}
								</DropdownMenuContent>
							</DropdownMenu>
						</div>
					)}

					{/* Recording Time */}
					<div className="text-center">
						<div className="text-3xl font-mono font-bold text-gray-900 dark:text-gray-100 mb-2">
							{formatTime(recordingTime)}
						</div>
						<div className="flex items-center justify-center gap-2 text-sm text-gray-600 dark:text-gray-400">
							{isRecording && !isPaused && (
								<div className="h-2 w-2 bg-red-500 rounded-full animate-pulse"></div>
							)}
							<span>
								{isRecording
									? isPaused
										? "Recording Paused"
										: "Recording..."
									: "Ready to Record"}
							</span>
						</div>
						{isRecording && (
							<div className="text-xs text-blue-600 dark:text-blue-400 mt-1">
								Recording continues even if you switch tabs
							</div>
						)}
					</div>

					{/* Waveform Container */}
					<div className="relative">
						<div
							ref={micContainerRef}
							className="w-full rounded-lg p-4 bg-gray-50 dark:bg-gray-800/50 min-h-[120px]"
						/>
						{!isRecording && !recordedBlob && (
							<div className="absolute inset-0 flex items-center justify-center pointer-events-none">
								<div className="text-gray-400 dark:text-gray-500 text-sm text-center">
									<Mic className="h-8 w-8 mx-auto mb-2 opacity-50" />
									<div>Waveform will appear here during recording</div>
									{!wavesurfer && (
										<div className="text-xs text-red-400 mt-1">
											Initializing recorder...
										</div>
									)}
									{wavesurfer && !record && (
										<div className="text-xs text-yellow-400 mt-1">
											Recorder plugin loading...
										</div>
									)}
									{wavesurfer && record && (
										<div className="text-xs text-green-400 mt-1">
											Ready to record
										</div>
									)}
								</div>
							</div>
						)}
					</div>

					{/* Recording Controls */}
					<div className="flex justify-center gap-4">
						{!isRecording && !recordedBlob && (
							<Button
								onClick={startRecording}
								size="lg"
								className="bg-red-500 hover:bg-red-600 text-white px-8 py-3 rounded-xl font-medium transition-all duration-300 hover:scale-105"
							>
								<Mic className="h-5 w-5 mr-2" />
								Start Recording
							</Button>
						)}

						{isRecording && (
							<>
								<Button
									onClick={togglePauseRecording}
									size="lg"
									variant="outline"
									className="border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700 px-6 py-3 rounded-xl"
								>
									{isPaused ? (
										<>
											<Play className="h-5 w-5 mr-2" />
											Resume
										</>
									) : (
										<>
											<Pause className="h-5 w-5 mr-2" />
											Pause
										</>
									)}
								</Button>
								<Button
									onClick={stopRecording}
									size="lg"
									className="bg-gray-600 hover:bg-gray-700 text-white px-6 py-3 rounded-xl"
								>
									<Square className="h-5 w-5 mr-2" />
									Stop
								</Button>
							</>
						)}

						{recordedBlob && (
							<Button
								onClick={handleUpload}
								size="lg"
								disabled={isUploading}
								className="bg-blue-500 hover:bg-blue-600 text-white px-8 py-3 rounded-xl font-medium transition-all duration-300 hover:scale-105"
							>
								{isUploading ? (
									<>
										<Loader2 className="h-5 w-5 mr-2 animate-spin" />
										Uploading...
									</>
								) : (
									<>
										<Upload className="h-5 w-5 mr-2" />
										Upload Recording
									</>
								)}
							</Button>
						)}
					</div>

					{recordedBlob && (
						<div className="text-center text-sm text-green-600 dark:text-green-400">
							✓ Recording completed! Review and upload when ready.
						</div>
					)}
				</div>
			</DialogContent>
		</Dialog>
	);
}
