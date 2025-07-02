<script lang="ts">
	import TemplateDialog from '$lib/components/page/TemplateDialog.svelte';
	import {
		ContextMenu,
		ContextMenuContent,
		ContextMenuItem,
		ContextMenuTrigger
	} from '$lib/components/ui/context-menu/index.js';
	import { FileText } from 'lucide-svelte';

	// --- TYPES ---
	export type Template = {
		id: string;
		title: string;
		prompt: string;
		created_at: string;
	};

	// --- PROPS ---
	let {
		templates,
		onEditTemplate,
		onDeleteTemplate,
		onUpdate
	}: {
		templates: Template[];
		onEditTemplate: (template: Template) => void;
		onDeleteTemplate: (id: string) => void;
		onUpdate: () => void;
	} = $props();

	let selectedTemplate: Template | null = $state(null);
	let isTemplateDialogOpen = $state(false);

	function showTemplateDetails(template: Template) {
		selectedTemplate = template;
		isTemplateDialogOpen = true;
	}

	function formatDate(dateString: string) {
		return new Date(dateString).toLocaleString();
	}
</script>

{#if templates.length === 0}
	<div class="py-10 text-center text-gray-500">
		<p>No templates yet. Click "New Template" to add your first one.</p>
	</div>
{:else}
	<div class="mt-4 space-y-3">
		{#each templates as template (template.id)}
			<ContextMenu>
				<ContextMenuTrigger
					class="flex cursor-pointer items-center justify-between gap-4 rounded-lg bg-gray-700/50 p-4 transition-colors hover:bg-gray-700"
					onclick={() => showTemplateDetails(template)}
				>
					<div class="flex min-w-0 flex-1 items-center gap-4">
						<FileText class="h-5 w-5 flex-shrink-0 text-blue-400" />
						<span class="truncate font-medium" title={template.title}>{template.title}</span>
					</div>
					<span class="flex-shrink-0 text-sm text-gray-400">
						{formatDate(template.created_at)}
					</span>
				</ContextMenuTrigger>
				<ContextMenuContent>
					<ContextMenuItem
						onclick={(e) => {
							e.stopPropagation();
							onEditTemplate(template);
						}}
					>
						Edit
					</ContextMenuItem>
					<ContextMenuItem
						onclick={(e) => {
							e.stopPropagation();
							onDeleteTemplate(template.id);
						}}
						class="text-red-500"
					>
						Delete
					</ContextMenuItem>
				</ContextMenuContent>
			</ContextMenu>
		{/each}
	</div>
{/if}

<TemplateDialog bind:open={isTemplateDialogOpen} template={selectedTemplate} {onUpdate} />
