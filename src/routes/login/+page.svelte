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

	let error = '';

	function handleSubmit(event: SubmitEvent) {
		return async ({ result }) => {
			if (result.type === 'success') {
				// Store the token in localStorage
				if (result.data?.token) {
					localStorage.setItem('sessionToken', result.data.token);
					if (result.data.expiresAt) {
						localStorage.setItem('sessionExpires', result.data.expiresAt);
					}
				}
				goto('/');
			} else {
				error = 'Invalid credentials';
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
				<Button type="submit" class="w-full">Sign in</Button>
			</form>
		</CardContent>
	</Card>
</div>
