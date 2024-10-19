<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount, afterUpdate } from 'svelte';
	import { Check, CircleCheck } from 'lucide-svelte';
	import { ScrollArea } from 'bits-ui';
	import { Label, Checkbox } from 'bits-ui';
	import { Button, Separator } from 'bits-ui';
	import { Progress } from 'bits-ui';
	import StatusSpinner from '$lib/components/StatusSpinner.svelte';
	import { redirect } from '@sveltejs/kit';

	let message = 'Not yet';
	let value = 0;
	let joblogs = '';

	async function hand() {
		console.log(settings);
		const formData = new FormData();
		formData.append('settings', JSON.stringify(settings));

		const resp = await fetch('/api/jobs/configure', {
			method: 'POST',
			body: formData
		});

		const jobj = await resp.json();
		console.log('JID: ', jobj.jobId);

		await trackJobProgress(jobj.jobId);

		let message;
		if (!resp.ok) {
			message = 'Error';
		} else {
			message = 'Submitted';
		}
	}

	// Function to poll the API for job progress
	async function trackJobProgress(jobId) {
		status = 'progress';
		const interval = 500; // Poll every 2 seconds

		const pollJobStatus = async () => {
			try {
				const resp = await fetch(`/api/jobs/configure/${jobId}`, {
					method: 'GET'
				});

				if (resp.ok) {
					const status = await resp.json();
					console.log('Job progress:', status.progress);
					value = status.progress;
					joblogs = status.logs;
					// console.log(joblogs);

					// If job is complete, stop polling
					if (status.progress === 100) {
						console.log('Job complete');
						clearInterval(polling);
					}
				} else {
					console.error('Failed to retrieve job status');
				}
			} catch (error) {
				console.error('Error:', error);
			}
		};

		// Set up polling at defined intervals
		const polling = setInterval(pollJobStatus, interval);
	}

	let status = 'wizard';

	$: if (value >= 100) {
		setTimeout(() => {
			status = 'done';
			goto('/');
		}, 1000);
	}

	let settings = {
		models: {
			small: false,
			tiny: false,
			medium: false,
			largev1: false,
			largev2: false,
			largev3: false,
			languages: {
				eng: false,
				others: false
			}
		}
	};

	let scrollArea;
	afterUpdate(() => {
		if (scrollArea) {
			scrollArea.scrollTop = scrollArea.scrollTopMax;
		}
	});
</script>

<div class="p-4">
	{#if status === 'wizard'}
		<div class="text-2xl">Configuration Wizard</div>
		<div class="my-4 flex items-start justify-start gap-2">
			<div class="flex w-[200px] flex-col items-start gap-2 rounded-md bg-carbongray-50 p-2">
				<div>Language</div>
				<Separator.Root
					class="shrink-0 bg-carbongray-100 data-[orientation=horizontal]:h-px data-[orientation=vertical]:h-full data-[orientation=horizontal]:w-full data-[orientation=vertical]:w-[1px]"
				/>
				<div class="flex items-center justify-center gap-2">
					<Checkbox.Root
						id="english"
						aria-labelledby="terms-label"
						class="bg-foreground active:scale-98 data-[state=unchecked]:border-border-input data-[state=unchecked]:bg-background data-[state=unchecked]:hover:border-dark-40 peer inline-flex size-[15px] items-center justify-center rounded-sm border border-carbongray-400 transition-all duration-150 ease-in-out"
						bind:checked={settings.models.languages.eng}
					>
						<Checkbox.Indicator
							let:isChecked
							let:isIndeterminate
							class="text-background inline-flex items-center justify-center"
						>
							{#if isChecked}
								<Check size={12} />
							{/if}
						</Checkbox.Indicator>
					</Checkbox.Root>
					<Label.Root id="light-label" for="light">English</Label.Root>
				</div>
				<div class="flex items-center justify-center gap-2">
					<Checkbox.Root
						id="others"
						aria-labelledby="terms-label"
						class="bg-foreground active:scale-98 data-[state=unchecked]:border-border-input data-[state=unchecked]:bg-background data-[state=unchecked]:hover:border-dark-40 peer inline-flex size-[15px] items-center justify-center rounded-sm border border-carbongray-400 transition-all duration-150 ease-in-out"
						bind:checked={settings.models.languages.eng}
					>
						<Checkbox.Indicator
							let:isChecked
							let:isIndeterminate
							class="text-background inline-flex items-center justify-center"
						>
							{#if isChecked}
								<Check size={12} />
							{/if}
						</Checkbox.Indicator>
					</Checkbox.Root>
					<Label.Root id="light-label" for="light">Others</Label.Root>
				</div>
			</div>
			<div class="flex w-[200px] flex-col items-start gap-2 rounded-md bg-carbongray-50 p-2">
				<div>Models to download</div>
				<Separator.Root
					class="shrink-0 bg-carbongray-100 data-[orientation=horizontal]:h-px data-[orientation=vertical]:h-full data-[orientation=horizontal]:w-full data-[orientation=vertical]:w-[1px]"
				/>
				<div class="flex items-center justify-center gap-2">
					<Checkbox.Root
						id="small"
						aria-labelledby="terms-label"
						class="bg-foreground active:scale-98 data-[state=unchecked]:border-border-input data-[state=unchecked]:bg-background data-[state=unchecked]:hover:border-dark-40 peer inline-flex size-[15px] items-center justify-center rounded-sm border border-carbongray-400 transition-all duration-150 ease-in-out"
						bind:checked={settings.models.small}
					>
						<Checkbox.Indicator
							let:isChecked
							let:isIndeterminate
							class="text-background inline-flex items-center justify-center"
						>
							{#if isChecked}
								<Check size={12} />
							{/if}
						</Checkbox.Indicator>
					</Checkbox.Root>
					<Label.Root id="light-label" for="light">Small</Label.Root>
				</div>
				<div class="flex items-center justify-center gap-2">
					<Checkbox.Root
						id="tiny"
						aria-labelledby="terms-label"
						class="bg-foreground active:scale-98 data-[state=unchecked]:border-border-input data-[state=unchecked]:bg-background data-[state=unchecked]:hover:border-dark-40 peer inline-flex size-[15px] items-center justify-center rounded-sm border border-carbongray-400 transition-all duration-150 ease-in-out"
						bind:checked={settings.models.tiny}
					>
						<Checkbox.Indicator
							let:isChecked
							let:isIndeterminate
							class="text-background inline-flex items-center justify-center"
						>
							{#if isChecked}
								<Check size={12} />
							{/if}
						</Checkbox.Indicator>
					</Checkbox.Root>
					<Label.Root id="light-label" for="light">Tiny</Label.Root>
				</div>
				<div class="flex items-center justify-center gap-2">
					<Checkbox.Root
						id="medium"
						aria-labelledby="terms-label"
						class="bg-foreground active:scale-98 data-[state=unchecked]:border-border-input data-[state=unchecked]:bg-background data-[state=unchecked]:hover:border-dark-40 peer inline-flex size-[15px] items-center justify-center rounded-sm border border-carbongray-400 transition-all duration-150 ease-in-out"
						bind:checked={settings.models.medium}
					>
						<Checkbox.Indicator
							let:isChecked
							let:isIndeterminate
							class="text-background inline-flex items-center justify-center"
						>
							{#if isChecked}
								<Check size={12} />
							{/if}
						</Checkbox.Indicator>
					</Checkbox.Root>
					<Label.Root id="light-label" for="light">Medium</Label.Root>
				</div>
				<div class="flex items-center justify-center gap-2">
					<Checkbox.Root
						id="largev1"
						aria-labelledby="terms-label"
						class="bg-foreground active:scale-98 data-[state=unchecked]:border-border-input data-[state=unchecked]:bg-background data-[state=unchecked]:hover:border-dark-40 peer inline-flex size-[15px] items-center justify-center rounded-sm border border-carbongray-400 transition-all duration-150 ease-in-out"
						bind:checked={settings.models.large}
					>
						<Checkbox.Indicator
							let:isChecked
							let:isIndeterminate
							class="text-background inline-flex items-center justify-center"
						>
							{#if isChecked}
								<Check size={12} />
							{/if}
						</Checkbox.Indicator>
					</Checkbox.Root>
					<Label.Root id="light-label" for="light">Large-V1</Label.Root>
				</div>
				<div class="flex items-center justify-center gap-2">
					<Checkbox.Root
						id="largev2"
						aria-labelledby="terms-label"
						class="bg-foreground active:scale-98 data-[state=unchecked]:border-border-input data-[state=unchecked]:bg-background data-[state=unchecked]:hover:border-dark-40 peer inline-flex size-[15px] items-center justify-center rounded-sm border border-carbongray-400 transition-all duration-150 ease-in-out"
						bind:checked={settings.models.large}
					>
						<Checkbox.Indicator
							let:isChecked
							let:isIndeterminate
							class="text-background inline-flex items-center justify-center"
						>
							{#if isChecked}
								<Check size={12} />
							{/if}
						</Checkbox.Indicator>
					</Checkbox.Root>
					<Label.Root id="light-label" for="light">Large-V2</Label.Root>
				</div>
				<div class="flex items-center justify-center gap-2">
					<Checkbox.Root
						id="largev3"
						aria-labelledby="terms-label"
						class="bg-foreground active:scale-98 data-[state=unchecked]:border-border-input data-[state=unchecked]:bg-background data-[state=unchecked]:hover:border-dark-40 peer inline-flex size-[15px] items-center justify-center rounded-sm border border-carbongray-400 transition-all duration-150 ease-in-out"
						bind:checked={settings.models.large}
					>
						<Checkbox.Indicator
							let:isChecked
							let:isIndeterminate
							class="text-background inline-flex items-center justify-center"
						>
							{#if isChecked}
								<Check size={12} />
							{/if}
						</Checkbox.Indicator>
					</Checkbox.Root>
					<Label.Root id="light-label" for="light">Large-V3</Label.Root>
				</div>
			</div>
		</div>
		<Button.Root
			on:click={hand}
			class="my-3 flex h-[30px] items-center justify-center rounded-md bg-black p-1 text-base text-carbongray-50"
			>Configure</Button.Root
		>
	{:else if status === 'progress'}
		<div class="text-2xl">Configuration Wizard</div>
		<div class="">
			<Progress.Root
				bind:value
				max={100}
				class="relative h-3 w-full overflow-hidden rounded-full bg-carbongray-100"
			>
				<div
					class="h-full w-full bg-carbongray-700 transition-all duration-1000 ease-in-out"
					style={`transform: translateX(-${100 - (100 * (value ?? 0)) / 100}%)`}
				></div>
			</Progress.Root>
			<div class="my-8 flex items-start">
				<div class="flex w-[300px] flex-shrink-0 flex-col justify-center gap-2">
					{#if value < 50}
						<StatusSpinner msg={'Compiling Whisper.cpp'} pos={'start'} />
					{:else if value > 50 && value < 100}
						<StatusSpinner msg={'Compiled Whisper.cpp'} pos={'start'} success="1" />
						<StatusSpinner msg={'Downloading whisper models'} pos="start" />
					{:else}
						<StatusSpinner msg={'Compiled Whisper.cpp'} pos={'start'} success="1" />
						<StatusSpinner msg={'Downloaded whisper models'} pos="start" success="1" />
					{/if}
				</div>
				<div class="rounded-md bg-carbongray-50 p-2 font-mono text-carbongray-600">
					<ScrollArea.Root class="h-[480px] w-full">
						<ScrollArea.Viewport class="h-full w-full" bind:el={scrollArea}>
							<ScrollArea.Content>
								{#each joblogs as log}
									<div class="text-sm">{log}</div>
								{/each}
							</ScrollArea.Content>
						</ScrollArea.Viewport>
						<ScrollArea.Scrollbar orientation="vertical">
							<ScrollArea.Thumb />
						</ScrollArea.Scrollbar>
						<ScrollArea.Corner />
					</ScrollArea.Root>
				</div>
			</div>
		</div>
	{:else}
		<div class="flex h-full w-full items-center justify-center">
			<div class="flex h-full w-full flex-col items-center justify-center">
				<CircleCheck size={40} />
				<div class="text-lg">Done</div>
			</div>
		</div>
	{/if}
</div>
