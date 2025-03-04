<!-- src/lib/components/ConfigWizard.svelte -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import * as Card from '$lib/components/ui/card';
	import WhisperSetup from './WhisperSetup.svelte';
	import { Button } from '$lib/components/ui/button';
	import { apiFetch } from '$lib/api';

	let currentStep = 1;
	let config = {
		// Your configuration options
	};

	async function completeSetup() {
		const response = await apiFetch('/api/check-config');
		if (!response.ok) throw new Error('Failed to complete startup check');
		const check = await response.json();

		if (!check.needsConfiguration) {
			goto('/');
		}
	}
</script>

<Card.Root class="mx-auto mt-16 h-[784px] w-[784px]">
	<Card.Header>
		<Card.Title>Setup Wizard</Card.Title>
		<Card.Description>Configure Scriberr</Card.Description>
	</Card.Header>
	<Card.Content>
		<WhisperSetup />

		{#if currentStep === 1}
			<div class="mt-6 flex justify-end">
				<Button variant="default" onclick={completeSetup}>Complete Setup</Button>
			</div>
		{/if}
	</Card.Content>
</Card.Root>
