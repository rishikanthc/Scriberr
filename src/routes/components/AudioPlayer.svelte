<script lang="ts">
	import WaveSurfer from 'wavesurfer.js';
	import { CirclePlay, CirclePause, Volume2, VolumeX } from 'lucide-svelte';
	import { onMount, onDestroy } from 'svelte';
	import { authToken } from '$lib/stores/config';
	import { get } from 'svelte/store';
	import { browser } from '$app/environment';

	const {
		audioSrc,
		peaks = null,
		height = 35,
		waveColor = '#8d8d8d',
		progressColor = '#0e61fe'
	} = $props<{
		audioSrc: string;
		peaks?: number[] | null;
		height?: number;
		waveColor?: string;
		progressColor?: string;
	}>();

	let wavesurfer: WaveSurfer | null = null;
	let waveformElement: HTMLDivElement | null = null;

	let isPlaying = $state(false);
	let isMuted = $state(false);
	let currentTime = $state('0:00');
	let duration = $state('0:00');
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let retryCount = 0;
	const MAX_RETRIES = 3;

	function formatTime(seconds: number): string {
		if (!seconds || isNaN(seconds)) return '0:00';
		const minutes = Math.floor(seconds / 60);
		const remainingSeconds = Math.floor(seconds % 60);
		return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
	}

	async function cleanupWaveSurfer() {
		if (wavesurfer) {
			wavesurfer.unAll();
			try {
				wavesurfer.pause();
				wavesurfer.destroy();
			} catch (err) {
				console.error('[AudioPlayer] Cleanup error:', err);
			}
			wavesurfer = null;
		}
		isPlaying = false;
		isMuted = false;
		currentTime = '0:00';
		duration = '0:00';
		error = null;
	}

	async function fetchAudioBlob(token: string | null): Promise<Blob> {
		const headers: HeadersInit = {};
		if (token) {
			headers['Authorization'] = `Bearer ${token}`;
		}

		const response = await fetch(audioSrc, { headers });
		if (!response.ok) {
			throw new Error(`HTTP error! status: ${response.status}`);
		}
		return await response.blob();
	}

	async function initializeWaveSurfer() {
		if (!waveformElement || !audioSrc || !browser) {
			error = 'Missing required elements';
			isLoading = false;
			return;
		}

		try {
			await cleanupWaveSurfer();

			wavesurfer = WaveSurfer.create({
				container: waveformElement,
				waveColor,
				progressColor,
				height,
				normalize: true,
				barWidth: 2,
				barGap: 1,
				barRadius: 2,
				interact: true,
				autoScroll: false,
				mediaControls: false
			});

			// Set up event handlers
			wavesurfer.on('ready', () => {
				isLoading = false;
				const dur = wavesurfer?.getDuration() || 0;
				duration = formatTime(dur);
			});

			wavesurfer.on('error', async (err) => {
				console.error('[AudioPlayer] WaveSurfer error:', err);
				if (retryCount < MAX_RETRIES) {
					retryCount++;
					console.log(`[AudioPlayer] Retrying (${retryCount}/${MAX_RETRIES})...`);
					await initializeWaveSurfer();
				} else {
					error = 'Error loading audio';
					isLoading = false;
				}
			});

			wavesurfer.on('play', () => (isPlaying = true));
			wavesurfer.on('pause', () => (isPlaying = false));
			wavesurfer.on('finish', () => (isPlaying = false));
			wavesurfer.on('timeupdate', (currentTimeVal) => {
				currentTime = formatTime(currentTimeVal);
			});

			// Fetch the audio blob with authentication
			const token = get(authToken);
			const audioBlob = await fetchAudioBlob(token);

			// Load the blob with peaks if available
			if (peaks && Array.isArray(peaks) && peaks.length > 0) {
				await wavesurfer.loadBlob(audioBlob, peaks);
			} else {
				await wavesurfer.loadBlob(audioBlob);
			}
		} catch (err) {
			if (err instanceof DOMException && err.name === 'AbortError') return;
			console.error('[AudioPlayer] Initialization error:', err);
			error = 'Failed to initialize audio player';
			isLoading = false;
		}
	}

	function togglePlayPause() {
		if (!wavesurfer || isLoading) return;
		wavesurfer.playPause();
	}

	function toggleMute() {
		if (!wavesurfer || isLoading) return;
		wavesurfer.setMuted(!isMuted);
		isMuted = !isMuted;
	}

	onDestroy(() => {
		cleanupWaveSurfer();
	});

	// Watch for audioSrc changes and auth token changes
	$effect(() => {
		const token = get(authToken);
		if (audioSrc && waveformElement && browser) {
			isLoading = true;
			error = null;
			retryCount = 0;
			initializeWaveSurfer();
		}
	});
</script>

<div
	class="rounded-lg border border-neutral-500/40 bg-neutral-900/25 p-3 shadow-sm backdrop-blur-sm"
>
	<div class="flex flex-col justify-center gap-4">
		{#if error}
			<div class="text-sm text-red-500">{error}</div>
		{:else if isLoading}
			<div class="flex h-2 items-center justify-center">
				<div class="h-1.5 w-1.5 animate-bounce rounded-full bg-gray-400 [animation-delay:-0.3s]" />
				<div
					class="mx-1 h-1.5 w-1.5 animate-bounce rounded-full bg-gray-400 [animation-delay:-0.15s]"
				/>
				<div class="h-1.5 w-1.5 animate-bounce rounded-full bg-gray-400" />
			</div>
		{/if}

		<div class="w-full max-w-3xl">
			<div bind:this={waveformElement} class="w-full" />
		</div>

		<div class="flex items-center justify-between">
			<div class="flex items-center gap-4">
				<button
					on:click={togglePlayPause}
					class="text-gray-200 transition-colors hover:text-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
					aria-label={isPlaying ? 'Pause' : 'Play'}
					disabled={isLoading || !!error}
				>
					<svelte:component
						this={isPlaying ? CirclePause : CirclePlay}
						size={24}
						class="text-gray-200"
					/>
				</button>

				<button
					on:click={toggleMute}
					class="text-gray-200 transition-colors hover:text-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
					aria-label={isMuted ? 'Unmute' : 'Mute'}
					disabled={isLoading || !!error}
				>
					<svelte:component this={isMuted ? VolumeX : Volume2} size={20} />
				</button>
			</div>

			<div class="text-sm text-gray-200">
				<span>{currentTime}</span>
				<span class="mx-1">/</span>
				<span>{duration}</span>
			</div>
		</div>
	</div>
</div>
