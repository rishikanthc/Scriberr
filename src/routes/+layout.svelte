<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { Button } from 'bits-ui';
	import { SunMedium, Moon } from 'lucide-svelte';
	import Processing from '$lib/components/Processing.svelte';

	let darkMode = false;

	function toggleDarkMode() {
		darkMode = !darkMode;
		if (darkMode) {
			document.body.classList.add('dark');
		} else {
			document.body.classList.remove('dark');
		}
	}

	onMount(() => {
		// Check for user's preference
		if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
			darkMode = true;
			document.body.classList.add('dark');
		}
	});
</script>

<div
	class="mx-auto flex h-[50px] items-center justify-between px-2 transition-colors duration-300 ease-in-out lg:w-[1000px]"
>
	<div class="font-[Megrim] text-3xl text-black dark:text-white">scriber</div>
	<div>
		<Button.Root
			on:click={toggleDarkMode}
			class="inline-flex items-center rounded-md bg-carbongray-100 p-2 text-gray-800 shadow-sm hover:bg-gray-300 active:transition-all dark:bg-carbongray-700 dark:text-white dark:hover:bg-gray-600"
		>
			{#if darkMode}
				<SunMedium class="h-5 w-5 transition-transform duration-300 ease-in-out hover:rotate-12" />
			{:else}
				<Moon class="h-5 w-5 transition-transform duration-300 ease-in-out hover:-rotate-12" />
			{/if}
		</Button.Root>
	</div>
</div>
<div
	class="container mx-auto h-[704px] max-h-screen border-carbongray-100 text-black shadow transition-colors duration-300 ease-in-out dark:border-carbongray-700 dark:text-white lg:max-w-[1000px] lg:rounded-xl lg:border 2xl:mt-[50px] 2xl:h-[900px]"
>
	<slot></slot>
</div>

<style lang="postcss">
	:global(body) {
		@apply bg-white text-black transition-colors duration-300 ease-in-out;
	}

	:global(body.dark) {
		@apply bg-carbongray-800 text-white;
	}
</style>
