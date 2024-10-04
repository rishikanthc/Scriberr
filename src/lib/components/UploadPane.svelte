<script lang="ts">
	import { AudioLines, Upload } from 'lucide-svelte';
	import { createEventDispatcher } from 'svelte';
	import { Button } from 'bits-ui';

	let draggedFiles: File[] = [];
	const dispatch = createEventDispatcher();

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
				throw new Error(`Failed to upload ${file.name}`);
			}
			const result = await response.json();

			console.log(`Upload successful for ${file.name}:`, result);
		} catch (error) {
			console.log('hit api ');
			console.error(`Error uploading file ${file.name}:`, error);
		}
	}

	// Function to iterate over all files and upload only valid audio files
	async function uploadFiles(files: File[]) {
		for (const file of files) {
			if (isValidAudioFile(file)) {
				await uploadFile(file); // Upload only if the file is valid
			} else {
				console.error(`Invalid file type: ${file.name} (${file.type})`);
			}
		}
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
</script>

<div class="w-full">
	<!-- Drag and Drop Area -->
	<div
		class="my-4 flex h-[150px] w-full items-center justify-center border border-dashed border-carbongray-200 text-carbongray-300 dark:border-carbongray-600"
		on:dragover={handleDragOver}
		on:drop={handleDrop}
	>
		<div class="flex flex-col items-center justify-center">
			<div>Drag and Drop audio</div>
			<Upload size={40} />
		</div>
	</div>

	<!-- Upload Button -->
	<Button.Root
		class="flex items-center gap-1 rounded-md bg-black p-1 text-white hover:bg-carbongray-600 dark:bg-carbongray-700 dark:hover:bg-carbongray-600"
	>
		<div class="text-base">Upload</div>
		<AudioLines size={20} />
	</Button.Root>
</div>
