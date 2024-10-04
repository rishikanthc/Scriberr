<script lang="ts">
	import { onMount, onDestroy, createEventDispatcher } from 'svelte';
	import { Progress } from 'bits-ui';
	import Processing from './Processing.svelte';

	export let state: boolean;
	const dispatch = createEventDispatcher();

	let jobs = {
		waiting: [],
		active: []
	};
	let interval;
	let totalPendingJobs = 0;
	let totalActiveJobs = 0;
	let overallProgress = 0;

	let totalJobs;
	let progressSum;

	// Function to fetch jobs from API
	async function fetchJobs() {
		try {
			const response = await fetch('/api/jobs');
			if (response.ok) {
				const data = await response.json();
				jobs.waiting = data.waiting;
				jobs.active = data.active;
				totalPendingJobs = jobs.waiting.length;
				totalActiveJobs = jobs.active.length;

				// Calculate overall progress
				totalJobs = totalPendingJobs + totalActiveJobs;
				progressSum = jobs.active.reduce((sum, job) => sum + job.progress, 0);
				overallProgress = totalJobs > 0 ? progressSum / totalJobs : 100;
				console.log(progressSum, totalJobs);
			}
		} catch (error) {
			console.error('Error fetching jobs:', error);
		}
	}

	// Polling mechanism
	function startPolling(pollingInterval: number) {
		interval = setInterval(async () => {
			await fetchJobs();

			// If no more pending jobs, stop polling and reset state
			if (jobs.waiting.length === 0 && jobs.active.length === 0) {
				clearInterval(interval);
				overallProgress = 100;
				state = false; // Reset state when no more pending jobs
				dispatch('finishedProcessing');
			}
		}, pollingInterval);
	}

	// Watch for state changes to trigger polling
	$: {
		if (state) {
			clearInterval(interval); // Clear any existing intervals
			startPolling(1000); // Poll every 1 second when state is active
		} else {
			clearInterval(interval); // Clear any existing intervals
			startPolling(5000); // Poll every 5 seconds when state is inactive
		}
	}

	// Clean up interval when component is destroyed
	onDestroy(() => {
		clearInterval(interval);
	});
</script>

<div class="flex flex-col justify-center p-4">
	{#if jobs.active.length > 0}
		<div>
			<Processing duration={1} message="Transcribing" color="#0043ce" />
		</div>
		<div class="grid grid-cols-2 gap-4">
			{#each jobs.active as job (job.id)}
				<div class="rounded-md p-4 shadow-md dark:bg-carbongray-700">
					<div class="text-sm">Record ID: {job.data.recordId}</div>
					<Progress.Root
						value={job.progress}
						max={100}
						class="relative h-1 w-full overflow-hidden rounded-full bg-carbongray-600"
					>
						<div
							class="h-full w-full bg-carbonblue-400 transition-all duration-1000 ease-in-out"
							style={`transform: translateX(-${100 - (100 * (job.progress ?? 0)) / 100}%)`}
						></div>
					</Progress.Root>
					<p class="mt-2 text-sm">Progress: {job.progress}%</p>
				</div>
			{/each}
		</div>
		<div></div>
	{:else}
		<p class="mt-4 text-center text-lg text-gray-500">No active jobs</p>
	{/if}

	<!-- Overall progress bar for pending jobs -->
	{#if totalPendingJobs > 0}
		<Progress.Root max={totalJobs} bind:value={progressSum} class="mt-4">
			<div
				class="h-2 rounded-full bg-blue-500 transition-all"
				style={`width: ${(progressSum / totalJobs) * 100}%`}
			></div>
		</Progress.Root>
	{/if}
</div>
