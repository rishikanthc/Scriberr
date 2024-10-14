<script lang="ts">
	import { Save, SaveOff, Library, CircleCheck, Timer, TypeOutline } from 'lucide-svelte';
	import { Separator } from 'bits-ui';
	import { Button } from 'bits-ui';
	import { Label } from 'bits-ui';
	import { Tabs } from 'bits-ui';
	import { SquareX } from 'lucide-svelte';
	import DisplayPane from '$lib/components/DisplayPane.svelte';
	import TemplateDisplay from '$lib/components/TemplateDisplay.svelte';
	import { Folder, Upload, Settings } from 'lucide-svelte';
	import SettingsPane from '$lib/components/SettingsPane.svelte';
	import TemplatesPane from '$lib/components/TemplatesPane.svelte';
	import UploadPane from '$lib/components/UploadPane.svelte';
	import { ScrollArea } from 'bits-ui';
	import SysStats from '$lib/components/SysStats.svelte';
	import { createEventDispatcher } from 'svelte';

	import { Loader2, Check } from 'lucide-svelte';
	let isLoading = false;
	let isUploaded = false;
	let errorMessage = '';

	export let data;
	export let fileUrls;
	export let templates;
	const dispatcher = createEventDispatcher();

	let clickedId;
	let selected;
	let fileurl;
	let newTemplateTitle;
	let newPrompt;

	let statsState = false;
	let raiseNewTemplate = false;
	let selectedTemplate;

	function newTemplate() {
		clickedId = null;
		selectedTemplate = null;
		raiseNewTemplate = true;
	}

	async function createTemplate() {
		if (newTemplateTitle && newPrompt) {
			try {
				isLoading = true;
				isUploaded = false;
				errorMessage = '';

				// Prepare the payload
				const payload = { title: newTemplateTitle, prompt: newPrompt };

				// Make the POST request to the API
				const response = await fetch('/api/templates', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json'
					},
					body: JSON.stringify(payload)
				});

				// Check if the response is successful
				if (response.ok) {
					const result = await response.json();
					console.log('Template created successfully:', result);
					isUploaded = true; // Set uploaded status

					// Add a small delay before clearing the state and removing the checkmark
					setTimeout(() => {
						clearStates();
						newTemplateTitle = null;
						newPrompt = null;
						isUploaded = false; // Reset uploaded status
					}, 2000); // Delay for 2 seconds

					refreshTemplates();
				} else {
					const error = await response.json();
					errorMessage = error.message || 'Error creating template';
					console.error('Error creating template:', error);
				}
			} catch (err) {
				errorMessage = 'Error during API call';
				console.error('Error during API call:', err);
			} finally {
				isLoading = false;
			}
		}
	}

	function clearStates() {
		clickedId = null;
		selectedTemplate = null;
		raiseNewTemplate = false;
	}

	async function refreshTemplates() {
		const response = await fetch('/api/templates');
		templates = await response.json();
	}

	function openTemplate(event) {
		selectedTemplate = event.detail;
	}

	function closeRecord() {
		// clickedId = null;
		// selectedTemplate = null;
		clearStates();
	}

	function doSomething() {
		statsState = true;
		dispatcher('onUpload');
	}

	function onClick(event) {
		clickedId = event.target.id;
		selected = data.find((value) => {
			return value.id === clickedId;
		});
		fileurl = fileUrls.find((value) => {
			return value.id === clickedId;
		});
	}
</script>

<div
	class="items-top z-0 h-[704px] w-full justify-start gap-2 p-1 lg:flex lg:h-[700px] 2xl:h-[900px]"
>
	<div
		class="flex h-full w-full flex-shrink-0 rounded bg-carbongray-50 p-2 text-lg dark:bg-carbongray-700 lg:w-[300px]"
	>
		<Tabs.Root value="files" class="w-full flex-shrink-0" onValueChange={clearStates}>
			<Tabs.List
				class="shadow-mini-inset grid w-full grid-cols-4 gap-1 rounded-t-md bg-carbongray-200 p-1 text-sm font-semibold leading-[0.01em] dark:border dark:border-neutral-600/30 dark:bg-carbongray-600"
			>
				<Tabs.Trigger
					value="files"
					class="data-[state=active]:shadow-xs flex h-7 items-center justify-center rounded-[7px] bg-transparent py-2 data-[state=active]:bg-white dark:data-[state=active]:bg-carbongray-700"
				>
					<Library size={20} />
				</Tabs.Trigger>
				<Tabs.Trigger
					value="upload"
					class="data-[state=active]:shadow-mini flex h-7 items-center justify-center rounded-[7px] bg-transparent py-2 data-[state=active]:bg-white dark:data-[state=active]:bg-carbongray-700"
				>
					<Upload size={20} />
				</Tabs.Trigger>
				<Tabs.Trigger
					value="templates"
					class="data-[state=active]:shadow-mini flex h-7 items-center justify-center rounded-[7px] bg-transparent py-2 data-[state=active]:bg-white dark:data-[state=active]:bg-carbongray-700"
				>
					<TypeOutline size={20} />
				</Tabs.Trigger>
				<Tabs.Trigger
					value="settings"
					class="data-[state=active]:shadow-mini flex h-7 items-center justify-center rounded-[7px] bg-transparent py-2 data-[state=active]:bg-white dark:data-[state=active]:bg-carbongray-700"
				>
					<Settings size={20} />
				</Tabs.Trigger>
			</Tabs.List>
			<Tabs.Content value="files">
				<ScrollArea.Root class="relative h-[632px] w-full px-0 2xl:h-[832px]">
					<ScrollArea.Viewport class="h-full w-full">
						<ScrollArea.Content>
							<div class="flex w-full flex-col gap-0">
								{#if data}
									{#each data as rec}
										<Button.Root
											class="border-b p-2 hover:bg-carbongray-100 dark:border-b-carbongray-800 dark:hover:bg-carbongray-800 {clickedId &&
											clickedId === rec.id
												? 'bg-carbongray-100 dark:bg-carbongray-800'
												: ''}"
											on:click={onClick}
										>
											<div class="justify-left flex items-center gap-4" id={rec.id}>
												{#if rec.processed}<CircleCheck
														color={'#24a147'}
														id={rec.id}
														size={15}
													/>{:else}<Timer id={rec.id} size={15} />{/if}
												<div class="gap-0.25 flex flex-col items-start justify-center" id={rec.id}>
													<div class="text-base" id={rec.id}>{rec.title}</div>
													<div class="text-xs text-carbongray-600" id={rec.id}>
														{rec.date.split(' ')[0]}
													</div>
												</div>
											</div>
										</Button.Root>
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
			<Tabs.Content value="upload">
				<UploadPane on:onUpload={doSomething} />
			</Tabs.Content>
			<Tabs.Content value="settings">
				<SettingsPane />
			</Tabs.Content>
			<Tabs.Content value="templates">
				<TemplatesPane
					{templates}
					on:onTemplateClick={openTemplate}
					on:openNewTemplate={newTemplate}
				/>
			</Tabs.Content>
		</Tabs.Root>
	</div>

	{#if clickedId}
		<div
			class="absolute left-0 top-[50px] z-50 h-full w-full bg-white dark:bg-carbongray-800 lg:relative lg:top-0 lg:z-10"
		>
			<div class="flex items-center justify-between p-3">
				<div class="text-4xl">
					{selected?.title}
				</div>
				<Button.Root class="hover:bg-carbongray-100" on:click={closeRecord}>
					<SquareX size={20} />
				</Button.Root>
			</div>
			<DisplayPane record={selected} {fileurl} {templates} />
		</div>
	{:else if selectedTemplate}
		<div
			class="absolute left-0 top-[50px] z-50 h-full w-full bg-white dark:bg-carbongray-800 lg:relative lg:top-0 lg:z-10"
		>
			<div class="flex items-center justify-between p-3">
				<div class="text-4xl">
					{selectedTemplate?.title}
				</div>
				<Button.Root class="hover:bg-carbongray-100" on:click={closeRecord}>
					<SquareX size={20} />
				</Button.Root>
			</div>
			<TemplateDisplay bind:record={selectedTemplate} on:onTemplatesUpdate={refreshTemplates} />
		</div>
	{:else if raiseNewTemplate}
		<div
			class="absolute left-0 top-[50px] z-50 h-full w-full bg-white dark:bg-carbongray-800 lg:relative lg:top-0 lg:z-10"
		>
			<div class="flex items-center justify-between p-3">
				<div class="text-4xl">New Template</div>
				<Button.Root class="hover:bg-carbongray-100" on:click={closeRecord}>
					<SquareX size={20} />
				</Button.Root>
			</div>

			<div class="relative flex w-full flex-col items-start gap-1 p-2">
				<Label.Root id="title-label" for="title">Title</Label.Root>
				<textarea
					id="title"
					bind:value={newTemplateTitle}
					class="h-[30px] w-[400px] rounded-md border border-carbongray-300 p-1 focus:outline-none focus:ring-2 focus:ring-carbongray-800 focus:ring-offset-2 focus:ring-offset-white dark:bg-carbongray-700"
				/>
			</div>

			<div class="relative flex w-full flex-col items-start gap-1 p-2">
				<Label.Root id="prompt-label" for="prompt">Prompt</Label.Root>
				<textarea
					id="prompt"
					bind:value={newPrompt}
					class="h-[320px] w-full rounded-md border border-carbongray-300 p-2 focus:outline-none focus:ring-2 focus:ring-carbongray-800 focus:ring-offset-2 focus:ring-offset-white dark:bg-carbongray-700"
				/>
			</div>
			<Button.Root
				on:click={createTemplate}
				class="m-2 flex items-center justify-center gap-1 rounded-md bg-carbongray-100 p-2 hover:bg-carbongray-200 disabled:bg-carbongray-50 dark:bg-carbongray-700"
				disabled={isLoading}
			>
				{#if isLoading}
					<Loader2 size={20} class="animate-spin" />
					<span>Uploading...</span>
				{:else if isUploaded}
					<Check size={20} class="animate-ping text-green-500" />
					<span>Uploaded!</span>
				{:else}
					Save <Save size={20} />
				{/if}
			</Button.Root>
		</div>
	{:else}
		<div class="hidden lg:block">
			<SysStats bind:state={statsState} on:finishedProcessing />
		</div>
	{/if}
</div>
