<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import * as Card from '$lib/components/ui/card';
	import { serverUrl, authToken } from '$lib/stores/config';
	import { get } from 'svelte/store';
	import { Progress } from '$lib/components/ui/progress';
	import { Capacitor } from '@capacitor/core';
	import SettingsPanel from './SettingsPanel.svelte';
	import { SpeechRecognition } from '@capacitor-community/speech-recognition';
	import AudioViz from './AudioViz.svelte';
	import { Button } from '$lib/components/ui/button';
	import { isRecording } from '$lib/stores/recstate';
	import { slide, fade } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { getContext } from 'svelte';
	import { CircleX, Play, Mic, Pause } from 'lucide-svelte';

	let audioPlayer: HTMLAudioElement | null = null;
	let recordedFilePath: string | null = $state(null);
	let isPlaying = $state(false);
	let duration = $state(0);
	let currentTime = $state(0);
	let error: string | null = $state(null);
	let mediaRecorder: MediaRecorder | null = null;
	let audioChunks: BlobPart[] = [];
	let platform: 'web' | 'mobile' = Capacitor.isNativePlatform() ? 'mobile' : 'web';
	let recState = getContext('recstate');

	let recordingTime = $state(0);
	let recordingInterval: number | null = $state(null);
	let isExpanded = $state(false);

	let { showSettings = $bindable() } = $props();

	let url;

	let transcriptionOptions = $state<TranscriptionOptions>({
		modelSize: 'base',
		language: 'en',
		threads: 4,
		processors: 1,
		diarization: false
	});

	let uploadStatus = $state<'idle' | 'uploading' | 'success' | 'error'>('idle');
	let uploadProgress = $state(0);
	let audioBlob = $state(null);

	async function uploadRecording(blob: Blob) {
		const file = new File([blob], `recording-${Date.now()}.wav`, { type: 'audio/wav' });
		const formData = new FormData();
		formData.append('file', file);
		formData.append('options', JSON.stringify(transcriptionOptions));

		try {
			uploadStatus = 'uploading';
			uploadProgress = 0;

			const xhr = new XMLHttpRequest();
			const promise = new Promise((resolve, reject) => {
				xhr.upload.addEventListener('progress', (event) => {
					if (event.lengthComputable) {
						uploadProgress = Math.round((event.loaded * 100) / event.total);
					}
				});

				xhr.addEventListener('load', () => {
					if (xhr.status >= 200 && xhr.status < 300) {
						resolve(xhr.response);
					} else {
						reject(new Error(`Upload failed: ${xhr.statusText}`));
					}
				});

				xhr.addEventListener('error', () => {
					reject(new Error('Upload failed'));
				});
			});

			xhr.open('POST', url);
			const token = get(authToken);
			xhr.setRequestHeader('Accept', 'application/json');
			xhr.setRequestHeader('Authorization', `Bearer ${token}`);
			xhr.responseType = 'json';
			xhr.send(formData);

			const response = await promise;
			uploadStatus = 'success';
			uploadProgress = 100;

			// Reset upload status after success
			setTimeout(() => {
				uploadStatus = 'idle';
				uploadProgress = 0;
			}, 1000);

			return response;
		} catch (err) {
			uploadStatus = 'error';
			uploadProgress = 0;
			error = err instanceof Error ? err.message : 'Failed to upload recording';
			console.error('Upload failed:', err);
		}
	}

	function startRecordingTimer() {
		recordingTime = 0;
		if (recordingInterval) clearInterval(recordingInterval);
		recordingInterval = setInterval(() => {
			recordingTime += 1;
			recordingTime = recordingTime; // trigger reactivity in Svelte 5
		}, 1000);
	}

	// Add function to stop recording timer
	function stopRecordingTimer() {
		if (recordingInterval) {
			clearInterval(recordingInterval);
			recordingInterval = null;
		}
	}

	function initializeAudioPlayer() {
		if (!audioPlayer) {
			audioPlayer = new Audio();
			audioPlayer.addEventListener('timeupdate', () => {
				currentTime = audioPlayer?.currentTime || 0;
			});
			audioPlayer.addEventListener('loadedmetadata', () => {
				duration = audioPlayer?.duration || 0;
				console.log('Audio duration loaded:', duration);
			});
			audioPlayer.addEventListener('ended', () => {
				isPlaying = false;
			});
			audioPlayer.addEventListener('error', (e) => {
				console.error('Audio player error:', e);
				error = 'Failed to load audio file';
			});
		}
	}

	async function startWebRecording() {
		try {
			const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
			mediaRecorder = new MediaRecorder(stream);
			audioChunks = [];

			mediaRecorder.ondataavailable = (event) => {
				if (event.data.size > 0) {
					audioChunks.push(event.data);
				}
			};

			mediaRecorder.onstop = async () => {
				audioBlob = new Blob(audioChunks, { type: 'audio/wav' });
				recordedFilePath = URL.createObjectURL(audioBlob);
				console.log('Web Recording saved at:', recordedFilePath);

				// await uploadRecording(audioBlob);

				// Initialize player and load the new recording
				// initializeAudioPlayer();
				// if (audioPlayer) {
				// 	audioPlayer.src = recordedFilePath;
				// 	try {
				// 		audioPlayer.load();
				// 		console.log('Audio loaded successfully');
				// 	} catch (err) {
				// 		console.error('Error loading audio:', err);
				// 	}
				// }
			};

			mediaRecorder.start();
			isRecording.set(true);
			recState.recording = true;
			startRecordingTimer();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to start recording';
			console.error('Recording failed:', err);
		}
	}

	async function stopWebRecording() {
		try {
			if (mediaRecorder && mediaRecorder.state !== 'inactive') {
				mediaRecorder.stop();
				mediaRecorder.stream.getTracks().forEach((track) => track.stop());
				isRecording.set(false);
				recState.recording = false;
				stopRecordingTimer();
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to stop recording';
			console.error('Failed to stop recording:', err);
		}
	}

	function formatTime(seconds: number): string {
		const mins = Math.floor(seconds / 60);
		const secs = Math.floor(seconds % 60);
		return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
	}

	function getFileUrl(path: string): string {
		if (Capacitor.getPlatform() === 'ios') {
			const cleanPath = path.replace('file://', '');
			return `capacitor://localhost/_capacitor_file_${cleanPath}`;
		}
		return path;
	}

	async function startRecording() {
		if (platform === 'web') {
			await startWebRecording();
		} else {
			try {
				recordedFilePath = null;
				error = null;
				const available = await SpeechRecognition.available();
				if (!available.available) {
					throw new Error('Speech Recognition not available');
				}

				const result = await SpeechRecognition.record();
				console.log('Recording started, path:', result.path);
				recordedFilePath = result.path;
				isRecording.set(true);
				recState.recording = true;
				startRecordingTimer();

				if (audioPlayer) {
					audioPlayer.src = '';
					audioPlayer = null;
					currentTime = 0;
					duration = 0;
				}
			} catch (err) {
				error = err instanceof Error ? err.message : 'Failed to start recording';
				console.error('Recording failed:', err);
			}
		}
		isExpanded = true;
	}

	async function stopRecording() {
		if (platform === 'web') {
			await stopWebRecording();
			showSettings = true;
		} else {
			try {
				error = null;
				await SpeechRecognition.stopRecording();
				isRecording.set(false);
				recState.recording = false;
				stopRecordingTimer();
				console.log('Recording stopped, path:', recordedFilePath);

				if (recordedFilePath) {
					const response = await fetch(getFileUrl(recordedFilePath));
					audioBlob = await response.blob();
					// await uploadRecording(blob);
				}

				const audioUrl = platform === 'mobile' ? getFileUrl(recordedFilePath) : recordedFilePath;
				console.log('Setting audio URL:', audioUrl);
				showSettings = true;

				// initializeAudioPlayer();
				// if (audioPlayer) {
				// 	audioPlayer.src = audioUrl;
				// 	try {
				// 		audioPlayer.load();
				// 		console.log('Audio loaded successfully');
				// 	} catch (err) {
				// 		console.error('Error loading audio:', err);
				// 	}
				// }
			} catch (err) {
				error = err instanceof Error ? err.message : 'Failed to stop recording';
				console.error('Failed to stop recording:', err);
			}
		}
		setTimeout(() => {
			isExpanded = false;
		}, 300);
	}

	async function togglePlayback() {
		if (!audioPlayer || !recordedFilePath) return;

		try {
			error = null;
			if (isPlaying) {
				audioPlayer.pause();
			} else {
				await audioPlayer.play();
			}
			isPlaying = !isPlaying;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to play audio';
			console.error('Playback failed:', err);
		}
	}

	async function handleOk() {
		showSettings = false;
		await uploadRecording(audioBlob);
	}

	onDestroy(() => {
		if (audioPlayer) {
			audioPlayer.pause();
			audioPlayer.src = '';
			audioPlayer = null;
		}
		if (platform === 'web' && mediaRecorder) {
			mediaRecorder.stream.getTracks().forEach((track) => track.stop());
		}
	});

	onMount(() => {
		const baseUrl = get(serverUrl);
		url = baseUrl ? `${baseUrl}/api/upload` : '/api/upload';
	});
</script>

{#if uploadStatus === 'idle'}
	<div
		class={`duration-20 mx-auto mb-6 transition-all ease-out ${
			isExpanded
				? 'w-[85svw] p-2 lg:w-[400px]'
				: 'flex h-[90px] w-[90px] items-center justify-center'
		} gap-4 rounded-xl border border-neutral-500/40 bg-neutral-900/15 shadow-lg backdrop-blur-md`}
	>
		<!-- Error display -->
		{#if error}
			<div class="rounded-lg bg-red-100 p-3 text-red-700">
				{error}
			</div>
		{/if}

		<!-- Recording controls -->
		<div class="mx-auto flex items-center gap-6 p-2">
			{#if isExpanded}
				<div
					class="flex flex-col items-start justify-start"
					transition:slide={{ duration: 20, easing: cubicOut }}
				>
					{#if $isRecording}
						<div class="flex w-full items-center justify-between" in:fade={{ duration: 20 }}>
							<div class="flex items-center gap-1 rounded-sm bg-white p-px">
								<div class="h-[8px] w-[8px] rounded-full bg-black"></div>
								<div class="text-xs text-gray-700">REC</div>
							</div>
							<div class="text-sm text-gray-100">{formatTime(recordingTime)}</div>
						</div>
					{:else}
						<div class="flex items-center gap-1 rounded-sm bg-gray-500 p-1">
							<div class="h-[6px] w-[6px] rounded-full bg-gray-600"></div>
							<div class="text-xs text-gray-600">REC</div>
						</div>
					{/if}
					<AudioViz />
				</div>
			{/if}
			<div class="flex items-center justify-center">
				{#if !$isRecording}
					<Button
						onclick={startRecording}
						class="h-[64px] w-[64px] rounded-full border-[8px] border-black bg-gray-600/80 transition-transform hover:scale-105"
					>
						<div class="text-gray-200">
							<Mic size="24" />
						</div>
					</Button>
				{:else}
					<Button
						onclick={stopRecording}
						class="h-[64px] w-[64px] rounded-full border-[8px] border-black bg-gray-600/80 px-0 py-0"
					>
						<div class="h-[16px] w-[16px] rounded-sm bg-gray-100"></div>
					</Button>
				{/if}
			</div>
		</div>
	</div>
	{#if showSettings}
		<div
			class="fixed left-1/2 top-10 z-[9999] mx-auto mt-8 w-[90svw] -translate-x-1/2 items-start justify-center rounded-lg border border-neutral-400/30 bg-neutral-900/15 p-6 shadow-lg backdrop-blur-lg lg:w-[400px]"
		>
			<div class="relative mt-2 rounded-lg p-0 lg:max-w-[784px]">
				<!-- Header -->
				<div class="mb-2 flex w-full flex-nowrap items-center justify-between">
					<div class="text-xl font-semibold text-gray-50">Transcription Settings</div>
					<Button
						variant="ghost"
						size="icon"
						class="text-300 flex items-center justify-center hover:bg-neutral-400/30"
						onclick={() => (showSettings = false)}
					>
						<CircleX class="h-5 w-5 text-gray-300" />
					</Button>
				</div>

				<SettingsPanel bind:transcriptionOptions />

				<div class="mt-8 flex w-full items-center justify-end">
					<Button class="secondary" onclick={handleOk}>OK</Button>
				</div>
			</div>
		</div>
	{/if}
{:else if uploadStatus === 'uploading'}
	<div class="mt-4 space-y-1">
		<Progress value={uploadProgress} class="h-2" />
		<p class="text-right text-sm text-gray-400">
			{uploadProgress}%
		</p>
	</div>
{:else if uploadStatus === 'success'}
	<div class="my-4 text-center text-sm text-blue-500">Upload complete!</div>
{:else if uploadStatus === 'error'}
	<div class="mt-2 text-center text-sm text-red-500">Upload failed. Please try again.</div>
{/if}
