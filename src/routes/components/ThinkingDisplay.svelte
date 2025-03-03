<script lang="ts">
	import { processThinkingSections } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import { cn } from '$lib/utils';
	import { Lightbulb, ChevronUp, ChevronDown } from 'lucide-svelte';

	const { summary, mode = 'process' } = $props<{
		summary: string;
		mode?: 'remove' | 'process';
	}>();

	let showThinkingSections = $state(false);
	let expandedSections = $state<Record<number, boolean>>({});

	$effect(() => {
		// Reset expanded sections whenever the summary changes
		expandedSections = {};
	}, [summary]);

	const toggleThinkingSections = () => {
		showThinkingSections = !showThinkingSections;
	};

	const toggleSection = (index: number) => {
		expandedSections[index] = !expandedSections[index];
	};

	const { processedText, hasThinkingSections, thinkingSections } = $derived(
		processThinkingSections(summary, mode)
	);

	// Split processed text by thinking section placeholders
	const formattedSections = $derived(() => {
		if (mode === 'remove' || !hasThinkingSections) {
			return [processedText];
		}

		const sections = processedText.split(/\[THINKING_SECTION_(\d+)\]/);
		const result = [];

		// The first element is always text
		result.push({ type: 'text', content: sections[0] });

		// Process the remaining sections
		for (let i = 1; i < sections.length; i += 2) {
			if (i < sections.length - 1) {
				const thinkingIndex = parseInt(sections[i]);
				result.push({
					type: 'thinking',
					index: thinkingIndex,
					content: thinkingSections[thinkingIndex]
				});
				result.push({ type: 'text', content: sections[i + 1] });
			}
		}

		return result;
	});
</script>

<div class="whitespace-pre-wrap text-gray-200">
	{#if hasThinkingSections && mode === 'process'}
		<div class="mb-3 flex items-center justify-between">
			<span class="text-sm text-gray-400">
				This response contains "thinking" sections from the AI.
			</span>
			<Button
				variant="outline"
				size="sm"
				class="border-gray-600 bg-neutral-700/20 text-gray-300 hover:bg-neutral-600/20"
				on:click={toggleThinkingSections}
			>
				<Lightbulb class="mr-2 h-4 w-4" />
				{showThinkingSections ? 'Hide Thinking' : 'Show Thinking'}
			</Button>
		</div>

		{#each formattedSections as section}
			{#if section.type === 'text'}
				<div>{section.content}</div>
			{:else if section.type === 'thinking' && showThinkingSections}
				<div class="my-2 rounded-md border border-gray-600 bg-gray-800/40">
					<button
						class="flex w-full items-center justify-between rounded-t-md bg-gray-700/40 px-3 py-2 text-left text-sm hover:bg-gray-700/60"
						on:click={() => toggleSection(section.index)}
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
						<div class="p-3 text-sm text-gray-300">{section.content}</div>
					{/if}
				</div>
			{/if}
		{/each}
	{:else}
		{processedText}
	{/if}
</div>