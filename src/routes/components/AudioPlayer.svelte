<script lang="ts">
	import { onMount, createEventDispatcher } from 'svelte';
	import { CirclePlay, CirclePause, VolumeX, Volume2 } from 'lucide-svelte';
	import { formatTime } from '$lib/utils';

	const dispatch = createEventDispatcher();

	const { audioSrc = '', peaks = [] } = $props();

	let audioElement: HTMLAudioElement;
	let containerRef = $state<HTMLDivElement | null>(null);
	let canvasRef = $state<HTMLCanvasElement | null>(null);
	let progressRef = $state<HTMLDivElement | null>(null);
	let seekRef = $state<HTMLDivElement | null>(null);

	let ctx: CanvasRenderingContext2D | null = null;
	let audioContext: AudioContext;
	let analyser: AnalyserNode;
	let dataArray: Uint8Array;
	let animationFrameId: number;

	let isPlaying = $state(false);
	let isMuted = $state(false);
	let currentTime = $state('0:00');
	let duration = $state('0:00');
	let progress = $state(0);
	let isLoading = $state(true);
	let error = $state<Error | null>(null);
	let isDrawn = false;

	// Initialize audio element and apply peaks if available
	onMount(() => {
		if (!audioSrc) {
			error = new Error('No audio source provided');
			isLoading = false;
			return;
		}

		audioElement = new Audio(audioSrc);
		audioElement.preload = 'metadata';

		audioElement.addEventListener('loadedmetadata', () => {
			duration = formatTime(audioElement.duration);
			isLoading = false;
			drawWaveform();
		});

		audioElement.addEventListener('timeupdate', updateProgress);

		audioElement.addEventListener('ended', () => {
			isPlaying = false;
			audioElement.currentTime = 0;
			updateProgress();
		});

		audioElement.addEventListener('error', (e) => {
			console.error('Audio error:', e);
			error = new Error('Failed to load audio');
			isLoading = false;
		});

		audioElement.addEventListener('playing', () => {
			isPlaying = true;
		});

		audioElement.addEventListener('pause', () => {
			isPlaying = false;
		});

		// Set up seeking functionality
		if (seekRef) {
			seekRef.addEventListener('click', handleSeek);
		}

		return () => {
			if (animationFrameId) {
				cancelAnimationFrame(animationFrameId);
			}
			if (audioElement) {
				audioElement.pause();
				audioElement.removeEventListener('timeupdate', updateProgress);
				audioElement.removeEventListener('ended', () => {});
				audioElement.removeEventListener('error', () => {});
				audioElement.removeEventListener('playing', () => {});
				audioElement.removeEventListener('pause', () => {});
			}
			if (seekRef) {
				seekRef.removeEventListener('click', handleSeek);
			}
		};
	});

	function togglePlayPause() {
		if (isLoading || error) return;

		if (isPlaying) {
			audioElement.pause();
		} else {
			audioElement.play().catch((e) => {
				console.error('Playback error:', e);
				error = e;
			});
		}
	}

	function toggleMute() {
		if (isLoading || error) return;
		isMuted = !isMuted;
		audioElement.muted = isMuted;
	}

	function updateProgress() {
		if (!audioElement) return;

		const currentTimeInSeconds = audioElement.currentTime;
		currentTime = formatTime(currentTimeInSeconds);

		// Calculate progress percentage
		if (audioElement.duration) {
			progress = (currentTimeInSeconds / audioElement.duration) * 100;
		}

		// Update progress bar width
		if (progressRef) {
			progressRef.style.width = `${progress}%`;
		}
	}

	function handleSeek(e: MouseEvent) {
		if (isLoading || error || !audioElement || !seekRef) return;

		const rect = seekRef.getBoundingClientRect();
		const pos = (e.clientX - rect.left) / rect.width;
		const newTime = pos * audioElement.duration;
		
		if (isNaN(newTime)) return;
		
		audioElement.currentTime = newTime;
		updateProgress();
	}

	function drawWaveform() {
		if (!canvasRef || !containerRef || isDrawn || peaks.length === 0) return;

		const containerWidth = containerRef.offsetWidth || 300;
		const containerHeight = containerRef.offsetHeight || 80;
		
		// Set canvas dimensions
		canvasRef.width = containerWidth;
		canvasRef.height = containerHeight;
		
		ctx = canvasRef.getContext('2d');
		if (!ctx) return;
		
		// Clear canvas
		ctx.clearRect(0, 0, canvasRef.width, canvasRef.height);
		
		// Set waveform style
		ctx.fillStyle = '#4a5568'; // Slate-600
		
		// Draw the waveform from peaks data
		const barWidth = Math.max(1, containerWidth / peaks.length);
		const barGap = Math.max(0, Math.min(1, barWidth * 0.2));
		const maxPeak = Math.max(...peaks, 1); // Avoid division by zero
		
		for (let i = 0; i < peaks.length; i++) {
			const peak = peaks[i];
			const x = i * barWidth;
			const barHeight = (peak / maxPeak) * containerHeight * 0.8; // 80% of container height
			
			// Center the bar vertically
			const y = (containerHeight - barHeight) / 2;
			
			// Draw the bar
			ctx.fillRect(x + barGap/2, y, barWidth - barGap, barHeight);
		}
		
		isDrawn = true;
	}
</script>

<div class="w-full">
	<div class="relative mb-2 rounded bg-neutral-800/60 p-4">
		{#if isLoading}
			<div class="flex h-16 items-center justify-center">
				<div class="text-sm text-gray-400">Loading audio...</div>
			</div>
		{:else if error}
			<div class="flex h-16 items-center justify-center">
				<div class="text-sm text-red-500">Failed to load audio: {error.message}</div>
			</div>
		{:else}
			<div
				bind:this={containerRef}
				class="relative h-16 w-full cursor-pointer"
				on:click={handleSeek}
				bind:this={seekRef}
			>
				<canvas
					bind:this={canvasRef}
					class="absolute left-0 top-0 h-full w-full"
				></canvas>
				<div class="absolute left-0 top-0 h-full bg-blue-500/20" style="width: {progress}%" bind:this={progressRef}></div>
			</div>
			
			<div class="mt-3 flex items-center justify-between">
				<div class="flex items-center gap-2">
					<button
						on:click={togglePlayPause}
						class="text-gray-200 transition-colors hover:text-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
						aria-label={isPlaying ? 'Pause' : 'Play'}
						disabled={isLoading || !!error}
					>
						{#if isPlaying}
							<CirclePause size={24} class="text-gray-200" />
						{:else}
							<CirclePlay size={24} class="text-gray-200" />
						{/if}
					</button>

					<button
						on:click={toggleMute}
						class="text-gray-200 transition-colors hover:text-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
						aria-label={isMuted ? 'Unmute' : 'Mute'}
						disabled={isLoading || !!error}
					>
						{#if isMuted}
							<VolumeX size={20} />
						{:else}
							<Volume2 size={20} />
						{/if}
					</button>
				</div>

				<div class="text-sm text-gray-200">
					<span>{currentTime}</span>
					<span class="mx-1">/</span>
					<span>{duration}</span>
				</div>
			</div>
		{/if}
	</div>
</div>