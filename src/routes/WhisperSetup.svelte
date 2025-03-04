<script lang="ts">
	import { Loader2 } from 'lucide-svelte';
	import * as Card from '$lib/components/ui/card';
	import { Progress } from '$lib/components/ui/progress/index.js';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import * as Switch from '$lib/components/ui/switch';
	import * as Select from '$lib/components/ui/select';
	import { Label } from '$lib/components/ui/label';
	// import { apiFetch, createEventSource } from '$lib/api';

	type ModelSize = 'tiny' | 'base' | 'small' | 'medium' | 'large';
	type LargeVersion = 'v1' | 'v2' | 'v3' | 'v3-turbo';

	interface ConfigOptions {
		modelSizes: ModelSize[];
		largeVersion?: LargeVersion;
		multilingual: boolean;
		quantization: 'none' | 'q5' | 'q8';
	}

	let status = $state<'initial' | 'installing' | 'complete' | 'error'>('initial');
	let progress = $state(0);
	let log = $state<string[]>([]);
	let eventSource = $state<EventSource | null>(null);

	let config = $state<ConfigOptions>({
		modelSizes: ['base'], // Initialize with base model selected
		multilingual: false,
		quantization: 'none'
	});

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

	async function startSetup() {
		status = 'installing';
		progress = 0;
		log = [];

		try {
			const params = new URLSearchParams({
				models: JSON.stringify(modelNames),
				multilingual: config.multilingual.toString()
			});

			eventSource = new EventSource(`/api/setup/whisper?${params}`);

			eventSource.onmessage = (event) => {
				const data = JSON.parse(event.data);
				log = [...log, data.message];
				console.log('LOGS --->', log);
				if (data.progress) progress = data.progress;
				if (data.status === 'complete') {
					eventSource.close();
					status = 'complete';
					eventSource = null;
				} else if (data.status === 'error') {
					eventSource.close();
					status = 'error';
					eventSource = null;
				}
			};

			eventSource.onerror = () => {
				eventSource.close();
				status = 'error';
				eventSource = null;
			};
		} catch (error) {
			status = 'error';
			log = [...log, `Error: ${error.message}`];
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
					<Label>Model Sizes</Label>
					<div class="grid gap-2">
						{#each ['tiny', 'base', 'small', 'medium', 'large'] as size}
							<div class="flex items-center space-x-2">
								<Switch.Root
									checked={config.modelSizes.includes(size)}
									onCheckedChange={(checked) => {
										if (checked) {
											config.modelSizes = [...config.modelSizes, size];
										} else {
											config.modelSizes = config.modelSizes.filter((s) => s !== size);
										}
									}}
								/>
								<Label>
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
						<Label>Multilingual Support</Label>
						<p class="text-sm text-muted-foreground">Enable support for multiple languages</p>
					</div>
					<Switch.Root
						checked={config.multilingual}
						onCheckedChange={(checked) => (config.multilingual = checked)}
					/>
				</div>

				<!-- Quantization Options -->
				<div class="space-y-2">
					<Label>Quantization</Label>
					<Select.Root
						value={config.quantization}
						onValueChange={(value) =>
							(config.quantization = value as ConfigOptions['quantization'])}
					>
						<Select.Trigger placeholder="Select quantization">
							{config.quantization}
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
						</ul>
					</div>
					<Button
						variant="default"
						class="w-full"
						onclick={startSetup}
						disabled={config.modelSizes.length === 0}
					>
						Start Installation
					</Button>
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
