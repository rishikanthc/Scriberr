<!-- AudioViz.svelte -->
<script lang="ts">
	import WaveSurfer from 'wavesurfer.js';
	import { Button } from 'bits-ui';
	import { CirclePlay, CirclePause } from 'lucide-svelte';
	import { onMount, afterUpdate } from 'svelte';

	export let audioSrc = '';
	export let peaks = [];

	let wavesurfer;
	let waveformElement;
	let playing = false;

	$: if (wavesurfer && audioSrc) {
		updateWaveSurfer();
	}

	function createWaveSurfer() {
		if (wavesurfer) {
			wavesurfer.destroy();
		}
		wavesurfer = WaveSurfer.create({
			container: waveformElement,
			waveColor: '#8d8d8d',
			progressColor: '#0e61fe',
			barWidth: 2,
			dragToSeek: true,
			height: 35,
			barRadius: 10,
			barGap: 2
		});

		wavesurfer.on('finish', () => {
			playing = false;
		});

		updateWaveSurfer();
	}

	function updateWaveSurfer() {
		if (wavesurfer && audioSrc) {
			if (peaks.length > 0) {
				wavesurfer.load(audioSrc, peaks);
			}
			resetPlayState();
		}
	}

	function resetPlayState() {
		playing = false;
		if (wavesurfer) {
			wavesurfer.stop(); // This stops and resets the wavesurfer progress
			wavesurfer.empty(); // This clears the waveform
			if (peaks.length > 0) {
				wavesurfer.load(audioSrc, peaks); // Reload the current audio
			}
		}
	}

	function togglePlayPause() {
		if (wavesurfer) {
			wavesurfer.playPause();
			playing = !playing;
		}
	}

	onMount(() => {
		if (waveformElement) {
			createWaveSurfer();
		}
	});

	afterUpdate(() => {
		if (!wavesurfer && waveformElement) {
			createWaveSurfer();
		}
	});
</script>

<div class="flex items-center gap-2 px-2">
	<Button.Root on:click={togglePlayPause}>
		{#if playing}
			<CirclePause />
		{:else}
			<CirclePlay />
		{/if}
	</Button.Root>
	<div bind:this={waveformElement} class="flex-1"></div>
</div>
