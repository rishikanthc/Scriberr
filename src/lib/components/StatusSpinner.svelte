<script lang="ts">
	import { Loader } from 'lucide-svelte';
	export let msg;
	export let size = 15;
</script>

<div class="flex w-fit items-center justify-center gap-1 p-2">
	<Loader {size} class="animate-spin" />
</div>
