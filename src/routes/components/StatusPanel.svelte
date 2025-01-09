<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import * as Card from '$lib/components/ui/card';
	import { Progress } from '$lib/components/ui/progress';
	import { apiFetch, createEventSource } from '$lib/api';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Loader2, Check, AlertCircle } from 'lucide-svelte';
	import { ScrollArea } from '$lib/components/ui/scroll-area';

	interface JobStatus {
		id: number;
		fileName: string;
		status: 'pending' | 'processing' | 'diarizing' | 'completed' | 'failed';
		progress: number;
		error?: string;
	}

	let activeJobs = $state<Record<number, JobStatus>>({});
	let eventSources: Record<number, EventSource> = {};

	function getStatusColor(status: string) {
		const colors = {
			pending: 'text-yellow-500',
			processing: 'text-blue-500',
			diarizing: 'text-purple-500',
			completed: 'text-green-500',
			failed: 'text-red-500'
		};
		return colors[status] || 'text-gray-500';
	}

	function getStatusIcon(status: string) {
		const icons = {
			pending: Loader2,
			processing: Loader2,
			diarizing: Loader2,
			completed: Check,
			failed: AlertCircle
		};
		return icons[status] || Loader2;
	}

	async function fetchActiveJobs() {
		try {
			const response = await apiFetch('/api/transcribe/active');
			if (!response.ok) throw new Error('Failed to fetch active jobs');

			const jobs = await response.json();
			jobs.forEach((job: JobStatus) => {
				activeJobs[job.id] = job;
				listenToTranscriptionProgress(job.id);
			});
		} catch (error) {
			console.error('Error fetching active jobs:', error);
		}
	}

	async function listenToTranscriptionProgress(id: number) {
		if (eventSources[id]) return;

		const eventSource = await createEventSource(`/api/transcribe/${id}`);
		eventSources[id] = eventSource;

		eventSource.onmessage = (event) => {
			const progress = JSON.parse(event.data);

			activeJobs[id] = {
				...activeJobs[id],
				status: progress.status,
				progress: progress.progress || 0,
				error: progress.error
			};

			if (progress.status === 'completed' || progress.status === 'failed') {
				eventSource.close();
				delete eventSources[id];
				delete activeJobs[id];
			}
		};

		eventSource.onerror = () => {
			eventSource.close();
			delete eventSources[id];
			activeJobs[id] = {
				...activeJobs[id],
				status: 'failed',
				error: 'Lost connection to server'
			};
		};
	}

	onMount(() => {
		fetchActiveJobs();
	});

	onDestroy(() => {
		Object.values(eventSources).forEach((es) => es.close());
	});
</script>

<Card.Root
	class="mx-auto rounded-xl border border-neutral-300/30 bg-neutral-400/15 p-4 shadow-lg backdrop-blur-xl 2xl:w-[500px]"
>
	<Card.Content class="p-2">
		<h2 class="prose-md prose text-gray-50">Active Transcription Jobs</h2>
		<ScrollArea class="h-[300px] pt-4">
			<div class="space-y-4">
				{#each Object.entries(activeJobs) as [id, job]}
					<Card.Root
						class="border border-neutral-300/30 bg-neutral-400/15 p-1 shadow-lg backdrop-blur-xl"
					>
						<Card.Content class="p-4">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-3">
									<svelte:component
										this={getStatusIcon(job.status)}
										class="h-5 w-5 {job.status === 'processing' || job.status === 'diarizing'
											? 'animate-spin'
											: ''} {getStatusColor(job.status)}"
									/>
									<div class="text-gray-100">
										<p class="font-medium">{job.fileName}</p>
									</div>
								</div>
							</div>

							{#if job.status === 'processing' || job.status === 'diarizing'}
								<div class="mt-4 space-y-1">
									<Progress value={job.progress} class="h-2" />
									<p class="text-right text-sm text-gray-300">
										{job.progress}% {job.status === 'diarizing' ? 'analyzing' : 'transcribed'}
									</p>
								</div>
							{/if}

							{#if job.error}
								<Alert variant="destructive" class="mt-4">
									<AlertDescription>{job.error}</AlertDescription>
								</Alert>
							{/if}
						</Card.Content>
					</Card.Root>
				{/each}

				{#if Object.keys(activeJobs).length === 0}
					<div class="text-center text-gray-500">No active transcription jobs</div>
				{/if}
			</div>
		</ScrollArea>
	</Card.Content>
</Card.Root>
