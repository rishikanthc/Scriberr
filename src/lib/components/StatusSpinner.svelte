<script lang="ts">
	import { Loader, CircleCheck, CircleX } from 'lucide-svelte';
	export let msg;
	export let size = 13;
	export let success = -1;
	export let pos = 'end';
</script>

<div class="flex w-fit items-center justify-start gap-2 p-0">
	{#if pos === 'end'}
		<div class="text-base">{msg}</div>
		{#if success < 0}
			<Loader {size} class="animate-spin text-carbonblue-500" />
		{:else if success > 0}
			<CircleCheck {size} class="animate-pulse text-[#24a147]" />
		{:else}
			<CircleX {size} class="animate-pulse" />
		{/if}
	{:else}
		{#if success < 0}
			<Loader {size} class="animate-spin text-carbonblue-500" />
		{:else if success > 0}
			<CircleCheck {size} class="animate-pulse text-[#24a147]" />
		{:else}
			<CircleX {size} class="animate-pulse" />
		{/if}
		<div class="text-base">{msg}</div>
	{/if}
</div>
