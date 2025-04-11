<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import { audioFiles } from '$lib/stores/audioFiles';
	import { Progress } from '$lib/components/ui/progress';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { get } from 'svelte/store';
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import { Loader2, Check, AlertCircle, CircleX, Upload, CircleCheck } from 'lucide-svelte';
	import { serverUrl, authToken } from '$lib/stores/config';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import { Button } from '$lib/components/ui/button';
	import Dropzone from 'svelte-file-dropzone';
	import SettingsPanel from './SettingsPanel.svelte';
	import type { TranscriptionOptions } from '$lib/types';

	interface FileStatus {
		uploadStatus: 'uploading' | 'success' | 'error';
		transcriptionStatus: 'pending' | 'processing' | 'completed' | 'failed';
		uploadProgress: number;
		transcriptionProgress: number;
		transcript?: string;
		error?: string;
		id?: number;
	}

	let { showUpload = $bindable() } = $props();

	let files = $state({
		accepted: [] as File[],
		rejected: [] as { file: File; errors: { message: string }[] }[]
	});

	let fileStatus = $state<Record<string, FileStatus>>({});
	let showSettings = $state(false);

	let url = $state('');
	let fadeIn = $state(true);

	let transcriptionOptions = $state<TranscriptionOptions>({
		modelSize: 'base',
		language: 'en',
		threads: 4,
		processors: 1,
		diarization: false
	});

	function getStatusColor(status: string) {
		const colors = {
			uploading: 'text-blue-500',
			success: 'text-green-500',
			error: 'text-red-500'
		};
		return colors[status] || 'text-gray-500';
	}

	function getStatusIcon(status: string) {
		const icons = {
			uploading: Loader2,
			success: CircleCheck,
			error: AlertCircle
		};
		return icons[status] || Loader2;
	}

	async function uploadFile(file: File) {
		console.log("Setting up upload with options:", transcriptionOptions);
		console.log("Diarization enabled:", transcriptionOptions.diarization);
		const formData = new FormData();
		formData.append('file', file);
		formData.append('options', JSON.stringify(transcriptionOptions));
		
		// For debugging
		const optionsJson = JSON.stringify(transcriptionOptions);
		console.log("Options being sent as JSON:", optionsJson);

		try {
			fileStatus[file.name] = {
				uploadStatus: 'uploading',
				transcriptionStatus: 'pending',
				uploadProgress: 0,
				transcriptionProgress: 0
			};

			// Create XHR to track upload progress
			const xhr = new XMLHttpRequest();
			const promise = new Promise((resolve, reject) => {
				xhr.upload.addEventListener('progress', (event) => {
					if (event.lengthComputable) {
						const progress = Math.round((event.loaded * 100) / event.total);
						fileStatus[file.name] = {
							...fileStatus[file.name],
							uploadProgress: progress
						};
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
			xhr.setRequestHeader('Accept', 'application/json');
			const token = get(authToken);
			xhr.setRequestHeader('Authorization', `Bearer ${token}`);
			xhr.responseType = 'json';
			xhr.send(formData);

			const response = await promise;
			const data = xhr.response;

			// Update file status on success
			fileStatus[file.name] = {
				...fileStatus[file.name],
				uploadStatus: 'success',
				uploadProgress: 100,
				id: data.id
			};

			// Automatically close after successful upload
			setTimeout(() => {
				delete fileStatus[file.name];
				if (Object.keys(fileStatus).length === 0) {
					showUpload = false;
				}
			}, 500);
			audioFiles.addFile(file);
		} catch (error) {
			fileStatus[file.name] = {
				...fileStatus[file.name],
				uploadStatus: 'error',
				transcriptionStatus: 'failed',
				error: error.message,
				uploadProgress: 0
			};
		}
	}

	function handleFilesSelect(e: CustomEvent<{ acceptedFiles: File[]; fileRejections: any[] }>) {
		const { acceptedFiles, fileRejections } = e.detail;
		files.rejected = fileRejections;

		acceptedFiles.forEach((file) => {
			files.accepted = [...files.accepted, file];
			uploadFile(file);
		});
	}

	onMount(() => {
		const base = get(serverUrl);
		url = base ? `${base}/api/upload` : '/api/upload';
		console.log('URL -->', url);
	});

	function handleBackdropClick(e) {
		// Only close if clicking directly on the backdrop, not on the card
		if (e.target === e.currentTarget) {
			showUpload = false;
		}
	}
</script>

{#if showUpload}
	<!-- Overlay backdrop -->
	<div 
		class="fixed inset-0 z-[9998] bg-black/50 backdrop-blur-sm" 
		onclick={handleBackdropClick}
	></div>

	<div>
		<Card.Root
			class="fixed left-1/2 top-1/2 z-[9999] mx-auto -translate-x-1/2 -translate-y-1/2 rounded-xl border border-neutral-300/30 bg-neutral-400/15 p-2 shadow-2xl backdrop-blur-xl w-[90%] max-w-[500px] max-h-[90vh] overflow-auto"
		>
			<Card.Content class="p-2">
				<div class="flex items-center justify-between">
					<h3 class="font-bold text-gray-50">Upload Audio</h3>
					<Button
						variant="ghost"
						size="icon"
						class="text-300 hover:bg-neutral-400/30"
						onclick={() => (showUpload = false)}
					>
						<CircleX class="h-5 w-5 text-gray-300" />
					</Button>
				</div>

				{#if Object.keys(fileStatus).length > 0}
					<div class="mt-4">
						<ScrollArea class="h-[500px]">
							<div class="space-y-4">
								{#each Object.entries(fileStatus) as [fileName, status]}
									<Card.Root
										class="border border-neutral-300/30 bg-neutral-400/15 p-1 shadow-lg backdrop-blur-xl"
									>
										<Card.Content class="p-1">
											<div class="flex items-center justify-between">
												<div class="flex items-center gap-3">
													<div class={`h-5 w-5 ${status.uploadStatus === 'uploading' ? 'animate-spin' : ''} ${getStatusColor(status.uploadStatus)}`}>
														<Loader2 class={status.uploadStatus === 'uploading' ? 'animate-spin' : 'hidden'} />
														<CircleCheck class={status.uploadStatus === 'success' ? '' : 'hidden'} />
														<AlertCircle class={status.uploadStatus === 'error' ? '' : 'hidden'} />
													</div>
													<div>
														<p class="font-medium text-gray-50">{fileName}</p>
														<p class="text-sm text-gray-400">
															{status.uploadStatus === 'uploading'
																? 'Uploading...'
																: status.uploadStatus === 'success'
																	? 'Upload complete'
																	: 'Upload failed'}
														</p>
													</div>
												</div>
											</div>

											{#if status.uploadStatus === 'uploading'}
												<div class="mt-4 space-y-1">
													<Progress value={status.uploadProgress} class="h-2" />
													<p class="text-right text-sm text-gray-400">
														{status.uploadProgress}%
													</p>
												</div>
											{:else if status.uploadStatus === 'success'}{/if}

											{#if status.error}
												<Alert variant="destructive" class="mt-4">
													<AlertDescription>{status.error}</AlertDescription>
												</Alert>
											{/if}
										</Card.Content>
									</Card.Root>
								{/each}
							</div>
						</ScrollArea>
					</div>
				{:else}
					<Dropzone
						on:drop={handleFilesSelect}
						accept="audio/*,video/*,audio/mpeg,audio/wav,audio/ogg,audio/mp3"
						disableDefaultStyles={false}
						class="mt-6 rounded-lg border border-neutral-500/30 bg-black bg-neutral-900/30 p-4 backdrop-blur-md"
					>
						<div
							class="flex h-[120px] items-center justify-center gap-4 text-muted-foreground md:h-[160px]"
						>
							<Upload class="h-8 w-8" />
							<div>
								<p class="text-lg font-medium">Drop audio files here</p>
								<p class="text-sm text-gray-500">or click to select files</p>
							</div>
						</div>
					</Dropzone>
					<SettingsPanel bind:transcriptionOptions />
				{/if}

				{#if files.rejected.length > 0}
					<div class="mt-4 space-y-2">
						{#each files.rejected as rejection}
							<Alert variant="destructive">
								<AlertDescription>
									{rejection.file.name} - {rejection.errors[0].message}
								</AlertDescription>
							</Alert>
						{/each}
					</div>
				{/if}
			</Card.Content>
		</Card.Root>
	</div>
{/if}