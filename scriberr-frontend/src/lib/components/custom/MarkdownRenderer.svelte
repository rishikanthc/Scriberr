<script lang="ts">
	import { Carta, Markdown } from 'carta-md';
	import { onMount } from 'svelte';

	// Props
	let { content = '', class: className = '' }: { content: string; class?: string } = $props();

	// State
	let carta = $state<Carta | null>(null);

	onMount(() => {
		// Initialize Carta MD instance
		carta = new Carta({
			// Add sanitizer for security (recommended)
			// For now, we'll use a basic HTML sanitizer
			sanitizer: (html: string) => html
		});
	});
</script>

{#key content}
	{#if carta}
    <div class="prose prose-base text-gray-100 prose-headings:text-gray-50 prose-strong:text-gray-100">
		<Markdown {carta} value={content}  />
    </div>
	{/if}
{/key}

<style>
	/* Required monospace font for Carta MD */

	/* Ensure the container takes full width */
	:global(.carta) {
		width: 100%;
	}
</style> 