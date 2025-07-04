<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Settings, Mic } from 'lucide-svelte';

	// --- TYPES ---
	type LiveTranscriptionConfig = {
		modelSize: string;
		language: string;
		translate: boolean;
		chunkSize: number;
		diarize: boolean;
	};

	type Props = {
		open: boolean;
		onStart: (config: LiveTranscriptionConfig) => void;
	};

	let { open = $bindable(), onStart }: Props = $props();

	// Default configuration
	let modelSize = $state('small');
	let language = $state('en');
	let translate = $state(false);
	let chunkSize = $state(500); // Changed from 250 to 500ms for better stability
	let diarize = $state(false);

	// Available model sizes with descriptions
	const modelSizes = [
		{ value: 'tiny', label: 'Tiny', description: 'Fastest, lowest accuracy' },
		{ value: 'base', label: 'Base', description: 'Fast, good accuracy' },
		{ value: 'small', label: 'Small', description: 'Balanced speed and accuracy' },
		{ value: 'medium', label: 'Medium', description: 'Slower, higher accuracy' },
		{ value: 'large', label: 'Large', description: 'Slowest, highest accuracy' }
	];

	// Available languages
	const languages = [
		{ value: 'en', label: 'English' },
		{ value: 'auto', label: 'Auto-detect' },
		{ value: 'es', label: 'Spanish' },
		{ value: 'fr', label: 'French' },
		{ value: 'de', label: 'German' },
		{ value: 'it', label: 'Italian' },
		{ value: 'pt', label: 'Portuguese' },
		{ value: 'ru', label: 'Russian' },
		{ value: 'ja', label: 'Japanese' },
		{ value: 'ko', label: 'Korean' },
		{ value: 'zh', label: 'Chinese' },
		{ value: 'hi', label: 'Hindi' }
	];

	// Available chunk sizes with descriptions (updated with larger options)
	const chunkSizes = [
		{ value: 100, label: '100ms', description: 'Ultra-low latency, may cause buffer overflow' },
		{ value: 250, label: '250ms', description: 'Low latency, may cause buffer overflow' },
		{ value: 500, label: '500ms', description: 'Recommended: Balanced latency and stability' },
		{ value: 1000, label: '1000ms', description: 'High stability, higher latency' },
		{ value: 2000, label: '2000ms', description: 'Maximum stability, highest latency' }
	];

	function handleStart() {
		const config: LiveTranscriptionConfig = {
			modelSize,
			language,
			translate,
			chunkSize,
			diarize
		};
		onStart(config);
		open = false;
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			handleStart();
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="border-none bg-gray-700 text-gray-200 sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<Settings class="h-5 w-5 text-blue-500" />
				Live Transcription Settings
			</Dialog.Title>
			<Dialog.Description class="text-gray-400">
				Configure your live transcription settings before starting to record.
			</Dialog.Description>
		</Dialog.Header>

		<div class="space-y-6 py-4">
			<!-- Model Size Selection -->
			<div class="space-y-2">
				<label for="model-size" class="text-sm font-medium text-gray-300">Model Size</label>
				<Select.Root bind:value={modelSize} type="single">
					<Select.Trigger class="w-full border-gray-600 bg-gray-800 text-gray-200">
						{modelSize ? modelSizes.find((m) => m.value === modelSize)?.label : 'Select model size'}
					</Select.Trigger>
					<Select.Content class="border-gray-600 bg-gray-800">
						{#each modelSizes as model}
							<Select.Item value={model.value} class="text-gray-200 hover:bg-gray-700">
								<div class="flex flex-col">
									<span class="font-medium">{model.label}</span>
									<span class="text-xs text-gray-400">{model.description}</span>
								</div>
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>

			<!-- Language Selection -->
			<div class="space-y-2">
				<label for="language" class="text-sm font-medium text-gray-300">Language</label>
				<Select.Root bind:value={language} type="single">
					<Select.Trigger class="w-full border-gray-600 bg-gray-800 text-gray-200">
						{language ? languages.find((l) => l.value === language)?.label : 'Select language'}
					</Select.Trigger>
					<Select.Content class="border-gray-600 bg-gray-800">
						{#each languages as lang}
							<Select.Item value={lang.value} class="text-gray-200 hover:bg-gray-700">
								{lang.label}
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>

			<!-- Chunk Size Selection -->
			<div class="space-y-2">
				<label for="chunk-size" class="text-sm font-medium text-gray-300">Audio Chunk Size</label>
				<Select.Root bind:value={chunkSize} type="single">
					<Select.Trigger class="w-full border-gray-600 bg-gray-800 text-gray-200">
						{chunkSize ? chunkSizes.find((c) => c.value === chunkSize)?.label : 'Select chunk size'}
					</Select.Trigger>
					<Select.Content class="border-gray-600 bg-gray-800">
						{#each chunkSizes as chunk}
							<Select.Item value={chunk.value} class="text-gray-200 hover:bg-gray-700">
								<div class="flex flex-col">
									<span class="font-medium">{chunk.label}</span>
									<span class="text-xs text-gray-400">{chunk.description}</span>
								</div>
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>

			<!-- Translation Option -->
			<div class="flex items-center space-x-2">
				<input
					type="checkbox"
					id="translate"
					bind:checked={translate}
					class="h-4 w-4 rounded border-gray-600 bg-gray-800 text-blue-600 focus:ring-blue-500"
				/>
				<label for="translate" class="text-sm text-gray-300"> Translate to English </label>
			</div>

			<!-- Speaker Diarization Option -->
			<div class="flex items-center space-x-2">
				<input
					type="checkbox"
					id="diarize"
					bind:checked={diarize}
					class="h-4 w-4 rounded border-gray-600 bg-gray-800 text-blue-600 focus:ring-blue-500"
				/>
				<label for="diarize" class="text-sm text-gray-300"> Enable Speaker Diarization </label>
			</div>
			{#if diarize}
				<div class="rounded-md border border-blue-500/30 bg-blue-900/20 p-3 text-sm text-blue-300">
					<p>
						<strong>Note:</strong> Speaker diarization will identify and label different speakers in
						the audio. This may increase processing time but provides more detailed transcriptions.
					</p>
				</div>
			{/if}

			<!-- Info Box -->
			<div class="rounded-md bg-gray-800 p-3 text-sm text-gray-400">
				<p>
					<strong>Note:</strong> Smaller models and chunk sizes are faster but less accurate. Larger
					models and chunk sizes provide better accuracy but may have higher latency.
				</p>
			</div>
		</div>

		<Dialog.Footer>
			<Button
				variant="outline"
				onclick={() => (open = false)}
				class="border-gray-600 text-gray-300 hover:bg-gray-600 hover:text-gray-100"
			>
				Cancel
			</Button>
			<Button onclick={handleStart} class="bg-blue-600 hover:bg-blue-700">
				<Mic class="mr-2 h-4 w-4" />
				Start Recording
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
