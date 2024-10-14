<script lang="ts">
	import { Button } from 'bits-ui';
	import { ScrollArea } from 'bits-ui';
	import { createEventDispatcher } from 'svelte';
	import { ContextMenu } from 'bits-ui';
	import { Plus, Trash } from 'lucide-svelte';

	export let templates;
	let dispatch = createEventDispatcher();

	let clickedId;
	let selected;
	$: console.log('TEMPLATES', selected);

	function newTemplate() {
		dispatch('openNewTemplate');
	}

	function onClick(event) {
		clickedId = event.target.id;
		selected = templates.find((value) => {
			return value.id === clickedId;
		});
		if (selected) {
			dispatch('onTemplateClick', selected);
		}
	}

	async function deleteTemplate(event) {
		const delId = event.target.id;

		if (!delId) {
			console.error('Template ID is missing');
			return;
		}

		try {
			// Delete the template
			const deleteResponse = await fetch(`/api/templates?id=${delId}`, {
				method: 'DELETE'
			});

			if (deleteResponse.ok) {
				const deleteResult = await deleteResponse.json();
				console.log('Template deleted successfully:', deleteResult);

				dispatch('templatesModified');
			} else {
				const error = await deleteResponse.json();
				console.error('Error deleting template:', error);
			}
		} catch (err) {
			console.error('Error during API call:', err);
		}
	}
</script>

<ScrollArea.Root class="h-[480px] w-full px-0 2xl:h-[784px]">
	<ScrollArea.Viewport class="h-full w-full">
		<div class="flex w-full justify-end">
			<Button.Root
				on:click={newTemplate}
				class="mt-2 flex w-[60px] items-center gap-1 rounded-md bg-carbongray-200 p-1 text-base hover:bg-carbongray-100"
				>New <Plus size={15} /></Button.Root
			>
		</div>
		<ScrollArea.Content>
			<div class="flex w-full flex-col items-start gap-0">
				{#if templates}
					{#each templates as rec}
						<ContextMenu.Root>
							<ContextMenu.Trigger class="w-full">
								<Button.Root
									class="w-full border-b border-carbongray-100 p-2 hover:bg-carbongray-100  dark:border-b-carbongray-800 dark:hover:bg-carbongray-800"
									id={rec.id}
									on:click={onClick}
									><div id={rec.id} class="flex items-center justify-start text-lg">
										{rec.title}
									</div></Button.Root
								>
							</ContextMenu.Trigger>
							<ContextMenu.Content
								class="border-muted z-50 w-full max-w-[229px] rounded-xl border bg-white px-1 py-1.5"
							>
								<ContextMenu.Item
									class="rounded-button flex h-10 select-none items-center py-3 pl-3 pr-1.5 text-sm font-medium outline-none !ring-0 !ring-transparent data-[highlighted]:bg-carbongray-50"
								>
									<Button.Root
										class="flex items-center justify-center gap-3"
										on:click={deleteTemplate}
									>
										<Trash size={15} id={rec.id} />

										<div class="text-base" id={rec.id}>Delete</div>
									</Button.Root>
								</ContextMenu.Item>
							</ContextMenu.Content>
						</ContextMenu.Root>
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
