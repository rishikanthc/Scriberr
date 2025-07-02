<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { LoaderCircle, Youtube } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';

	type Props = {
		open: boolean;
		onDownload: (url: string, title: string) => Promise<void>;
	};

	let { open = $bindable(), onDownload }: Props = $props();

	let youtubeUrl = $state('');
	let title = $state('');
	let isDownloading = $state(false);

	function handleSubmit() {
		if (!youtubeUrl.trim()) {
			toast.error('Please enter a YouTube URL');
			return;
		}

		if (!isValidYouTubeUrl(youtubeUrl)) {
			toast.error('Please enter a valid YouTube URL');
			return;
		}

		downloadYouTube();
	}

	async function downloadYouTube() {
		isDownloading = true;
		try {
			await onDownload(youtubeUrl, title || 'YouTube Video');
			// Reset form
			youtubeUrl = '';
			title = '';
			open = false;
		} catch (error) {
			console.error('YouTube download error:', error);
		} finally {
			isDownloading = false;
		}
	}

	function isValidYouTubeUrl(url: string): boolean {
		const youtubePatterns = [
			/youtube\.com\/watch/,
			/youtu\.be\//,
			/youtube\.com\/embed\//,
			/youtube\.com\/v\//,
			/youtube\.com\/shorts\//
		];

		return youtubePatterns.some(pattern => pattern.test(url));
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !isDownloading) {
			handleSubmit();
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="border-none bg-gray-700 text-gray-200 sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<Youtube class="h-5 w-5 text-red-500" />
				Download YouTube Audio
			</Dialog.Title>
			<Dialog.Description class="text-gray-400">
				Enter a YouTube URL to download and transcribe the audio.
			</Dialog.Description>
		</Dialog.Header>

		<div class="space-y-4 py-4">
			<div class="space-y-2">
				<label for="youtube-url" class="text-sm font-medium text-gray-300">YouTube URL *</label>
				<Input
					id="youtube-url"
					type="url"
					placeholder="https://www.youtube.com/watch?v=..."
					bind:value={youtubeUrl}
					onkeydown={handleKeydown}
					disabled={isDownloading}
				/>
			</div>

			<div class="space-y-2">
				<label for="title" class="text-sm font-medium text-gray-300">Title (optional)</label>
				<Input
					id="title"
					type="text"
					placeholder="Enter a title for this audio"
					bind:value={title}
					onkeydown={handleKeydown}
					disabled={isDownloading}
				/>
			</div>
		</div>

		<Dialog.Footer>
			<Button
				variant="outline"
				onclick={() => (open = false)}
				disabled={isDownloading}
				class="border-gray-600 text-gray-300 hover:bg-gray-600 hover:text-gray-100"
			>
				Cancel
			</Button>
			<Button
				onclick={handleSubmit}
				disabled={isDownloading || !youtubeUrl.trim()}
				class="bg-red-500 hover:bg-red-600"
			>
				{#if isDownloading}
					<LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
					Downloading...
				{:else}
					<Youtube class="mr-2 h-4 w-4" />
					Download
				{/if}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root> 