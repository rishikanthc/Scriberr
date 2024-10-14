<script lang="ts">
	import { Slider } from 'bits-ui';
	import { Select } from 'bits-ui';
	import { Label } from 'bits-ui';
	import { Check, CircleCheck, CircleX, Loader } from 'lucide-svelte';
	import { onMount, tick } from 'svelte';

	const models = [
		{ value: 'small', label: 'Small' },
		{ value: 'tiny', label: 'Tiny' },
		{ value: 'base', label: 'Base' },
		{ value: 'medium', label: 'Medium' }
	];
	const summaryModels = [
		{ value: 'gpt-4o', label: 'GPT-4O' },
		{ value: 'gpt-4o-mini', label: 'GPT-4O Mini' },
		{ value: 'gpt-4o-turbo', label: 'GPT-4O Turbo' },
		{ value: 'gpt-3.5-turbo-0125', label: 'GPT-3.5 Turbo' }
	];

	let currSettings;
	let selWhisperModel;
	let selSummaryModel;
	let cpus;
	let threads;

	async function getSettings() {
		const response = await fetch('/api/settings', {
			method: 'GET'
		});

		const currSetting = await response.json();
		return currSetting;
	}

	let initLoad = false;
	onMount(async () => {
		currSettings = await getSettings();
		console.log(currSettings);
		threads = [currSettings.threads];
		cpus = [currSettings.processors];
		selWhisperModel = models.filter((rec) => {
			return rec.value === currSettings.model;
		})[0];
		selSummaryModel = summaryModels.filter((rec) => {
			return rec.value === currSettings.default_openai_model;
		})[0];
		console.log(selWhisperModel);
		await tick();
		initLoad = true;
	});

	let newSettings;
	$: updateSettings(selWhisperModel, selSummaryModel, threads, cpus);
	let updating = false;
	let success = false;
	let error = false;

	async function updateSettings(whisper, summModel, threads, cpus) {
		if (!initLoad) {
			updating = false;
			console.log('settings not loaded yet');
		} else {
			const settingsUpd = {
				model: whisper.value,
				default_openai_model: summModel.value,
				threads: threads[0],
				processors: cpus[0]
			};

			try {
				updating = true;
				const response = await fetch('/api/settings', {
					method: 'POST',
					body: JSON.stringify(settingsUpd)
				});
				updating = false;
				if (!response.ok) {
					throw new Error('Request to update settings failed');
					error = true;
				} else {
					success = true;
				}
			} catch (error) {
				updating = false;
				console.error('Error updating settings:', error);
			} finally {
				// Add 1-second delay before resetting flags
				await new Promise((resolve) => setTimeout(resolve, 1000));
				error = false;
				success = false;
			}
		}
		currSettings = await getSettings();
		updating = false;
	}
</script>

<div class="flex h-full flex-grow flex-col gap-6 p-2">
	<div>
		<Label.Root
			id="select-model"
			for="model"
			class="text-sm peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
		>
			Summarizer Model
		</Label.Root>
		<Select.Root items={summaryModels} bind:selected={selSummaryModel} disabled={updating}>
			<Select.Trigger
				class="bg-background placeholder:text-foreground-alt/50 focus:ring-foreground focus:ring-offset-background inline-flex h-[35px] w-full items-center rounded-md border border-carbongray-200 px-[11px] text-sm  transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 dark:border-carbongray-600"
				aria-label="Select a theme"
			>
				<Select.Value class="text-sm" placeholder="Select a theme" />
			</Select.Trigger>
			<Select.Content
				class="shadow-popover w-full rounded-xl  border border-carbongray-100 bg-white px-1 py-3 outline-none dark:border-carbongray-700 dark:bg-carbongray-800"
				sideOffset={8}
			>
				{#each summaryModels as model}
					<Select.Item
						class="flex h-10 w-full select-none items-center rounded-md py-3 pl-5 pr-1.5 text-sm outline-none transition-all duration-75 data-[highlighted]:bg-carbongray-50 dark:data-[highlighted]:bg-carbongray-700"
						value={model.value}
						label={model.label}
					>
						{model.label}
						<Select.ItemIndicator class="ml-auto" asChild={false}>
							<Check />
						</Select.ItemIndicator>
					</Select.Item>
				{/each}
			</Select.Content>
		</Select.Root>
	</div>
	<div>
		<Label.Root
			id="select-model"
			for="model"
			class="text-sm peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
		>
			Whisper Model
		</Label.Root>
		<Select.Root items={models} bind:selected={selWhisperModel} disabled={updating}>
			<Select.Trigger
				class="bg-background placeholder:text-foreground-alt/50 focus:ring-foreground focus:ring-offset-background inline-flex h-[35px] w-full items-center rounded-md border border-carbongray-200 px-[11px] text-sm  transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 dark:border-carbongray-600"
				aria-label="Select a theme"
			>
				<Select.Value class="text-sm" placeholder="Select a theme" />
			</Select.Trigger>
			<Select.Content
				class="shadow-popover w-full rounded-xl  border border-carbongray-50 bg-white px-1 py-3 outline-none dark:border-carbongray-700 dark:bg-carbongray-800"
				sideOffset={8}
			>
				{#each models as theme}
					<Select.Item
						class="rounded-button flex h-10 w-full select-none items-center py-3 pl-5 pr-1.5 text-sm outline-none transition-all duration-75 data-[highlighted]:bg-carbongray-50 dark:data-[highlighted]:bg-carbongray-700"
						value={theme.value}
						label={theme.label}
					>
						{theme.label}
						<Select.ItemIndicator class="ml-auto" asChild={false}>
							<Check />
						</Select.ItemIndicator>
					</Select.Item>
				{/each}
			</Select.Content>
		</Select.Root>
	</div>
	<div class="flex items-center justify-between">
		<div class="flex w-[80%] flex-col justify-center gap-2">
			<Label.Root
				id="terms-label"
				for="terms"
				class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
			>
				# Processors
			</Label.Root>
			<Slider.Root
				bind:value={cpus}
				min={1}
				max={10}
				step={1}
				let:thumbs
				class="relative flex w-full touch-none select-none items-center"
				disabled={updating}
			>
				<span
					class="relative h-2 w-full grow overflow-hidden rounded-full bg-carbongray-100 dark:bg-carbongray-600"
				>
					<Slider.Range class="absolute h-full bg-black dark:bg-carbonblue-500" />
				</span>
				{#each thumbs as thumb}
					<Slider.Thumb
						{thumb}
						class="border-border-input active:scale-98 block size-[20px] cursor-pointer rounded-full border  bg-carbongray-600 shadow transition-colors hover:border-carbongray-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-carbongray-600 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 dark:bg-carbongray-700"
					/>
				{/each}
			</Slider.Root>
		</div>
		<div
			class="flex h-[20px] w-[20px] items-center justify-center rounded-sm bg-white p-1 text-base dark:bg-carbongray-800"
		>
			{#if cpus}
				{cpus[0]}
			{/if}
		</div>
	</div>

	<div class="flex items-center justify-between">
		<div class="flex w-[80%] flex-col justify-center gap-2">
			<Label.Root
				id="terms-label"
				for="terms"
				class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
			>
				# Threads
			</Label.Root>
			<Slider.Root
				bind:value={threads}
				min={1}
				max={10}
				step={1}
				let:thumbs
				class="relative flex w-full touch-none select-none items-center"
				disabled={updating}
			>
				<span
					class="relative h-2 w-full grow overflow-hidden rounded-full bg-carbongray-100 dark:bg-carbongray-600"
				>
					<Slider.Range class="absolute h-full bg-black dark:bg-carbonblue-500" />
				</span>
				{#each thumbs as thumb}
					<Slider.Thumb
						{thumb}
						class="border-border-input active:scale-98 block size-[20px] cursor-pointer rounded-full border  bg-carbongray-600 shadow transition-colors hover:border-carbongray-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-carbongray-600 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 dark:bg-carbongray-700"
					/>
				{/each}
			</Slider.Root>
		</div>
		<div
			class="flex h-[20px] w-[20px] items-center justify-center rounded-sm bg-white p-1 text-base dark:bg-carbongray-800"
		>
			{#if threads}
				{threads[0]}
			{/if}
		</div>
	</div>
	{#if updating}
		<div class="flex items-center justify-center gap-2">
			<div class="text-base text-carbongray-600 dark:text-carbongray-200">Upating settings</div>
			<Loader size={15} class="animate-spin" />
		</div>
	{:else if success}
		<div class="flex items-center justify-center gap-2">
			<div class="text-base text-carbongray-600 dark:text-carbongray-200">Updated</div>
			<CircleCheck size={15} class="animate-ping" />
		</div>
	{:else if error}
		<div class="flex items-center justify-center gap-2">
			<div class="text-base text-carbongray-600 dark:text-carbongray-200">Failed</div>
			<CircleX size={15} class="animate-ping" />
		</div>
	{/if}
</div>
