<script lang="ts">
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import * as Popover from '$lib/components/ui/popover/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Download, FileText, FileJson, FileVideo } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import MarkdownRenderer from './MarkdownRenderer.svelte';

	// Props
	let { recordId }: { recordId: string } = $props();

	// Types
	type FullAudioRecord = {
		id: string;
		title: string;
		transcript: string; // JSON string
		speaker_map: string; // JSON string
		summary: string;
		created_at: string;
	};

	type Word = {
		word: string;
		start: number;
		end: number;
		score: number;
	};

	type TranscriptSegment = {
		start: number;
		end: number;
		text: string;
		words: Word[];
	};

	type JSONTranscript = {
		segments: TranscriptSegment[];
		word_segments: Word[];
		language: string;
	};

	// State
	let record = $state<FullAudioRecord | null>(null);
	let segments = $state<TranscriptSegment[]>([]);
	let activeTab = $state<'transcript' | 'summary'>('transcript');
	let isLoading = $state(true);
	let errorMessage = $state<string | null>(null);
	let audioPlayer = $state<HTMLAudioElement | null>(null);
	let currentTime = $state(0);
	let isDownloadPopoverOpen = $state(false);

	async function fetchRecordDetails() {
		if (!recordId) return;

		isLoading = true;
		errorMessage = null;

		try {
			const response = await fetch(`/api/audio/${recordId}`, { credentials: 'include' });
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || 'Failed to fetch audio details.');
			}
			const data: FullAudioRecord = await response.json();
			record = data;

			// Parse transcript if it exists and is not an empty object string
			if (data.transcript && data.transcript !== '{}') {
				try {
					const transcriptData: JSONTranscript = JSON.parse(data.transcript);
					segments = transcriptData.segments || [];
				} catch (error) {
					console.error('Error parsing transcript:', error);
					segments = [];
				}
			} else {
				segments = [];
			}

			// The summary is now part of the record object, so no separate fetch is needed.
			if (data.summary) {
				// The summary is directly available on the record object.
				// We can also set the tab to 'summary' if a summary exists and no transcript.
				if (!data.transcript || data.transcript === '{}' || data.transcript === '[]' || segments.length === 0) {
					activeTab = 'summary';
				}
			}
		} catch (error) {
			const msg = error instanceof Error ? error.message : 'An unknown error occurred.';
			errorMessage = msg;
			toast.error('Failed to load details', { description: msg });
		} finally {
			isLoading = false;
		}
	}

	// Fetch data when the component mounts or recordId changes
	$effect(() => {
		fetchRecordDetails();
	});

	function formatTime(timeInSeconds: number) {
		const minutes = Math.floor(timeInSeconds / 60);
		const seconds = Math.floor(timeInSeconds % 60);
		return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
	}

	function handleTimeUpdate() {
		if (!audioPlayer) return;
		currentTime = audioPlayer.currentTime;
	}

	function seekTo(time: number) {
		if (audioPlayer) {
			audioPlayer.currentTime = time;
			audioPlayer.play();
		}
	}

	async function downloadTranscript(format: string) {
		if (!record) return;

		try {
			const response = await fetch(`/api/audio/${record.id}/transcript/download?format=${format}`, {
				credentials: 'include'
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || 'Failed to download transcript');
			}

			// Get filename from response headers
			const contentDisposition = response.headers.get('content-disposition');
			let filename = `${record.title}_transcript.${format}`;
			if (contentDisposition) {
				const match = contentDisposition.match(/filename="([^"]+)"/);
				if (match) {
					filename = match[1];
				}
			}

			// Create blob and download
			const blob = await response.blob();
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = filename;
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			window.URL.revokeObjectURL(url);

			toast.success('Transcript downloaded successfully');
			isDownloadPopoverOpen = false;
		} catch (error) {
			const msg = error instanceof Error ? error.message : 'An unknown error occurred.';
			toast.error('Download failed', { description: msg });
		}
	}

	async function downloadSummary() {
		if (!record || !record.summary) return;

		try {
			// Create a blob with the markdown content
			const blob = new Blob([record.summary], { type: 'text/markdown' });
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `${record.title}_summary.md`;
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			window.URL.revokeObjectURL(url);

			toast.success('Summary downloaded successfully');
		} catch (error) {
			const msg = error instanceof Error ? error.message : 'An unknown error occurred.';
			toast.error('Download failed', { description: msg });
		}
	}
</script>

{#if isLoading}
	<div class="flex h-64 items-start justify-center p-8">
		<p>Loading details...</p>
	</div>
{:else if errorMessage}
	<div class="p-8 text-center text-red-500">
		<p>{errorMessage}</p>
	</div>
{:else if record}
	<div class="grid gap-6">
		<audio
			bind:this={audioPlayer}
			src={`/api/audio/file/${record.id}`}
			controls
			class="w-full"
			ontimeupdate={handleTimeUpdate}
		>
			Your browser does not support the audio element.
		</audio>

		<div class="flex-1 overflow-auto">
			<div class="flex items-center justify-between border-b border-gray-700">
				<div class="flex">
					<button
						class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'transcript'
							? 'border-b-2 border-blue-500 text-white'
							: 'text-gray-400 hover:text-white'}"
						onclick={() => (activeTab = 'transcript')}
					>
						Transcript
					</button>
					{#if record.summary}
						<button
							class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'summary'
								? 'border-b-2 border-blue-500 text-white'
								: 'text-gray-400 hover:text-white'}"
							onclick={() => (activeTab = 'summary')}
						>
							Summary
						</button>
					{/if}
				</div>
				
				{#if activeTab === 'transcript' && segments.length > 0}
					<Popover.Root bind:open={isDownloadPopoverOpen}>
						<Popover.Trigger
							class="inline-flex h-8 w-8 items-center justify-center rounded-md text-gray-400 hover:text-white hover:bg-gray-700"
							title="Download transcript"
						>
							<Download class="h-4 w-4" />
						</Popover.Trigger>
						<Popover.Content class="w-48 border-none bg-gray-800 p-2" side="bottom" align="end">
							<div class="space-y-1">
								<button
									class="flex w-full items-center gap-2 rounded px-2 py-2 text-sm text-gray-200 hover:bg-gray-700"
									onclick={() => downloadTranscript('txt')}
								>
									<FileText class="h-4 w-4" />
									Download as TXT
								</button>
								<button
									class="flex w-full items-center gap-2 rounded px-2 py-2 text-sm text-gray-200 hover:bg-gray-700"
									onclick={() => downloadTranscript('json')}
								>
									<FileJson class="h-4 w-4" />
									Download as JSON
								</button>
								<button
									class="flex w-full items-center gap-2 rounded px-2 py-2 text-sm text-gray-200 hover:bg-gray-700"
									onclick={() => downloadTranscript('srt')}
								>
									<FileVideo class="h-4 w-4" />
									Download as SRT
								</button>
							</div>
						</Popover.Content>
					</Popover.Root>
				{:else if activeTab === 'summary' && record.summary}
					<button
						class="inline-flex h-8 w-8 items-center justify-center rounded-md text-gray-400 hover:text-white hover:bg-gray-700"
						title="Download summary"
						onclick={downloadSummary}
					>
						<Download class="h-4 w-4" />
					</button>
				{/if}
			</div>
			<ScrollArea
				class="max-h-[700px] rounded-md border-none bg-gray-800 p-4 shadow-sm shadow-gray-800 lg:h-[700px]"
			>
							{#if activeTab === 'transcript'}
				<div class="space-y-4">
					{#if segments.length > 0}
						{#each segments as segment, index (index)}
							{@const isActive = currentTime >= segment.start && currentTime < segment.end}
							<div
								class="flex cursor-pointer flex-col gap-1 rounded-sm p-1 transition-colors {isActive
									? 'bg-gray-700'
									: 'hover:bg-gray-700'}"
								onclick={() => seekTo(segment.start)}
								role="button"
								tabindex="0"
								onkeypress={(e) => e.key === 'Enter' && seekTo(segment.start)}
							>
								<div class="text-sm font-medium {isActive ? 'text-neon-100' : 'text-gray-400'}">
									{formatTime(segment.start)}
								</div>
								<p class="text-gray-200">{segment.text}</p>
							</div>
						{/each}
					{:else}
						<div class="flex h-full items-center justify-center text-center text-gray-500">
							<p>
								No transcript available for this recording. <br />Right-click to transcribe.
							</p>
						</div>
					{/if}
				</div>
				{:else if activeTab === 'summary'}
					<div class="rounded-md bg-gray-800 p-4">
						<MarkdownRenderer content={record.summary} />
					</div>
				{/if}
			</ScrollArea>
		</div>
	</div>
{/if}
