import { useState, useEffect, useRef } from "react";
import {
	MonitorSpeaker,
	Mic,
	Square,
	Upload,
	Loader2,
	ChevronDown,
	Settings,
	XCircle,
	AlertCircle,
	CheckCircle,
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
import { Card, CardContent } from "@/components/ui/card";
import { Slider } from "@/components/ui/slider";
import { useToast } from "@/components/ui/toast";

interface SystemAudioRecorderProps {
	isOpen: boolean;
	onClose: () => void;
	onRecordingComplete: (blob: Blob, title: string) => void;
}

export function SystemAudioRecorder({
	isOpen,
	onClose,
	onRecordingComplete,
}: SystemAudioRecorderProps) {
	// Recording state
	const [isRecording, setIsRecording] = useState(false);
	const [recordingTime, setRecordingTime] = useState(0);
	const [title, setTitle] = useState("");
	const [recordedBlob, setRecordedBlob] = useState<Blob | null>(null);
	const [isUploading, setIsUploading] = useState(false);
	const [mediaRecorder, setMediaRecorder] = useState<MediaRecorder | null>(null);
	const recordingChunksRef = useRef<Blob[]>([]);
	const timerIntervalRef = useRef<number | null>(null);

	// Audio Streams & Web Audio API
	const [systemStream, setSystemStream] = useState<MediaStream | null>(null);
	const [micStream, setMicStream] = useState<MediaStream | null>(null);
	const [audioContext, setAudioContext] = useState<AudioContext | null>(null);
	const [systemGainNode, setSystemGainNode] = useState<GainNode | null>(null);
	const [micGainNode, setMicGainNode] = useState<GainNode | null>(null);

	// Volume Controls
	const [systemVolume, setSystemVolume] = useState(100);
	const [micVolume, setMicVolume] = useState(100);

	// Device Selection
	const [availableDevices, setAvailableDevices] = useState<MediaDeviceInfo[]>([]);
	const [selectedDevice, setSelectedDevice] = useState("");

	// Error & Compatibility
	const [compatibilityError, setCompatibilityError] = useState<string | null>(null);
	const [permissionDenied, setPermissionDenied] = useState(false);
	const [micAvailable, setMicAvailable] = useState(true);

	const { toast } = useToast();

	// Browser compatibility check
	const checkCompatibility = (): { supported: boolean; error?: string } => {
		if (!navigator.mediaDevices?.getDisplayMedia) {
			return {
				supported: false,
				error:
					"Your browser doesn't support system audio capture. Please use Chrome, Edge, or Firefox.",
			};
		}
		return { supported: true };
	};

	// Initialize microphone device list when dialog opens
	useEffect(() => {
		if (!isOpen) return;

		let activeStream: MediaStream | null = null;

		const init = async () => {
			try {
				// Check browser compatibility
				const compatibility = checkCompatibility();
				if (!compatibility.supported) {
					setCompatibilityError(compatibility.error || null);
					return;
				}

				// Request permission to get device labels
				activeStream = await navigator.mediaDevices.getUserMedia({
					audio: true,
				});

				// Get available microphones
				const devices = await navigator.mediaDevices.enumerateDevices();
				const audioDevices = devices.filter((d) => d.kind === "audioinput");
				setAvailableDevices(audioDevices);

				// Set default device if none selected
				if (audioDevices.length > 0) {
					const deviceExists = audioDevices.some(
						(d) => d.deviceId === selectedDevice,
					);
					if (!selectedDevice || !deviceExists) {
						setSelectedDevice(audioDevices[0].deviceId);
					}
				}
			} catch (error) {
				console.error("Failed to enumerate devices:", error);
				toast({
					title: "Initialization Error",
					description: "Failed to get microphone devices. You can still record system audio only.",
				});
			} finally {
				// Stop the temporary stream used for permissions
				if (activeStream) {
					activeStream.getTracks().forEach((track) => track.stop());
				}
			}
		};

		init();
	}, [isOpen]); // eslint-disable-line react-hooks/exhaustive-deps

	// Simple timer that increments every second while recording
	useEffect(() => {
		if (isRecording) {
			timerIntervalRef.current = window.setInterval(() => {
				setRecordingTime((prev) => prev + 1000);
			}, 1000);
		} else {
			if (timerIntervalRef.current) {
				clearInterval(timerIntervalRef.current);
				timerIntervalRef.current = null;
			}
		}

		return () => {
			if (timerIntervalRef.current) {
				clearInterval(timerIntervalRef.current);
			}
		};
	}, [isRecording]);

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

		if (isRecording) {
			// Update page title to show recording status
			document.title = `🔴 Recording System Audio... - ${originalTitle}`;
			window.addEventListener("beforeunload", handleBeforeUnload);
		} else {
			// Restore original title
			document.title = originalTitle;
		}

		return () => {
			document.title = originalTitle;
			window.removeEventListener("beforeunload", handleBeforeUnload);
		};
	}, [isRecording]);

	// Create mixed audio stream using Web Audio API
	const createMixedAudioStream = (
		sysStream: MediaStream,
		mStream: MediaStream,
	): MediaStream => {
		try {
			// Create audio context
			const ctx = new AudioContext();
			setAudioContext(ctx);

			// Create source nodes from MediaStreams
			const systemSource = ctx.createMediaStreamSource(sysStream);
			const micSource = ctx.createMediaStreamSource(mStream);

			// Create gain nodes for volume control
			const systemGain = ctx.createGain();
			const micGain = ctx.createGain();

			// Set initial volumes
			systemGain.gain.value = systemVolume / 100;
			micGain.gain.value = micVolume / 100;

			// Store gain nodes for real-time control
			setSystemGainNode(systemGain);
			setMicGainNode(micGain);

			// Create destination for mixed output
			const destination = ctx.createMediaStreamDestination();

			// Connect: sources → gains → destination
			systemSource.connect(systemGain);
			micSource.connect(micGain);
			systemGain.connect(destination);
			micGain.connect(destination);

			return destination.stream;
		} catch (error) {
			console.error("Audio mixing failed:", error);
			toast({
				title: "Audio Mixing Unavailable",
				description: "Recording system audio only. Browser doesn't support mixing.",
			});
			// Fallback: return system stream only
			return sysStream;
		}
	};

	// Start recording
	const startRecording = async () => {
		try {
			setPermissionDenied(false);

			// Step 1: Request system audio via getDisplayMedia
			// Note: video is required by the API, we'll stop it immediately
			const displayStream = await navigator.mediaDevices.getDisplayMedia({
				video: true,
				audio: {
					echoCancellation: false,
					noiseSuppression: false,
					autoGainControl: false,
				},
			});

			// Stop the video track immediately since we only want audio
			const videoTrack = displayStream.getVideoTracks()[0];
			if (videoTrack) {
				videoTrack.stop();
				displayStream.removeTrack(videoTrack);
			}

			// Create a new MediaStream with only audio tracks
			const audioTracks = displayStream.getAudioTracks();
			if (audioTracks.length === 0) {
				throw new Error("NotFoundError");
			}
			const sysStream = new MediaStream(audioTracks);

			setSystemStream(sysStream);

			// Handle stream end (user stops sharing via browser UI)
			sysStream.getAudioTracks()[0].addEventListener("ended", () => {
				if (isRecording) {
					stopRecording();
					toast({
						title: "Screen Sharing Stopped",
						description: "Recording has been saved.",
					});
				}
			});

			// Step 2: Request microphone
			let mStream: MediaStream | null = null;
			try {
				mStream = await navigator.mediaDevices.getUserMedia({
					audio: {
						deviceId: selectedDevice ? { exact: selectedDevice } : undefined,
						echoCancellation: true,  // CRITICAL: Prevent mic from capturing system audio output
						noiseSuppression: true,   // Remove background noise
						autoGainControl: true,    // Normalize volume levels
					},
				});

				setMicStream(mStream);
				setMicAvailable(true);
			} catch (micError) {
				console.error("Microphone permission denied:", micError);
				toast({
					title: "Microphone Unavailable",
					description: "Recording system audio only.",
				});
				setMicAvailable(false);
			}

			// Step 3: Mix streams or use system-only
			let streamToRecord: MediaStream;
			if (mStream) {
				streamToRecord = createMixedAudioStream(sysStream, mStream);
			} else {
				streamToRecord = sysStream;
			}

			// Step 4: Create MediaRecorder directly
			const recorder = new MediaRecorder(streamToRecord);
			recordingChunksRef.current = [];

			recorder.ondataavailable = (e) => {
				if (e.data.size > 0) {
					recordingChunksRef.current.push(e.data);
				}
			};

			recorder.onstop = () => {
				const blob = new Blob(recordingChunksRef.current, {
					type: recordingChunksRef.current[0]?.type || 'audio/webm'
				});
				setRecordedBlob(blob);
				setIsRecording(false);
			};

			recorder.start(1000); // Capture in 1-second chunks
			setMediaRecorder(recorder);

			setIsRecording(true);
			setRecordingTime(0);
			setRecordedBlob(null);
		} catch (error) {
			console.error("Failed to start recording:", error);

			// Handle specific errors
			if (error instanceof Error && error.name === "NotAllowedError") {
				setPermissionDenied(true);
			} else if (error instanceof Error && error.name === "NotFoundError") {
				alert(
					"The selected source doesn't support audio sharing. Please choose a tab or window with audio.",
				);
			} else {
				alert("Failed to start screen sharing. Please try again.");
			}

			// Cleanup if failed
			cleanupStreams();
		}
	};

	// Stop recording
	const stopRecording = () => {
		if (mediaRecorder && mediaRecorder.state !== 'inactive') {
			mediaRecorder.stop();
		}
		cleanupStreams();
	};

	// Update system volume in real-time
	const updateSystemVolume = (value: number[]) => {
		const vol = value[0];
		setSystemVolume(vol);
		if (systemGainNode && isRecording) {
			systemGainNode.gain.value = vol / 100;
		}
	};

	// Update microphone volume in real-time
	const updateMicVolume = (value: number[]) => {
		const vol = value[0];
		setMicVolume(vol);
		if (micGainNode && isRecording) {
			micGainNode.gain.value = vol / 100;
		}
	};

	// Cleanup streams and audio context
	const cleanupStreams = () => {
		if (systemStream) {
			systemStream.getTracks().forEach((track) => track.stop());
			setSystemStream(null);
		}
		if (micStream) {
			micStream.getTracks().forEach((track) => track.stop());
			setMicStream(null);
		}
		if (audioContext && audioContext.state !== "closed") {
			audioContext.close();
			setAudioContext(null);
		}
		setSystemGainNode(null);
		setMicGainNode(null);
		setMediaRecorder(null);
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
				title || `System Recording ${new Date().toISOString()}`,
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
		cleanupStreams();
		setRecordedBlob(null);
		setTitle("");
		setRecordingTime(0);
		setIsRecording(false);
		setPermissionDenied(false);
		setCompatibilityError(null);
		onClose();
	};

	// Render browser compatibility error
	if (compatibilityError) {
		return (
			<Dialog open={isOpen} onOpenChange={handleClose}>
				<DialogContent className="sm:max-w-[600px] bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700">
					<DialogHeader>
						<DialogTitle className="flex items-center gap-2 text-carbon-900 dark:text-carbon-100">
							<MonitorSpeaker className="h-5 w-5 text-cyan-600" />
							Record System Audio
						</DialogTitle>
					</DialogHeader>

					<Card className="bg-red-500/10 border-red-200 dark:border-red-800">
						<CardContent className="pt-6">
							<div className="flex gap-4">
								<XCircle className="h-6 w-6 text-red-600 flex-shrink-0" />
								<div>
									<h3 className="font-semibold mb-2 text-carbon-900 dark:text-carbon-100">
										Browser Not Supported
									</h3>
									<p className="text-sm mb-3 text-carbon-700 dark:text-carbon-300">
										{compatibilityError}
									</p>
									<p className="text-xs text-carbon-600 dark:text-carbon-400">
										You can use "Record Audio" for microphone-only recording.
									</p>
								</div>
							</div>
						</CardContent>
					</Card>

					<div className="flex justify-end">
						<Button variant="outline" onClick={handleClose}>
							Close
						</Button>
					</div>
				</DialogContent>
			</Dialog>
		);
	}

	// Render permission denied error
	if (permissionDenied && !isRecording) {
		return (
			<Dialog open={isOpen} onOpenChange={handleClose}>
				<DialogContent className="sm:max-w-[600px] bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700">
					<DialogHeader>
						<DialogTitle className="flex items-center gap-2 text-carbon-900 dark:text-carbon-100">
							<MonitorSpeaker className="h-5 w-5 text-cyan-600" />
							Record System Audio
						</DialogTitle>
					</DialogHeader>

					<Card className="bg-amber-500/10 border-amber-200 dark:border-amber-800">
						<CardContent className="pt-6">
							<div className="flex gap-4">
								<AlertCircle className="h-6 w-6 text-amber-600 flex-shrink-0" />
								<div>
									<h3 className="font-semibold mb-2 text-carbon-900 dark:text-carbon-100">
										Screen Sharing Permission Required
									</h3>
									<p className="text-sm mb-3 text-carbon-700 dark:text-carbon-300">
										You denied screen sharing permission. Please click "Try Again"
										and allow access when prompted.
									</p>
									<p className="text-xs font-medium text-amber-700 dark:text-amber-600">
										Make sure to check "Share system audio" or "Share tab audio"
										in the browser picker!
									</p>
								</div>
							</div>
						</CardContent>
					</Card>

					<div className="flex justify-end gap-3">
						<Button variant="outline" onClick={handleClose}>
							Cancel
						</Button>
						<Button
							onClick={() => {
								setPermissionDenied(false);
								startRecording();
							}}
							className="bg-cyan-500 hover:bg-cyan-600 text-white"
						>
							Try Again
						</Button>
					</div>
				</DialogContent>
			</Dialog>
		);
	}

	// Render recording complete state
	if (recordedBlob && !isRecording) {
		return (
			<Dialog open={isOpen} onOpenChange={handleClose}>
				<DialogContent className="sm:max-w-[600px] bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700">
					<DialogHeader>
						<DialogTitle className="flex items-center gap-2 text-carbon-900 dark:text-carbon-100">
							<MonitorSpeaker className="h-5 w-5 text-cyan-600" />
							Recording Complete
						</DialogTitle>
					</DialogHeader>

					<div className="space-y-6 py-4">
						{/* Success Message */}
						<div className="flex items-center gap-3 p-4 bg-green-500/10 rounded-lg border border-green-200 dark:border-green-800">
							<CheckCircle className="h-5 w-5 text-green-600 flex-shrink-0" />
							<div>
								<h3 className="font-semibold text-carbon-900 dark:text-carbon-100">
									Recording Complete!
								</h3>
								<p className="text-sm text-carbon-600 dark:text-carbon-400">
									Duration: {formatTime(recordingTime)}
								</p>
							</div>
						</div>

						{/* Title Input */}
						<div className="space-y-2">
							<label className="text-sm font-medium text-carbon-700 dark:text-carbon-300">
								Recording Title
							</label>
							<Input
								value={title}
								onChange={(e) => setTitle(e.target.value)}
								placeholder="Enter a title for your recording..."
								className="bg-white dark:bg-carbon-800 border-carbon-300 dark:border-carbon-600 text-carbon-900 dark:text-carbon-100"
							/>
						</div>

						{/* Upload Button */}
						<Button
							onClick={handleUpload}
							disabled={isUploading}
							className="w-full bg-cyan-500 hover:bg-cyan-600 text-white px-8 py-3 rounded-xl font-medium transition-all duration-300 hover:scale-105"
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
					</div>
				</DialogContent>
			</Dialog>
		);
	}

	// Render active recording state
	if (isRecording) {
		return (
			<Dialog open={isOpen} onOpenChange={handleClose}>
				<DialogContent className="sm:max-w-[700px] bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700">
					<DialogHeader>
						<DialogTitle className="flex items-center gap-2 text-carbon-900 dark:text-carbon-100">
							<MonitorSpeaker className="h-5 w-5 text-cyan-600" />
							Recording System Audio
						</DialogTitle>
					</DialogHeader>

					<div className="space-y-6 py-4">
						{/* Recording Status Banner */}
						<div className="flex items-center gap-3 p-4 bg-cyan-500/10 rounded-lg border border-cyan-200 dark:border-cyan-800">
							<div className="h-3 w-3 bg-red-500 rounded-full animate-pulse flex-shrink-0" />
							<div>
								<h3 className="font-semibold text-cyan-900 dark:text-cyan-100">
									Recording System Audio{micAvailable ? " + Microphone" : " Only"}
								</h3>
								<p className="text-xs text-cyan-700 dark:text-cyan-300">
									Recording continues even if you switch tabs
								</p>
							</div>
						</div>

						{/* Recording Time */}
						<div className="text-center">
							<div className="text-6xl font-mono font-bold text-carbon-900 dark:text-carbon-100 mb-2">
								{formatTime(recordingTime)}
							</div>
							<div className="flex items-center justify-center gap-2 text-sm text-carbon-600 dark:text-carbon-400">
								<div className="h-2 w-2 bg-red-500 rounded-full animate-pulse" />
								<span>Recording...</span>
							</div>
						</div>

						{/* Volume Controls */}
						{micAvailable && (
							<div className="grid grid-cols-2 gap-4">
								<div className="space-y-2">
									<div className="flex items-center gap-2">
										<MonitorSpeaker className="h-4 w-4 text-cyan-600" />
										<label className="text-sm font-medium text-carbon-700 dark:text-carbon-300">
											System Audio
										</label>
									</div>
									<Slider
										min={0}
										max={100}
										step={1}
										value={[systemVolume]}
										onValueChange={updateSystemVolume}
										className="cursor-pointer"
									/>
									<span className="text-xs text-carbon-500 dark:text-carbon-400">
										{systemVolume}%
									</span>
								</div>
								<div className="space-y-2">
									<div className="flex items-center gap-2">
										<Mic className="h-4 w-4 text-cyan-600" />
										<label className="text-sm font-medium text-carbon-700 dark:text-carbon-300">
											Microphone
										</label>
									</div>
									<Slider
										min={0}
										max={100}
										step={1}
										value={[micVolume]}
										onValueChange={updateMicVolume}
										className="cursor-pointer"
									/>
									<span className="text-xs text-carbon-500 dark:text-carbon-400">
										{micVolume}%
									</span>
								</div>
							</div>
						)}

						{/* Recording Controls */}
						<div className="flex justify-center">
							<Button
								onClick={stopRecording}
								size="lg"
								className="bg-carbon-600 hover:bg-carbon-700 text-white px-8 py-3 rounded-xl"
							>
								<Square className="h-5 w-5 mr-2" />
								Stop Recording
							</Button>
						</div>
					</div>
				</DialogContent>
			</Dialog>
		);
	}

	// Render initial instructions state
	return (
		<Dialog open={isOpen} onOpenChange={handleClose}>
			<DialogContent className="sm:max-w-[700px] bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2 text-carbon-900 dark:text-carbon-100 text-xl font-bold">
						<MonitorSpeaker className="h-5 w-5 text-cyan-600" />
						Record System Audio
					</DialogTitle>
					<DialogDescription className="text-carbon-600 dark:text-carbon-400">
						Capture system audio from your screen/tab along with your microphone
						for meeting recordings.
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-6 py-4">
					{/* Instructions Card */}
					<Card className="bg-cyan-500/5 border-cyan-200 dark:border-cyan-800">
						<CardContent className="pt-6">
							<h3 className="font-semibold mb-3 text-cyan-900 dark:text-cyan-100">
								How it works:
							</h3>
							<ol className="space-y-3 text-sm text-carbon-700 dark:text-carbon-300">
								<li className="flex gap-3">
									<span className="font-bold text-cyan-600 flex-shrink-0">
										1.
									</span>
									<span>Click "Start Recording" below</span>
								</li>
								<li className="flex gap-3">
									<span className="font-bold text-cyan-600 flex-shrink-0">
										2.
									</span>
									<div className="flex-1">
										<div className="mb-2">
											<strong className="text-amber-700 dark:text-amber-500">Chrome/Edge:</strong> Select "Chrome Tab", choose your tab, and check "Share tab audio"
										</div>
										<div>
											<strong className="text-amber-700 dark:text-amber-500">Firefox:</strong> Select "Application Window", choose the window with audio (e.g., browser window), and check "Share system audio"
										</div>
									</div>
								</li>
								<li className="flex gap-3">
									<span className="font-bold text-cyan-600 flex-shrink-0">
										3.
									</span>
									<span>Allow microphone access when prompted (optional)</span>
								</li>
							</ol>
							<div className="mt-4 p-3 bg-amber-500/10 rounded-lg border border-amber-200 dark:border-amber-800">
								<p className="text-xs text-amber-800 dark:text-amber-300">
									<strong>💡 Tip:</strong> Use headphones to prevent echo and ensure the best recording quality!
								</p>
							</div>
						</CardContent>
					</Card>

					{/* Title Input */}
					<div className="space-y-2">
						<label className="text-sm font-medium text-carbon-700 dark:text-carbon-300">
							Recording Title (Optional)
						</label>
						<Input
							value={title}
							onChange={(e) => setTitle(e.target.value)}
							placeholder="Enter a title for your recording..."
							className="bg-white dark:bg-carbon-800 border-carbon-300 dark:border-carbon-600 text-carbon-900 dark:text-carbon-100"
							disabled={isRecording}
						/>
					</div>

					{/* Microphone Selection */}
					{availableDevices.length > 1 && (
						<div className="space-y-2">
							<label className="text-sm font-medium text-carbon-700 dark:text-carbon-300">
								Microphone
							</label>
							<DropdownMenu>
								<DropdownMenuTrigger asChild disabled={isRecording}>
									<Button
										variant="outline"
										className="w-full justify-between bg-white dark:bg-carbon-800 border-carbon-300 dark:border-carbon-600 hover:bg-carbon-50 dark:hover:bg-carbon-700"
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
								<DropdownMenuContent className="w-full min-w-[400px] bg-white dark:bg-carbon-900 border-carbon-200 dark:border-carbon-700">
									{availableDevices.map((device) => (
										<DropdownMenuItem
											key={device.deviceId}
											onClick={() => setSelectedDevice(device.deviceId)}
											className="flex items-center gap-3 px-3 py-2 cursor-pointer hover:bg-carbon-100 dark:hover:bg-carbon-700"
										>
											<Mic className="h-4 w-4 text-carbon-500" />
											<div className="flex-1">
												<div className="text-sm font-medium text-carbon-900 dark:text-carbon-100">
													{device.label ||
														`Microphone ${device.deviceId.slice(0, 8)}`}
												</div>
												<div className="text-xs text-carbon-500 dark:text-carbon-400">
													Device ID: {device.deviceId.slice(0, 20)}...
												</div>
											</div>
											{selectedDevice === device.deviceId && (
												<div className="h-2 w-2 bg-cyan-500 rounded-full"></div>
											)}
										</DropdownMenuItem>
									))}
								</DropdownMenuContent>
							</DropdownMenu>
						</div>
					)}

					{/* Start Button */}
					<Button
						onClick={startRecording}
						size="lg"
						className="w-full bg-cyan-500 hover:bg-cyan-600 text-white px-8 py-3 rounded-xl font-medium transition-all duration-300 hover:scale-105"
					>
						<MonitorSpeaker className="h-5 w-5 mr-2" />
						Start Recording
					</Button>
				</div>
			</DialogContent>
		</Dialog>
	);
}
