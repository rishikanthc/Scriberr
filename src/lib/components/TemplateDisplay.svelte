<script lang="ts">
	import { ScrollArea } from 'bits-ui';
	import { Button } from 'bits-ui';
	import { SaveOff, Pencil, Save } from 'lucide-svelte';
	export let record;
	let editMode = false;
	let inputValue;
	let recordId;
	let newPrompt;

	$: prompt = record?.prompt;

	function enableEditMode() {
		editMode = true;
		inputValue = record.prompt;
	}

	function disableEditMode() {
		editMode = false;
	}

	async function saveChanges() {
		console.log('save');
		editMode = false;
		recordId = record.id;
		newPrompt = inputValue;

		// Make a POST request to update the prompt
		try {
			const postResponse = await fetch(`/api/templates/${recordId}`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ prompt: newPrompt })
			});

			if (!postResponse.ok) {
				console.error('Error updating the prompt:', await postResponse.text());
				return;
			}

			console.log('Update prompt being displayed');

			// After updating, make a GET request to fetch the updated record
			const getResponse = await fetch(`/api/templates/${recordId}`);
			if (!getResponse.ok) {
				console.error('Error fetching the updated record:', await getResponse.text());
				return;
			}

			const updatedRecord = await getResponse.json();
			record = updatedRecord; // Update the record with the fetched data

			console.log('Updated record:', record);
		} catch (err) {
			console.error('An error occurred while saving changes:', err);
		}
	}
</script>

<div>
	<h4 class="p-3 text-lg font-semibold text-carbongray-500">PROMPT</h4>
	<ScrollArea.Root class="h-full px-3">
		<ScrollArea.Viewport class="h-full w-full">
			<ScrollArea.Content class="relative">
				<div class="relative flex w-full items-start justify-between gap-4 p-2">
					{#if editMode}
						<textarea
							type="text"
							bind:value={inputValue}
							class="h-[320px] w-full rounded-xl border border-carbongray-300 p-2 focus:outline-none focus:ring-2 focus:ring-carbongray-800 focus:ring-offset-2 focus:ring-offset-white"
						/>
					{:else}
						<div class="text-foreground-alt text-lg">{record.prompt}</div>
					{/if}
					<div class="flex flex-col justify-center gap-2">
						{#if editMode}
							<Button.Root on:click={saveChanges}><Save size={20} /></Button.Root>
							<Button.Root on:click={disableEditMode}><SaveOff size={20} /></Button.Root>
						{:else}
							<Button.Root on:click={enableEditMode}><Pencil size={18} /></Button.Root>
						{/if}
					</div>
				</div></ScrollArea.Content
			>
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
</div>
