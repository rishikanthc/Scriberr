<script lang="ts">
	import { Loader2 } from 'lucide-svelte';
	import * as Card from '$lib/components/ui/card';
	import { Progress } from '$lib/components/ui/progress/index.js';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import * as Switch from '$lib/components/ui/switch';
	import * as Select from '$lib/components/ui/select';
	import { Label } from '$lib/components/ui/label';
	import { Input } from '$lib/components/ui/input';
	import { createEventDispatcher } from 'svelte';

	// Set up event dispatcher for parent component communication
	const dispatch = createEventDispatcher();

	type ModelSize = 'tiny' | 'base' | 'small' | 'medium' | 'large' | 'large-v2' | 'large-v3';
	type LargeVersion = 'v1' | 'v2' | 'v3' | 'v3-turbo';

	interface ConfigOptions {
		modelSizes: ModelSize[];
		largeVersion?: LargeVersion;
		multilingual: boolean;
		quantization: 'none' | 'q5' | 'q8';
		hfApiKey?: string; // HuggingFace API key for diarization
	}

	let status = $state<'initial' | 'installing' | 'complete' | 'error'>('initial');
	let progress = $state(0);
	let log = $state<string[]>([]);
	let eventSource = $state<EventSource | null>(null);

	let config = $state<ConfigOptions>({
		modelSizes: ['base'], // Initialize with base model selected
		multilingual: false,
		quantization: 'none',
		hfApiKey: 'hf_DEBUGMODE' // Set a dummy key for testing
	});
	
	// TEMP: Set diarization to false by default for testing
	let enableDiarization = $state(false);

	let modelNames: string[] = $state([]);
	$effect(() => {
		modelNames = config.modelSizes.map((size) => {
			const lang = config.multilingual ? '' : '.en';
			let quant = '';

			if (config.quantization === 'q5') {
				quant = size === 'medium' || (size === 'large' && config.largeVersion) ? '-q5_0' : '-q5_1';
			} else if (config.quantization === 'q8') {
				quant = '-q8_0';
			}

			if (size === 'large') {
				const version = config.largeVersion || 'v1';
				return `large-${version}${quant}`;
			}

			// Special handling for large-v2 and large-v3
			if (size === 'large-v2' || size === 'large-v3') {
				return `${size}${quant}`;
			}
			return `${size}${lang}${quant}`;
		});
	});

	// Validate if HF API key is needed
	let needsHfKey = $derived(enableDiarization && !config.hfApiKey);

	$effect(() => {
		// When status changes to complete, notify parent component
		if (status === 'complete') {
			console.log("Setup complete, notifying parent component");
			dispatch('setupcomplete', { complete: true });
		} else if (status === 'error') {
			console.log("Setup error, notifying parent component");
			dispatch('setupcomplete', { complete: false });
		}
	});

	// Function for installation button
	function startInstallation() {
		console.log("INSTALLATION BUTTON CLICKED");
		log = [...log, "Installation started..."];
		
		// Set up installation
		status = 'installing';
		progress = 0;
		
		// Build URL with parameters
		const params = new URLSearchParams({
			models: JSON.stringify(modelNames),
			multilingual: config.multilingual.toString(),
			diarization: enableDiarization.toString(),
			compute_type: config.quantization === 'none' ? 'float32' : 'int8'
		});

		// Add HF API key if provided
		if (config.hfApiKey) {
			params.append('hf_api_key', config.hfApiKey);
		}
		
		// Start direct installation with params
		fetch(`/api/direct-setup?${params}`)
			.then(response => {
				console.log("Installation API response:", response.status);
				log = [...log, `API response status: ${response.status}`];
				return response.json();
			})
			.then(data => {
				console.log("Installation response data:", data);
				
				// Process model download results
				if (data.modelResults) {
					data.modelResults.forEach(result => {
						if (result.success) {
							log = [...log, `✅ Downloaded model: ${result.model}`];
						} else {
							log = [...log, `❌ Failed to download model ${result.model}: ${result.error}`];
						}
					});
				}
				
				// Process diarization results if applicable
				if (data.diarizationResult) {
					if (data.diarizationResult.success) {
						log = [...log, "✅ Downloaded diarization model successfully"];
					} else {
						log = [...log, `❌ Failed to download diarization model: ${data.diarizationResult.error}`];
					}
				}
				
				if (data.success) {
					status = 'complete';
					progress = 100;
					log = [...log, "Setup completed successfully!"];
					dispatch('setupcomplete', { complete: true });
				} else {
					status = 'error';
					log = [...log, `Setup failed: ${data.error || "Unknown error"}`];
					dispatch('setupcomplete', { complete: false });
				}
			})
			.catch(error => {
				console.error("Installation error:", error);
				status = 'error';
				log = [...log, `Error: ${error.message}`];
				dispatch('setupcomplete', { complete: false });
			});
	}

	// Cleanup on component destroy
	$effect.root(() => {
		return () => {
			if (eventSource) {
				eventSource.close();
				eventSource = null;
			}
		};
	});
</script>

<div class="space-y-6">
	<div class="space-y-2">
		<p class="text-sm text-muted-foreground">
			Configure and install whisper.cpp for speech recognition
		</p>
	</div>

	{#if status === 'installing'}
		<div class="space-y-4">
			<div class="flex items-center space-x-2">
				<Loader2 class="h-4 w-4 animate-spin" />
				<span>Installing Whisper...</span>
			</div>
			<Progress value={progress} />
		</div>
	{/if}

	{#if status === 'initial'}
		<Card.Root>
			<Card.Content class="space-y-6 pt-6">
				<!-- Model Size Selection -->
				<div class="space-y-2">
					<Label for="model-sizes">Model Sizes</Label>
					<div id="model-sizes" class="grid gap-2">
						{#each ['tiny', 'base', 'small', 'medium', 'large', 'large-v2', 'large-v3'] as size}
							<div class="flex items-center space-x-2">
								<Switch.Root
									id={`model-${size}`}
									name={`model-${size}`}
									checked={config.modelSizes.includes(size)}
									onCheckedChange={(checked) => {
										if (checked) {
											config.modelSizes = [...config.modelSizes, size];
										} else {
											config.modelSizes = config.modelSizes.filter((s) => s !== size);
										}
									}}
								/>
								<Label for={`model-${size}`}>
									{size}
									{#if size === 'tiny'}(75MB)
									{:else if size === 'base'}(142MB)
									{:else if size === 'small'}(466MB)
									{:else if size === 'medium'}(1.5GB)
									{:else if size === 'large'}(2.9GB)
									{:else if size === 'large-v2'}(3.1GB)
									{:else if size === 'large-v3'}(3.6GB){/if}
								</Label>
							</div>
						{/each}
					</div>
				</div>

				<!-- Multilingual Switch -->
				<div class="flex items-center justify-between">
					<div class="space-y-1">
						<Label for="multilingual-switch">Multilingual Support</Label>
						<p class="text-sm text-muted-foreground">Enable support for multiple languages</p>
					</div>
					<Switch.Root
						id="multilingual-switch"
						name="multilingual"
						checked={config.multilingual}
						onCheckedChange={(checked) => (config.multilingual = checked)}
					/>
				</div>

				<!-- Diarization Switch -->
				<div class="flex items-center justify-between">
					<div class="space-y-1">
						<Label for="diarization-switch">Speaker Diarization</Label>
						<p class="text-sm text-muted-foreground">Install model to identify different speakers</p>
					</div>
					<Switch.Root
						id="diarization-switch"
						name="diarization"
						checked={enableDiarization}
						onCheckedChange={(checked) => (enableDiarization = checked)}
					/>
				</div>

				<!-- HuggingFace API Key Input (visible only when diarization is enabled) -->
				{#if enableDiarization}
					<div class="space-y-2">
						<Label for="hf-api-key" class="text-sm font-medium">HuggingFace API Key <span class="text-red-500">*</span></Label>
						<p class="text-xs text-muted-foreground mb-2">
							Required for diarization model download. Get your free API key at 
							<a href="https://huggingface.co/settings/tokens" class="text-blue-600 hover:underline" target="_blank" rel="noopener noreferrer">huggingface.co/settings/tokens</a>
						</p>
						<Input
							id="hf-api-key"
							name="hfApiKey"
							type="password"
							bind:value={config.hfApiKey}
							placeholder="hf_..."
							class="w-full"
							aria-required="true"
						/>
					</div>
				{/if}

				<!-- Quantization Options -->
				<div class="space-y-2">
					<Label for="quantization-select">Quantization</Label>
					<Select.Root
						id="quantization-select"
						name="quantization"
						value={config.quantization}
						onValueChange={(value) => (config.quantization = value as ConfigOptions['quantization'])}
						defaultValue="none"
					>
						<Select.Trigger id="quantization-trigger" placeholder="Select quantization">
							{#if config.quantization === 'none'}
								None - Full precision
							{:else if config.quantization === 'q5'}
								Q5 - Reduced size, slightly lower quality
							{:else if config.quantization === 'q8'}
								Q8 - Minimal quality loss
							{/if}
						</Select.Trigger>
						<Select.Content>
							<Select.Item value="none">None - Full precision</Select.Item>
							<Select.Item value="q5">Q5 - Reduced size, slightly lower quality</Select.Item>
							<Select.Item value="q8">Q8 - Minimal quality loss</Select.Item>
						</Select.Content>
					</Select.Root>
				</div>

				<div class="pt-4">
					<div class="mb-4 text-sm text-muted-foreground">
						Selected models:
						<ul class="mt-2 list-disc pl-4">
							{#each modelNames as model}
								<li>{model}</li>
							{/each}
							{#if enableDiarization}
								<li>Speaker diarization model</li>
							{/if}
						</ul>
					</div>
					
					<!-- Installation button -->
					<button
						id="install-button"
						type="button"
						class="bg-primary text-primary-foreground inline-flex h-9 w-full items-center justify-center rounded-md px-4 py-2 text-sm font-medium shadow transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50"
						on:click={startInstallation}
					>
						Install Selected Models
					</button>
						
					{#if enableDiarization && !config.hfApiKey}
						<p class="text-xs text-red-500 mt-2">HuggingFace API key is required for diarization</p>
					{/if}
				</div>
			</Card.Content>
		</Card.Root>
	{:else}
		<div class="h-48 overflow-y-auto rounded-md border bg-muted p-4 font-mono text-sm">
			{#each log as message}
				<div class="py-1">
					{message}
				</div>
			{/each}
		</div>
	{/if}

	{#if status === 'complete'}
		<Alert variant="default">
			<AlertDescription class="font-medium text-green-600">
				Setup completed successfully!
			</AlertDescription>
		</Alert>
	{/if}

	{#if status === 'error'}
		<Alert variant="destructive">
			<AlertDescription>
				An error occurred during setup. Please check the logs above.
			</AlertDescription>
		</Alert>
	{/if}
</div>