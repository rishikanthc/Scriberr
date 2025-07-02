<script lang="ts">
	import { goto } from '$app/navigation';
	import { LoaderCircle } from 'lucide-svelte';
	import { isAuthenticated, checkAuthWithRedirect } from '$lib/stores';
	import { onMount } from 'svelte';

	let { children } = $props();
	let isChecking = $state(true);
	let isAuthenticatedLocal = $state(false);

	// Immediately check authentication when the component mounts
	onMount(async () => {
		try {
			const authenticated = await checkAuthWithRedirect();
			isAuthenticatedLocal = authenticated;
			
			// If not authenticated, the redirect should have already happened
			// but we can also handle it here as a fallback
			if (!authenticated) {
				goto('/login', { replaceState: true });
			}
		} catch (error) {
			console.error('Auth check failed:', error);
			goto('/login', { replaceState: true });
		} finally {
			isChecking = false;
		}
	});

	// Watch for auth status changes
	$effect(() => {
		if (!isChecking && !$isAuthenticated && isAuthenticatedLocal) {
			isAuthenticatedLocal = false;
			goto('/login', { replaceState: true });
		}
	});
</script>

{#if isChecking || !isAuthenticatedLocal}
	<div class="flex h-screen w-full items-center justify-center bg-gray-900">
		<LoaderCircle class="h-8 w-8 animate-spin text-gray-400" />
	</div>
{:else}
	<!-- Only render if authenticated -->
	{@render children()}
{/if}
