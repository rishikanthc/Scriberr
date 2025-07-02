<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
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
	}

	// --- RECORDING ---
	async function startRecording() {
		reset(); // Start from a clean slate
		try {
			mediaStream = await navigator.mediaDevices.getUserMedia({
				audio: {
					echoCancellation: true,
					noiseSuppression: true,
					autoGainControl: true
				}
			});

			// Find a supported MIME type to ensure compatibility
			const mimeTypes = [
				'audio/webm;codecs=opus',
				'audio/webm',
				'audio/ogg;codecs=opus',
				'audio/mp4'
			];
			const selectedMimeType =
				mimeTypes.find((type) => MediaRecorder.isTypeSupported(type)) ?? null;

			if (!selectedMimeType) {
				toast.error('No supported audio format found for recording.');
				reset();
				return;
			}

			mediaRecorder = new MediaRecorder(mediaStream, {
				mimeType: selectedMimeType,
				audioBitsPerSecond: 256000 // 256kbps for high quality
			});
			audioChunks = [];

			mediaRecorder.ondataavailable = (event) => {
				if (event.data.size > 0) {
					audioChunks.push(event.data);
				}
			};

			mediaRecorder.onstop = () => {
				const blob = new Blob(audioChunks, { type: selectedMimeType });
				audioBlob = blob;
				audioUrl = URL.createObjectURL(blob);
				duration = recordingTimer; // Set duration from recording timer as an initial estimate
				stopMediaStream();
				recordingState = 'stopped'; // Transition state only when blob is ready
			};

			// Start recording with a timeslice to get data in chunks, which is more robust.
			mediaRecorder.start(100);
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
		if (mediaRecorder) {
			mediaRecorder.stop(); // This will trigger 'onstop' where the state transition happens
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

			<!-- Recording / Playback Controls -->
			<div class="flex h-16 items-center justify-center gap-4">
				{#if recordingState === 'idle'}
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
</Dialog.Root>
