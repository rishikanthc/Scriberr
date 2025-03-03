<script lang="ts">
	import { enhance } from '$app/forms';
	import { goto } from '$app/navigation';
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle
	} from '$lib/components/ui/card';
	import { Label } from '$lib/components/ui/label';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { authToken, isAuthenticated } from '$lib/stores/config';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';

	let error = '';
	let isSubmitting = false;

	// Check if user is already authenticated on page load
	onMount(() => {
		if (browser) {
			const storedToken = localStorage.getItem('sessionToken');
			const storedExpires = localStorage.getItem('sessionExpires');
			
			if (storedToken && storedExpires) {
				const expiresAt = new Date(storedExpires).getTime();
				const now = Date.now();
				
				// If token is valid, redirect to home
				if (expiresAt > now) {
					authToken.set(storedToken);
					isAuthenticated.set(true);
					goto('/');
				}
			}
		}
	});

	function handleSubmit(event: SubmitEvent) {
		isSubmitting = true;
		
		return async ({ result }) => {
			isSubmitting = false;
			
			if (result.type === 'success') {
				// Store the token in localStorage and update auth state
				if (result.data?.token) {
					const token = result.data.token;
					const expiresAt = result.data.expiresAt || 
						new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(); // Default to 30 days
					
					// Update localStorage
					localStorage.setItem('sessionToken', token);
					localStorage.setItem('sessionExpires', expiresAt);
					
					// Update auth state
					authToken.set(token);
					isAuthenticated.set(true);
					
					// Navigate to home
					goto('/');
				} else {
					error = 'Authentication failed: No token received';
				}
			} else {
				error = result.data?.message || 'Invalid credentials';
			}
		};
	}
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-50 p-4">
	<Card class="w-full max-w-md">
		<CardHeader>
			<CardTitle>Admin Login</CardTitle>
			<CardDescription>Enter your credentials to access the dashboard</CardDescription>
		</CardHeader>
		<CardContent>
			{#if error}
				<Alert variant="destructive" class="mb-4">
					<AlertDescription>{error}</AlertDescription>
				</Alert>
			{/if}
			<form method="POST" use:enhance={handleSubmit} class="space-y-4">
				<div class="space-y-2">
					<Label for="username">Username</Label>
					<Input type="text" id="username" name="username" required />
				</div>
				<div class="space-y-2">
					<Label for="password">Password</Label>
					<Input type="password" id="password" name="password" required />
				</div>
				<Button type="submit" class="w-full" disabled={isSubmitting}>
					{isSubmitting ? 'Signing in...' : 'Sign in'}
				</Button>
			</form>
		</CardContent>
	</Card>
</div>