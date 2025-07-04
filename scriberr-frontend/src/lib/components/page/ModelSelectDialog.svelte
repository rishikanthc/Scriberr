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
		onStartTranscription: (diarize: boolean, minSpeakers: number, maxSpeakers: number) => void;
	} = $props();

	// Add diarization state
	let enableDiarization = $state(false);
	let minSpeakers = $state(1);
	let maxSpeakers = $state(2);
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

			<!-- Speaker Diarization Option -->
			<div class="flex items-center space-x-2">
				<input
					type="checkbox"
					id="diarization"
					bind:checked={enableDiarization}
					class="h-4 w-4 rounded border-gray-600 bg-gray-800 text-blue-600 focus:ring-blue-500"
				/>
				<label for="diarization" class="text-sm text-gray-300"> Enable Speaker Diarization </label>
			</div>
			{#if enableDiarization}
				<div class="space-y-3">
					<div
						class="rounded-md border border-blue-500/30 bg-blue-900/20 p-3 text-sm text-blue-300"
					>
						<p>
							<strong>Note:</strong> Speaker diarization will identify and label different speakers in
							the audio. This may increase processing time but provides more detailed transcriptions.
						</p>
					</div>

					<!-- Speaker Count Configuration -->
					<div class="grid grid-cols-2 gap-4">
						<div class="space-y-2">
							<label for="min-speakers" class="text-sm font-medium text-gray-300">
								Minimum Speakers
							</label>
							<input
								type="number"
								id="min-speakers"
								bind:value={minSpeakers}
								min="1"
								max="10"
								class="w-full rounded-md border border-gray-600 bg-gray-800 px-3 py-2 text-sm text-gray-200 focus:border-blue-500 focus:outline-none"
							/>
						</div>
						<div class="space-y-2">
							<label for="max-speakers" class="text-sm font-medium text-gray-300">
								Maximum Speakers
							</label>
							<input
								type="number"
								id="max-speakers"
								bind:value={maxSpeakers}
								min="1"
								max="10"
								class="w-full rounded-md border border-gray-600 bg-gray-800 px-3 py-2 text-sm text-gray-200 focus:border-blue-500 focus:outline-none"
							/>
						</div>
					</div>

					{#if minSpeakers > maxSpeakers}
						<div class="rounded-md border border-red-500/30 bg-red-900/20 p-2 text-sm text-red-300">
							Minimum speakers cannot be greater than maximum speakers.
						</div>
					{/if}
				</div>
			{/if}
		</div>
		<Dialog.Footer>
			<Button
				onclick={() => onStartTranscription(enableDiarization, minSpeakers, maxSpeakers)}
				class="bg-neon-100 hover:bg-neon-200 w-full text-gray-800"
				disabled={enableDiarization && minSpeakers > maxSpeakers}
			>
				Start Transcription
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
