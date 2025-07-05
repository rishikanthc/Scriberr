<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { isAuthenticated } from '$lib/stores';
	import { LoaderCircle } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import Logo from '$lib/components/Logo.svelte';

	// --- STATE ---
	let username = $state('');
	let password = $state('');
	let isLoading = $state(false);

	// --- LOGIC ---
	async function handleLogin() {
		if (!username || !password) {
			toast.error('Please enter both username and password.');
			return;
		}

		isLoading = true;

		try {
			const response = await fetch('/api/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ username, password }),
				credentials: 'include'
			});

			const result = await response.json();

			if (!response.ok) {
				throw new Error(result.error || 'Login failed. Please check your credentials.');
			}

			toast.success('Login successful!');
			isAuthenticated.set(true);
			await goto('/app', { replaceState: true });
		} catch (error) {
			const errorMessage = error instanceof Error ? error.message : 'An unknown error occurred.';
			toast.error('Login Failed', { description: errorMessage });
			password = ''; // Clear password field on failure
		} finally {
			isLoading = false;
		}
	}

	function handleKeyPress(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			handleLogin();
		}
	}
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-900 p-4">
	<div class="w-full max-w-sm rounded-xl border border-gray-700 bg-gray-800 p-8 shadow-lg">
		<div class="mb-8 text-center">
			<h1 class="text-3xl font-bold text-gray-100 flex items-center gap-3">
				<Logo size={40} strokeColor="#f0f9ff" />
				Scriberr
			</h1>
			<p class="text-gray-400">Sign in to your account</p>
		</div>

		<form class="space-y-6" on:submit|preventDefault={handleLogin} on:keypress={handleKeyPress}>
			<div class="space-y-2">
				<label for="username" class="text-sm font-medium text-gray-300">Username</label>
				<Input
					id="username"
					type="text"
					placeholder="admin"
					bind:value={username}
					disabled={isLoading}
					class="border-gray-600 bg-gray-700 text-gray-200 focus:border-blue-500 focus:ring-blue-500"
					required
				/>
			</div>

			<div class="space-y-2">
				<label for="password" class="text-sm font-medium text-gray-300">Password</label>
				<Input
					id="password"
					type="password"
					placeholder="••••••••"
					bind:value={password}
					disabled={isLoading}
					class="border-gray-600 bg-gray-700 text-gray-200 focus:border-blue-500 focus:ring-blue-500"
					required
				/>
			</div>

			<div>
				<Button
					type="submit"
					class="w-full bg-blue-600 text-white hover:bg-blue-700"
					disabled={isLoading}
				>
					{#if isLoading}
						<LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
						<span>Signing in...</span>
					{:else}
						<span>Sign In</span>
					{/if}
				</Button>
			</div>
		</form>
		<div class="mt-4 text-center">
			<p class="text-xs text-gray-500">Default credentials: admin / password</p>
		</div>
	</div>
</div>
