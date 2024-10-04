<script lang="ts">
	import { Button } from 'bits-ui';
	import { ScrollArea } from 'bits-ui';
	import { createEventDispatcher } from 'svelte';

	export let templates;
	let dispatch = createEventDispatcher();

	let clickedId;
	let selected;
	$: console.log('TEMPLATES', selected);

	function onClick(event) {
		clickedId = event.target.id;
		selected = templates.find((value) => {
			return value.id === clickedId;
		});
		if (selected) {
			dispatch('onTemplateClick', selected);
		}
	}
</script>

<ScrollArea.Root class="h-[480px] w-full px-0 2xl:h-[784px]">
	<ScrollArea.Viewport class="h-full w-full">
		<ScrollArea.Content>
			<div class="flex w-full flex-col items-start gap-0">
				{#if templates}
					{#each templates as rec}
						<Button.Root
							class="w-full border-b border-carbongray-100 p-2 hover:bg-carbongray-50 dark:border-b-carbongray-800 dark:hover:bg-carbongray-800"
							id={rec.id}
							on:click={onClick}
							><div id={rec.id} class="flex items-center justify-start text-lg">
								{rec.title}
							</div></Button.Root
						>
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
