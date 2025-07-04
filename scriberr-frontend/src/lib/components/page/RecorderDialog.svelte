<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { ScrollArea } from '$lib/components/ui/scroll-area/index.js';
	import { Mic, Pause, Play, StopCircle, Trash2 } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';

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

	// Audio visualization
	let audioContext: AudioContext | null = $state(null);
	let analyser: AnalyserNode | null = $state(null);
	let microphone: MediaStreamAudioSourceNode | null = $state(null);
	let animationFrame: number | null = $state(null);
	let waveCanvas: HTMLCanvasElement | null = $state(null);

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
	}

	// --- RECORDING ---
	async function startRecording() {
		console.log('startRecording called');

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
				}
			};

				mediaRecorder.onstop = () => {
		console.log('MediaRecorder stopped, audioChunks length:', audioChunks.length);
		const blob = new Blob(audioChunks, { type: selectedMimeType });
		audioBlob = blob;
		audioUrl = URL.createObjectURL(blob);
		// Set initial duration from recording timer as fallback
		duration = recordingTimer;
		stopMediaStream();
		recordingState = 'stopped';
	};

			// Start recording with 1 second chunks
			mediaRecorder.start(1000);
			recordingState = 'recording';
			startRecordingTimer();
			drawWaveform();
		} catch (err) {
			toast.error('Microphone access denied.');
			reset();
		}
	}

	function pauseRecording() {
		if (mediaRecorder && mediaRecorder.state === 'recording') {
			mediaRecorder.pause();
			recordingState = 'paused';
			pauseRecordingTimer();
		}
	}

	function resumeRecording() {
		if (mediaRecorder && mediaRecorder.state === 'paused') {
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
			const actualDuration = audioElement.duration;
			if (!isNaN(actualDuration) && actualDuration > 0 && isFinite(actualDuration)) {
				duration = Math.floor(actualDuration);
			} else {
				// Fallback to recording timer if audio duration is not available
				if (recordingTimer > 0) {
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
		if (isNaN(seconds) || !isFinite(seconds) || seconds < 0) {
			return '00:00';
		}
		const min = Math.floor(seconds / 60);
		const sec = Math.floor(seconds % 60);
		return `${String(min).padStart(2, '0')}:${String(sec).padStart(2, '0')}`;
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
	<Dialog.Content class="border-none bg-gray-700 text-gray-200 sm:max-w-2xl">
		<Dialog.Header>
			<Dialog.Title>Record Audio</Dialog.Title>
		</Dialog.Header>

		<div class="flex flex-col items-center justify-center gap-6 py-4">
			<!-- Recording Controls -->
			<div class="flex items-center gap-4">
				<!-- Start/Stop Button -->
				{#if recordingState === 'idle'}
					<Button
						onclick={startRecording}
						class="h-16 w-16 rounded-full bg-red-600 shadow-lg transition-all hover:bg-red-700"
						aria-label="Start recording"
					>
						<Mic class="h-8 w-8" />
					</Button>
				{:else if recordingState === 'recording'}
					<div class="flex items-center gap-2">
						<Button
							onclick={pauseRecording}
							class="h-12 w-12 rounded-full bg-yellow-600 shadow-lg transition-all hover:bg-yellow-700"
							aria-label="Pause recording"
						>
							<Pause class="h-6 w-6" />
						</Button>
						<Button
							onclick={stopRecording}
							class="h-12 w-12 rounded-full bg-red-600 shadow-lg transition-all hover:bg-red-700"
							aria-label="Stop recording"
						>
							<StopCircle class="h-6 w-6" />
						</Button>
					</div>
				{:else if recordingState === 'paused'}
					<div class="flex items-center gap-2">
						<Button
							onclick={resumeRecording}
							class="h-12 w-12 rounded-full bg-green-600 shadow-lg transition-all hover:bg-green-700"
							aria-label="Resume recording"
						>
							<Play class="h-6 w-6" />
						</Button>
						<Button
							onclick={stopRecording}
							class="h-12 w-12 rounded-full bg-red-600 shadow-lg transition-all hover:bg-red-700"
							aria-label="Stop recording"
						>
							<StopCircle class="h-6 w-6" />
						</Button>
					</div>
				{/if}

				<!-- Recording Timer and Waveform -->
				{#if recordingState === 'recording' || recordingState === 'paused'}
					<div class="flex items-center gap-4">
						<div class="wave-container">
							<canvas bind:this={waveCanvas} width="60" height="30" class="wave-canvas"></canvas>
						</div>
						<div class="font-mono text-lg font-medium text-gray-200">
							{formatTime(recordingTimer)}
						</div>
					</div>
				{/if}
			</div>

			<!-- Playback Controls (when recording is stopped) -->
			{#if recordingState === 'stopped' && audioUrl}
				<div class="flex items-center gap-4">
					<Button
						onclick={togglePlayback}
						class="h-12 w-12 rounded-full bg-blue-600 shadow-lg transition-all hover:bg-blue-700"
						aria-label={playbackState === 'playing' ? 'Pause playback' : 'Play recording'}
					>
						{#if playbackState === 'playing'}
							<Pause class="h-6 w-6" />
						{:else}
							<Play class="h-6 w-6" />
						{/if}
					</Button>
					<div class="font-mono text-sm text-gray-400">
						{formatTime(playbackTime)} / {formatTime(duration)}
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
						class="border-input ring-offset-background focus:ring-ring w-full rounded-md bg-gray-800 p-3 text-gray-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
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
			oncanplay={handleMetadataLoaded}
			ontimeupdate={handleTimeUpdate}
			onplay={() => (playbackState = 'playing')}
			onpause={() => (playbackState = 'paused')}
			onended={() => (playbackState = 'paused')}
			style="display: none;"
		></audio>
	{/if}
</Dialog.Root>
