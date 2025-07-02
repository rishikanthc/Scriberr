<script lang="ts">
	import { Button, buttonVariants } from '$lib/components/ui/button/index.js';
	import * as Popover from '$lib/components/ui/popover/index.js';
	import * as Tabs from '$lib/components/ui/tabs/index.js';
	import { FilePlus, LoaderCircle, LogOut, Mic, Upload, Youtube } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import { cn } from '$lib/utils';
	import { isAuthenticated } from '$lib/stores';

	// Page-specific components
	import AudioTab from '$lib/components/page/AudioTab.svelte';
	import DetailsDialog from '$lib/components/page/DetailsDialog.svelte';
	import JobsTab from '$lib/components/page/JobsTab.svelte';
	import ModelSelectDialog from '$lib/components/page/ModelSelectDialog.svelte';
	import RecorderDialog from '$lib/components/page/RecorderDialog.svelte';
	import TemplateTab from '$lib/components/page/TemplateTab.svelte';
	import type { SummaryTemplate } from '$lib/components/page/TemplateDialog.svelte';
	import TemplateDialog from '$lib/components/page/TemplateDialog.svelte';
	import SummarizeDialog from '$lib/components/page/SummarizeDialog.svelte';
	import YouTubeDialog from '$lib/components/page/YouTubeDialog.svelte';

	// --- TYPES ---
	type AudioRecord = {
		id: string;
		title: string;
		created_at: string;
		transcript: string;
		downloading?: boolean;
	};

	type ActiveJob = {
		id: string;
		audio_id: string;
		audio_title: string;
		status: string;
		created_at: string;
		job_type: 'transcription' | 'summarization';
	};

	type JobStatus = 'processing' | 'completed' | 'failed';

	// --- STATE ---
	let fileInput: HTMLInputElement;
	let responseText = $state('');
	let isUploading = $state(false);
	let records: AudioRecord[] = $state([]);
	let activeJobs: ActiveJob[] = $state([]);
	let transcriptionStatus: Record<string, JobStatus> = $state({});
	let summarizationStatus: Record<string, JobStatus> = $state({});
	let templates: SummaryTemplate[] = $state([]);
	let activeTab = $state('audio');

	// Dialog states
	let selectedRecord: AudioRecord | null = $state(null);
	let isDetailDialogOpen = $state(false);
	let isModelSelectOpen = $state(false);
	let recordToTranscribe: AudioRecord | null = $state(null);
	let isRecorderOpen = $state(false);
	let isYouTubeDialogOpen = $state(false);
	let isTemplateDialogOpen = $state(false);
	let selectedTemplate: SummaryTemplate | null = $state(null);
	let isSummarizeDialogOpen = $state(false);
	let recordToSummarize: AudioRecord | null = $state(null);

	// Model/Template selection state
	let selectedModel = $state('small'); // Default model for transcription
	let selectedSummaryModel = $state('gpt-3.5-turbo'); // Default model for summarization
	let selectedTemplateId = $state('');
	const modelSizes = ['tiny', 'base', 'small', 'medium', 'large-v1', 'large-v2', 'large-v3'];
	const summaryModelOptions = ['gpt-3.5-turbo', 'gpt-4', 'gpt-4-turbo', 'gpt-4o', 'gpt-4o-mini'];

	// --- EFFECTS ---
	$effect(() => {
		// Fetch initial data and set up polling
		fetchRecords();
		fetchActiveJobs();
		fetchTemplates();
		const jobsInterval = setInterval(fetchActiveJobs, 5000);
		const recordsInterval = setInterval(fetchRecords, 5000);

		return () => {
			clearInterval(jobsInterval);
			clearInterval(recordsInterval);
		};
	});

	// Side-effect for closing dialogs
	$effect(() => {
		if (!isDetailDialogOpen) {
			selectedRecord = null;
		}
	});

	$effect(() => {
		if (!isModelSelectOpen) {
			recordToTranscribe = null;
		}
	});

	$effect(() => {
		if (!isTemplateDialogOpen) {
			selectedTemplate = null;
		}
	});

	// --- API CALLS ---

	async function fetchTemplates() {
		try {
			const response = await fetch('/api/summary-templates', { credentials: 'include' });
			if (!response.ok) throw new Error('Failed to fetch templates');
			const data = (await response.json()) || [];
			templates = data;
		} catch (error) {
			console.error('Error fetching templates:', error);
			toast.error('Failed to load templates.');
			templates = []; // reset on error
		}
	}

	async function handleDeleteTemplate(id: string) {
		try {
			const response = await fetch(`/api/summary-templates/${id}`, {
				method: 'DELETE',
				credentials: 'include'
			});
			if (!response.ok) {
				const result = await response.json();
				throw new Error(result.error || 'Unknown server error.');
			}
			toast.success('Template deleted successfully.');
			await fetchTemplates(); // Refresh the list
		} catch (error) {
			const errorMessage = error instanceof Error ? error.message : 'An unknown error occurred.';
			toast.error('Deletion error', { description: errorMessage });
		}
	}

	async function fetchRecords() {
		try {
			const response = await fetch('/api/audio/all', { credentials: 'include' });
			if (!response.ok) {
				toast.error('Failed to fetch recordings.');
				return;
			}
			const data: AudioRecord[] = (await response.json()) || [];
			records = data;

			const newStatus = { ...transcriptionStatus };
			for (const record of data) {
				if (
					record.transcript &&
					record.transcript !== '{}' &&
					newStatus[record.id] !== 'processing'
				) {
					newStatus[record.id] = 'completed';
				}
			}
			transcriptionStatus = newStatus;
		} catch (error) {
			toast.error('An error occurred while fetching recordings.');
		}
	}

	async function fetchActiveJobs() {
		try {
			const response = await fetch('/api/transcribe/jobs/active', { credentials: 'include' });
			if (!response.ok) {
				toast.error('Failed to fetch active jobs.');
				return;
			}
			const data: ActiveJob[] = (await response.json()) || [];
			activeJobs = data;

			const newStatus = { ...transcriptionStatus };
			for (const job of data) {
				if (newStatus[job.audio_id] !== 'completed') {
					newStatus[job.audio_id] = 'processing';
				}
			}
			transcriptionStatus = newStatus;
		} catch (error) {
			toast.error('An error occurred while fetching active jobs.');
		}
	}

	function pollStatus(jobId: string, audioId: string) {
		const interval = setInterval(async () => {
			try {
				const res = await fetch(`/api/transcribe/status/${jobId}`, { credentials: 'include' });
				if (!res.ok) {
					clearInterval(interval);
					transcriptionStatus[audioId] = 'failed';
					toast.error('Polling failed', { description: 'Could not retrieve job status.' });
					return;
				}

				const job = await res.json();
				if (job.status === 'completed') {
					clearInterval(interval);
					transcriptionStatus[audioId] = 'completed';
					toast.success('Transcription complete!');
					await fetchRecords();
					await fetchActiveJobs();
				} else if (job.status === 'failed') {
					clearInterval(interval);
					transcriptionStatus[audioId] = 'failed';
					toast.error('Transcription failed', { description: job.error });
					await fetchActiveJobs();
				}
			} catch (e) {
				clearInterval(interval);
				transcriptionStatus[audioId] = 'failed';
				toast.error('Polling error', {
					description: 'Could not connect to server to get status.'
				});
			}
		}, 3000);
	}

	function pollSummaryStatus(jobId: string, audioId: string) {
		const interval = setInterval(async () => {
			try {
				const res = await fetch(`/api/summarize/status/job/${jobId}`, { credentials: 'include' });
				if (!res.ok) {
					clearInterval(interval);
					const newStatus = { ...summarizationStatus };
					newStatus[audioId] = 'failed';
					summarizationStatus = newStatus;
					toast.error('Polling failed', {
						description: 'Could not retrieve summarization job status.'
					});
					return;
				}

				const job = await res.json();
				if (job.status === 'completed') {
					clearInterval(interval);
					const newStatus = { ...summarizationStatus };
					delete newStatus[audioId];
					summarizationStatus = newStatus;
					toast.success('Summarization complete!');
					await fetchRecords(); // refetch to get new summary
					await fetchActiveJobs();
				} else if (job.status === 'failed') {
					clearInterval(interval);
					const newStatus = { ...summarizationStatus };
					newStatus[audioId] = 'failed';
					summarizationStatus = newStatus;
					toast.error('Summarization failed', { description: job.error });
					await fetchActiveJobs();
				}
			} catch (e) {
				clearInterval(interval);
				const newStatus = { ...summarizationStatus };
				newStatus[audioId] = 'failed';
				summarizationStatus = newStatus;
				toast.error('Polling error', {
					description: 'Could not connect to server to get status.'
				});
			}
		}, 3000);
	}

	async function transcribe(audioId: string, modelSize: string) {
		if (transcriptionStatus[audioId] === 'processing') {
			toast.info('Transcription is already in progress.');
			return;
		}
		transcriptionStatus[audioId] = 'processing';
		activeTab = 'jobs';

		try {
			const response = await fetch('/api/transcribe', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ audio_id: audioId, model_size: modelSize }),
				credentials: 'include'
			});

			if (!response.ok) {
				const result = await response.json();
				throw new Error(result.error || 'Failed to start transcription job.');
			}

			const result = await response.json();
			toast.info('Transcription started...', { description: `Job ID: ${result.job_id}` });
			await fetchActiveJobs();
			pollStatus(result.job_id, audioId);
		} catch (error) {
			const errorMessage = error instanceof Error ? error.message : 'An unknown error occurred.';
			toast.error('Transcription error', { description: errorMessage });
			delete transcriptionStatus[audioId];
		}
	}

	async function summarize() {
		if (!recordToSummarize || !selectedTemplateId) return;
		const audioId = recordToSummarize.id;
		const recordTitle = recordToSummarize.title;

		// Optimistically update UI
		const newStatus = { ...summarizationStatus };
		newStatus[audioId] = 'processing';
		summarizationStatus = newStatus;
		activeTab = 'jobs';

		try {
			const response = await fetch('/api/summarize', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					audio_id: audioId,
					template_id: selectedTemplateId,
					model: selectedSummaryModel
				}),
				credentials: 'include'
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || 'Failed to start summarization');
			}

			const result = await response.json();
			toast.info(`Summarization started for ${recordTitle}`);
			await fetchActiveJobs();
			pollSummaryStatus(result.job_id, audioId);
		} catch (error) {
			const errorMessage = error instanceof Error ? error.message : 'An unknown error occurred.';
			toast.error('Summarization error', { description: errorMessage });
			// Rollback optimistic update
			const newStatus = { ...summarizationStatus };
			delete newStatus[audioId];
			summarizationStatus = newStatus;
		}
	}

	async function handleLogout() {
		try {
			const response = await fetch('/api/auth/logout', { method: 'POST', credentials: 'include' });
			if (!response.ok) {
				const result = await response.json();
				throw new Error(result.error || 'Logout failed');
			}
			toast.success('Logged out successfully.');
			isAuthenticated.set(false);
			// The layout will handle the redirect.
		} catch (error) {
			const errorMessage = error instanceof Error ? error.message : 'An unknown error occurred.';
			toast.error('Logout failed', { description: errorMessage });
		}
	}

	async function handleSaveRecording(blob: Blob, title: string) {
		isUploading = true;
		responseText = 'Uploading and processing recording...';
		const formData = new FormData();
		formData.append('audio', blob, `${title}.webm`);
		formData.append('title', title);

		try {
			const response = await fetch('/api/audio', {
				method: 'POST',
				body: formData,
				credentials: 'include'
			});
			const result = await response.json();

			if (!response.ok) {
				throw new Error(result.error || 'Unknown upload error');
			}
			toast.success('Upload successful!', { description: `Recording ID: ${result.id}` });
			await fetchRecords();
		} catch (error) {
			const errorMessage = error instanceof Error ? error.message : 'An unknown error occurred.';
			toast.error('Upload failed', { description: errorMessage });
		} finally {
			isUploading = false;
			responseText = '';
		}
	}

	async function handleYouTubeDownload(url: string, title: string) {
		isUploading = true;
		responseText = 'Downloading YouTube audio...';
		
		// Create a temporary record to show in the UI
		const tempId = `temp-${Date.now()}`;
		const tempRecord: AudioRecord = {
			id: tempId,
			title: title,
			created_at: new Date().toISOString(),
			transcript: '{}',
			downloading: true
		};
		
		// Add to records list
		records = [tempRecord, ...records];

		try {
			const response = await fetch('/api/youtube', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ url, title }),
				credentials: 'include'
			});
			const result = await response.json();

			if (!response.ok) {
				throw new Error(result.error || 'Unknown download error');
			}
			
			toast.success('YouTube download successful!', { description: `Audio ID: ${result.id}` });
			
			// Remove temporary record and fetch updated records
			records = records.filter(r => r.id !== tempId);
			await fetchRecords();
		} catch (error) {
			const errorMessage = error instanceof Error ? error.message : 'An unknown error occurred.';
			toast.error('YouTube download failed', { description: errorMessage });
			
			// Remove temporary record on error
			records = records.filter(r => r.id !== tempId);
		} finally {
			isUploading = false;
			responseText = '';
		}
	}

	async function handleFileSelect(event: Event) {
		const target = event.target as HTMLInputElement;
		const files = target.files;
		if (!files || files.length === 0) return;

		isUploading = true;
		responseText = `Uploading and processing ${files.length} file(s)...`;

		const uploadPromises = Array.from(files).map(async (file) => {
			try {
				const formData = new FormData();
				formData.append('audio', file);
				formData.append('title', file.name.replace(/\.[^/.]+$/, ''));

				const response = await fetch('/api/audio', {
					method: 'POST',
					body: formData,
					credentials: 'include'
				});
				const result = await response.json();

				if (!response.ok) {
					throw new Error(result.error || `Unknown upload error for ${file.name}`);
				}
				toast.success('Upload successful!', { description: file.name });
			} catch (error) {
				const errorMessage = error instanceof Error ? error.message : 'An unknown error occurred.';
				toast.error(`Upload failed for ${file.name}`, { description: errorMessage });
			}
		});

		await Promise.all(uploadPromises);

		await fetchRecords(); // Refresh records after all uploads are attempted.

		isUploading = false;
		responseText = '';
		if (target) {
			target.value = ''; // Reset the file input
		}
	}

	async function handleDelete(id: string) {
		try {
			const response = await fetch(`/api/audio/${id}`, {
				method: 'DELETE',
				credentials: 'include'
			});
			if (!response.ok) {
				const result = await response.json();
				throw new Error(result.error || 'Unknown server error.');
			}
			toast.success('Recording deleted successfully.');
			await fetchRecords();
		} catch (error) {
			const errorMessage = error instanceof Error ? error.message : 'An unknown error occurred.';
			toast.error('Deletion error', { description: errorMessage });
		}
	}

	// --- UI HANDLERS ---
	async function handleTerminateJob(jobId: string) {
		const jobToTerminate = activeJobs.find((job) => job.id === jobId);
		if (!jobToTerminate) {
			toast.error('Job not found locally.');
			return;
		}

		try {
			const response = await fetch(`/api/transcribe/job/${jobId}`, {
				method: 'DELETE',
				credentials: 'include'
			});
			if (!response.ok) {
				const result = await response.json();
				throw new Error(result.error || 'Failed to terminate job.');
			}
			toast.success('Job terminated.');

			// Update UI reactively
			await fetchActiveJobs();

			// Also update the status for the specific audio record
			const newStatus = { ...transcriptionStatus };
			delete newStatus[jobToTerminate.audio_id];
			transcriptionStatus = newStatus;
		} catch (error) {
			const errorMessage = error instanceof Error ? error.message : 'An unknown error occurred.';
			toast.error('Termination error', { description: errorMessage });
		}
	}

	function openDetailDialog(record: AudioRecord) {
		selectedRecord = record;
		isDetailDialogOpen = true;
	}

	function openModelSelectDialog(record: AudioRecord) {
		recordToTranscribe = record;
		isModelSelectOpen = true;
	}

	function handleStartTranscription() {
		if (!recordToTranscribe) return;
		transcribe(recordToTranscribe.id, selectedModel);
		isModelSelectOpen = false;
	}

	function openNewTemplateDialog() {
		selectedTemplate = null; // Ensure we are creating a new one
		isTemplateDialogOpen = true;
	}

	function openEditTemplateDialog(template: SummaryTemplate) {
		selectedTemplate = template;
		isTemplateDialogOpen = true;
	}

	function openSummarizeDialog(record: AudioRecord) {
		recordToSummarize = record;
		selectedTemplateId = '';
		selectedSummaryModel = 'gpt-3.5-turbo'; // Reset to default
		isSummarizeDialogOpen = true;
	}

	function handleStartSummarization() {
		summarize();
		isSummarizeDialogOpen = false;
	}
</script>

<div class="min-h-screen w-full bg-gray-800 p-8 text-gray-200">
	<div class="mx-auto max-w-4xl">
		<header class="flex items-center justify-between border-b border-gray-700 pb-4">
			<h1 class="text-2xl font-bold">Scriberr</h1>
			<div class="flex items-center gap-2">
				{#if activeTab === 'templates'}
					<Button
						class="bg-neon-100 hover:bg-neon-200 text-gray-800"
						onclick={openNewTemplateDialog}
					>
						<FilePlus class="mr-2 h-4 w-4" />
						New Template
					</Button>
				{:else}
					<input
						type="file"
						bind:this={fileInput}
						onchange={handleFileSelect}
						style="display: none;"
						accept="audio/*"
						disabled={isUploading}
						multiple
					/>
					<Popover.Root>
						<Popover.Trigger
							class={cn(buttonVariants(), 'bg-neon-100 hover:bg-neon-200 text-gray-800')}
							disabled={isUploading}
						>
							{#if isUploading}
								<LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
								<span>Processing...</span>
							{:else}
								<span>New Recording</span>
							{/if}
						</Popover.Trigger>
						<Popover.Content class="w-48 border-gray-600 bg-gray-800 p-2 text-gray-200">
							<div class="grid gap-2">
								<Button
									variant="ghost"
									class="w-full justify-start gap-2 px-2 outline-none hover:bg-gray-700 hover:text-gray-100"
									onclick={() => fileInput.click()}
									disabled={isUploading}
								>
									<Upload class="h-4 w-4" />
									Upload Files
								</Button>
								<Button
									variant="ghost"
									class="w-full justify-start gap-2 px-2 hover:bg-gray-700 hover:text-gray-100"
									onclick={() => (isRecorderOpen = true)}
									disabled={isUploading}
								>
									<Mic class="h-4 w-4" />
									Record Audio
								</Button>
								<Button
									variant="ghost"
									class="w-full justify-start gap-2 px-2 hover:bg-gray-700 hover:text-gray-100"
									onclick={() => (isYouTubeDialogOpen = true)}
									disabled={isUploading}
								>
									<Youtube class="h-4 w-4" />
									YouTube
								</Button>
							</div>
						</Popover.Content>
					</Popover.Root>
				{/if}

				<Button
					variant="ghost"
					size="icon"
					class="text-gray-400 hover:bg-gray-700 hover:text-gray-100"
					onclick={handleLogout}
					title="Log Out"
				>
					<LogOut class="h-5 w-5" />
				</Button>
			</div>
		</header>

		{#if responseText}
			<div class="mt-4 rounded-md bg-gray-700 p-3 text-center text-sm">
				<p>{responseText}</p>
			</div>
		{/if}

		<main class="mt-8">
			<Tabs.Root bind:value={activeTab} class="w-full">
				<Tabs.List class="grid w-full grid-cols-3 bg-gray-900 text-gray-100">
					<Tabs.Trigger
						value="audio"
						class="text-gray-100 data-[state=active]:bg-gray-800 data-[state=active]:text-blue-400"
						>Audio</Tabs.Trigger
					>
					<Tabs.Trigger
						value="jobs"
						class="text-gray-100 data-[state=active]:bg-gray-800 data-[state=active]:text-blue-400"
					>
						Active Jobs
						{#if activeJobs.length > 0}
							<span
								class="ml-2 inline-flex h-6 w-6 items-center justify-center rounded-full bg-yellow-400 text-xs font-bold text-gray-800"
							>
								{activeJobs.length}
							</span>
						{/if}
					</Tabs.Trigger>
					<Tabs.Trigger
						value="templates"
						class="text-gray-100 data-[state=active]:bg-gray-800 data-[state=active]:text-blue-400"
						>Templates</Tabs.Trigger
					>
				</Tabs.List>
				<Tabs.Content value="audio">
					<AudioTab
						{records}
						{transcriptionStatus}
						{summarizationStatus}
						{isUploading}
						onOpenDetail={openDetailDialog}
						onOpenModelSelect={openModelSelectDialog}
						onOpenSummarizeDialog={openSummarizeDialog}
						onDeleteRecord={handleDelete}
					/>
				</Tabs.Content>
				<Tabs.Content value="jobs">
					<JobsTab {activeJobs} onTerminateJob={handleTerminateJob} />
				</Tabs.Content>
				<Tabs.Content value="templates">
					<TemplateTab
						{templates}
						onEditTemplate={openEditTemplateDialog}
						onDeleteTemplate={handleDeleteTemplate}
						onUpdate={fetchTemplates}
					/>
				</Tabs.Content>
			</Tabs.Root>
		</main>
	</div>
</div>

<DetailsDialog bind:open={isDetailDialogOpen} record={selectedRecord} />

<ModelSelectDialog
	bind:open={isModelSelectOpen}
	bind:selectedModel
	{recordToTranscribe}
	{modelSizes}
	onStartTranscription={handleStartTranscription}
/>

<SummarizeDialog
	bind:open={isSummarizeDialogOpen}
	{recordToSummarize}
	bind:selectedTemplateId
	bind:selectedModel={selectedSummaryModel}
	{templates}
	modelOptions={summaryModelOptions}
	onStartSummarization={handleStartSummarization}
/>

<RecorderDialog bind:open={isRecorderOpen} onSave={handleSaveRecording} />

<YouTubeDialog bind:open={isYouTubeDialogOpen} onDownload={handleYouTubeDownload} />

<TemplateDialog
	bind:open={isTemplateDialogOpen}
	template={selectedTemplate}
	onUpdate={fetchTemplates}
/>
