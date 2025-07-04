<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { ScrollArea } from '$lib/components/ui/scroll-area/index.js';
	import { Mic, Pause, Play, StopCircle, Trash2, Settings } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import LiveTranscriptionDialog from './LiveTranscriptionDialog.svelte';

	// --- TYPES ---
	type LiveTranscriptionConfig = {
		modelSize: string;
		language: string;
		translate: boolean;
	};

	type TranscriptLine = {
		text: string;
		speaker: number;
		beg?: number;
		end?: number;
	};

	// --- PROPS ---
	let {
		open = $bindable(),
		onSave
	}: {
		open: boolean;
		onSave: (blob: Blob, title: string) => void;
	} = $props();

	// --- STATE ---
	let recordingState: 'idle' | 'recording' | 'paused' | 'stopped' = $state('idle');
	let playbackState: 'playing' | 'paused' = $state('paused');
	let title = $state('');
	let mediaRecorder: MediaRecorder | null = $state(null);
	let mediaStream: MediaStream | null = $state(null);
	let audioChunks: Blob[] = $state([]);
	let audioBlob: Blob | null = $state(null);
	let audioUrl: string | null = $state(null);
	let audioElement: HTMLAudioElement | null = $state(null);
	let recordingTimer = $state(0);
	let playbackTime = $state(0);
	let duration = $state(0);
	let recordingIntervalId: ReturnType<typeof setInterval> | null = null;

	// Live transcription state
	let isLiveTranscriptionEnabled = $state(false);
	let isLiveTranscriptionDialogOpen = $state(false);
	let liveTranscriptionConfig: LiveTranscriptionConfig | null = $state(null);
	let websocket: WebSocket | null = $state(null);
	let liveTranscriptionText = $state('');
	let isWebSocketConnected = $state(false);
	let connectionStatus: 'connecting' | 'connected' | 'disconnected' | 'error' =
		$state('disconnected');
	let lastTranscriptionText = $state('');
	let processedTranscriptSegments = $state(new Set<string>());

	// Performance monitoring
	let audioQueueSize = $state(0);
	let processingLag = $state(0);
	let lastProcessedTime = $state(0);
	let audioChunkQueue: Blob[] = $state([]);
	let isProcessingAudio = $state(false);
	let connectionHealth = $state('unknown');
	let pingInterval: ReturnType<typeof setInterval> | null = null;

	// Audio visualization
	let audioContext: AudioContext | null = $state(null);
	let analyser: AnalyserNode | null = $state(null);
	let microphone: MediaStreamAudioSourceNode | null = $state(null);
	let animationFrame: number | null = $state(null);
	let waveCanvas: HTMLCanvasElement | null = $state(null);

	// Transcription display
	let transcriptLines: TranscriptLine[] = $state([]);
	let bufferTranscription = $state('');
	let bufferDiarization = $state('');
	let remainingTimeTranscription = $state(0);
	let remainingTimeDiarization = $state(0);
	let waitingForStop = $state(false);
	let lastReceivedData: any = $state(null);

	// --- LIFECYCLE & CLEANUP ---
	$effect(() => {
		if (!open) {
			reset();
		}
		return () => {
			reset();
		};
	});

	function stopMediaStream() {
		if (mediaStream) {
			mediaStream.getTracks().forEach((track) => track.stop());
			mediaStream = null;
		}
	}

	function reset() {
		if (mediaRecorder && mediaRecorder.state !== 'inactive') {
			mediaRecorder.stop();
		}
		stopMediaStream();
		stopRecordingTimer();
		if (audioUrl) {
			URL.revokeObjectURL(audioUrl);
		}
		recordingState = 'idle';
		playbackState = 'paused';
		mediaRecorder = null;
		audioChunks = [];
		audioBlob = null;
		audioUrl = null;
		audioElement = null;
		title = '';
		recordingTimer = 0;
		playbackTime = 0;
		duration = 0;

		// Clean up live transcription
		stopWebSocketConnection();
		isLiveTranscriptionEnabled = false;
		liveTranscriptionConfig = null;

		// Clean up audio visualization
		if (animationFrame) {
			cancelAnimationFrame(animationFrame);
			animationFrame = null;
		}
		if (audioContext && audioContext.state !== 'closed') {
			audioContext.close();
		}
		if (microphone) {
			microphone.disconnect();
		}
		audioContext = null;
		analyser = null;
		microphone = null;

		// Reset transcription state
		transcriptLines = [];
		bufferTranscription = '';
		bufferDiarization = '';
		remainingTimeTranscription = 0;
		remainingTimeDiarization = 0;
		waitingForStop = false;
		lastReceivedData = null;

		// Clean up connection health monitoring
		if (pingInterval) {
			clearInterval(pingInterval);
			pingInterval = null;
		}
		connectionHealth = 'unknown';
	}

	// --- RECORDING ---
	async function startRecording() {
		console.log('startRecording called');

		if (isLiveTranscriptionEnabled) {
			if (!websocket || websocket.readyState !== WebSocket.OPEN) {
				console.log('WebSocket not ready, attempting to connect...');
				toast.info('Connecting to live transcription service...');
				startWebSocketConnection();

				await new Promise((resolve) => setTimeout(resolve, 1000));

				if (!websocket || websocket.readyState !== WebSocket.OPEN) {
					console.log('WebSocket still not ready after waiting');
					toast.error('Live transcription not ready. Please try again.');
					return;
				}
			}
		}

		if (isLiveTranscriptionEnabled) {
			clearTranscriptionState();
		}

		try {
			mediaStream = await navigator.mediaDevices.getUserMedia({
				audio: {
					echoCancellation: true,
					noiseSuppression: true,
					autoGainControl: true
				}
			});

			// Set up audio visualization
			audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();
			analyser = audioContext.createAnalyser();
			analyser.fftSize = 256;
			microphone = audioContext.createMediaStreamSource(mediaStream);
			microphone.connect(analyser);

			const mimeTypes = [
				'audio/webm;codecs=opus',
				'audio/webm',
				'audio/ogg;codecs=opus',
				'audio/wav'
			];
			const selectedMimeType =
				mimeTypes.find((type) => MediaRecorder.isTypeSupported(type)) ?? null;

			if (!selectedMimeType) {
				toast.error(
					'No supported audio format found for recording. Please use a browser that supports WebM audio.'
				);
				reset();
				return;
			}

			mediaRecorder = new MediaRecorder(mediaStream, {
				mimeType: selectedMimeType,
				audioBitsPerSecond: 256000
			});
			audioChunks = [];

			mediaRecorder.ondataavailable = (event) => {
				console.log('MediaRecorder data available, size:', event.data.size);
				if (event.data.size > 0) {
					audioChunks.push(event.data);
					if (isLiveTranscriptionEnabled) {
						console.log('Sending audio chunk for live transcription');
						sendAudioChunk(event.data);
					}
				}
			};

			mediaRecorder.onstop = () => {
				console.log('MediaRecorder stopped, audioChunks length:', audioChunks.length);
				const blob = new Blob(audioChunks, { type: selectedMimeType });
				audioBlob = blob;
				audioUrl = URL.createObjectURL(blob);
				duration = recordingTimer;
				stopMediaStream();
				recordingState = 'stopped';
			};

			// Start recording with smaller chunks for better real-time performance
			mediaRecorder.start(500);
			recordingState = 'recording';
			startRecordingTimer();
			drawWaveform();
		} catch (err) {
			toast.error('Microphone access denied.');
			reset();
		}
	}

	function pauseRecording() {
		if (mediaRecorder) {
			mediaRecorder.pause();
			recordingState = 'paused';
			pauseRecordingTimer();
		}
	}

	function resumeRecording() {
		if (mediaRecorder) {
			mediaRecorder.resume();
			recordingState = 'recording';
			startRecordingTimer();
		}
	}

	function stopRecording() {
		console.log('stopRecording called, recordingState:', recordingState);
		if (mediaRecorder) {
			mediaRecorder.stop();
			stopRecordingTimer();
			if (isLiveTranscriptionEnabled) {
				console.log('Stopping live transcription due to recording stop');
				stopWebSocketConnection();
			}
		}
	}

	// --- PLAYBACK ---
	function togglePlayback() {
		if (!audioElement) return;
		if (playbackState === 'playing') {
			audioElement.pause();
		} else {
			if (audioElement.ended) {
				audioElement.currentTime = 0;
			}
			audioElement.play();
		}
	}

	function handleMetadataLoaded() {
		if (audioElement) {
			const actualDuration = Math.floor(audioElement.duration);
			if (!isNaN(actualDuration) && actualDuration > 0) {
				duration = actualDuration;
			} else {
				if (duration === 0 || isNaN(duration)) {
					duration = recordingTimer;
				}
			}
		}
	}

	function handleTimeUpdate() {
		if (audioElement) playbackTime = Math.floor(audioElement.currentTime);
	}

	// --- ACTIONS ---
	function handleSaveClick() {
		if (!audioBlob) {
			toast.error('No recording available to save.');
			return;
		}
		onSave(audioBlob, title || `Recording ${new Date().toLocaleString()}`);
		open = false;
	}

	function discardRecording() {
		reset();
	}

	// --- TIMER ---
	function startRecordingTimer() {
		stopRecordingTimer();
		recordingIntervalId = setInterval(() => (recordingTimer += 1), 1000);
	}
	function pauseRecordingTimer() {
		if (recordingIntervalId) clearInterval(recordingIntervalId);
	}
	function stopRecordingTimer() {
		pauseRecordingTimer();
		recordingIntervalId = null;
	}
	function formatTime(seconds: number) {
		if (isNaN(seconds) || seconds < 0) {
			return '00:00';
		}
		const min = Math.floor(seconds / 60);
		const sec = Math.floor(seconds % 60);
		return `${String(min).padStart(2, '0')}:${String(sec).padStart(2, '0')}`;
	}

	// --- LIVE TRANSCRIPTION ---
	function openLiveTranscriptionDialog() {
		isLiveTranscriptionDialogOpen = true;
	}

	function clearTranscriptionState() {
		liveTranscriptionText = '';
		lastTranscriptionText = '';
		processedTranscriptSegments.clear();
		transcriptLines = [];
		bufferTranscription = '';
		bufferDiarization = '';
		remainingTimeTranscription = 0;
		remainingTimeDiarization = 0;
		console.log('Cleared transcription state for new recording');
	}

	async function testWebSocketConnection() {
		return new Promise((resolve) => {
			const testWs = new WebSocket(`ws://localhost:9090/ws/transcribe`);

			testWs.onopen = () => {
				console.log('Test WebSocket connection successful');
				testWs.close();
				resolve(true);
			};

			testWs.onerror = (error) => {
				console.log('Test WebSocket connection failed:', error);
				resolve(false);
			};

			setTimeout(() => {
				console.log('Test WebSocket connection timeout');
				testWs.close();
				resolve(false);
			}, 3000);
		});
	}

	function handleLiveTranscriptionStart(config: LiveTranscriptionConfig) {
		liveTranscriptionConfig = config;
		isLiveTranscriptionEnabled = true;

		testWebSocketConnection().then((success) => {
			if (success) {
				startWebSocketConnection();
				toast.info('Connecting to live transcription service...');
			} else {
				toast.error(
					'Cannot connect to live transcription service. Please check if the server is running.'
				);
				isLiveTranscriptionEnabled = false;
			}
		});
	}

	function startWebSocketConnection() {
		if (!liveTranscriptionConfig) {
			console.log('No live transcription config available');
			return;
		}

		if (websocket) {
			console.log('Closing existing WebSocket connection');
			websocket.close();
			websocket = null;
		}

		const wsUrl = `ws://localhost:9090/ws/transcribe`;

		console.log('Starting WebSocket connection to:', wsUrl);
		console.log('Live transcription config:', liveTranscriptionConfig);

		try {
			websocket = new WebSocket(wsUrl);
			console.log('WebSocket object created:', websocket);

			const connectionTimeout = setTimeout(() => {
				if (websocket && websocket.readyState === WebSocket.CONNECTING) {
					console.log('WebSocket connection timeout, closing...');
					websocket.close();
					toast.error('WebSocket connection timeout');
				}
			}, 10000);

			websocket.onopen = () => {
				clearTimeout(connectionTimeout);
				console.log('WebSocket connection opened successfully');
				isWebSocketConnected = true;
				connectionStatus = 'connected';
				connectionHealth = 'healthy';
				toast.success('Live transcription connected!');

				// Start connection health monitoring
				pingInterval = setInterval(() => {
					if (websocket && websocket.readyState === WebSocket.OPEN) {
						try {
							websocket.send(JSON.stringify({ type: 'ping', client_id: `client_${Date.now()}` }));
						} catch (e) {
							console.warn('Failed to send ping:', e);
							connectionHealth = 'unhealthy';
						}
					}
				}, 15000); // Ping every 15 seconds

				if (liveTranscriptionConfig) {
					const initMessage = {
						type: 'init',
						client_id: `client_${Date.now()}`,
						model_size: liveTranscriptionConfig.modelSize,
						language: liveTranscriptionConfig.language,
						translate: liveTranscriptionConfig.translate
					};
					console.log('Sending init message:', initMessage);
					websocket?.send(JSON.stringify(initMessage));
				}
			};

			websocket.onmessage = (event) => {
				console.log('Received WebSocket message:', event.data);
				const data = JSON.parse(event.data);

				if (data.type === 'ready') {
					toast.success('Live transcription ready!');
				} else if (data.type === 'init_success') {
					console.log('Transcription service initialized successfully');
					toast.success('Live transcription initialized!');
				} else if (data.type === 'error') {
					toast.error('Transcription error', { description: data.message });
				} else if (data.type === 'pong') {
					// Connection health check response
					connectionHealth = 'healthy';
				} else if (data.type === 'ping') {
					// Respond to server ping
					if (websocket && websocket.readyState === WebSocket.OPEN) {
						websocket.send(JSON.stringify({ type: 'pong', client_id: data.client_id }));
					}
				} else if (data.type === 'ready_to_stop') {
					console.log('Ready to stop received, finalizing display');
					waitingForStop = false;

					if (lastReceivedData) {
						renderLinesWithBuffer(
							lastReceivedData.lines || [],
							lastReceivedData.buffer_diarization || '',
							lastReceivedData.buffer_transcription || '',
							0,
							0,
							true
						);
					}
					toast.success('Finished processing audio! Ready to record again.');

					if (websocket) {
						websocket.close();
					}
				} else if (data.type === 'transcription' || data.status === 'active_transcription') {
					lastReceivedData = data;

					const {
						lines = [],
						buffer_transcription = '',
						buffer_diarization = '',
						remaining_time_transcription = 0,
						remaining_time_diarization = 0,
						status = 'active_transcription'
					} = data;

					renderLinesWithBuffer(
						lines,
						buffer_diarization,
						buffer_transcription,
						remaining_time_diarization,
						remaining_time_transcription,
						false,
						status
					);
				} else if (data.type === 'status' && data.status === 'no_audio_detected') {
					console.log('No audio detected by transcription service');
				} else {
					console.log('Unhandled WebSocket message type:', data.type || data.status, data);
				}
			};

			websocket.onclose = (event) => {
				console.log('WebSocket connection closed:', event.code, event.reason);
				connectionStatus = 'disconnected';
				isWebSocketConnected = false;
				websocket = null;

				if (event.code !== 1000 && isLiveTranscriptionEnabled) {
					console.log('Unexpected WebSocket close, attempting to reconnect...');
					setTimeout(() => {
						if (isLiveTranscriptionEnabled) {
							startWebSocketConnection();
						}
					}, 2000);
				}
			};

			websocket.onerror = (error) => {
				console.error('WebSocket error:', error);
				toast.error('WebSocket connection failed');
				connectionStatus = 'error';

				setTimeout(() => {
					if (isLiveTranscriptionEnabled && !isWebSocketConnected) {
						console.log('Attempting to reconnect WebSocket...');
						startWebSocketConnection();
					}
				}, 2000);
			};
		} catch (error) {
			console.error('Error creating WebSocket:', error);
			toast.error('Failed to create WebSocket connection');
			return;
		}
	}

	function stopWebSocketConnection() {
		console.log('Stopping WebSocket connection');
		if (websocket) {
			const stopMessage = { type: 'stop' };
			console.log('Sending stop message:', stopMessage);
			websocket.send(JSON.stringify(stopMessage));
			websocket.close();
			websocket = null;
		}
		isWebSocketConnected = false;
		clearTranscriptionState();
	}

	async function sendAudioChunk(audioBlob: Blob) {
		audioChunkQueue.push(audioBlob);
		audioQueueSize = audioChunkQueue.length;

		if (!isProcessingAudio) {
			processAudioQueue();
		}
	}

	async function processAudioQueue() {
		if (isProcessingAudio || audioChunkQueue.length === 0) {
			return;
		}

		isProcessingAudio = true;
		const startTime = Date.now();

		while (audioChunkQueue.length > 0 && websocket?.readyState === WebSocket.OPEN) {
			const audioBlob = audioChunkQueue.shift()!;
			audioQueueSize = audioChunkQueue.length;

			if (audioQueueSize > 5) {
				console.warn(`⚠️ Audio queue too large (${audioQueueSize}), dropping oldest chunks`);
				while (audioChunkQueue.length > 2) {
					audioChunkQueue.shift();
				}
				audioQueueSize = audioChunkQueue.length;
			}

			try {
				await sendSingleAudioChunk(audioBlob);
				lastProcessedTime = Date.now();
				processingLag = lastProcessedTime - startTime;
			} catch (error) {
				console.error('Error processing audio chunk:', error);
				break;
			}

			await new Promise((resolve) => setTimeout(resolve, 50));
		}

		isProcessingAudio = false;
	}

	async function sendSingleAudioChunk(audioBlob: Blob) {
		if (!websocket || websocket.readyState !== WebSocket.OPEN) {
			console.log('WebSocket not ready for audio transmission, skipping chunk');
			return;
		}

		if (audioBlob.type === 'audio/mp4') {
			console.error('❌ WARNING: Sending audio/mp4 format - this will cause transcription issues!');
			return;
		}

		return new Promise<void>((resolve, reject) => {
			const reader = new FileReader();
			reader.onload = () => {
				const result = reader.result as string;
				if (result && result.includes(',')) {
					const base64Audio = result.split(',')[1];
					try {
						websocket?.send(
							JSON.stringify({
								type: 'audio_data',
								audio: base64Audio,
								format: audioBlob.type
							})
						);
						resolve();
					} catch (error) {
						console.error('Error sending audio data:', error);
						reject(error);
					}
				} else {
					console.error('Failed to convert audio blob to base64');
					reject(new Error('Failed to convert audio blob to base64'));
				}
			};
			reader.onerror = () => {
				console.error('Error reading audio blob');
				reject(new Error('Error reading audio blob'));
			};
			reader.readAsDataURL(audioBlob);
		});
	}

	// --- TRANSCRIPTION RENDERING ---
	function renderLinesWithBuffer(
		lines: TranscriptLine[],
		bufferDiarizationText: string,
		bufferTranscriptionText: string,
		remainingTimeDiarizationMs: number,
		remainingTimeTranscriptionMs: number,
		isFinalizing = false,
		currentStatus = 'active_transcription'
	) {
		if (currentStatus === 'no_audio_detected') {
			transcriptLines = [];
			bufferTranscription = '';
			bufferDiarization = '';
			return;
		}

		transcriptLines = lines;
		bufferTranscription = bufferTranscriptionText;
		bufferDiarization = bufferDiarizationText;
		remainingTimeTranscription = remainingTimeTranscriptionMs;
		remainingTimeDiarization = remainingTimeDiarizationMs;
	}

	// --- AUDIO VISUALIZATION ---
	function drawWaveform() {
		if (!analyser || !waveCanvas) return;

		const bufferLength = analyser.frequencyBinCount;
		const dataArray = new Uint8Array(bufferLength);
		analyser.getByteTimeDomainData(dataArray);

		const ctx = waveCanvas.getContext('2d');
		if (!ctx) return;

		ctx.clearRect(0, 0, waveCanvas.width, waveCanvas.height);
		ctx.lineWidth = 1;
		ctx.strokeStyle = 'rgb(0, 0, 0)';
		ctx.beginPath();

		const sliceWidth = waveCanvas.width / bufferLength;
		let x = 0;

		for (let i = 0; i < bufferLength; i++) {
			const v = dataArray[i] / 128.0;
			const y = (v * waveCanvas.height) / 2;

			if (i === 0) {
				ctx.moveTo(x, y);
			} else {
				ctx.lineTo(x, y);
			}

			x += sliceWidth;
		}

		ctx.lineTo(waveCanvas.width, waveCanvas.height / 2);
		ctx.stroke();

		animationFrame = requestAnimationFrame(drawWaveform);
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="border-none bg-gray-700 text-gray-200 sm:max-w-4xl">
		<Dialog.Header>
			<Dialog.Title>Record Audio</Dialog.Title>
		</Dialog.Header>

		<div class="flex flex-col items-center justify-center gap-6 py-4">
			<!-- Recording Button with Waveform -->
			<div class="flex items-center gap-4">
				<button
					class="recorder-button {recordingState === 'recording'
						? 'recording recording-pulse'
						: ''}"
					disabled={waitingForStop}
					onclick={() => {
						if (recordingState === 'idle') {
							startRecording();
						} else if (recordingState === 'recording') {
							stopRecording();
						}
					}}
				>
					<div class="flex h-6 w-6 flex-shrink-0 items-center justify-center">
						<div class="recorder-shape {recordingState === 'recording' ? 'recording' : ''}"></div>
					</div>

					{#if recordingState === 'recording'}
						<div class="ml-4 flex flex-grow items-center">
							<div class="wave-container">
								<canvas bind:this={waveCanvas} width="60" height="30" class="wave-canvas"></canvas>
							</div>
							<div class="ml-3 font-mono text-sm font-medium text-gray-800">
								{formatTime(recordingTimer)}
							</div>
						</div>
					{/if}
				</button>

				<!-- Live Transcription Settings -->
				{#if recordingState === 'idle'}
					<Button
						onclick={openLiveTranscriptionDialog}
						size="icon"
						class="h-12 w-12 rounded-full bg-blue-600 shadow-lg transition-all hover:bg-blue-700"
						aria-label="Live transcription settings"
					>
						<Settings class="h-6 w-6" />
					</Button>
				{/if}
			</div>

			<!-- Live Transcription Status -->
			{#if isLiveTranscriptionEnabled}
				<div class="flex flex-col gap-2 text-sm">
					<div class="flex items-center gap-2">
						<div class="flex items-center gap-1">
							<div
								class="h-2 w-2 rounded-full {isWebSocketConnected ? 'bg-green-500' : 'bg-red-500'}"
							></div>
							<span class="text-gray-400">
								{isWebSocketConnected ? 'Live transcription active' : 'Connecting...'}
							</span>
						</div>
						<Button onclick={testWebSocketConnection} size="sm" class="text-xs" variant="outline">
							Test Connection
						</Button>
					</div>

					<!-- Performance Monitoring -->
					{#if isWebSocketConnected}
						<div class="flex items-center gap-4 text-xs text-gray-500">
							<span>Queue: {audioQueueSize}</span>
							<span>Lag: {processingLag}ms</span>
							<span
								class="h-1 w-8 rounded-full {audioQueueSize > 3
									? 'bg-red-500'
									: audioQueueSize > 1
										? 'bg-yellow-500'
										: 'bg-green-500'}"
							></span>
							<span class="flex items-center gap-1">
								<div
									class="h-2 w-2 rounded-full {connectionHealth === 'healthy'
										? 'bg-green-500'
										: connectionHealth === 'unhealthy'
											? 'bg-red-500'
											: 'bg-yellow-500'}"
								></div>
								<span>{connectionHealth}</span>
							</span>
						</div>
					{/if}
				</div>
			{/if}

			<!-- Live Transcription Display -->
			{#if isLiveTranscriptionEnabled}
				<div class="w-full space-y-2">
					<div class="flex items-center justify-between">
						<label class="text-sm font-medium text-gray-300">
							Live Transcription {isWebSocketConnected ? '(Connected)' : '(Connecting...)'}
						</label>
						{#if transcriptLines.length > 0 || bufferTranscription || bufferDiarization}
							<Button
								onclick={clearTranscriptionState}
								size="sm"
								variant="outline"
								class="h-6 px-2 text-xs"
							>
								Clear
							</Button>
						{/if}
					</div>

					<div class="h-80 w-full rounded-md bg-gray-800 p-3 text-sm text-gray-200">
						<ScrollArea class="transcript-scrollbar h-full w-full">
							<div class="pr-4">
								{#if transcriptLines.length > 0 || bufferTranscription || bufferDiarization}
									{#each transcriptLines as line, idx}
										<div class="mb-3">
											<!-- Speaker Label -->
											{#if line.speaker === -2}
												<span class="speaker-label silence">
													Silence
													{#if line.beg !== undefined && line.end !== undefined}
														<span class="time-info">{line.beg} - {line.end}</span>
													{/if}
												</span>
											{:else if line.speaker === 0 && !waitingForStop}
												<span class="speaker-label diarization">
													<span class="spinner"></span>
													{remainingTimeDiarization}s diarization lag
												</span>
											{:else if line.speaker === -1}
												<span class="speaker-label speaker-1">
													Speaker 1
													{#if line.beg !== undefined && line.end !== undefined}
														<span class="time-info speaker">{line.beg} - {line.end}</span>
													{/if}
												</span>
											{:else if line.speaker > 0}
												<span class="speaker-label speaker-other">
													Speaker {line.speaker}
													{#if line.beg !== undefined && line.end !== undefined}
														<span class="time-info speaker">{line.beg} - {line.end}</span>
													{/if}
												</span>
											{/if}

											<!-- Transcription Lag Indicators -->
											{#if idx === transcriptLines.length - 1 && !waitingForStop}
												{#if remainingTimeTranscription > 0}
													<span class="lag-indicator transcription">
														<span class="spinner"></span>
														Transcription lag {remainingTimeTranscription}s
													</span>
												{/if}
												{#if bufferDiarization && remainingTimeDiarization > 0}
													<span class="lag-indicator diarization">
														<span class="spinner"></span>
														Diarization lag {remainingTimeDiarization}s
													</span>
												{/if}
											{/if}

											<!-- Text Content -->
											<div class="text-content">
												{line.text}
												{#if idx === transcriptLines.length - 1}
													{#if bufferDiarization}
														<span class="buffer-text">{bufferDiarization}</span>
													{/if}
													{#if bufferTranscription}
														<span class="buffer-transcription">{bufferTranscription}</span>
													{/if}
												{/if}
											</div>
										</div>
									{/each}
								{:else}
									<span class="italic text-gray-500">Waiting for speech...</span>
								{/if}
							</div>
						</ScrollArea>
					</div>
				</div>
			{/if}

			<!-- Title Input -->
			{#if recordingState === 'stopped'}
				<div class="w-full space-y-4 pt-4">
					<input
						type="text"
						placeholder="Enter title (optional)"
						bind:value={title}
						class="border-input ring-offset-background focus:ring-ring w-full rounded-md bg-transparent p-2 text-gray-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
					/>
				</div>
			{/if}
		</div>

		<!-- Footer Actions -->
		{#if recordingState === 'stopped'}
			<Dialog.Footer class="sm:justify-between">
				<Button
					variant="ghost"
					onclick={discardRecording}
					class="text-gray-400 hover:bg-gray-600 hover:text-gray-200"
				>
					<Trash2 class="mr-2 h-4 w-4" />
					Discard
				</Button>
				<Button
					onclick={handleSaveClick}
					class="bg-neon-100 hover:bg-neon-200 text-gray-800"
					disabled={!audioBlob}
				>
					Save Recording
				</Button>
			</Dialog.Footer>
		{/if}
	</Dialog.Content>

	<!-- Hidden audio element for playback control -->
	{#if audioUrl}
		<audio
			bind:this={audioElement}
			src={audioUrl}
			onloadedmetadata={handleMetadataLoaded}
			ontimeupdate={handleTimeUpdate}
			onplay={() => (playbackState = 'playing')}
			onpause={() => (playbackState = 'paused')}
			onended={() => (playbackState = 'paused')}
			style="display: none;"
		></audio>
	{/if}

	<!-- Live Transcription Dialog -->
	<LiveTranscriptionDialog
		bind:open={isLiveTranscriptionDialogOpen}
		onStart={handleLiveTranscriptionStart}
	/>
</Dialog.Root>
