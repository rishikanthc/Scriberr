<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Capacitor } from '@capacitor/core';
	import AudioMotionAnalyzer from 'audiomotion-analyzer';
	import { SpeechRecognition } from '@capacitor-community/speech-recognition';
	import { getContext } from 'svelte';

	let containerRef: HTMLDivElement;
	let audioMotion: AudioMotionAnalyzer;
	let isStreaming = $state(false);
	let platform: 'web' | 'mobile' = Capacitor.isNativePlatform() ? 'mobile' : 'web';
	let micStream: MediaStream | null = null;
	let audioSource: MediaStreamAudioSourceNode | null = null;

	let recState = getContext('recstate');

	$inspect(recState);
	$effect(() => {
		if (recState.recording) {
			if (platform === 'web') {
				startWebStream();
			} else {
				startMobileStream();
			}
		} else {
			if (platform === 'web') {
				stopWebStream();
			} else {
				stopMobileStream();
			}
		}
	});

	async function startWebStream() {
		try {
			micStream = await navigator.mediaDevices.getUserMedia({ audio: true });
			if (audioMotion.audioCtx.state === 'suspended') {
				await audioMotion.audioCtx.resume();
			}
			audioSource = audioMotion.audioCtx.createMediaStreamSource(micStream);
			audioMotion.connectInput(audioSource);
			isStreaming = true;
			audioMotion.volume = 0;
		} catch (error) {
			console.error('Failed to start web stream:', error);
			alert('Failed to start stream: ' + error.message);
		}
	}

	async function stopWebStream() {
		if (audioSource) {
			audioMotion.disconnectInput(audioSource, true);
			audioSource = null;
		}
		if (micStream) {
			micStream.getTracks().forEach((track) => track.stop());
			micStream = null;
		}
		isStreaming = false;
	}

	async function requestMobilePermissions() {
		try {
			const permissionStatus = await SpeechRecognition.requestPermissions();
			return permissionStatus.speechRecognition === 'granted';
		} catch (error) {
			console.error('Error requesting permissions:', error);
			return false;
		}
	}

	async function startMobileStream() {
		const hasPermission = await requestMobilePermissions();
		if (!hasPermission) return;

		try {
			await SpeechRecognition.startMicrophoneStream();
			isStreaming = true;

			SpeechRecognition.addListener('audioData', (data: { buffer: number[] }) => {
				if (audioMotion) {
					const buffer = audioMotion.audioCtx.createBuffer(
						1,
						data.buffer.length,
						audioMotion.audioCtx.sampleRate
					);
					const channelData = buffer.getChannelData(0);
					channelData.set(data.buffer);

					const source = audioMotion.audioCtx.createBufferSource();
					source.buffer = buffer;
					audioMotion.volume = 0;
					audioMotion.connectInput(source);
					source.start();
				}
			});
		} catch (error) {
			console.error('Failed to start microphone stream:', error);
		}
	}

	async function stopMobileStream() {
		try {
			await SpeechRecognition.stopMicrophoneStream();
			isStreaming = false;
			SpeechRecognition.removeAllListeners();
		} catch (error) {
			console.error('Failed to stop mobile stream:', error);
		}
	}

	async function toggleStream() {
		if (isStreaming) {
			if (platform === 'web') {
				await stopWebStream();
			} else {
				await stopMobileStream();
			}
		} else {
			if (platform === 'web') {
				await startWebStream();
			} else {
				await startMobileStream();
			}
		}
	}

	onMount(() => {
		// Adjust settings based on platform
		const mobileSettings =
			platform === 'mobile'
				? {
						barSpace: 0.5, // Increase space between bars
						linearBoost: 3.5, // Boost amplitude for better visibility
						smoothing: 0.5, // Faster response
						minFreq: 60, // Adjust frequency range
						maxFreq: 14000
					}
				: {};

		audioMotion = new AudioMotionAnalyzer(containerRef, {
			mode: 2,
			alphaBars: false,
			ansiBands: false,
			barSpace: 0.25,
			channelLayout: 'single',
			colorMode: 'bar-level',
			frequencyScale: 'log',
			gradient: 'prism',
			ledBars: false,
			linearAmplitude: true,
			linearBoost: 2.5,
			lumiBars: false,
			maxFreq: 16000,
			minFreq: 30,
			mirror: 0,
			radial: false,
			reflexRatio: 0.5,
			reflexAlpha: 1,
			roundBars: true,
			showPeaks: false,
			showScaleX: false,
			showBgColor: false,
			overlay: true,
			smoothing: 0.7,
			weightingFilter: 'D',
			...mobileSettings
		});

		audioMotion.volume = 0;

		// Set canvas size for mobile
		// if (platform === 'mobile') {
		// 	const width = window.innerWidth;
		// 	const height = Math.min(window.innerHeight * 0.3, 200); // 30% of viewport height, max 200px
		// 	audioMotion.setCanvasSize(width, height);
		// }
	});

	onDestroy(() => {
		if (platform === 'web') {
			stopWebStream();
		} else {
			stopMobileStream();
		}
		if (audioMotion) {
			audioMotion.destroy();
		}
	});
</script>

<div bind:this={containerRef} class="h-[40px] overflow-hidden rounded-lg p-0"></div>
