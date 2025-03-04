<script lang="ts">
	import { processThinkingSections } from '$lib/utils';
	import { Lightbulb, ChevronUp, ChevronDown } from 'lucide-svelte';
	import { marked } from 'marked';

	// Props
	const { summary = '', initialShowThinking = true } = $props<{
		summary: string;
		initialShowThinking?: boolean;
	}>();

	// Local state
	let showThinkingSections = $state(initialShowThinking);
	let expandedSections = $state<Record<number, boolean>>({});

	// Sync with parent
	$effect(() => {
		showThinkingSections = initialShowThinking;
		console.log('ThinkingDisplay - showThinkingSections updated:', showThinkingSections);
	});

	// Get the processed output (with or without thinking sections)
	function getProcessedText() {
		if (!summary) return "";
		
		if (showThinkingSections) {
			// Show thinking sections
			return processThinkingSections(summary, 'process').processedText;
		} else {
			// Hide thinking sections
			return processThinkingSections(summary, 'remove').processedText;
		}
	}

	// Check if summary has thinking sections
	function hasThinkingSections() {
		if (!summary) return false;
		return processThinkingSections(summary, 'process').hasThinkingSections;
	}

	// Get thinking sections array
	function getThinkingSections() {
		if (!summary) return [];
		return processThinkingSections(summary, 'process').thinkingSections;
	}

	// Convert text to HTML with markdown rendering
	function renderMarkdown(text: string) {
		if (!text) return '';
		return marked(text);
	}

	// Format sections for display
	function getFormattedSections() {
		if (!summary || !hasThinkingSections()) {
			return [{ type: 'text', content: summary }];
		}

		const processedText = getProcessedText();
		const thinkingSections = getThinkingSections();
		const parts = processedText.split(/\[THINKING_SECTION_(\d+)\]/);
		const result = [];
		
		// Add the first text part
		result.push({ type: 'text', content: parts[0] });
		
		// Add the thinking sections and remaining text parts
		for (let i = 1; i < parts.length; i += 2) {
			const index = parseInt(parts[i]);
			// Add thinking section
			result.push({ 
				type: 'thinking', 
				index, 
				content: thinkingSections[index] 
			});
			
			// Add next text part if it exists
			if (i + 1 < parts.length) {
				result.push({ type: 'text', content: parts[i + 1] });
			}
		}
		
		return result;
	}

	// Toggle a specific thinking section
	function toggleSection(index: number) {
		expandedSections[index] = !expandedSections[index];
	}

	// Initialize expanded state for all sections
	$effect(() => {
		if (hasThinkingSections()) {
			const newSections = {};
			getThinkingSections().forEach((_, i) => {
				newSections[i] = true; // Auto-expand
			});
			expandedSections = newSections;
		}
	});
</script>

<div class="whitespace-pre-wrap text-gray-200">
	{#if !summary}
		<!-- Empty state -->
		<div class="text-gray-400">No summary available</div>
	{:else if !hasThinkingSections()}
		<!-- No thinking sections - render markdown -->
		<div class="prose prose-invert prose-sm max-w-none" class:prose-a:text-blue-400={true}>
			{@html renderMarkdown(summary)}
		</div>
	{:else if !showThinkingSections}
		<!-- Plain text without thinking sections - render markdown -->
		<div class="prose prose-invert prose-sm max-w-none" class:prose-a:text-blue-400={true}>
			{@html renderMarkdown(processThinkingSections(summary, 'remove').processedText)}
		</div>
	{:else}
		<!-- Display with thinking sections -->
		{#each getFormattedSections() as section}
			{#if section.type === 'text'}
				<!-- Regular text content with markdown -->
				<div class="prose prose-invert prose-sm max-w-none mb-2" class:prose-a:text-blue-400={true}>
					{@html renderMarkdown(section.content)}
				</div>
			{:else if section.type === 'thinking'}
				<!-- Thinking section -->
				<div class="my-3 rounded-md border border-gray-600 bg-gray-800/40">
					<button
						class="flex w-full items-center justify-between rounded-t-md bg-gray-700/40 px-3 py-2 text-left text-sm hover:bg-gray-700/60"
						onclick={() => toggleSection(section.index)}
					>
						<span class="flex items-center">
							<Lightbulb class="mr-2 h-4 w-4 text-amber-500" />
							AI's Thinking Process
						</span>
						{#if expandedSections[section.index]}
							<ChevronUp class="h-4 w-4" />
						{:else}
							<ChevronDown class="h-4 w-4" />
						{/if}
					</button>
					{#if expandedSections[section.index]}
						<div class="p-3 text-sm text-gray-300">
							<div class="prose prose-invert prose-sm max-w-none" class:prose-a:text-blue-400={true}>
								{@html renderMarkdown(section.content)}
							</div>
						</div>
					{/if}
				</div>
			{/if}
		{/each}
	{/if}
</div>