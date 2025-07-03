<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { LoaderCircle, StopCircle } from 'lucide-svelte';

	// --- TYPES ---
	type ActiveJob = {
		id: string;
		audio_id: string;
		audio_title: string;
		status: string;
		created_at: string;
		job_type: 'transcription' | 'summarization';
	};

	// --- PROPS ---
	let {
		activeJobs,
		onTerminateJob
	}: { activeJobs: ActiveJob[]; onTerminateJob: (jobId: string) => void } = $props();

	function formatDate(dateString: string) {
		return new Date(dateString).toLocaleString();
	}
</script>

{#if activeJobs.length === 0}
	<div class="py-10 text-center text-gray-500">
		<p>No active jobs.</p>
	</div>
{:else}
	<div class="mt-4 space-y-3">
		{#each activeJobs as job (job.id)}
			<div class="flex items-center justify-between gap-4 rounded-lg bg-gray-700/50 p-4">
				<div class="flex min-w-0 flex-1 items-center gap-4">
					<LoaderCircle
						class="h-5 w-5 flex-shrink-0 animate-spin {job.job_type === 'summarization'
							? 'text-blue-400'
							: 'text-yellow-400'}"
					/>
					<div class="flex flex-col truncate">
						<span class="truncate font-medium" title={job.audio_title}>
							{job.audio_title}
						</span>
						<span class="text-xs text-gray-400 capitalize">{job.job_type}</span>
					</div>
				</div>
				<div class="flex flex-shrink-0 items-center gap-4">
					<span class="text-sm text-gray-400">{formatDate(job.created_at)}</span>
					<Button
						variant="ghost"
						size="icon"
						class="h-8 w-8 text-red-400 hover:bg-red-400/10 hover:text-red-400"
						title="Stop Job"
						onclick={() => onTerminateJob(job.id)}
					>
						<StopCircle class="h-7 w-7" />
					</Button>
				</div>
			</div>
		{/each}
	</div>
{/if}
