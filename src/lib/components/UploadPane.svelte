<script lang="ts">
	import { CircleX, AudioLines, CircleCheck, Upload } from 'lucide-svelte';
	import { createEventDispatcher } from 'svelte';
	import { Button, Progress } from 'bits-ui';
	import StatusSpinner from './StatusSpinner.svelte';

	let draggedFiles: File[] = [];
	let selectedFiles: File[] = [];
	const dispatch = createEventDispatcher();
	let uploading = false;
	let error = false;

	// Function to validate if the file is an audio file
	function isValidAudioFile(file: File): boolean {
		return file.type.startsWith('audio/');
	}

	async function uploadFile(file: File) {
		const formData = new FormData();
		console.log(file);
		formData.append('audio', file); // Append individual file to form data

		try {
			const response = await fetch('/api/files', {
				method: 'POST',
				body: formData
			});

			if (!response.ok) {
				error = true;
				throw new Error(`Failed to upload ${file.name}`);
			}
			const result = await response.json();

			console.log(`Upload successful for ${file.name}:`, result);
		} catch (error) {
			error = true;
			console.error(`Error uploading file ${file.name}:`, error);
		}
	}

	let nfiles = 0;
	let ctr = 0;
	let currFile;

	// Function to iterate over all files and upload only valid audio files
	async function uploadFiles(files: File[]) {
		nfiles = files.length;
		uploading = true;
		for (const file of files) {
			currFile = file.name;
			if (isValidAudioFile(file)) {
				await uploadFile(file); // Upload only if the file is valid
			} else {
				error = true;
				console.error(`Invalid file type: ${file.name} (${file.type})`);
			}
			ctr = ctr + 1;
		}
		setTimeout(() => {
			uploading = false;
			ctr = 0;
			nfiles = 0;
		}, 1000); // Delay for 2 seconds
	}

	// Drag & Drop Handlers
	function handleDragOver(event: DragEvent) {
		event.preventDefault();
		const files = Array.from(event.dataTransfer!.items);
		const isAudio = files.every((item) => item.kind === 'file' && item.type.startsWith('audio/'));
		event.dataTransfer!.dropEffect = isAudio ? 'copy' : 'none'; // Allow only if all files are audio
	}
	async function handleDrop(event: DragEvent) {
		event.preventDefault(); // Prevent default behavior (browser redirects)
		const files = Array.from(event.dataTransfer!.files); // Get files from the drop
		draggedFiles = files;
		console.log('Files dropped:', draggedFiles);
		await uploadFiles(draggedFiles);
		dispatch('onUpload');
	}

	// Handle file selection from input dialog
	function handleFileSelect(event: Event) {
		const input = event.target as HTMLInputElement;
		if (input.files) {
			selectedFiles = Array.from(input.files); // Get files from the input
			uploadFiles(selectedFiles);
			dispatch('onUpload');
		}
	}

	// Trigger the file input dialog when the box is clicked
	function triggerFileInput() {
		document.getElementById('fileInput')!.click(); // Simulate click on hidden input
	}
</script>

<div class="w-full">
	<!-- Drag and Drop Area -->
	<div
		class="my-4 flex h-[150px] w-full cursor-pointer items-center justify-center border border-dashed border-carbongray-200 text-carbongray-300 dark:border-carbongray-600"
		on:dragover={handleDragOver}
		on:drop={handleDrop}
		on:click={triggerFileInput}
	>
		<div class="flex flex-col items-center justify-center">
			{#if uploading}
				{#if ctr < nfiles}
					<StatusSpinner msg={'Uploading'} />
					<Progress.Root
						bind:value={ctr}
						max={nfiles}
						class="relative h-1 w-[100px] overflow-hidden rounded-full bg-carbongray-100"
					>
						<div
							class="h-full w-full bg-carbongray-700 transition-all duration-1000 ease-in-out"
							style={`transform: translateX(-${100 - (100 * (ctr ?? 0)) / nfiles}%)`}
						></div>
					</Progress.Root>
					<div class="text-sm">
						Uploading {currFile}
					</div>
				{:else if !error}
					<CircleCheck size={40} class="animate-ping" />
				{:else}
					<div class="flex flex-col items-center gap-2">
						<div class="text-base">Upload failed</div>
						<CircleX size={40} class="animate-pulse" color={'red'} />
					</div>
				{/if}
			{:else}
				<div>Drag and Drop audio</div>
				<Upload size={40} />
				<div class="mt-6">Or click to select</div>
			{/if}
		</div>
	</div>
	<input
		id="fileInput"
		type="file"
		accept="audio/*"
		class="hidden"
		on:change={handleFileSelect}
		multiple
	/>
</div>
