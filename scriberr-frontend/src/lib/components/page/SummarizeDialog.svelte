<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Button } from '$lib/components/ui/button/index.js';

	// --- TYPES ---
	type AudioRecord = {
		id: string;
		title: string;
		created_at: string;
		transcript: string;
	};

	type Template = {
		id: string;
		title: string;
		prompt: string;
	};

	// --- PROPS ---
	let {
		open = $bindable(),
		recordToSummarize,
		selectedTemplateId = $bindable(),
		selectedModel = $bindable(),
		templates,
		modelOptions,
		onStartSummarization
	}: {
		open: boolean;
		recordToSummarize: AudioRecord | null;
		selectedTemplateId: string;
		selectedModel: string;
		templates: Template[];
		modelOptions: string[];
		onStartSummarization: () => void;
	} = $props();

	function getSelectedTemplateName() {
		if (!selectedTemplateId) return 'Select a prompt';
		const template = templates.find((t) => t.id === selectedTemplateId);
		return template ? template.title : 'Select a prompt';
	}

	function getSelectedModelName() {
		if (!selectedModel) return 'Select a model';
		return selectedModel;
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="border-none bg-gray-700 text-gray-200 sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Select Summarization Prompt</Dialog.Title>
			<Dialog.Description class="pt-2 text-gray-400">
				Choose a prompt to guide the summarization for your transcript.
			</Dialog.Description>
		</Dialog.Header>
		<div class="grid gap-4 py-4">
			<p class="text-sm text-gray-400">
				For: <span class="font-medium text-gray-200">{recordToSummarize?.title}</span>
			</p>
			<Select.Root bind:value={selectedTemplateId} type="single">
				<Select.Trigger class="w-full">
					{getSelectedTemplateName()}
				</Select.Trigger>
				<Select.Content>
					{#each templates as template (template.id)}
						<Select.Item value={template.id}>{template.title}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
			<Select.Root bind:value={selectedModel} type="single">
				<Select.Trigger class="w-full">
					{getSelectedModelName()}
				</Select.Trigger>
				<Select.Content>
					{#each modelOptions as model}
						<Select.Item value={model}>{model}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>
		<Dialog.Footer>
			<Button
				onclick={onStartSummarization}
				disabled={!selectedTemplateId || !selectedModel}
				class="w-full bg-blue-500 text-gray-100 hover:bg-blue-600"
			>
				Start Summarization
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
