<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Checkbox } from '$lib/components/ui/checkbox/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';
	import * as Tabs from '$lib/components/ui/tabs/index.js';

	// --- TYPES ---
	type AudioRecord = {
		id: string;
		title: string;
		created_at: string;
		transcript: string;
	};

	type TranscriptionParams = {
		model_size: string;
		batch_size: number;
		compute_type: string;
		vad_onset: number;
		vad_offset: number;
		condition_on_previous_text: boolean;
		compression_ratio_threshold: number;
		logprob_threshold: number;
		no_speech_threshold: number;
		temperature: number;
		best_of: number;
		beam_size: number;
		patience: number;
		length_penalty: number;
		suppress_numerals: boolean;
		initial_prompt: string;
		temperature_increment_on_fallback: number;
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
		onStartTranscription: (
			diarize: boolean,
			minSpeakers: number,
			maxSpeakers: number,
			params: TranscriptionParams
		) => void;
	} = $props();

	// Add diarization state
	let enableDiarization = $state(false);
	let minSpeakers = $state(1);
	let maxSpeakers = $state(2);

	// Advanced parameters state
	let transcriptionParams = $state<TranscriptionParams>({
		model_size: selectedModel,
		batch_size: 16,
		compute_type: 'int8',
		vad_onset: 0.5,
		vad_offset: 0.5,
		condition_on_previous_text: true,
		compression_ratio_threshold: 2.4,
		logprob_threshold: -1.0,
		no_speech_threshold: 0.6,
		temperature: 0.0,
		best_of: 5,
		beam_size: 5,
		patience: 1.0,
		length_penalty: 1.0,
		suppress_numerals: false,
		initial_prompt: '',
		temperature_increment_on_fallback: 0.2
	});

	// Update model size when selectedModel changes
	$effect(() => {
		transcriptionParams.model_size = selectedModel;
	});

	function handleStartTranscription() {
		onStartTranscription(enableDiarization, minSpeakers, maxSpeakers, transcriptionParams);
	}

	function resetToDefaults() {
		transcriptionParams = {
			model_size: selectedModel,
			batch_size: 16,
			compute_type: 'int8',
			vad_onset: 0.5,
			vad_offset: 0.5,
			condition_on_previous_text: true,
			compression_ratio_threshold: 2.4,
			logprob_threshold: -1.0,
			no_speech_threshold: 0.6,
			temperature: 0.0,
			best_of: 5,
			beam_size: 5,
			patience: 1.0,
			length_penalty: 1.0,
			suppress_numerals: false,
			initial_prompt: '',
			temperature_increment_on_fallback: 0.2
		};
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content
		class="max-h-[90vh] overflow-hidden border-none bg-gray-700 text-gray-200 sm:max-w-4xl"
	>
		<Dialog.Header>
			<Dialog.Title>Transcription Settings</Dialog.Title>
			<Dialog.Description class="pt-2 text-gray-400">
				Configure transcription settings for optimal results.
			</Dialog.Description>
		</Dialog.Header>

		<div class="flex h-full flex-col">
			<Tabs.Root value="basic" class="flex-1 overflow-hidden">
				<Tabs.List class="grid w-full grid-cols-2">
					<Tabs.Trigger value="basic">Basic Settings</Tabs.Trigger>
					<Tabs.Trigger value="advanced">Advanced Settings</Tabs.Trigger>
				</Tabs.List>

				<Tabs.Content value="basic" class="max-h-[60vh] space-y-4 overflow-y-auto">
					<div class="grid gap-4 py-4">
						<p class="text-sm text-gray-400">
							For: <span class="font-medium text-gray-200">{recordToTranscribe?.title}</span>
						</p>

						<div class="space-y-2">
							<Label for="model">Model Size</Label>
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

						<!-- Speaker Diarization Option -->
						<div class="flex items-center space-x-2">
							<Checkbox id="diarization" bind:checked={enableDiarization} />
							<Label for="diarization">Enable Speaker Diarization</Label>
						</div>

						{#if enableDiarization}
							<div class="space-y-3">
								<div
									class="rounded-md border border-blue-500/30 bg-blue-900/20 p-3 text-sm text-blue-300"
								>
									<p>
										<strong>Note:</strong> Speaker diarization will identify and label different speakers
										in the audio. This may increase processing time but provides more detailed transcriptions.
									</p>
								</div>

								<!-- Speaker Count Configuration -->
								<div class="grid grid-cols-2 gap-4">
									<div class="space-y-2">
										<Label for="min-speakers">Minimum Speakers</Label>
										<Input
											type="number"
											id="min-speakers"
											bind:value={minSpeakers}
											min="1"
											max="10"
											class="w-full"
										/>
									</div>
									<div class="space-y-2">
										<Label for="max-speakers">Maximum Speakers</Label>
										<Input
											type="number"
											id="max-speakers"
											bind:value={maxSpeakers}
											min="1"
											max="10"
											class="w-full"
										/>
									</div>
								</div>

								{#if minSpeakers > maxSpeakers}
									<div
										class="rounded-md border border-red-500/30 bg-red-900/20 p-2 text-sm text-red-300"
									>
										Minimum speakers cannot be greater than maximum speakers.
									</div>
								{/if}
							</div>
						{/if}
					</div>
				</Tabs.Content>

				<Tabs.Content value="advanced" class="max-h-[60vh] space-y-4 overflow-y-auto">
					<div class="grid gap-4 py-4">
						<div class="flex items-center justify-between">
							<h3 class="text-lg font-medium">Advanced Transcription Parameters</h3>
							<Button variant="outline" size="sm" onclick={resetToDefaults}>
								Reset to Defaults
							</Button>
						</div>

						<Separator />

						<!-- Performance Settings -->
						<div class="space-y-4">
							<h4 class="text-md font-medium text-gray-300">Performance Settings</h4>
							<div class="grid grid-cols-2 gap-4">
								<div class="space-y-2">
									<Label for="batch-size">Batch Size</Label>
									<Input
										type="number"
										id="batch-size"
										bind:value={transcriptionParams.batch_size}
										min="1"
										max="32"
										class="w-full"
									/>
									<p class="text-xs text-gray-400">
										Higher values use more memory but may be faster
									</p>
								</div>
								<div class="space-y-2">
									<Label for="compute-type">Compute Type</Label>
									<Select.Root bind:value={transcriptionParams.compute_type} type="single">
										<Select.Trigger class="w-full">
											{transcriptionParams.compute_type}
										</Select.Trigger>
										<Select.Content>
											<Select.Item value="int8">int8 (CPU)</Select.Item>
											<Select.Item value="float16">float16 (GPU)</Select.Item>
											<Select.Item value="float32">float32 (GPU)</Select.Item>
										</Select.Content>
									</Select.Root>
									<p class="text-xs text-gray-400">int8 for CPU, float16/32 for GPU</p>
								</div>
							</div>
						</div>

						<!-- Voice Activity Detection -->
						<div class="space-y-4">
							<h4 class="text-md font-medium text-gray-300">Voice Activity Detection</h4>
							<div class="grid grid-cols-2 gap-4">
								<div class="space-y-2">
									<Label for="vad-onset">VAD Onset Threshold</Label>
									<Input
										type="number"
										id="vad-onset"
										bind:value={transcriptionParams.vad_onset}
										min="0.1"
										max="1.0"
										step="0.1"
										class="w-full"
									/>
									<p class="text-xs text-gray-400">Lower = more sensitive to speech start</p>
								</div>
								<div class="space-y-2">
									<Label for="vad-offset">VAD Offset Threshold</Label>
									<Input
										type="number"
										id="vad-offset"
										bind:value={transcriptionParams.vad_offset}
										min="0.1"
										max="1.0"
										step="0.1"
										class="w-full"
									/>
									<p class="text-xs text-gray-400">Lower = more sensitive to speech end</p>
								</div>
							</div>
						</div>

						<!-- Quality Thresholds -->
						<div class="space-y-4">
							<h4 class="text-md font-medium text-gray-300">Quality Thresholds</h4>
							<div class="grid grid-cols-2 gap-4">
								<div class="space-y-2">
									<Label for="compression-ratio">Compression Ratio Threshold</Label>
									<Input
										type="number"
										id="compression-ratio"
										bind:value={transcriptionParams.compression_ratio_threshold}
										min="0.1"
										max="10.0"
										step="0.1"
										class="w-full"
									/>
									<p class="text-xs text-gray-400">Higher = more aggressive filtering</p>
								</div>
								<div class="space-y-2">
									<Label for="logprob-threshold">Log Probability Threshold</Label>
									<Input
										type="number"
										id="logprob-threshold"
										bind:value={transcriptionParams.logprob_threshold}
										min="-5.0"
										max="0.0"
										step="0.1"
										class="w-full"
									/>
									<p class="text-xs text-gray-400">Lower = more permissive</p>
								</div>
							</div>
							<div class="space-y-2">
								<Label for="no-speech-threshold">No Speech Threshold</Label>
								<Input
									type="number"
									id="no-speech-threshold"
									bind:value={transcriptionParams.no_speech_threshold}
									min="0.1"
									max="1.0"
									step="0.1"
									class="w-full"
								/>
								<p class="text-xs text-gray-400">Higher = more likely to skip silence</p>
							</div>
						</div>

						<!-- Generation Parameters -->
						<div class="space-y-4">
							<h4 class="text-md font-medium text-gray-300">Generation Parameters</h4>
							<div class="grid grid-cols-2 gap-4">
								<div class="space-y-2">
									<Label for="temperature">Temperature</Label>
									<Input
										type="number"
										id="temperature"
										bind:value={transcriptionParams.temperature}
										min="0.0"
										max="2.0"
										step="0.1"
										class="w-full"
									/>
									<p class="text-xs text-gray-400">0.0 = deterministic, higher = more random</p>
								</div>
								<div class="space-y-2">
									<Label for="best-of">Best Of</Label>
									<Input
										type="number"
										id="best-of"
										bind:value={transcriptionParams.best_of}
										min="1"
										max="10"
										class="w-full"
									/>
									<p class="text-xs text-gray-400">Number of candidates to consider</p>
								</div>
							</div>
							<div class="grid grid-cols-2 gap-4">
								<div class="space-y-2">
									<Label for="beam-size">Beam Size</Label>
									<Input
										type="number"
										id="beam-size"
										bind:value={transcriptionParams.beam_size}
										min="1"
										max="10"
										class="w-full"
									/>
									<p class="text-xs text-gray-400">Beam search width</p>
								</div>
								<div class="space-y-2">
									<Label for="patience">Patience</Label>
									<Input
										type="number"
										id="patience"
										bind:value={transcriptionParams.patience}
										min="0.1"
										max="5.0"
										step="0.1"
										class="w-full"
									/>
									<p class="text-xs text-gray-400">Beam search patience</p>
								</div>
							</div>
							<div class="space-y-2">
								<Label for="length-penalty">Length Penalty</Label>
								<Input
									type="number"
									id="length-penalty"
									bind:value={transcriptionParams.length_penalty}
									min="0.1"
									max="5.0"
									step="0.1"
									class="w-full"
								/>
								<p class="text-xs text-gray-400">Penalty for longer sequences</p>
							</div>
						</div>

						<!-- Additional Options -->
						<div class="space-y-4">
							<h4 class="text-md font-medium text-gray-300">Additional Options</h4>
							<div class="space-y-2">
								<Label for="initial-prompt">Initial Prompt (Optional)</Label>
								<Input
									type="text"
									id="initial-prompt"
									bind:value={transcriptionParams.initial_prompt}
									placeholder="Enter context or prompt for transcription..."
									class="w-full"
								/>
								<p class="text-xs text-gray-400">
									Provide context to improve transcription accuracy
								</p>
							</div>
							<div class="space-y-2">
								<Label for="temperature-increment">Temperature Increment on Fallback</Label>
								<Input
									type="number"
									id="temperature-increment"
									bind:value={transcriptionParams.temperature_increment_on_fallback}
									min="0.0"
									max="2.0"
									step="0.1"
									class="w-full"
								/>
								<p class="text-xs text-gray-400">
									Temperature increase when retrying failed segments
								</p>
							</div>
							<div class="flex items-center space-x-2">
								<Checkbox
									id="condition-on-previous"
									bind:checked={transcriptionParams.condition_on_previous_text}
								/>
								<Label for="condition-on-previous">Condition on Previous Text</Label>
							</div>
							<div class="flex items-center space-x-2">
								<Checkbox
									id="suppress-numerals"
									bind:checked={transcriptionParams.suppress_numerals}
								/>
								<Label for="suppress-numerals">Suppress Numerals</Label>
							</div>
						</div>
					</div>
				</Tabs.Content>
			</Tabs.Root>

			<Dialog.Footer class="mt-4">
				<Button
					onclick={handleStartTranscription}
					class="bg-neon-100 hover:bg-neon-200 w-full text-gray-800"
					disabled={enableDiarization && minSpeakers > maxSpeakers}
				>
					Start Transcription
				</Button>
			</Dialog.Footer>
		</div>
	</Dialog.Content>
</Dialog.Root>
