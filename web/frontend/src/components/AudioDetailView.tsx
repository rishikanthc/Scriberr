import { useState, useEffect, useRef } from "react";
import { ArrowLeft, Play, Pause, List, AlignLeft } from "lucide-react";
import WaveSurfer from "wavesurfer.js";
import { Button } from "./ui/button";
import { useRouter } from "../contexts/RouterContext";
import { useTheme } from "../contexts/ThemeContext";

interface AudioFile {
	id: string;
	title?: string;
	status: "uploaded" | "pending" | "processing" | "completed" | "failed";
	created_at: string;
	audio_path: string;
}

interface Transcript {
	text: string;
	segments?: Array<{
		start: number;
		end: number;
		text: string;
	}>;
}

interface AudioDetailViewProps {
	audioId: string;
}

export function AudioDetailView({ audioId }: AudioDetailViewProps) {
	const { navigate } = useRouter();
	const { theme } = useTheme();
	const [audioFile, setAudioFile] = useState<AudioFile | null>(null);
	const [transcript, setTranscript] = useState<Transcript | null>(null);
	const [loading, setLoading] = useState(true);
	const [isPlaying, setIsPlaying] = useState(false);
	const [transcriptMode, setTranscriptMode] = useState<"compact" | "expanded">(
		"compact",
	);
	const waveformRef = useRef<HTMLDivElement>(null);
	const wavesurferRef = useRef<WaveSurfer | null>(null);

	useEffect(() => {
		console.log("AudioDetailView mounted, audioId:", audioId);
		fetchAudioDetails();
	}, [audioId]);

	// Initialize WaveSurfer when audioFile is available - with proper DOM timing
	useEffect(() => {
		if (!audioFile) {
			console.log("No audioFile yet, skipping WaveSurfer initialization");
			return;
		}

		console.log("AudioFile available, checking DOM readiness...");

		// Use a longer timeout and multiple checks to ensure DOM is ready
		const checkAndInitialize = () => {
			console.log("Checking DOM and WaveSurfer state:", {
				audioFile: !!audioFile,
				waveformRefElement: !!waveformRef.current,
				wavesurferInstance: !!wavesurferRef.current,
				waveformRefType: waveformRef.current?.tagName,
			});

			if (waveformRef.current && !wavesurferRef.current) {
				console.log("✅ All conditions met! Initializing WaveSurfer...");
				initializeWaveSurfer();
			} else if (!waveformRef.current) {
				console.log("❌ Waveform DOM element not ready yet, will retry...");
				// Retry after a bit more time
				setTimeout(checkAndInitialize, 200);
			} else if (wavesurferRef.current) {
				console.log("WaveSurfer already initialized");
			}
		};

		// Initial check with a small delay
		const timer = setTimeout(checkAndInitialize, 50);

		return () => {
			clearTimeout(timer);
			if (wavesurferRef.current) {
				console.log("Cleaning up WaveSurfer");
				wavesurferRef.current.destroy();
				wavesurferRef.current = null;
			}
		};
	}, [audioFile]);

	const fetchAudioDetails = async () => {
		console.log("Fetching audio details for ID:", audioId);
		try {
			// Fetch audio file details
			const audioResponse = await fetch(`/api/v1/transcription/${audioId}`, {
				headers: {
					"X-API-Key": "dev-api-key-123",
				},
			});

			console.log("Audio response status:", audioResponse.status);

			if (audioResponse.ok) {
				const audioData = await audioResponse.json();
				console.log("Audio data received:", audioData);
				setAudioFile(audioData);

				// Fetch transcript if completed
				if (audioData.status === "completed") {
					const transcriptResponse = await fetch(
						`/api/v1/transcription/${audioId}/transcript`,
						{
							headers: {
								"X-API-Key": "dev-api-key-123",
							},
						},
					);

					if (transcriptResponse.ok) {
						const transcriptData = await transcriptResponse.json();
						// The API returns transcript data in a nested structure
						if (transcriptData.transcript) {
							// Check if transcript has segments or text
							if (typeof transcriptData.transcript === "string") {
								setTranscript({ text: transcriptData.transcript });
							} else if (transcriptData.transcript.text) {
								setTranscript({
									text: transcriptData.transcript.text,
									segments: transcriptData.transcript.segments,
								});
							} else if (transcriptData.transcript.segments) {
								// If only segments, combine them into text
								const fullText = transcriptData.transcript.segments
									.map((segment: any) => segment.text)
									.join(" ");
								setTranscript({
									text: fullText,
									segments: transcriptData.transcript.segments,
								});
							}
						}
					} else {
						console.error(
							"Failed to fetch transcript:",
							transcriptResponse.status,
						);
					}
				}
			} else {
				console.error("Failed to fetch audio details:", audioResponse.status);
			}
		} catch (error) {
			console.error("Failed to fetch audio details:", error);
		} finally {
			setLoading(false);
		}
	};

	const initializeWaveSurfer = async () => {
		if (!waveformRef.current || !audioFile) return;

		try {
			// First, try to load the audio file manually to check if it's accessible
			const audioUrl = `/api/v1/transcription/${audioId}/audio`;
			console.log("Testing audio URL:", audioUrl);

			const response = await fetch(audioUrl, {
				headers: {
					"X-API-Key": "dev-api-key-123",
				},
			});

			if (!response.ok) {
				console.error(
					"Audio file request failed:",
					response.status,
					response.statusText,
				);
				const errorText = await response.text();
				console.error("Error response body:", errorText);
				return;
			}

			console.log("Audio file accessible, creating WaveSurfer...");

			// Theme-aware colors
			const isDark = theme === "dark";
			const waveColor = isDark ? "#4b5563" : "#d1d5db"; // dark: gray-600, light: gray-300
			const progressColor = "#3b82f6"; // blue-500 for both themes

			// Create WaveSurfer instance
			wavesurferRef.current = WaveSurfer.create({
				container: waveformRef.current,
				waveColor,
				progressColor,
				barWidth: 2,
				barGap: 1,
				barRadius: 2,
				height: 80,
				normalize: true,
				backend: "WebAudio",
			});

			// Load the audio blob
			const audioBlob = await response.blob();
			const audioObjectURL = URL.createObjectURL(audioBlob);

			console.log("Loading audio blob into WaveSurfer...");
			await wavesurferRef.current.load(audioObjectURL);

			wavesurferRef.current.on("ready", () => {
				console.log("WaveSurfer is ready");
			});

			wavesurferRef.current.on("load", (url) => {
				console.log("WaveSurfer loading:", url);
			});

			wavesurferRef.current.on("error", (error) => {
				console.error("WaveSurfer error:", error);
			});

			wavesurferRef.current.on("play", () => {
				setIsPlaying(true);
			});

			wavesurferRef.current.on("pause", () => {
				setIsPlaying(false);
			});

			wavesurferRef.current.on("finish", () => {
				setIsPlaying(false);
			});
		} catch (error) {
			console.error("Failed to initialize WaveSurfer:", error);
		}
	};

	const togglePlayPause = () => {
		if (wavesurferRef.current) {
			wavesurferRef.current.playPause();
		}
	};

	const handleBack = () => {
		navigate({ path: "home" });
	};

	const getFileName = (audioPath: string) => {
		const parts = audioPath.split("/");
		return parts[parts.length - 1];
	};

	const formatTimestamp = (seconds: number): string => {
		const minutes = Math.floor(seconds / 60);
		const remainingSeconds = Math.floor(seconds % 60);
		return `${minutes}:${remainingSeconds.toString().padStart(2, "0")}`;
	};

	const formatDate = (dateString: string) => {
		return new Date(dateString).toLocaleDateString("en-US", {
			year: "numeric",
			month: "long",
			day: "numeric",
			hour: "2-digit",
			minute: "2-digit",
		});
	};

	if (loading) {
		return (
			<div className="min-h-screen bg-gray-50 dark:bg-gray-900">
				<div className="mx-auto px-8 py-6" style={{ width: "60vw" }}>
					<div className="bg-white dark:bg-gray-800 rounded-xl p-6">
						<div className="animate-pulse">
							<div className="h-6 bg-gray-200 dark:bg-gray-600 rounded w-1/4 mb-4"></div>
							<div className="h-4 bg-gray-200 dark:bg-gray-600 rounded w-1/2 mb-8"></div>
							<div className="h-20 bg-gray-200 dark:bg-gray-600 rounded mb-8"></div>
							<div className="space-y-3">
								{[...Array(5)].map((_, i) => (
									<div
										key={i}
										className="h-4 bg-gray-200 dark:bg-gray-600 rounded"
									></div>
								))}
							</div>
						</div>
					</div>
				</div>
			</div>
		);
	}

	if (!audioFile) {
		return (
			<div className="min-h-screen bg-gray-50 dark:bg-gray-900">
				<div className="mx-auto px-8 py-6" style={{ width: "60vw" }}>
					<div className="bg-white dark:bg-gray-800 rounded-xl p-6 text-center">
						<h1 className="text-xl font-semibold text-gray-900 dark:text-gray-50 mb-4">
							Audio file not found
						</h1>
						<Button onClick={handleBack} variant="outline">
							<ArrowLeft className="mr-2 h-4 w-4" />
							Back to Audio Files
						</Button>
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="min-h-screen bg-gray-50 dark:bg-gray-900">
			<div className="mx-auto px-8 py-6" style={{ width: "60vw" }}>
				{/* Header with back button */}
				<div className="mb-6">
					<Button onClick={handleBack} variant="outline" className="mb-4">
						<ArrowLeft className="mr-2 h-4 w-4" />
						Back to Audio Files
					</Button>
				</div>

				{/* Audio Player Section */}
				<div className="bg-white dark:bg-gray-800 rounded-xl p-6 mb-6">
					<div className="mb-6">
						<h1 className="text-2xl font-bold text-gray-900 dark:text-gray-50 mb-2">
							{audioFile.title || getFileName(audioFile.audio_path)}
						</h1>
						<p className="text-gray-600 dark:text-gray-400 text-sm">
							Added on {formatDate(audioFile.created_at)}
						</p>
					</div>

					{/* Audio Player Controls */}
					<div className="mb-6">
						<div className="flex items-center gap-4">
							{/* Circular Play/Pause Button */}
							<button
								onClick={togglePlayPause}
								className="w-16 h-16 rounded-full bg-blue-500 hover:bg-blue-600 text-white shadow-lg hover:shadow-xl transition-all duration-200 hover:scale-105 flex items-center justify-center group"
							>
								{isPlaying ? (
									<Pause className="h-6 w-6 group-hover:scale-110 transition-transform" />
								) : (
									<Play className="h-6 w-6 ml-0.5 group-hover:scale-110 transition-transform" />
								)}
							</button>

							{/* WaveSurfer Container */}
							<div className="flex-1">
								<div
									ref={waveformRef}
									className="w-full bg-gray-50 dark:bg-gray-700 rounded-lg p-4"
									style={{ minHeight: "80px" }}
								/>
							</div>
						</div>
					</div>
				</div>

				{/* Transcript Section */}
				{audioFile.status === "completed" && transcript && (
					<div className="bg-white dark:bg-gray-800 rounded-xl p-6">
						<div className="flex items-center justify-between mb-6">
							<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-50">
								Transcript
							</h2>

							{/* View Toggle Buttons */}
							<div className="flex items-center bg-gray-100 dark:bg-gray-600 rounded-lg p-1">
								<button
									onClick={() => setTranscriptMode("compact")}
									className={`px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-200 flex items-center gap-2 ${
										transcriptMode === "compact"
											? "bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 shadow-sm"
											: "text-gray-600 dark:text-gray-200 hover:text-gray-900 dark:hover:text-gray-200"
									}`}
								>
									<AlignLeft className="h-4 w-4" />
									<span className="hidden sm:inline">Compact</span>
								</button>
								<button
									onClick={() => setTranscriptMode("expanded")}
									className={`px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-200 flex items-center gap-2 ${
										transcriptMode === "expanded"
											? "bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 shadow-sm"
											: "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200"
									}`}
								>
									<List className="h-4 w-4" />
									<span className="hidden sm:inline">Timeline</span>
								</button>
							</div>
						</div>

						{/* Transcript Content with Smooth Transition */}
						<div className="relative overflow-hidden">
							<div
								className={`transition-all duration-300 ease-in-out ${
									transcriptMode === "compact"
										? "opacity-100 translate-y-0"
										: "opacity-0 -translate-y-4 absolute inset-0"
								}`}
							>
								{transcriptMode === "compact" && (
									<div className="prose prose-gray dark:prose-invert max-w-none">
										<p className="text-gray-700 dark:text-gray-300 leading-relaxed">
											{transcript.text}
										</p>
									</div>
								)}
							</div>

							<div
								className={`transition-all duration-300 ease-in-out ${
									transcriptMode === "expanded"
										? "opacity-100 translate-y-0"
										: "opacity-0 translate-y-4 absolute inset-0"
								}`}
							>
								{transcriptMode === "expanded" && transcript.segments && (
									<div className="space-y-4">
										{transcript.segments.map((segment, index) => (
											<div
												key={index}
												className="flex gap-4 p-3 rounded-lg bg-gray-50 dark:bg-gray-700 hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors duration-150"
											>
												<div className="flex-shrink-0">
													<span className="inline-block px-2 py-1 text-xs font-mono bg-blue-100 dark:bg-blue-700 text-blue-800 dark:text-blue-200 rounded">
														{formatTimestamp(segment.start)}
													</span>
												</div>
												<div className="flex-1">
													<p className="text-gray-700 dark:text-gray-200 leading-relaxed">
														{segment.text.trim()}
													</p>
												</div>
											</div>
										))}
									</div>
								)}
							</div>
						</div>
					</div>
				)}

				{/* Status Messages */}
				{audioFile.status !== "completed" && (
					<div className="bg-white dark:bg-gray-700 rounded-xl p-6">
						<div className="text-center">
							<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-50 mb-2">
								{audioFile.status === "processing" &&
									"Transcription in Progress"}
								{audioFile.status === "pending" && "Transcription Queued"}
								{audioFile.status === "uploaded" && "Ready for Transcription"}
								{audioFile.status === "failed" && "Transcription Failed"}
							</h2>
							<p className="text-gray-600 dark:text-gray-400">
								{audioFile.status === "processing" &&
									"Please wait while we process your audio file..."}
								{audioFile.status === "pending" &&
									"Your audio file is in the transcription queue."}
								{audioFile.status === "uploaded" &&
									"Start transcription from the audio files list."}
								{audioFile.status === "failed" &&
									"There was an error processing your audio file."}
							</p>
						</div>
					</div>
				)}
			</div>
		</div>
	);
}
