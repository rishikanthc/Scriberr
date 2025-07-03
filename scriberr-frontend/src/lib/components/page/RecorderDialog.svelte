<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Mic, Pause, Play, StopCircle, Trash2, Settings } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import LiveTranscriptionDialog from './LiveTranscriptionDialog.svelte';

	// --- TYPES ---
	type LiveTranscriptionConfig = {
		modelSize: string;
		language: string;
		translate: boolean;
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
	let lastAudioSendTime = $state(0);
	let connectionStatus: 'connecting' | 'connected' | 'disconnected' | 'error' =
		$state('disconnected');
	let lastTranscriptionText = $state(''); // Track last transcription to avoid duplicates

	// --- LIFECYCLE & CLEANUP ---
	$effect(() => {
		// When the dialog is closed, ensure everything is torn down.
		if (!open) {
			reset();
		}
		// On component unmount, also ensure cleanup.
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
	}

	// --- RECORDING ---
	async function startRecording() {
		console.log('startRecording called');

		// Check if WebSocket is ready for live transcription
		if (isLiveTranscriptionEnabled) {
			console.log('Live transcription enabled, checking WebSocket state...');
			console.log('WebSocket object:', websocket);
			console.log('WebSocket readyState:', websocket?.readyState);
			console.log('isWebSocketConnected:', isWebSocketConnected);

			if (!websocket || websocket.readyState !== WebSocket.OPEN) {
				console.log('WebSocket not ready, attempting to connect...');
				toast.info('Connecting to live transcription service...');
				startWebSocketConnection();

				// Wait a moment for connection to establish
				await new Promise((resolve) => setTimeout(resolve, 1000));

				// Check again
				if (!websocket || websocket.readyState !== WebSocket.OPEN) {
					console.log('WebSocket still not ready after waiting');
					toast.error('Live transcription not ready. Please try again.');
					return;
				}
			}
		}

		// Only reset if not using live transcription to avoid closing the WebSocket
		if (!isLiveTranscriptionEnabled) {
			reset(); // Start from a clean slate
		} else {
			// For live transcription, only reset recording-specific state
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
			// Reset transcription text for new recording
			liveTranscriptionText = '';
			lastTranscriptionText = '';
			// Don't reset live transcription state
			connectionStatus = 'connecting';
		}
		try {
			mediaStream = await navigator.mediaDevices.getUserMedia({
				audio: {
					echoCancellation: true,
					noiseSuppression: true,
					autoGainControl: true
				}
			});

			// Try to use WebM format first for better compatibility with transcription
			const mimeTypes = [
				'audio/webm;codecs=opus',
				'audio/webm',
				'audio/ogg;codecs=opus',
				'audio/wav'
			];
			const selectedMimeType =
				mimeTypes.find((type) => MediaRecorder.isTypeSupported(type)) ?? null;

			console.log('Available MIME types:', mimeTypes);
			console.log('Selected MIME type:', selectedMimeType);
			console.log('MediaRecorder.isTypeSupported for each type:');
			mimeTypes.forEach((type) => {
				console.log(`  ${type}: ${MediaRecorder.isTypeSupported(type)}`);
			});

			if (!selectedMimeType) {
				toast.error(
					'No supported audio format found for recording. Please use a browser that supports WebM audio.'
				);
				reset();
				return;
			}

			// Warn if we're not using WebM
			if (!selectedMimeType.startsWith('audio/webm')) {
				console.warn('Warning: Not using WebM format, transcription may not work properly');
				toast.warning('Using non-WebM format - live transcription may not work properly');
			}

			mediaRecorder = new MediaRecorder(mediaStream, {
				mimeType: selectedMimeType,
				audioBitsPerSecond: 256000 // 256kbps for high quality
			});
			audioChunks = [];

			mediaRecorder.ondataavailable = (event) => {
				console.log('MediaRecorder data available, size:', event.data.size);
				if (event.data.size > 0) {
					audioChunks.push(event.data);
					// Send audio chunk for live transcription
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
				duration = recordingTimer; // Set duration from recording timer as an initial estimate
				stopMediaStream();
				recordingState = 'stopped'; // Transition state only when blob is ready
			};

			// Start recording with a timeslice to get data in chunks, which is more robust.
			// Using 500ms chunks to reduce frequency and avoid overwhelming the WebSocket
			mediaRecorder.start(500);
			recordingState = 'recording';
			startRecordingTimer();
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
			mediaRecorder.stop(); // This will trigger 'onstop' where the state transition happens
			stopRecordingTimer();
			// Stop live transcription
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
		if (audioElement) duration = Math.floor(audioElement.duration); // Refine duration with actual value
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
		stopRecordingTimer(); // Ensure no multiple intervals
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
		const min = Math.floor(seconds / 60);
		const sec = seconds % 60;
		return `${String(min).padStart(2, '0')}:${String(sec).padStart(2, '0')}`;
	}

	// --- LIVE TRANSCRIPTION ---
	function openLiveTranscriptionDialog() {
		isLiveTranscriptionDialogOpen = true;
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

			// Timeout after 3 seconds
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

		// Test connection first
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

		// Close any existing connection
		if (websocket) {
			console.log('Closing existing WebSocket connection');
			websocket.close();
			websocket = null;
		}

		// Connect directly to the FastAPI server on port 9090
		const wsUrl = `ws://localhost:9090/ws/transcribe`;

		console.log('Starting WebSocket connection to:', wsUrl);
		console.log('Current location:', window.location.href);
		console.log('Live transcription config:', liveTranscriptionConfig);

		try {
			websocket = new WebSocket(wsUrl);
			console.log('WebSocket object created:', websocket);

			// Add connection timeout
			const connectionTimeout = setTimeout(() => {
				if (websocket && websocket.readyState === WebSocket.CONNECTING) {
					console.log('WebSocket connection timeout, closing...');
					websocket.close();
					toast.error('WebSocket connection timeout');
				}
			}, 10000); // 10 second timeout

			// Clear timeout when connection opens
			websocket.onopen = () => {
				clearTimeout(connectionTimeout);
				console.log('WebSocket connection opened successfully');
				isWebSocketConnected = true;
				connectionStatus = 'connected';
				toast.success('Live transcription connected!');

				// Send initialization message
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
		} catch (error) {
			console.error('Error creating WebSocket:', error);
			toast.error('Failed to create WebSocket connection');
			return;
		}

		// Note: onopen is now handled in the try block above

		websocket.onmessage = (event) => {
			console.log('Received WebSocket message:', event.data);
			const data = JSON.parse(event.data);
			if (data.type === 'ready') {
				toast.success('Live transcription ready!');
			} else if (data.type === 'transcription') {
				console.log('Received transcription text:', data.text);
				// Only add new text if it's not empty and not repetitive
				if (data.text && data.text.trim() && data.text.trim() !== lastTranscriptionText) {
					liveTranscriptionText = liveTranscriptionText + ' ' + data.text;
					lastTranscriptionText = data.text.trim();
					console.log('Updated liveTranscriptionText:', liveTranscriptionText);
				} else {
					console.log('Skipping duplicate or empty transcription text');
				}
			} else if (data.type === 'error') {
				toast.error('Transcription error', { description: data.message });
			}
		};

		websocket.onclose = (event) => {
			console.log('WebSocket connection closed:', event.code, event.reason);
			connectionStatus = 'disconnected';
			isWebSocketConnected = false;
			websocket = null;

			// If this was an unexpected close and live transcription is still enabled, try to reconnect
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

			// Try to reconnect after a short delay
			setTimeout(() => {
				if (isLiveTranscriptionEnabled && !isWebSocketConnected) {
					console.log('Attempting to reconnect WebSocket...');
					startWebSocketConnection();
				}
			}, 2000);
		};
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
		liveTranscriptionText = '';
	}

	function sendAudioChunk(audioBlob: Blob) {
		console.log(
			'sendAudioChunk called, websocket state:',
			websocket?.readyState,
			'connected:',
			isWebSocketConnected
		);

		// Don't try to send if WebSocket isn't ready
		if (!websocket || websocket.readyState !== WebSocket.OPEN) {
			console.log('WebSocket not ready for audio transmission, skipping chunk');
			return;
		}

		if (websocket && isWebSocketConnected && websocket.readyState === WebSocket.OPEN) {
			// Throttle audio sending to prevent overwhelming the WebSocket
			const now = Date.now();
			if (now - lastAudioSendTime < 200) {
				// Minimum 200ms between sends
				console.log('Throttling audio chunk send');
				return;
			}
			lastAudioSendTime = now;

			// Log detailed audio format information
			console.log('=== AUDIO CHUNK DEBUG ===');
			console.log(`Audio blob type: ${audioBlob.type}`);
			console.log(`Audio blob size: ${audioBlob.size} bytes`);
			console.log(`Audio blob lastModified: ${audioBlob.lastModified}`);
			console.log(`Audio blob name: ${audioBlob.name || 'unnamed'}`);

			// Check if this is the problematic MP4 format
			if (audioBlob.type === 'audio/mp4') {
				console.error(
					'❌ WARNING: Sending audio/mp4 format - this will cause transcription issues!'
				);
				console.error('❌ The server does not support audio/mp4 format properly');
			} else if (audioBlob.type.startsWith('audio/webm')) {
				console.log('✅ Using WebM format - this should work with transcription');
			} else {
				console.warn(`⚠️ Using format: ${audioBlob.type} - may or may not work with transcription`);
			}

			// Convert blob to base64 for transmission
			const reader = new FileReader();
			reader.onload = () => {
				const result = reader.result as string;
				if (result && result.includes(',')) {
					const base64Audio = result.split(',')[1];
					console.log(`Sending audio chunk of size: ${base64Audio.length} characters`);
					console.log(`Audio MIME type: ${audioBlob.type}`);
					console.log(`Audio blob size: ${audioBlob.size} bytes`);
					try {
						websocket?.send(
							JSON.stringify({
								type: 'audio_data',
								audio: base64Audio,
								format: audioBlob.type
							})
						);
						console.log('✅ Audio chunk sent successfully');
					} catch (error) {
						console.error('Error sending audio data:', error);
					}
				} else {
					console.error('Failed to convert audio blob to base64');
				}
			};
			reader.onerror = () => {
				console.error('Error reading audio blob');
			};
			reader.readAsDataURL(audioBlob);
		} else {
			console.log('WebSocket not ready for audio transmission');
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="border-none bg-gray-700 text-gray-200 sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Record Audio</Dialog.Title>
		</Dialog.Header>

		<div class="flex flex-col items-center justify-center gap-6 py-4">
			<!-- Timer/Progress Display -->
			<div class="font-mono text-4xl text-gray-100" style="font-variant-numeric: tabular-nums;">
				{#if recordingState === 'stopped'}
					{formatTime(playbackTime)} / {formatTime(duration)}
				{:else}
					{formatTime(recordingTimer)}
				{/if}
			</div>

			<!-- Live Transcription Status -->
			{#if isLiveTranscriptionEnabled}
				<div class="flex items-center gap-2 text-sm">
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
			{/if}

			<!-- Recording / Playback Controls -->
			<div class="flex h-16 items-center justify-center gap-4">
				{#if recordingState === 'idle'}
					<!-- Live Transcription Settings Button -->
					<Button
						onclick={openLiveTranscriptionDialog}
						size="icon"
						class="h-12 w-12 rounded-full bg-blue-600 shadow-lg transition-all hover:bg-blue-700 focus:ring-blue-500"
						aria-label="Live transcription settings"
					>
						<Settings class="h-6 w-6" />
					</Button>

					<Button
						onclick={startRecording}
						size="icon"
						class="h-16 w-16 rounded-full bg-red-600 shadow-lg transition-all hover:bg-red-700 focus:ring-red-500"
						aria-label="Start recording"
					>
						<Mic class="h-8 w-8" />
					</Button>
				{/if}

				{#if recordingState === 'recording'}
					<Button
						onclick={pauseRecording}
						size="icon"
						class="h-12 w-12 rounded-full bg-blue-600 shadow-lg transition-all hover:bg-blue-700 focus:ring-blue-500"
						aria-label="Pause recording"
					>
						<Pause class="h-6 w-6" />
					</Button>
					<Button
						onclick={stopRecording}
						size="icon"
						class="h-16 w-16 rounded-full bg-gray-600 shadow-lg transition-all hover:bg-gray-500"
						aria-label="Stop recording"
					>
						<StopCircle class="h-8 w-8" />
					</Button>
				{/if}

				{#if recordingState === 'paused'}
					<Button
						onclick={resumeRecording}
						size="icon"
						class="h-12 w-12 rounded-full bg-blue-600 shadow-lg transition-all hover:bg-blue-700"
						aria-label="Resume recording"
					>
						<Play class="h-6 w-6" />
					</Button>
					<Button
						onclick={stopRecording}
						size="icon"
						class="h-16 w-16 rounded-full bg-gray-600 shadow-lg transition-all hover:bg-gray-500"
						aria-label="Stop recording"
					>
						<StopCircle class="h-8 w-8" />
					</Button>
				{/if}

				{#if recordingState === 'stopped'}
					<Button
						onclick={togglePlayback}
						size="icon"
						class="h-16 w-16 rounded-full bg-blue-600 shadow-lg transition-all hover:bg-blue-700 focus:ring-blue-500"
						aria-label={playbackState === 'playing' ? 'Pause playback' : 'Play recording'}
					>
						{#if playbackState === 'playing'}
							<Pause class="h-8 w-8" />
						{:else}
							<Play class="h-8 w-8" />
						{/if}
					</Button>
				{/if}
			</div>

			<!-- Live Transcription Text -->
			{#if isLiveTranscriptionEnabled}
				<div class="w-full space-y-2">
					<div class="flex items-center justify-between">
						<label class="text-sm font-medium text-gray-300">
							Live Transcription {isWebSocketConnected ? '(Connected)' : '(Connecting...)'}
						</label>
						{#if liveTranscriptionText}
							<Button
								onclick={() => (liveTranscriptionText = '')}
								size="sm"
								variant="outline"
								class="h-6 px-2 text-xs"
							>
								Clear
							</Button>
						{/if}
					</div>
					<div
						class="max-h-32 min-h-[4rem] overflow-y-auto rounded-md bg-gray-800 p-3 text-sm text-gray-200"
					>
						{#if liveTranscriptionText}
							{liveTranscriptionText}
						{:else}
							<span class="italic text-gray-500">Waiting for speech...</span>
						{/if}
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
			on:loadedmetadata={handleMetadataLoaded}
			on:timeupdate={handleTimeUpdate}
			on:play={() => (playbackState = 'playing')}
			on:pause={() => (playbackState = 'paused')}
			on:ended={() => (playbackState = 'paused')}
			style="display: none;"
		></audio>
	{/if}

	<!-- Live Transcription Dialog -->
	<LiveTranscriptionDialog
		bind:open={isLiveTranscriptionDialogOpen}
		onStart={handleLiveTranscriptionStart}
	/>
</Dialog.Root>
