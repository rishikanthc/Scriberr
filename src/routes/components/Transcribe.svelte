<script lang="ts">
	import Dropzone from 'svelte-file-dropzone';
	import { apiFetch, createEventSource } from '$lib/api';
	import { Switch } from '$lib/components/ui/switch';
	import * as Sheet from '$lib/components/ui/sheet';
	import * as Card from '$lib/components/ui/card/index.js';
	import { ScrollArea } from '$lib/components/ui/scroll-area/index.js';
	import { Button } from '$lib/components/ui/button';
	import * as Select from '$lib/components/ui/select';
	import { Label } from '$lib/components/ui/label';
	import { Progress } from '$lib/components/ui/progress';
	import { AlertCircle, Check, CheckCircle2, Loader2, Settings2, Upload } from 'lucide-svelte';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { audioFiles } from '$lib/stores/audioFiles';
	import SettingsPanel from './SettingsPanel.svelte';
	import type { AudioFile } from '$lib/types';

	interface FileStatus {
		uploadStatus: 'uploading' | 'success' | 'error';
		transcriptionStatus: 'pending' | 'processing' | 'completed' | 'failed';
		uploadProgress: number;
		transcriptionProgress: number;
		transcript?: string;
		error?: string;
		id?: number;
	}

	interface TranscriptionOptions {
		modelSize: 'tiny' | 'base' | 'small' | 'medium' | 'large';
		language: string;
		threads: number;
		processors: number;
		diarization: boolean;
	}

	let files = $state({
		accepted: [] as File[],
		rejected: [] as { file: File; errors: { message: string }[] }[]
	});

	let fileStatus = $state<Record<string, FileStatus>>({});
	let showSettings = $state(false);

	let transcriptionOptions = $state<TranscriptionOptions>({
		modelSize: 'base',
		language: 'en',
		threads: 4,
		processors: 1,
		diarization: false
	});

	async function listenToTranscriptionProgress(fileName: string, id: number) {
		const eventSource = await createEventSource(`/api/transcribe/${id}`);

		eventSource.onmessage = (event) => {
			const progress = JSON.parse(event.data);

			fileStatus[fileName] = {
				...fileStatus[fileName],
				transcriptionStatus: progress.status,
				transcriptionProgress: progress.progress || 0,
				transcript: progress.transcript,
				error: progress.error
			};

			if (progress.status === 'completed' || progress.status === 'failed') {
				audioFiles.updateFile(id, {
					transcriptionStatus: progress.status,
					transcript: progress.transcript,
					transcribedAt: new Date().toISOString()
				});
				eventSource.close();
			}
		};

		eventSource.onerror = () => {
			eventSource.close();
			fileStatus[fileName] = {
				...fileStatus[fileName],
				transcriptionStatus: 'failed',
				error: 'Lost connection to server'
			};
		};
	}

	async function uploadFile(file: File) {
		const formData = new FormData();
		formData.append('file', file);
		formData.append('options', JSON.stringify(transcriptionOptions));

		try {
			fileStatus[file.name] = {
				uploadStatus: 'uploading',
				transcriptionStatus: 'pending',
				uploadProgress: 0,
				transcriptionProgress: 0
			};

			const response = await apiFetch('/api/upload', {
				method: 'POST',
				body: formData
			});

			if (!response.ok) throw new Error('Upload failed');

			const data = await response.json();

			// Add the new file to the store
			audioFiles.addFile({
				id: data.id,
				fileName: file.name,
				title: file.name,
				peaks: data.peaks || [],
				transcriptionStatus: 'pending',
				duration: null,
				language: transcriptionOptions.language,
				uploadedAt: new Date().toISOString(),
				transcribedAt: null,
				transcript: null,
				diarization: transcriptionOptions.diarization
			});

			fileStatus[file.name] = {
				...fileStatus[file.name],
				uploadStatus: 'success',
				uploadProgress: 100,
				id: data.id
			};

			listenToTranscriptionProgress(file.name, data.id);
		} catch (error) {
			fileStatus[file.name] = {
				...fileStatus[file.name],
				uploadStatus: 'error',
				transcriptionStatus: 'failed',
				error: error.message,
				transcriptionProgress: 0
			};
		}
	}

	function handleFilesSelect(e: CustomEvent<{ acceptedFiles: File[]; fileRejections: any[] }>) {
		const { acceptedFiles, fileRejections } = e.detail;
		files.rejected = [...files.rejected, ...fileRejections];

		acceptedFiles.forEach((file) => {
			files.accepted = [...files.accepted, file];
			uploadFile(file);
		});
	}

	function getStatusColor(status: string) {
		const colors = {
			pending: 'text-yellow-500',
			processing: 'text-blue-500',
			completed: 'text-green-500',
			failed: 'text-red-500'
		};
		return colors[status] || 'text-gray-500';
	}

	function getStatusIcon(status: string) {
		const icons = {
			uploading: Loader2,
			processing: Loader2,
			completed: Check,
			failed: AlertCircle
		};
		return icons[status] || Loader2;
	}
</script>

<Card.Root class="w-full border-gray-50 2xl:w-[1280px]">
	<Card.Content class="p-4 md:p-6">
		<Dropzone
			on:drop={handleFilesSelect}
			accept="audio/*,video/*,audio/mpeg,audio/wav,audio/ogg,audio/mp3"
			containerClasses="border-2 border-dashed border-gray-300 rounded-lg p-4 hover:border-primary cursor-pointer transition-colors"
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

		<div class="my-2 flex justify-center md:hidden">
			<Button onclick={() => (showSettings = true)}>Transcription Settings</Button>
		</div>

		<div class="mt-6">
			<div class="md:grid md:grid-cols-[350px_1fr] md:gap-6">
				<Card.Root class="hidden border-gray-100 md:block">
					<Card.Header>
						<Card.Title>Transcription Settings</Card.Title>
					</Card.Header>
					<Card.Content class="space-y-4">
						<SettingsPanel bind:transcriptionOptions />
					</Card.Content>
				</Card.Root>

				<ScrollArea
					class="h-[420px] rounded-lg bg-gray-50 p-4 text-base leading-relaxed md:h-[500px]"
				>
					<div class="space-y-3">
						{#each Object.entries(fileStatus) as [fileName, status]}
							<Card.Root class="border border-gray-50 shadow-md">
								<Card.Content class="p-4">
									<div class="flex items-center justify-between">
										<div class="flex items-center gap-3">
											<svelte:component
												this={getStatusIcon(status.transcriptionStatus)}
												class="h-5 w-5 {status.transcriptionStatus === 'processing' ||
												status.transcriptionStatus === 'uploading'
													? 'animate-spin'
													: ''} {getStatusColor(status.transcriptionStatus)}"
											/>
											<div>
												<p class="font-medium">{fileName}</p>
												<p class="text-sm text-gray-500">
													{status.transcriptionStatus === 'uploading'
														? 'Uploading...'
														: status.transcriptionStatus}
												</p>
											</div>
										</div>
									</div>

									{#if status.uploadStatus === 'uploading'}
										<div class="mt-4 space-y-1">
											<Progress value={status.uploadProgress} class="h-2" />
											<p class="text-right text-sm text-gray-500">{status.uploadProgress}%</p>
										</div>
									{/if}

									{#if status.transcriptionStatus === 'processing'}
										<div class="mt-4 space-y-1">
											<Progress value={status.transcriptionProgress} class="h-2" />
											<p class="text-right text-sm text-gray-500">
												{status.transcriptionProgress}% transcribed
											</p>
										</div>
									{/if}

									{#if status.error}
										<Alert variant="destructive" class="mt-4">
											<AlertDescription>{status.error}</AlertDescription>
										</Alert>
									{/if}
								</Card.Content>
							</Card.Root>
						{/each}

						{#if files.rejected.length > 0}
							<div class="space-y-2">
								<h4 class="font-medium text-red-500">Rejected Files:</h4>
								{#each files.rejected as rejection}
									<Alert variant="destructive">
										<AlertDescription>
											{rejection.file.name} - {rejection.errors[0].message}
										</AlertDescription>
									</Alert>
								{/each}
							</div>
						{/if}
					</div>
				</ScrollArea>
			</div>
		</div>
	</Card.Content>
</Card.Root>

<Sheet.Root bind:open={showSettings}>
	<Sheet.Content side="bottom" class="h-[80vh]">
		<Sheet.Header>
			<Sheet.Title>Transcription Settings</Sheet.Title>
		</Sheet.Header>
		<SettingsPanel bind:transcriptionOptions />
	</Sheet.Content>
</Sheet.Root>
