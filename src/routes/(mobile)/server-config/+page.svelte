<!-- src/routes/server-config/+page.svelte -->
<script lang="ts">
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Loader2 } from 'lucide-svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Preferences } from '@capacitor/preferences';
	import { setServerConfig, validateServerConfig } from '$lib/stores/config';

	const KEYS = {
		SERVER_URL: 'server_url',
		USERNAME: 'username',
		PASSWORD: 'password'
	} as const;

	let serverUrl = $state('');
	let username = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state('');

	async function validateAndConnect() {
		try {
			const response = await fetch(`${serverUrl}/api/auth`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ username, password })
			});

			console.log(response);

			if (!response.ok) throw new Error('Invalid credentials or server unreachable');

			await Promise.all([
				Preferences.set({ key: KEYS.SERVER_URL, value: serverUrl }),
				Preferences.set({ key: KEYS.USERNAME, value: username }),
				Preferences.set({ key: KEYS.PASSWORD, value: password })
			]);

			goto('/');
		} catch (e) {
			error = e.message || 'Failed to connect to server';
			throw e;
		}
	}

	async function handleSubmit() {
		loading = true;
		error = '';

		try {
			const isValid = await validateServerConfig({ url: serverUrl, username, password });
			if (isValid) {
				await setServerConfig({ url: serverUrl, username, password });
				goto('/');
			} else {
				error = 'Invalid credentials or server unreachable';
			}
		} catch (e) {
			error = 'Failed to save configuration';
			console.error('Config error:', e);
		} finally {
			loading = false;
		}
	}

	onMount(async () => {
		loading = true;
		try {
			const [storedUrl, storedUsername, storedPassword] = await Promise.all([
				Preferences.get({ key: KEYS.SERVER_URL }),
				Preferences.get({ key: KEYS.USERNAME }),
				Preferences.get({ key: KEYS.PASSWORD })
			]);

			if (storedUrl.value) serverUrl = storedUrl.value;
			if (storedUsername.value) username = storedUsername.value;
			if (storedPassword.value) password = storedPassword.value;

			if (storedUrl.value && storedUsername.value && storedPassword.value) {
				await validateAndConnect();
			}
		} catch (e) {
			console.error('Error loading stored credentials:', e);
		} finally {
			loading = false;
		}
	});
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-100 p-4">
	<Card class="w-full max-w-md">
		<CardHeader>
			<CardTitle class="text-center">Connect to Server</CardTitle>
		</CardHeader>
		<CardContent>
			{#if loading}
				<div class="flex justify-center py-4">
					<Loader2 class="h-6 w-6 animate-spin text-gray-600" />
				</div>
			{:else}
				<form on:submit|preventDefault={handleSubmit} class="space-y-4">
					<div class="space-y-2">
						<Label for="serverUrl">Server URL</Label>
						<Input
							id="serverUrl"
							type="url"
							placeholder="https://your-server.com"
							bind:value={serverUrl}
							required
						/>
					</div>

					<div class="space-y-2">
						<Label for="username">Username</Label>
						<Input id="username" type="text" bind:value={username} required />
					</div>

					<div class="space-y-2">
						<Label for="password">Password</Label>
						<Input id="password" type="password" bind:value={password} required />
					</div>

					{#if error}
						<Alert variant="destructive">
							<AlertDescription>{error}</AlertDescription>
						</Alert>
					{/if}

					<Button type="submit" class="w-full">Connect</Button>
				</form>
			{/if}
		</CardContent>
	</Card>
</div>
