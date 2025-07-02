<script lang="ts">
	import { goto } from '$app/navigation';
	import { isAuthenticated, checkAuthStatus } from '$lib/stores';
	import { LoaderCircle } from 'lucide-svelte';

	// This page acts as a router. It checks the auth status
	// and redirects the user to the correct page.
	$effect(() => {
		async function performRedirect() {
			await checkAuthStatus();

			if ($isAuthenticated) {
				goto('/app', { replaceState: true });
			} else {
				goto('/login', { replaceState: true });
			}
		}

		performRedirect();
	});
</script>

<!-- Show a loading spinner while the check and redirect is happening -->
<div class="flex h-screen w-full items-center justify-center bg-gray-900">
	<LoaderCircle class="h-8 w-8 animate-spin text-gray-400" />
</div>
