<script lang="ts">
	import { Library, CircleCheck, Timer, TypeOutline } from 'lucide-svelte';
	import { Separator } from 'bits-ui';
	import { Button } from 'bits-ui';
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

	export let data;
	export let fileUrls;
	export let templates;
	const dispatcher = createEventDispatcher();

	let clickedId;
	let selected;
	let fileurl;

	let statsState = false;
	let selectedTemplate;

	function clearStates() {
		clickedId = null;
		selectedTemplate = null;
	}

	function openTemplate(event) {
		selectedTemplate = event.detail;
	}

	function closeRecord() {
		clickedId = null;
		selectedTemplate = null;
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
				<TemplatesPane {templates} on:onTemplateClick={openTemplate} />
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
			<TemplateDisplay record={selectedTemplate} />
		</div>
	{:else}
		<div class="hidden lg:block">
			<SysStats bind:state={statsState} on:finishedProcessing />
		</div>
	{/if}
</div>
