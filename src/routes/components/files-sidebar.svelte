<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import { toast } from 'svelte-sonner';
	import * as Dialog from '$lib/components/ui/dialog';
	import AudioRec from './AudioRec.svelte';
	import { onMount, onDestroy } from 'svelte';
	import { browser } from '$app/environment';
	import * as Sheet from '$lib/components/ui/sheet';
	import { Button } from '$lib/components/ui/button';
	import { get } from 'svelte/store';
	import * as ContextMenu from '$lib/components/ui/context-menu/index.js';
	import { Clock, Upload, ChevronRight, CircleCheck, CircleX, Loader2 } from 'lucide-svelte';
	import { audioFiles } from '$lib/stores/audioFiles';
	import FilePanel from './FilePanel.svelte';
	import { setContext } from 'svelte';
	import UploadPanel from './UploadPanel.svelte';

	let { showAudioRec = $bindable(), selectedFileId = $bindable(), showUpload = $bindable() } = $props();
	let showSettings = $state(false);

	interface AudioFile {
		id: number;
		fileName: string;
		title?: string;
		duration: number | null;
		peaks: number[];
		transcriptionStatus: 'pending' | 'processing' | 'completed' | 'failed';
		language: string;
		uploadedAt: string;
		transcribedAt: string | null;
		transcript: TranscriptSegment[] | null;
		diarization: boolean;
		lastError?: string;
	}

	// selectedFileId is now a prop
	let isLoading = $state(true);
	let isFileOpen = $state(false);
	let refreshInterval: ReturnType<typeof setInterval>;

	// Subscribe to the store value
	let files = $derived($audioFiles);
	let fileOpen = $state(false);

	let selectedFile = $derived(selectedFileId ? files.find((f) => f.id === selectedFileId) : null);
	let hidden = $derived(selectedFileId !== null);

	$inspect(isFileOpen);

	$effect(() => {
		if (!isFileOpen) {
			selectedFileId = null;
		}
	});

	function getStatusIcon(status: 'pending' | 'processing' | 'completed' | 'failed') {
		const icons = {
			pending: Clock,
			processing: Loader2,
			completed: CircleCheck,
			failed: CircleX
		} as const;
		return icons[status] || Clock;
	}

	function getStatusColor(status: 'pending' | 'processing' | 'completed' | 'failed'): string {
		const colors = {
			pending: 'bg-yellow-100 text-yellow-600',
			processing: 'bg-blue-100 text-blue-600',
			completed: 'bg-green-100 text-green-600',
			failed: 'bg-red-100 text-red-600'
		};
		return colors[status] || 'bg-gray-100 text-gray-600';
	}

	async function deleteFile(fileId: number, event?: Event) {
		if (event) {
			event.stopPropagation(); // Prevent event bubbling
			event.preventDefault();
		}

		try {
			await audioFiles.deleteFile(fileId);
			toast.success('File deleted successfully');
		} catch (error) {
			console.error('Delete failed:', error);
			toast.error('Failed to delete file');
		}
	}

	function handleFileClick(file: AudioFile) {
		selectedFileId = file.id;
		fileOpen = true;
		isFileOpen = true;
	}

	function handleUploadClick() {
		showUpload = true;
	}

	let height = 'h-[70svh]';

	// Initialize data
	onMount(async () => {
		try {
			await audioFiles.refresh();
			isLoading = false;
		} catch (error) {
			console.error('Failed to load audio files:', error);
			isLoading = false;
		}

		if (window.Capacitor?.isNative) {
			height = 'h-[55svh]';
		}
		
		// Setup auto-refresh for file list
		refreshInterval = setInterval(async () => {
			// Only refresh if we're not in a detail view and looking at the file list
			if (!selectedFileId && !showUpload) {
				console.log('Auto-refreshing file list...');
				try {
					await audioFiles.refresh();
				} catch (error) {
					console.error('Auto-refresh failed:', error);
				}
			}
		}, 5000); // Check every 5 seconds
	});
	
	// Clean up on component destroy
	onDestroy(() => {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}
	});
</script>

<Card.Root class="mx-auto rounded-xl border-none bg-neutral-400/15 p-4 shadow-lg backdrop-blur-xl w-full {showUpload ? 'pointer-events-none opacity-0' : 'opacity-100'}">
	<Card.Content class="p-2">
		<div class="w-full rounded-md">
			{#if isLoading}
				<div class="flex h-32 items-center justify-center">
					<div class="flex gap-1">
						<div
							class="h-1.5 w-1.5 animate-bounce rounded-full bg-gray-400 [animation-delay:-0.3s]"
						></div>
						<div
							class="h-1.5 w-1.5 animate-bounce rounded-full bg-gray-400 [animation-delay:-0.15s]"
						></div>
						<div class="h-1.5 w-1.5 animate-bounce rounded-full bg-gray-400"></div>
					</div>
				</div>
			{:else}
				{#if showAudioRec}
					<AudioRec bind:showSettings />
				{/if}
				<div
					class="{showAudioRec ? 'mt-12' : 'mt-1'} {showSettings
						? 'opacity-0'
						: 'opacity-100'} mb-2 text-lg font-bold text-white"
				>
					<div class="flex items-center justify-between">
						<h3>Recordings</h3>
						<Button
							variant="secondary"
							size="sm"
							onclick={handleUploadClick}
						>
							<div>Upload</div>
							<Upload size={16} class="mr-1 text-blue-500" />
						</Button>
					</div>
				</div>
				<div
					class="{showAudioRec ? 'h-[54svh]' : 'h-[65svh]'} {showSettings
						? 'opacity-0'
						: 'opacity-100'} divide-y divide-neutral-500/15 overflow-y-scroll"
				>
					{#each files as file (file.id)}
						<button
							type="button"
							class="w-full cursor-pointer rounded-md p-3 text-left transition-colors hover:bg-neutral-400/30
                  {selectedFileId === file.id ? 'bg-neutral-400/30' : ''}"
							onclick={() => handleFileClick(file)}
						>
							<ContextMenu.Root>
								<ContextMenu.Trigger>
									<div>
										<div class="flex items-center justify-between">
											<span class="truncate text-sm text-gray-50">
												{file.title || file.fileName}
											</span>
											<ChevronRight class="text-neutral-200/70" size="18" />
										</div>
										<div class="flex items-center justify-between">
											<span class="text-xs text-gray-400">
												{new Date(file.uploadedAt).toLocaleDateString()}
											</span>
										</div>
									</div>
								</ContextMenu.Trigger>
								<ContextMenu.Content>
									<ContextMenu.Item
										class="data-[highlighted]:bg-gray-200"
										onSelect={(event) => deleteFile(file.id, event)}
									>
										Delete
									</ContextMenu.Item>
								</ContextMenu.Content>
							</ContextMenu.Root>
						</button>
					{/each}
				</div>
			{/if}
		</div>
	</Card.Content>
</Card.Root>