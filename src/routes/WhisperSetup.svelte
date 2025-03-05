<script lang="ts">
	import { Loader2 } from 'lucide-svelte';
	import * as Card from '$lib/components/ui/card';
	import { Progress } from '$lib/components/ui/progress/index.js';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import * as Switch from '$lib/components/ui/switch';
	import * as Select from '$lib/components/ui/select';
	import { Label } from '$lib/components/ui/label';
	import { Input } from '$lib/components/ui/input';
	import { createEventDispatcher } from 'svelte';
	// import { apiFetch, createEventSource } from '$lib/api';

	// Set up event dispatcher for parent component communication
	const dispatch = createEventDispatcher();

	type ModelSize = 'tiny' | 'base' | 'small' | 'medium' | 'large';
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
			log = [...log, "Setup encountered an error."];
			dispatch('setupcomplete', { complete: false });
		}
	});

	async function startSetup() {
		// Log button press and state for debugging
		log = [...log, "START SETUP BUTTON PRESSED!"];
		log = [...log, "Current config:", JSON.stringify({
			modelSizes: config.modelSizes,
			multilingual: config.multilingual,
			enableDiarization: enableDiarization,
			hfApiKey: config.hfApiKey ? "provided" : "not provided",
			quantization: config.quantization
		})];
		
		// Re-enable validation for HuggingFace API key
		if (enableDiarization && !config.hfApiKey) {
			log = [...log, "HuggingFace API key is required for diarization. Please enter a valid API key."];
			return;
		}

		log = [...log, "Starting setup..."];
		status = 'installing';
		progress = 0;
		dispatch('setupcomplete', { complete: false });

		try {
			const params = new URLSearchParams({
				models: JSON.stringify(modelNames),
				multilingual: config.multilingual.toString(),
			 диarization: enableDiarization.toString(),
				compute_type: config.quantization === 'none' ? 'float32' : 'int8'
			});

			// Add HF API key if provided
			if (config.hfApiKey) {
				params.append('hf_api_key', config.hfApiKey);
			}

			// Also try a direct fetch to see if the endpoint is reachable
			try {
				const testFetch = await fetch(`/api/setup/whisper?${params}`, { method: 'HEAD' });
				log = [...log, `API endpoint is reachable (status: ${testFetch.status})`];
			} catch (fetchError) {
				log = [...log, `Warning: API endpoint test failed: ${fetchError.message}`];
			}

			log = [...log, "Starting installation with params:", JSON.stringify(Object.fromEntries(params.entries()))];
			log = [...log, "Connecting to server..."];

			try {
				eventSource = new EventSource(`/api/setup/whisper?${params}`);
				log = [...log, "EventSource created with readyState:", eventSource.readyState];
				
				// Add onopen handler to verify connection is established
				eventSource.onopen = (event) => {
					log = [...log, "EventSource connection opened successfully"];
				};
			} catch (esError) {
				log = [...log, `Failed to create EventSource: ${esError.message}`];
				status = 'error';
				dispatch('setupcomplete', { complete: false });
				return;
			}

			eventSource.onmessage = (event) => {
				try {
					const data = JSON.parse(event.data);
					log = [...log, data.message];
					
					if (data.progress) progress = data.progress;
					
					if (data.status === 'complete') {
						log = [...log, "Received COMPLETE status"];
						eventSource.close();
						status = 'complete';
						dispatch('setupcomplete', { complete: true });
						eventSource = null;
					} else if (data.status === 'error') {
						log = [...log, "Received ERROR status"];
						log = [...log, `Error: ${data.message}`];
						eventSource.close();
						status = 'error';
						dispatch('setupcomplete', { complete: false });
						eventSource = null;
					}
				} catch (error) {
					log = [...log, `Error processing EventSource message: ${error.message}`];
				}
			};

			eventSource.onerror = (error) => {
				log = [...log, `EventSource error: ${error.message}`];
				log = [...log, `EventSource readyState: ${eventSource.readyState}`];
				log = [...log, `Connection error: The server might be unavailable`];
				
				// Try to get more error details
				if (error instanceof Event) {
					log = [...log, `Error type: ${error.type}`];
				}
				
				eventSource.close();
				status = 'error';
				dispatch('setupcomplete', { complete: false });
				eventSource = null;
			};
			
			// Set a timeout to detect if connection is not established
			setTimeout(() => {
				if (eventSource && eventSource.readyState !== 1) { // 1 = OPEN
					log = [...log, "EventSource connection not established after timeout"];
					log = [...log, "Timeout: Connection to server not established"];
					eventSource.close();
					status = 'error';
					dispatch('setupcomplete', { complete: false });
					eventSource = null;
				}
			}, 5000);
			
		} catch (error) {
			log = [...log, `Setup error: ${error.message}`];
			status = 'error';
			dispatch('setupcomplete', { complete: false });
		}
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
						{#each ['tiny', 'base', 'small', 'medium', 'large'] as size}
							<div class="flex items-center space-x-2">
								<Switch.Root
									id={`model-${size}`}
									name={`model-${size}`}
									checked={config.modelSizes.includes(size)}
								(onCheckedChange={(checked) => {
										if (checked) {
											config.modelSizes = [...config.modelSizes, size];
										} else {
											config.modelSizes = config.modelSizes.filter((s) => s !== size);
										}
									}}
								/>
								<Label for={`model-${size}`}>
									{size}
									{#if size === 'tiny'}(75MB){:else if size === 'base'}(142MB)
									{:else if size === 'small'}(466MB){:else if size === 'medium'}(1.5GB)
									{:else}(2.9GB+){/if}
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
						(onCheckedChange={(checked) => (config.multilingual = checked)}
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
						(onCheckedChange={(checked) => (enableDiarization = checked)}
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
						(onValueChange={(value) => (config.quantization = value as ConfigOptions['quantization'])}
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
					<!-- Re-enable validation for debugging -->
					<Button
						id="start-installation"
						name="start-installation"
						type="button"
						variant="default"
						class="w-full"
						(on:click={startSetup}
						disabled={enableDiarization && !config.hfApiKey}
					>
						Start Installation
					</Button>
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
