<script>
	import { ScrollArea } from 'bits-ui';
	import { Button } from 'bits-ui';
	import { Tabs } from 'bits-ui';
	import AudioViz from '$lib/components/AudioViz.svelte';
	import { Combobox } from 'bits-ui';
	import { Sparkles, ChevronsUpDown, Search, Check } from 'lucide-svelte';

	export let record;
	export let fileurl;
	export let templates;

	$: transcript = record.transcript !== '' ? JSON.parse(record.transcript).transcription : null;
	$: summary = record.summary;
	$: audioSrc = fileurl?.selected_file || '';
	$: audioPeaks = record.peaks?.data || [];

	$: templateList = templates.map((val) => {
		return { value: val.title, label: val.title, id: val.id };
	});

	let inputValue = '';
	let touchedInput = false;
	let selectedTemplate;

	$: filteredTemplates =
		inputValue && touchedInput
			? templateList?.filter((template) =>
					template.value.toLowerCase().includes(inputValue.toLowerCase())
				)
			: templateList;

	async function generateSummary() {
		const recordId = record.id;
		const templateId = templates.find((val) => {
			return val.title.toLowerCase() === selectedTemplate.value.toLowerCase();
		}).id;
		const tscript = transcript.map((obj) => obj.text).join(' ');

		const response = await fetch('/api/summarize', {
			method: 'POST',
			body: JSON.stringify({
				templateId: templateId,
				transcript: tscript,
				id: recordId
			})
		});

		// Parse the summary result
		const data = await response.json();
		summary = data.message.content;
		const resp = await fetch(`/api/records/${recordId}`, {
			method: 'GET'
		});
		const jresp = await resp.json();
		record = jresp.record;
	}
</script>

<div class="flex w-full flex-col justify-center gap-2">
	<div class="p-3">
		<AudioViz {audioSrc} bind:peaks={audioPeaks} />
	</div>
	<Tabs.Root
		value="transcript"
		class="rounded-card bg-background-alt shadow-card h-[90%] w-full px-3"
	>
		<Tabs.List
			class="shadow-mini-inset grid w-full grid-cols-2 gap-1 rounded-md bg-carbongray-200 p-1 text-sm font-semibold leading-[0.01em] dark:border dark:border-neutral-600/30 dark:bg-carbongray-600"
		>
			<Tabs.Trigger
				value="transcript"
				class="data-[state=active]:shadow-xs h-7 rounded-[7px] bg-transparent py-2 data-[state=active]:bg-white dark:data-[state=active]:bg-carbongray-700"
				>Transcript</Tabs.Trigger
			>
			<Tabs.Trigger
				value="summary"
				class="data-[state=active]:shadow-mini h-7 rounded-[7px] bg-transparent py-2 data-[state=active]:bg-white dark:data-[state=active]:bg-carbongray-700"
				>Summary</Tabs.Trigger
			>
		</Tabs.List>
		<Tabs.Content value="transcript" class="pt-3">
			<ScrollArea.Root class="relative h-[480px] px-4 2xl:h-[672px]">
				<ScrollArea.Viewport class="h-full w-full">
					<ScrollArea.Content>
						<div>
							{#if transcript}
								{#each transcript as t}
									<div class="my-3 flex flex-col items-start">
										<div class="text-xs text-carbongray-500">
											{t.timestamps.from.split(',')[0]}
										</div>
										<div class="text-base leading-relaxed">
											<p id={t.timestamps.from}>{t.text}</p>
										</div>
									</div>
								{/each}
							{/if}
						</div>
					</ScrollArea.Content>
				</ScrollArea.Viewport>
				<ScrollArea.Scrollbar
					orientation="vertical"
					class="hover:bg-dark-10 flex h-full w-2.5 touch-none select-none rounded-full border-l border-l-transparent p-px transition-all hover:w-3"
				>
					<ScrollArea.Thumb
						class="relative flex-1 rounded-full bg-carbongray-200 opacity-40 transition-opacity hover:opacity-100"
					/>
				</ScrollArea.Scrollbar>
				<ScrollArea.Corner />
			</ScrollArea.Root>
		</Tabs.Content>
		<Tabs.Content value="summary" class="pt-3">
			<div class="flex items-center justify-start gap-2">
				<Combobox.Root
					items={filteredTemplates}
					bind:inputValue
					bind:touchedInput
					bind:selected={selectedTemplate}
				>
					<div class="relative w-[320px] overflow-visible p-2">
						<!-- Left icon -->
						<Search
							class="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-carbongray-400 dark:text-carbongray-600"
						/>

						<!-- Search input -->
						<Combobox.Input
							class="h-10 w-full rounded-lg border border-carbongray-500 bg-transparent pl-10 pr-10 text-sm text-carbongray-700 placeholder-carbongray-600 transition focus:outline-none focus:ring-2 focus:ring-carbonblue-500 focus:ring-offset-2 focus:ring-offset-white dark:border-carbongray-700 dark:text-carbongray-100 dark:placeholder-carbongray-600 dark:focus:ring-offset-carbongray-900"
							placeholder="Search a template"
							aria-label="Search a template"
						/>
						<ChevronsUpDown
							class="absolute right-4 top-1/2 h-5 w-5 -translate-y-1/2 text-carbongray-400"
						/>
					</div>

					<!-- Dropdown content -->
					<Combobox.Content
						class="z-40 mt-2 w-full rounded-lg border border-carbongray-300 bg-white p-2  shadow-lg dark:border-carbongray-700 dark:bg-carbongray-700"
						sideOffset={8}
					>
						<!-- List items -->
						{#each filteredTemplates as template (template.value)}
							<Combobox.Item
								class="flex h-10 w-full cursor-pointer select-none items-center rounded-lg px-4 py-2 text-sm capitalize text-carbongray-700 hover:bg-carbongray-100 data-[highlighted]:bg-carbongray-100 dark:text-carbongray-50 dark:hover:bg-carbongray-700 dark:data-[highlighted]:bg-carbongray-800"
								value={template.value}
								label={template.label}
							>
								{template.label}
								<Combobox.ItemIndicator class="ml-auto" asChild={false}>
									<Check class="h-4 w-4 text-blue-500" />
								</Combobox.ItemIndicator>
							</Combobox.Item>
						{:else}
							<span class="block px-4 py-2 text-sm text-gray-400"> No results found </span>
						{/each}
					</Combobox.Content>

					<!-- Hidden input -->
					<Combobox.HiddenInput name="favoriteFruit" />
				</Combobox.Root>
				<Button.Root
					class="rounded-md border border-carbongray-100 p-2 shadow-sm hover:bg-carbongray-100 hover:text-carbonblue-500 dark:border-carbongray-800"
					on:click={generateSummary}><Sparkles size={20} /></Button.Root
				>
			</div>

			{#if summary}
				<ScrollArea.Root class="relative z-30 h-[480px] px-4">
					<ScrollArea.Viewport class="h-full w-full">
						<ScrollArea.Content>
							{summary}
						</ScrollArea.Content>
					</ScrollArea.Viewport>
					<ScrollArea.Scrollbar
						orientation="vertical"
						class="hover:bg-dark-10 flex h-full w-2.5 touch-none select-none rounded-full border-l border-l-transparent p-px transition-all hover:w-3"
					>
						<ScrollArea.Thumb
							class="bg-muted-foreground relative flex-1 rounded-full opacity-40 transition-opacity hover:opacity-100"
						/>
					</ScrollArea.Scrollbar>
					<ScrollArea.Corner />
				</ScrollArea.Root>
			{/if}
		</Tabs.Content>
	</Tabs.Root>
</div>
