<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import { toast } from 'svelte-sonner';

	// --- TYPES ---
	export type SummaryTemplate = {
		id: string;
		title: string;
		prompt: string;
		created_at: string;
	};

	// --- PROPS ---
	let {
		open = $bindable(),
		template,
		onUpdate
	}: {
		open: boolean;
		template: SummaryTemplate | null;
		onUpdate: () => void;
	} = $props();

	// --- STATE ---
	let editableTitle = $state('');
	let editablePrompt = $state('');

	// When the dialog opens or the template to edit changes, update the form fields.
	$effect(() => {
		if (open && template) {
			editableTitle = template.title;
			editablePrompt = template.prompt;
		} else {
			// Reset for a new template
			editableTitle = '';
			editablePrompt = '';
		}
	});

	const isEditing = $derived(!!template);

	// --- LOGIC ---
	async function handleSave() {
		const title = editableTitle.trim();
		const prompt = editablePrompt.trim();

		if (!title || !prompt) {
			toast.error('Both title and prompt are required.');
			return;
		}

		try {
			const response = await fetch('/api/summary-templates', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					id: template?.id ?? null,
					title: title,
					prompt: prompt
				}),
				credentials: 'include'
			});

			if (!response.ok) {
				const result = await response.json();
				throw new Error(result.error || 'Failed to save template.');
			}

			toast.success(`Template ${isEditing ? 'updated' : 'created'} successfully!`);
			onUpdate(); // Notify parent to refresh data
			open = false; // Close the dialog
		} catch (error) {
			const errorMessage = error instanceof Error ? error.message : 'An unknown error occurred.';
			toast.error('Save failed', { description: errorMessage });
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="border-none bg-gray-700 text-gray-200 sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>{isEditing ? 'Edit Template' : 'Create New Template'}</Dialog.Title>
			<Dialog.Description class="pt-2 text-gray-400">
				{isEditing
					? 'Update the details for your summary template.'
					: 'Provide a title and a prompt for your new template.'}
			</Dialog.Description>
		</Dialog.Header>
		<div class="grid gap-4 py-4">
			<div class="grid grid-cols-4 items-center gap-4">
				<label for="title" class="text-right text-sm font-medium"> Title </label>
				<Input
					id="title"
					bind:value={editableTitle}
					class="col-span-3 border-gray-600 bg-gray-800 text-gray-200 focus:border-blue-500 focus:ring-blue-500"
					placeholder="e.g., Meeting Summary"
				/>
			</div>
			<div class="grid grid-cols-4 items-start gap-4">
				<label for="prompt" class="pt-2 text-right text-sm font-medium"> Prompt </label>
				<Textarea
					id="prompt"
					bind:value={editablePrompt}
					class="col-span-3 min-h-[150px] border-gray-600 bg-gray-800 text-gray-200 focus:border-blue-500 focus:ring-blue-500"
					placeholder="e.g., Summarize the following transcript..."
				/>
			</div>
		</div>
		<Dialog.Footer>
			<Button
				variant="ghost"
				class="hover:bg-gray-600"
				onclick={() => {
					open = false;
				}}>Cancel</Button
			>
			<Button onclick={handleSave} class="bg-neon-100 hover:bg-neon-200 text-gray-800">
				{isEditing ? 'Save Changes' : 'Create Template'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
