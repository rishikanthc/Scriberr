<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import AudioDetails from '$lib/components/custom/AudioDetails.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Button } from '$lib/components/ui/button/index.js';

	// --- TYPES ---
	type AudioRecord = {
		id: string;
		title: string;
		created_at: string;
		transcript: string;
	};

	// --- PROPS ---
	let {
		open = $bindable(),
		record
	}: {
		open: boolean;
		record: AudioRecord | null;
	} = $props();

	// --- STATE ---
	let editableTitle = $state(record?.title ?? '');

	$effect(() => {
		if (record) {
			editableTitle = record.title;
		}
	});

	// --- LOGIC ---
	async function handleSaveTitle() {
		if (!record || !editableTitle.trim() || editableTitle.trim() === record.title) {
			return; // No change, no record, or empty title
		}

		try {
			const response = await fetch(`/api/audio/${record.id}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ title: editableTitle.trim() }),
				credentials: 'include'
			});

			if (response.ok) {
				record.title = editableTitle.trim();
				// Here you might want to dispatch an event to notify the parent list to update.
			} else {
				console.error('Failed to update title:', await response.text());
				editableTitle = record.title; // Revert on failure
			}
		} catch (error) {
			console.error('Error saving title:', error);
			if (record) {
				editableTitle = record.title;
			}
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="border-none bg-gray-700 text-gray-200 sm:max-w-3xl">
		<Dialog.Header>
			<Dialog.Title>
				<div class="mt-1 flex items-center gap-2 p-2">
					<Input
						class="h-auto grow border-0 bg-transparent p-0 text-lg font-semibold focus-visible:ring-0 focus-visible:ring-offset-0"
						bind:value={editableTitle}
					/>
					{#if record && editableTitle.trim() !== record.title}
						<Button
							size="sm"
							variant="default"
							class="bg-neon-100 h-7 text-black"
							disabled={!editableTitle.trim()}
							onclick={handleSaveTitle}>Save</Button
						>
					{/if}
				</div>
			</Dialog.Title>
		</Dialog.Header>
		{#if record}
			<div class="grid gap-4 py-4">
				<AudioDetails recordId={record.id} />
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
