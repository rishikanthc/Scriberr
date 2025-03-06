<!-- src/routes/+layout.svelte -->
<script lang="ts">
	import '../app.css';
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import ConfigWizard from './ConfigWizard.svelte';
	import { browser } from '$app/environment';
	import { Preferences } from '@capacitor/preferences';
	import { apiFetch, checkAndRefreshToken } from '$lib/api';
	import { authToken, isAuthenticated } from '$lib/stores/config';

	let { children, data } = $props();
	let check = $state(null);
	let tokenRefreshInterval: number | undefined;

	// Initialize auth state from localStorage on mount
	function initAuthState() {
		if (browser) {
			const storedToken = localStorage.getItem('sessionToken');
			const storedExpires = localStorage.getItem('sessionExpires');
			
			if (storedToken && storedExpires) {
				const expiresAt = new Date(storedExpires).getTime();
				const now = Date.now();
				
				// Check if token is valid (not expired)
				if (expiresAt > now) {
					authToken.set(storedToken);
					isAuthenticated.set(true);
				} else {
					// Token is expired, clear it
					localStorage.removeItem('sessionToken');
					localStorage.removeItem('sessionExpires');
					authToken.set('');
					isAuthenticated.set(false);
					
					// Redirect to login if not already there
					const currentPath = window.location.pathname;
					if (currentPath !== '/login') {
						goto('/login');
					}
				}
			} else {
				// No token found
				authToken.set('');
				isAuthenticated.set(false);
			}
		}
	}

	onMount(async () => {
		// Initialize auth state from localStorage
		initAuthState();
		
		// Setup token refresh interval (check every 5 minutes)
		if (browser) {
			tokenRefreshInterval = window.setInterval(() => {
				checkAndRefreshToken().catch(console.error);
			}, 300000); // 5 minutes
		}
		
		try {
			console.log("Calling config check API...");
			const response = await apiFetch('/api/check-config');
			console.log("Config check API response status:", response.status);
			
			if (!response.ok) {
				throw new Error(`Failed to complete startup check, status: ${response.status}`);
			}
			
			const data = await response.json();
			console.log("Config check API response data:", data);
			check = data;
			
			// Force ConfigWizard to show if system is not configured
			if (!data.isConfigured) {
				console.log("System not configured, showing setup wizard");
			} else {
				console.log("System already configured, showing main application");
			}
		} catch (error) {
			console.error('Error with check-config api', error);
			// Set check to show ConfigWizard on error
			check = { isConfigured: false };
			console.log("Showing setup wizard due to API error");
		}

		if (browser && window.Capacitor?.isNative) {
			const { value } = await Preferences.get({ key: 'server_url' });
			if (!value) {
				goto('/server-config');
			}
		}
	});
	
	onDestroy(() => {
		// Clear the token refresh interval
		if (browser && tokenRefreshInterval) {
			clearInterval(tokenRefreshInterval);
		}
	});
</script>

{#if !check?.isConfigured}
	<ConfigWizard />
{:else}
	{@render children()}
{/if}