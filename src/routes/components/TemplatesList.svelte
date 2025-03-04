<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Pencil, Upload, CirclePlus, Trash2, ChevronUp, ChevronDown } from 'lucide-svelte';
	import { onMount } from 'svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { templates } from '$lib/stores/templateStore';
	import { toast } from 'svelte-sonner';

	let selectedTemplate = $state(false);
	let title = $state('');
	let prompt = $state('');
	let dialogOpen = $state(false);
	let expandedId = $state(null);
	let editingTemplate = $state<{ id: string; title: string; prompt: string } | null>(null);
	let editDialogOpen = $state(false);

	async function handleSubmit(event) {
		if (event) event.preventDefault();
		await templates.add({ title, prompt });
		dialogOpen = false;
		title = '';
		prompt = '';
		toast.success('Template created');
	}

	let editingTitle = $state('');
	let editingPrompt = $state('');

	function startEdit(template: any) {
		editingTitle = template.title;
		editingPrompt = template.prompt;
		editingTemplate = { ...template };
		editDialogOpen = true;
	}

	async function handleEdit(event) {
		if (event) event.preventDefault();
		if (editingTemplate) {
			await templates.update(editingTemplate.id, {
				title: editingTitle,
				prompt: editingPrompt
			});
			editDialogOpen = false;
			editingTemplate = null;
		}
	}

	function getFirstFiveWords(text: string) {
		return text.split(' ').slice(0, 10).join(' ') + '...';
	}

	async function deleteTemplate(id: string) {
		await templates.remove(id);
	}

	function toggleExpand(id: string) {
		expandedId = expandedId === id ? null : id;
	}

	onMount(() => {
		templates.refresh();
	});
</script>

<Card.Root
	class="mx-auto rounded-xl border-none bg-neutral-400/15 p-4 shadow-lg backdrop-blur-xl 2xl:w-[500px] {selectedTemplate
		? 'pointer-events-none opacity-0'
		: 'opacity-100'}"
>
	<Card.Content class="p-2">
		<div class="mb-4 text-lg font-bold text-white">
			<div class="flex items-center justify-between">
				<h3>Summary Templates</h3>
				<Button variant="secondary" size="sm" onclick={() => (dialogOpen = true)}>
					<div>New</div>
					<CirclePlus size={16} class="text-blue-500" />
				</Button>
			</div>
		</div>

		<ScrollArea class="h-[45svh] rounded-lg p-4 min-[390px]:h-[50svh] lg:h-[55svh]">
			<div class="space-y-2">
				{#each $templates as template}
					<div
						class="rounded-lg border border-neutral-500/40 bg-neutral-900/25 p-3 shadow-sm backdrop-blur-sm"
					>
						<div class="flex items-center justify-between gap-4">
							<div class="flex-1">
								<div class="mb-2 flex items-center justify-between">
									<h4 class="font-medium text-white">{template.title}</h4>
									{#if expandedId === template.id}
										<div class="flex items-center">
											<Button
												variant="ghost"
												size="icon"
												class="m-0 hover:bg-neutral-600 hover:text-red-500"
												onclick={() => deleteTemplate(template.id)}
											>
												<Trash2 size={16} class="text-gray-300" />
											</Button>
											<Button
												variant="ghost"
												size="icon"
												class="hover:bg-neutral-600 hover:text-blue-500"
												onclick={() => startEdit(template)}
											>
												<Pencil size={16} class="m-0 text-gray-300" />
											</Button>
										</div>
									{/if}
								</div>
								<p class="text-sm text-gray-200">
									{expandedId === template.id
										? template.prompt
										: getFirstFiveWords(template.prompt)}
								</p>
							</div>
							<div class="flex gap-2">
								<Button
									variant="ghost"
									size="icon"
									class="hover:bg-neutral-600"
									onclick={() => toggleExpand(template.id)}
								>
									{#if expandedId === template.id}
										<ChevronUp size={16} class="text-gray-100" />
									{:else}
										<ChevronDown size={16} class="text-gray-100" />
									{/if}
								</Button>
							</div>
						</div>
					</div>
				{/each}
			</div>
		</ScrollArea>
	</Card.Content>
</Card.Root>

<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Content class="sm:max-w-[425px]">
		<Dialog.Header>
			<Dialog.Title>New Template</Dialog.Title>
		</Dialog.Header>

		<form class="grid gap-4 py-4" onsubmit={handleSubmit}>
			<Input
				placeholder="Template Title"
				bind:value={title}
				required
				class="text-gray-100 placeholder:text-gray-200"
			/>
			<Textarea
				placeholder="Summarization Prompt"
				bind:value={prompt}
				required
				class="min-h-[100px] text-gray-100 placeholder:text-gray-200"
			/>
			<Dialog.Footer>
				<Button type="submit" class="bg-gray-300 text-gray-700 hover:bg-gray-400"
					>Save Template</Button
				>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={editDialogOpen}>
	<Dialog.Content class="sm:max-w-[425px]">
		<Dialog.Header>
			<Dialog.Title>Edit Template</Dialog.Title>
		</Dialog.Header>

		<form class="grid gap-4 py-4" onsubmit={handleEdit}>
			<Input
				placeholder="Template Title"
				bind:value={editingTitle}
				required
				class="text-gray-100"
			/>
			<Textarea
				placeholder="Summarization Prompt"
				bind:value={editingPrompt}
				required
				class="min-h-[100px] text-gray-100"
			/>
			<Dialog.Footer>
				<Button type="submit" class="bg-gray-300 text-gray-700 hover:bg-gray-400 "
					>Update Template</Button
				>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>