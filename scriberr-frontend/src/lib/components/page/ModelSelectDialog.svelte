<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Button } from '$lib/components/ui/button/index.js';

	// --- TYPES ---
	type AudioRecord = {
		id: string;
		title: string;
		created_at: string;
		transcript: string;
	};

	// --- PROPS ---
	let {
		open = $bindable(),
		recordToTranscribe,
		selectedModel = $bindable(),
		modelSizes,
		onStartTranscription
	}: {
		open: boolean;
		recordToTranscribe: AudioRecord | null;
		selectedModel: string;
		modelSizes: string[];
		onStartTranscription: () => void;
	} = $props();
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="border-none bg-gray-700 text-gray-200 sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Select Transcription Model</Dialog.Title>
			<Dialog.Description class="pt-2 text-gray-400">
				Larger models are more accurate but take longer to process.
			</Dialog.Description>
		</Dialog.Header>
		<div class="grid gap-4 py-4">
			<p class="text-sm text-gray-400">
				For: <span class="font-medium text-gray-200">{recordToTranscribe?.title}</span>
			</p>
			<Select.Root bind:value={selectedModel} type="single">
				<Select.Trigger class="w-full">
					{selectedModel ? selectedModel : 'Select a model'}
				</Select.Trigger>
				<Select.Content>
					{#each modelSizes as model}
						<Select.Item value={model}>{model}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>
		<Dialog.Footer>
			<Button
				onclick={onStartTranscription}
				class="bg-neon-100 hover:bg-neon-200 w-full text-gray-800"
			>
				Start Transcription
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
