import { useState, useEffect, useRef, useCallback } from "react";
import WaveSurfer from "wavesurfer.js";
import RecordPlugin from "wavesurfer.js/dist/plugins/record.js";
import {
    Mic,
    Square,
    Home,
    Settings,
    ChevronDown,
    AlertCircle,
    Copy,
    Check
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ScriberrLogo } from "../components/ScriberrLogo";
import { ThemeSwitcher } from "../components/ThemeSwitcher";
import { useRouter } from "../contexts/RouterContext";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

interface TranscriptionSegment {
    start: number;
    end: number;
    text: string;
}

export function RealtimeTranscriptionPage() {
    const { navigate, currentRoute } = useRouter();
    const [isRecording, setIsRecording] = useState(false);
    const [isConnected, setIsConnected] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [transcript, setTranscript] = useState<TranscriptionSegment[]>([]);
    const [currentText, setCurrentText] = useState("");
    const [availableDevices, setAvailableDevices] = useState<MediaDeviceInfo[]>([]);
    const [selectedDevice, setSelectedDevice] = useState("");
    // Removed unused wavesurfer state
    const [recordPlugin, setRecordPlugin] = useState<RecordPlugin | null>(null);
    const [copied, setCopied] = useState(false);

    const micContainerRef = useRef<HTMLDivElement>(null);
    const websocketRef = useRef<WebSocket | null>(null);
    const transcriptEndRef = useRef<HTMLDivElement>(null);

    // Get params
    const model = currentRoute.params?.model || "base";
    const device = currentRoute.params?.device || "cpu";

    // Initialize WaveSurfer
    useEffect(() => {
        let activeStream: MediaStream | null = null;
        let ws: WaveSurfer | null = null;

        const init = async () => {
            try {
                // Request permission
                activeStream = await navigator.mediaDevices.getUserMedia({ audio: true });

                // Get devices
                const devices = await RecordPlugin.getAvailableAudioDevices();
                setAvailableDevices(devices);

                if (devices.length > 0 && !selectedDevice) {
                    setSelectedDevice(devices[0].deviceId);
                }

                if (!micContainerRef.current) return;

                ws = WaveSurfer.create({
                    container: micContainerRef.current,
                    waveColor: "rgb(168, 85, 247)", // purple-500
                    progressColor: "rgb(147, 51, 234)", // purple-600
                    height: 60,
                    normalize: true,
                    interact: false,
                    cursorWidth: 0,
                });

                // Removed setWavesurfer(ws);

                const record = ws.registerPlugin(
                    RecordPlugin.create({
                        renderRecordedAudio: false,
                        scrollingWaveform: true,
                        continuousWaveform: true,
                        continuousWaveformDuration: 30,
                        mediaRecorderTimeslice: 500, // Send chunks every 500ms
                    })
                );

                setRecordPlugin(record);

                // Handle audio data
                record.on("record-data-available", (blob: Blob) => {
                    if (websocketRef.current && websocketRef.current.readyState === WebSocket.OPEN) {
                        websocketRef.current.send(blob);
                    }
                });

            } catch (err) {
                console.error("Failed to initialize recorder:", err);
                setError("Failed to access microphone. Please check permissions.");
            } finally {
                if (activeStream) {
                    activeStream.getTracks().forEach(track => track.stop());
                }
            }
        };

        init();

        return () => {
            if (ws) ws.destroy();
            if (activeStream) activeStream.getTracks().forEach(track => track.stop());
        };
    }, []);

    // Auto-scroll to bottom of transcript
    useEffect(() => {
        transcriptEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, [transcript, currentText]);

    const connectWebSocket = useCallback(() => {
        try {
            // Use relative path to allow proxying through same origin
            const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
            const host = window.location.host;
            const wsUrl = `${protocol}//${host}/api/v1/realtime/asr`;

            const ws = new WebSocket(wsUrl);

            ws.onopen = () => {
                console.log("Connected to transcription server");
                setIsConnected(true);
                setError(null);

                // Send configuration
                ws.send(JSON.stringify({
                    model: model,
                    device: device
                }));
            };

            ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);

                    if (data.text) {
                        // Logic to handle partial vs final
                        // For now, we update currentText if it looks partial (no punctuation end)
                        // and move to transcript if it looks final.
                        // But without explicit flags from WLK, we'll stick to the previous simple logic:
                        // Update last segment if start time matches.

                        // We use setCurrentText to show what's being processed if we had partials.
                        // Since we don't have partials explicitly, we'll just use setTranscript.
                        // But to satisfy the linter and potential future use:
                        setCurrentText(""); // Reset current text as we are committing to transcript

                        setTranscript(prev => {
                            const last = prev[prev.length - 1];
                            if (last && Math.abs(last.start - data.start) < 0.1) {
                                // Update last segment
                                return [...prev.slice(0, -1), data];
                            } else {
                                // New segment
                                return [...prev, data];
                            }
                        });
                    }
                } catch (e) {
                    console.error("Error parsing message:", e);
                }
            };

            ws.onerror = (event) => {
                console.error("WebSocket error:", event);
                setError("Failed to connect to transcription server. Is the Docker container running?");
                setIsConnected(false);
                setIsRecording(false);
            };

            ws.onclose = () => {
                console.log("Disconnected from transcription server");
                setIsConnected(false);
                setIsRecording(false);
            };

            websocketRef.current = ws;
        } catch (err) {
            console.error("Connection error:", err);
            setError("Failed to create WebSocket connection.");
        }
    }, [model, device]);

    const startRecording = async () => {
        if (!recordPlugin) return;

        // Connect if not connected
        if (!isConnected) {
            connectWebSocket();
            // Wait for connection? 
            // The onopen handler will handle sending config.
            // But we should wait before starting recording?
            // Let's wait a bit or check state.
        }

        try {
            const constraints: MediaTrackConstraints = {
                deviceId: selectedDevice ? { exact: selectedDevice } : undefined,
                channelCount: 1,
                sampleRate: 16000, // Whisper prefers 16kHz
            };

            await recordPlugin.startRecording(constraints);
            setIsRecording(true);

            // If we just initiated connection, we might need to wait for it to be open before sending data.
            // The 'record-data' handler checks for OPEN state, so it's safe.

        } catch (err) {
            console.error("Failed to start recording:", err);
            setError("Failed to start microphone.");
        }
    };

    const stopRecording = () => {
        if (recordPlugin) {
            recordPlugin.stopRecording();
        }
        setIsRecording(false);

        // Close connection?
        // Maybe keep it open for a bit?
        // For now, let's close it to be safe and clean.
        if (websocketRef.current) {
            websocketRef.current.close();
        }
    };

    const handleHomeClick = () => {
        if (isRecording) {
            if (confirm("Recording in progress. Stop and leave?")) {
                stopRecording();
                navigate({ path: "home" });
            }
        } else {
            navigate({ path: "home" });
        }
    };

    const copyTranscript = () => {
        const text = transcript.map(t => t.text).join(" ");
        navigator.clipboard.writeText(text);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    return (
        <div className="min-h-screen bg-carbon-50 dark:bg-black text-carbon-900 dark:text-carbon-100 font-sans selection:bg-blue-500/30">
            {/* Header */}
            <header className="sticky top-0 z-50 glass border-b border-carbon-200 dark:border-carbon-800">
                <div className="container mx-auto px-4 h-16 flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={handleHomeClick}
                            className="hover:bg-carbon-100 dark:hover:bg-carbon-800"
                        >
                            <Home className="h-5 w-5" />
                        </Button>
                        <ScriberrLogo />
                        <div className="hidden md:flex items-center gap-2 px-3 py-1 rounded-full bg-carbon-100 dark:bg-carbon-800 text-xs font-medium">
                            <span className="text-carbon-500">Model:</span>
                            <span>{model}</span>
                            <span className="w-1 h-1 rounded-full bg-carbon-400"></span>
                            <span className="text-carbon-500">Device:</span>
                            <span>{device}</span>
                        </div>
                    </div>
                    <ThemeSwitcher />
                </div>
            </header>

            <main className="container mx-auto px-4 py-6 max-w-4xl">
                {/* Error Alert */}
                {error && (
                    <Alert variant="destructive" className="mb-6">
                        <AlertCircle className="h-4 w-4" />
                        <AlertTitle>Error</AlertTitle>
                        <AlertDescription>{error}</AlertDescription>
                    </Alert>
                )}

                {/* Recorder Controls */}
                <div className="bg-white dark:bg-carbon-900 rounded-2xl p-6 shadow-sm border border-carbon-200 dark:border-carbon-800 mb-6">
                    <div className="flex flex-col md:flex-row items-center gap-6">
                        {/* Mic Selection */}
                        <div className="w-full md:w-64">
                            <DropdownMenu>
                                <DropdownMenuTrigger asChild disabled={isRecording}>
                                    <Button
                                        variant="outline"
                                        className="w-full justify-between bg-carbon-50 dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700"
                                    >
                                        <div className="flex items-center gap-2 truncate">
                                            <Settings className="h-4 w-4" />
                                            <span className="truncate">
                                                {availableDevices.find(d => d.deviceId === selectedDevice)?.label || "Select Mic"}
                                            </span>
                                        </div>
                                        <ChevronDown className="h-4 w-4 opacity-50" />
                                    </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent className="w-64">
                                    {availableDevices.map(device => (
                                        <DropdownMenuItem
                                            key={device.deviceId}
                                            onClick={() => setSelectedDevice(device.deviceId)}
                                        >
                                            {device.label || `Mic ${device.deviceId.slice(0, 8)}`}
                                        </DropdownMenuItem>
                                    ))}
                                </DropdownMenuContent>
                            </DropdownMenu>
                        </div>

                        {/* Waveform */}
                        <div className="flex-1 w-full h-[60px] bg-carbon-50 dark:bg-carbon-800/50 rounded-lg overflow-hidden relative">
                            <div ref={micContainerRef} className="w-full h-full" />
                            {!isRecording && (
                                <div className="absolute inset-0 flex items-center justify-center text-xs text-carbon-400">
                                    Waveform visualization
                                </div>
                            )}
                        </div>

                        {/* Record Button */}
                        <Button
                            size="lg"
                            onClick={isRecording ? stopRecording : startRecording}
                            className={`w-full md:w-auto min-w-[140px] h-12 rounded-xl font-medium transition-all ${isRecording
                                ? "bg-carbon-100 dark:bg-carbon-800 text-carbon-900 dark:text-carbon-100 hover:bg-carbon-200 dark:hover:bg-carbon-700"
                                : "bg-red-500 hover:bg-red-600 text-white shadow-lg shadow-red-500/20"
                                }`}
                        >
                            {isRecording ? (
                                <>
                                    <Square className="h-5 w-5 mr-2 fill-current" />
                                    Stop
                                </>
                            ) : (
                                <>
                                    <Mic className="h-5 w-5 mr-2" />
                                    Start
                                </>
                            )}
                        </Button>
                    </div>
                </div>

                {/* Transcript Area */}
                <div className="bg-white dark:bg-carbon-900 rounded-2xl shadow-sm border border-carbon-200 dark:border-carbon-800 min-h-[500px] flex flex-col">
                    <div className="p-4 border-b border-carbon-100 dark:border-carbon-800 flex items-center justify-between">
                        <h2 className="font-semibold text-lg">Live Transcript</h2>
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={copyTranscript}
                            className="text-carbon-500 hover:text-carbon-900 dark:hover:text-carbon-100"
                        >
                            {copied ? <Check className="h-4 w-4 mr-1" /> : <Copy className="h-4 w-4 mr-1" />}
                            {copied ? "Copied" : "Copy Text"}
                        </Button>
                    </div>

                    <div className="flex-1 p-6 font-mono text-lg leading-relaxed whitespace-pre-wrap">
                        {transcript.length === 0 && !currentText && (
                            <div className="text-carbon-400 dark:text-carbon-600 italic text-center mt-20">
                                Start recording to see transcription...
                            </div>
                        )}

                        {transcript.map((seg, i) => (
                            <span key={i} className="text-carbon-900 dark:text-carbon-100 transition-colors duration-300">
                                {seg.text}{" "}
                            </span>
                        ))}

                        {currentText && (
                            <span className="text-carbon-400 dark:text-carbon-500">
                                {currentText}
                            </span>
                        )}

                        <div ref={transcriptEndRef} />
                    </div>
                </div>
            </main>
        </div>
    );
}
