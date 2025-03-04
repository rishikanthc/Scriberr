<!-- src/routes/+layout.svelte -->
<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import ConfigWizard from './ConfigWizard.svelte';
	import { browser } from '$app/environment';
	import { Preferences } from '@capacitor/preferences';
	import { apiFetch } from '$lib/api';

	let { children, data } = $props();
	let check = $state(null);

	onMount(async () => {
		try {
			const response = await apiFetch('/api/check-config');
			if (!response.ok) throw new Error('Failed to complete startup check');
			check = await response.json();
			console.log(check);
		} catch (error) {
			console.error('Error with check-config api', error);
		}

		if (browser && window.Capacitor?.isNative) {
			const { value } = await Preferences.get({ key: 'server_url' });
			if (!value) {
				goto('/server-config');
			}
		}
	});
</script>

{#if !check?.isConfigured}
	<ConfigWizard />
{:else}
	{@render children()}
{/if}
