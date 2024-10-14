<script lang="ts">
	import { Save, SaveOff, Library, CircleCheck, Timer, TypeOutline, Loader } from 'lucide-svelte';
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
	import { ContextMenu } from 'bits-ui';
	import { Trash, Pencil } from 'lucide-svelte';

	import { Loader2, Check, X } from 'lucide-svelte';
	import StatusSpinner from './StatusSpinner.svelte';
	let isLoading = false;
	let isUploaded = false;
	let isError = false;
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
				isError = false;
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

					dispatcher('templatesModified');
				} else {
					const error = await response.json();
					errorMessage = error.message || 'Error creating template';
					isError = true;
					console.error('Error creating template:', error);
				}
			} catch (err) {
				errorMessage = 'Error during API call';
				isError = true;
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

	let deleting = false;
	let delMessage = null;
	let success = -1;

	async function deleteRecording(event) {
		const delId = event.currentTarget.id;

		if (!delId) {
			console.error('Record ID is missing');
			return;
		}

		try {
			delMessage = `Deleting record ${delId}`;
			deleting = true;
			// Delete the template
			const deleteResponse = await fetch(`/api/records?id=${delId}`, {
				method: 'DELETE'
			});

			if (deleteResponse.ok) {
				const deleteResult = await deleteResponse.json();
				console.log('Record deleted successfully:', deleteResult);
				delMessage = 'Deleted';
				success = 1;

				// dispatch('templatesModified');
				dispatcher('recordsModified');
			} else {
				const error = await deleteResponse.json();
				console.error('Error deleting record:', error);
				delMessage = `Delete failed`;
				success = 0;
			}
		} catch (err) {
			console.error('Error during API call:', err);
			delMessage = `Delete failed`;
			success = 0;
		} finally {
			setInterval(() => {
				deleting = false;
				success = -1;
			}, 3000);
		}
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

	let renaming = false;
	async function renameTempl(event) {
		renaming = true;
		console.log('Double clicked');
	}

	let renameTitle;
	let renameLoading = false;

	async function saveTitle() {
		renameLoading = true;
		try {
			// Make a POST request to the API with the updated title only
			const response = await fetch(`/api/templates/${selectedTemplate.id}`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					title: renameTitle // Send only the updated title or any other field
				})
			});

			// Check if the response is successful
			if (response.ok) {
				const updatedRecord = await response.json();
				console.log('Template updated successfully:', updatedRecord);
				selectedTemplate = updatedRecord;
				dispatcher('templatesModified');
				// Update the selected template title locally
			} else {
				const error = await response.json();
				console.error('Error updating title:', error);
			}
		} catch (err) {
			console.error('Error during API call:', err);
		} finally {
			// Exit rename mode regardless of success or failure
			renaming = false;
			renameLoading = false;
		}
	}

	function handleKeydown(event) {
		if (event.key === 'Enter' && event.shiftKey) {
			event.preventDefault();
			saveTitle();
			dispatcher('templatesModified');
		} else if (event.key === 'Escape') {
			renaming = false;
		}
	}

	let renameTitleRecord;
	async function saveTitleRecord() {
		renameLoading = true;
		try {
			// Make a POST request to the API with the updated title only
			const response = await fetch(`/api/records/${selected.id}`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					title: renameTitleRecord // Send only the updated title or any other field
				})
			});

			if (response.ok) {
				const updatedRecord = await response.json();
				console.log('Template updated successfully:', updatedRecord);
				selected = updatedRecord;
				dispatcher('recordsModified');
			} else {
				const error = await response.json();
				console.error('Error updating title:', error);
			}
		} catch (err) {
			console.error('Error during API call:', err);
		} finally {
			// Exit rename mode regardless of success or failure
			renaming = false;
			renameLoading = false;
		}
	}

	function handleKeydownRecord(event) {
		if (event.key === 'Enter' && event.shiftKey) {
			event.preventDefault();
			saveTitleRecord();
		} else if (event.key === 'Escape') {
			renaming = false;
		}
	}
</script>

<div
	class="items-top relative z-0 h-[704px] w-full justify-start gap-2 p-1 lg:flex lg:h-[700px] 2xl:h-[900px]"
>
	{#if deleting}
		<div class="absolute bottom-0 right-0">
			<StatusSpinner bind:msg={delMessage} {success} />
		</div>
	{/if}
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
											<ContextMenu.Root>
												<ContextMenu.Trigger class="w-full">
													<div class="justify-left flex items-center gap-4" id={rec.id}>
														{#if rec.processed}<CircleCheck
																color={'#24a147'}
																id={rec.id}
																size={15}
															/>{:else}<Timer id={rec.id} size={15} />{/if}
														<div
															class="gap-0.25 flex flex-col items-start justify-center"
															id={rec.id}
														>
															<div class="text-base" id={rec.id}>{rec.title}</div>
															<div class="text-xs text-carbongray-600" id={rec.id}>
																{rec.date.split(' ')[0]}
															</div>
														</div>
													</div>
												</ContextMenu.Trigger>
												<ContextMenu.Content
													class="border-muted z-50 w-full max-w-[229px] rounded-xl border bg-white px-1 py-1.5"
												>
													<ContextMenu.Item
														class="rounded-button flex h-10 select-none items-center py-3 pl-3 pr-1.5 text-sm font-medium outline-none !ring-0 !ring-transparent data-[highlighted]:bg-carbongray-50"
													>
														<Button.Root
															class="flex items-center justify-center gap-3"
															on:click={deleteRecording}
															id={rec.id}
														>
															<Trash size={15} />

															<div class="text-base">Delete</div>
														</Button.Root>
													</ContextMenu.Item>
												</ContextMenu.Content>
											</ContextMenu.Root>
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
					on:templatesModified
				/>
			</Tabs.Content>
		</Tabs.Root>
	</div>

	{#if clickedId}
		<div
			class="absolute left-0 top-[50px] z-50 h-full w-full bg-white dark:bg-carbongray-800 lg:relative lg:top-0 lg:z-10"
		>
			<div class="flex items-center justify-between p-3">
				{#if renaming}
					<div class="justify-left flex w-full items-center gap-1">
						<textarea
							id="newtitle"
							bind:value={renameTitleRecord}
							class="h-[30px] w-[300px] rounded-md border border-carbongray-300 p-1 focus:outline-none focus:ring-2 focus:ring-carbongray-800 focus:ring-offset-2 focus:ring-offset-white disabled:bg-carbongray-50 dark:bg-carbongray-700 disabled:dark:bg-carbongray-600"
							on:keydown={handleKeydownRecord}
							autofocus
							disabled={renameLoading}
						/>
						{#if renameLoading}
							<Loader class="animate-spin" size={15} />
						{/if}
					</div>
				{:else}
					<div class="text-4xl" on:dblclick={renameTempl}>
						{selected?.title}
					</div>
				{/if}
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
				{#if renaming}
					<div class="justify-left flex w-full items-center gap-1">
						<textarea
							id="newtitle"
							bind:value={renameTitle}
							class="h-[30px] w-[300px] rounded-md border border-carbongray-300 p-1 focus:outline-none focus:ring-2 focus:ring-carbongray-800 focus:ring-offset-2 focus:ring-offset-white disabled:bg-carbongray-50 dark:bg-carbongray-700 disabled:dark:bg-carbongray-600"
							on:keydown={handleKeydown}
							autofocus
							disabled={renameLoading}
						/>
						{#if renameLoading}
							<Loader class="animate-spin" size={15} />
						{/if}
					</div>
				{:else}
					<div class="text-4xl" on:dblclick={renameTempl}>
						{selectedTemplate?.title}
					</div>
				{/if}
				<Button.Root class="hover:bg-carbongray-100" on:click={closeRecord}>
					<SquareX size={20} />
				</Button.Root>
			</div>
			<TemplateDisplay bind:record={selectedTemplate} on:templatesModified />
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
				class="m-2 flex items-center justify-center gap-1 rounded-md bg-carbongray-100 p-2 hover:bg-carbongray-200 dark:bg-carbongray-700"
				disabled={isLoading || isUploaded || isError}
			>
				{#if isLoading}
					<Loader2 size={20} class="animate-spin" />
					<span>Uploading...</span>
				{:else if isUploaded}
					<Check size={20} class="animate-ping text-green-500" />
					<span>Uploaded!</span>
				{:else if isError}
					<X size={20} class="text-red-500" />
					<span>Failed!</span>
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
