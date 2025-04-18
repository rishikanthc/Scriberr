<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as AlertDialog from '$lib/components/ui/alert-dialog';
	import * as ContextMenu from '$lib/components/ui/context-menu/index.js';
	import { apiFetch } from '$lib/api';
	import * as Dialog from '$lib/components/ui/dialog';
	import { get } from 'svelte/store';
	import { tick } from 'svelte';
	import * as Tabs from '$lib/components/ui/tabs/index.js';
	import { templates } from '$lib/stores/templateStore';
	import * as Command from '$lib/components/ui/command/index.js';
	import * as Popover from '$lib/components/ui/popover/index.js';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { onMount, onDestroy } from 'svelte';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import { ChevronsUpDown, TextQuote, Check, Mic2, Settings, BrainCircuit, Star } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import { audioFiles } from '$lib/stores/audioFiles';
	import { speakerLabels } from '$lib/stores/speakerLabels';
	import { getSpeakerColor } from '$lib/speakerColors';
	import { processThinkingSections, formatTime } from '$lib/utils';
	import AudioPlayer from './AudioPlayer.svelte';
	import SpeakerLabels from './SpeakerLabels.svelte';
	import ThinkingDisplay from './ThinkingDisplay.svelte';
	import { serverUrl } from '$lib/stores/config';
	import type { TranscriptSegment } from '$lib/types';

	interface FileProps {
		id: number;
		fileName: string;
		title?: string;
		uploadedAt: string;
		peaks: number[];
		transcript?: TranscriptSegment[];
		transcriptionStatus: string;
		diarization?: boolean;
		summary?: string;
		originalFileName?: string;
	}

	let { file, isOpen = $bindable() } = $props();
	let audioUrl = $state('');
	let summary = $state('');
	let isSummarizing = $state(false);
	let isGeneratingTitle = $state(false);
	let selectedTemplateId = $state(null);
	let selectedTemplate = $state('Select a template...');
	let isDialogOpen = $state(false);
	let titleDialogOpen = $state(false);
	let newTitle = $state('');
	let error = $state(null);
	let templateOpen = $state(false);
	let triggerRef = null;
	let showThinkingSections = $state(true);
	let summaryHasThinking = $derived(Boolean(summary && typeof summary === 'string' && summary.includes('<think>')));
	let fileSummaryHasThinking = $derived(Boolean(file?.summary && typeof file.summary === 'string' && file.summary.includes('<think>')));
	let hasThinkingSections = $derived(summaryHasThinking || fileSummaryHasThinking);

	function logError(error: any, context: string) {
		console.error(`${context}:`, error);
		return error.message || 'An unexpected error occurred';
	}

	function selectTemplate(templateId: string, templateTitle: string) {
		console.log("Template selected:", templateTitle, templateId);
		selectedTemplateId = templateId;
		selectedTemplate = templateTitle;
		templateOpen = false;
	}

	let refreshInterval: ReturnType<typeof setInterval>;

	onMount(async () => {
		if (window.Capacitor?.isNative) {
			audioUrl = get(serverUrl);
		}
		if (file?.id) {
			try {
				await speakerLabels.loadLabels(file.id);
			} catch (err) {
				console.error("Failed to load speaker labels:", err);
			}
			if (file.title) {
				newTitle = file.title;
			}
			refreshInterval = setInterval(async () => {
				if (file?.id && isOpen) {
					if (file.transcriptionStatus !== 'completed' && file.transcriptionStatus !== 'failed') {
						console.log(`Auto-refreshing file ${file.id} details while viewing`);
						await audioFiles.refresh();
					} else {
						if (Math.random() < 0.2) {
							await audioFiles.refresh();
						}
					}
				}
			}, 2000);
		}
	});

	onDestroy(() => {
		if (refreshInterval) {
			clearInterval(refreshInterval);
			console.log('Cleaned up file refresh interval');
		}
	});

	const currentLabels = $derived(get(speakerLabels)[file?.id] || {});

	$effect(() => {
		if (file) {
			summary = '';
			if (file.title) {
				newTitle = file.title;
			}
		}
	});

	function openTitleDialog() {
		titleDialogOpen = true;
		newTitle = file.title || '';
	}

	async function handleTitleUpdate() {
		if (!newTitle.trim()) {
			error = 'Title cannot be empty';
			return;
		}

		try {
			error = null;
			const response = await apiFetch(`/api/audio/${file.id}`, {
				method: 'PATCH',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					title: newTitle
				})
			});

			if (!response.ok) {
				throw new Error('Failed to update title');
			}

			titleDialogOpen = false;
			audioFiles.refresh();
			toast.success('Title updated successfully');
		} catch (error) {
			const errorMessage = logError(error, 'Title update failed');
			error = errorMessage;
			toast.error('Failed to rename file. Please try again.');
		}
	}

	async function deleteFile(fileId) {
		let temp = file.title;
		isOpen = false;
		await audioFiles.deleteFile(fileId);
		await audioFiles.refresh();
		toast.success(`${temp} deleted`);
	}

	function handleSpeakerLabelsClose() {
		isDialogOpen = false;
		error = null;
	}

    async function generateTitle() {
      if (!file?.transcript || file.transcript.length === 0) {
        toast.error('Cannot generate title: Transcript is not available');
        return;
      }

      try {
        isGeneratingTitle = true;
        const transcriptText = file.transcript.map(segment => segment.text).join(' ');
        const system_prompt = "You are a concise title generator. Always provide titles that are 3-8 words long. Never explain your reasoning. Never use quotes. Never provide multiple options.";
        const titlePrompt = "Create a short, descriptive title that captures the main subject or theme of this transcript.";
        
        const response = await apiFetch('/api/summarize', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            fileId: file.id,
            prompt: titlePrompt,
            system_prompt: system_prompt,
            transcript: transcriptText,
            processThinking: true,
            maxLength: 50
          })
        });

        if (!response.ok) {
          throw new Error('Failed to generate title');
        }

        const data = await response.json();
        let generatedTitle = data.summary.trim();

        // Remove quotes if present
        if (generatedTitle.startsWith('"') && generatedTitle.endsWith('"')) {
          generatedTitle = generatedTitle.substring(1, generatedTitle.length - 1);
        }

        await apiFetch(`/api/audio/${file.id}`, {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            title: generatedTitle
          })
        });

        await audioFiles.refresh();
        toast.success('Generated new title');
        newTitle = generatedTitle;
      } catch (error) {
        const errorMessage = logError(error, 'Title generation failed');
        toast.error(errorMessage);
      } finally {
        isGeneratingTitle = false;
      }
    }

	function doSummary() {
		console.log("doSummary called");
		if (!file?.transcript || !selectedTemplateId) {
			toast.error('Please select a template and ensure transcript is available');
			return;
		}

		isSummarizing = true;
		console.log("Setting isSummarizing to true");

		try {
			const template = $templates.find((t) => t.id === selectedTemplateId);
			if (!template) {
				throw new Error('Template not found');
			}

			const transcriptText = file.transcript.map((segment) => segment.text).join(' ');

			apiFetch('/api/summarize', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					fileId: file.id,
					prompt: template.prompt,
					transcript: transcriptText,
					processThinking: false
				})
			})
			.then(response => {
				console.log("API response received", response.status);
				if (!response.ok) {
					throw new Error('Failed to generate summary');
				}
				return response.json();
			})
			.then(data => {
				console.log("Data parsed successfully");
				summary = data.summary;
				return audioFiles.refresh();
			})
			.then(() => {
				console.log("Summary generated and files refreshed");
				toast.success('Summary generated successfully');
			})
			.catch(error => {
				console.error("Promise chain error:", error);
				const errorMessage = logError(error, 'Summary generation failed');
				toast.error(errorMessage);
			})
			.finally(() => {
				console.log("Setting isSummarizing to false");
				isSummarizing = false;
			});
		} catch (error) {
			console.error("Initial setup error:", error);
			const errorMessage = logError(error, 'Summary generation failed');
			toast.error(errorMessage);
			isSummarizing = false;
		}
	}
</script>

{#if file}
	<AudioPlayer
		audioSrc={`${audioUrl}/api/audio/${file.id}`}
		originalAudioSrc={file.originalFileName ? `${audioUrl}/api/audio/${file.id}?original=true` : null}
		peaks={file.peaks}
	/>
	{#if file.transcriptionStatus === 'completed' && file.transcript}
		<div class="mt-6 h-[calc(100%-120px)] flex flex-col">
			<Tabs.Root value="transcript">
				<div class="mb-2 flex items-center justify-between">
					<Tabs.List class="mb-2 bg-neutral-800/60">
						<Tabs.Trigger
							value="transcript"
							class="data-[state=active]:bg-neutral-600/90 data-[state=active]:text-gray-200"
							>Transcript</Tabs.Trigger>
						<Tabs.Trigger
							value="summary"
							class="data-[state=active]:bg-neutral-600/90 data-[state=active]:text-gray-200"
							>Summary</Tabs.Trigger>
					</Tabs.List>
					<div class="flex justify-end">
						<DropdownMenu.Root>
							<DropdownMenu.Trigger asChild>
								<Button variant="secondary" size="sm">
									<Settings size={16} class="mr-1 text-blue-500" />
									Options
								</Button>
							</DropdownMenu.Trigger>
							<DropdownMenu.Content>
								<DropdownMenu.Item class="data-[highlighted]:bg-gray-100" onclick={openTitleDialog}
									>Rename</DropdownMenu.Item>
								<DropdownMenu.Item
									class="data-[highlighted]:bg-gray-100"
									onclick={generateTitle}
								>
									{isGeneratingTitle ? 'Generating Title...' : 'Auto-Generate Title'}
								</DropdownMenu.Item>
								<DropdownMenu.Item
									class="data-[highlighted]:bg-gray-100"
									onclick={() => {
										deleteFile(file.id);
									}}>Delete</DropdownMenu.Item>
								{#if file.diarization}
									<DropdownMenu.Item
										class="data-[highlighted]:bg-gray-100"
										onclick={() => (isDialogOpen = true)}
									>
										Label Speakers
									</DropdownMenu.Item>
								{/if}
							</DropdownMenu.Content>
						</DropdownMenu.Root>
					</div>
				</div>
				<Tabs.Content value="transcript" class="h-full">
					<ScrollArea class="h-[40svh] rounded-lg p-4 text-base min-[390px]:h-[45svh] lg:h-[50svh] xl:h-[55svh] 2xl:h-[60svh] max-h-[calc(100vh-320px)]">
						<div class="flex flex-col gap-6 max-w-[1000px] mx-auto pb-10">
							{#each file.transcript as segment}
								<div class="flex flex-col gap-1">
									<div class="flex items-center gap-2 text-xs font-medium text-gray-400">
										{#if file.diarization && segment.speaker && segment.speaker.trim() !== ""}
											<div
												class="flex items-center gap-1 text-sm font-bold"
												style="color: {getSpeakerColor(segment.speaker)}"
											>
												<Mic2 size={16} />
												{#if currentLabels[segment.speaker]}
													{currentLabels[segment.speaker].charAt(0).toUpperCase() +
													currentLabels[segment.speaker].slice(1)}
												{:else if segment.speaker.startsWith("SPEAKER_")}
													Speaker {segment.speaker.split("_")[1]}
												{:else}
													{segment.speaker.charAt(0).toUpperCase() + segment.speaker.slice(1)}
												{/if}
											</div>
										{/if}
										<div>{formatTime(segment.start)}</div>
									</div>
									<div class="text-lg leading-relaxed text-gray-100">
										{segment.text}
									</div>
								</div>
							{/each}
						</div>
					</ScrollArea>
				</Tabs.Content>
				<Tabs.Content value="summary" class="h-full">
					<div class="flex items-center gap-2">
						<div class="space-y-4">
							<Popover.Root bind:open={templateOpen}>
								<Popover.Trigger bind:ref={triggerRef}>
									{#snippet child({ props })}
										<Button
											variant="outline"
											class="w-[300px] justify-between border-gray-600 bg-neutral-700/55 text-gray-300 hover:bg-neutral-600/40 hover:text-gray-200"
											{...props}
											role="combobox"
											aria-expanded={templateOpen}
										>
											{selectedTemplate}
											<ChevronsUpDown class="opacity-50" />
										</Button>
									{/snippet}
								</Popover.Trigger>
								<Popover.Content class="w-full border-gray-600 bg-gray-700 p-0">
									<Command.Root class="border-gray-600 bg-neutral-700">
										<Command.Input placeholder="Search templates..." class="h-9 text-gray-100" />
										<Command.List>
											<Command.Empty>No templates found.</Command.Empty>
											{#each $templates as template}
												<div
													class="flex items-center gap-2 px-2 py-1.5 text-sm text-gray-200 hover:bg-neutral-600 hover:text-gray-50 cursor-pointer"
													onclick={() => selectTemplate(template.id, template.title)}
												>
													<Check class={selectedTemplateId !== template.id ? "text-transparent" : ""} />
													{template.title}
												</div>
											{/each}
										</Command.List>
									</Command.Root>
								</Popover.Content>
							</Popover.Root>
						</div>
						<div class="flex gap-2">
							<Button
								variant="ghost"
								size="icon"
								class="bg-neutral-700 p-1 disabled:bg-neutral-500"
								onclick={() => doSummary()}
								disabled={isSummarizing || !selectedTemplateId}
							>
								<Star size="20" class="text-gray-300" />
							</Button>
							{#if summaryHasThinking || fileSummaryHasThinking}
								<Button
									variant="ghost"
									size="icon"
									class="bg-neutral-700 p-1"
									onclick={() => showThinkingSections = !showThinkingSections}
									title={showThinkingSections ? "Hide AI's thinking process" : "Show AI's thinking process"}
								>
									<BrainCircuit size="20" class={showThinkingSections ? "text-amber-400" : "text-gray-500"} />
								</Button>
							{/if}
						</div>
					</div>
					<ScrollArea class="h-[40svh] rounded-lg p-4 text-base min-[390px]:h-[45svh] lg:h-[50svh] xl:h-[55svh] 2xl:h-[60svh] max-h-[calc(100vh-380px)]">
						<div class="max-w-[1000px] mx-auto pb-10">
							{#if file.summary}
								<div class="mt-6">
									<ThinkingDisplay summary={file.summary} initialShowThinking={showThinkingSections} />
								</div>
							{:else if isSummarizing}
								<div class="flex h-full items-center justify-center">
									<div class="text-gray-400">Generating summary...</div>
								</div>
							{:else if summary}
								<div>
									<ThinkingDisplay summary={summary} initialShowThinking={showThinkingSections} />
								</div>
							{:else}
								<div class="flex h-full items-center justify-center text-gray-400">
									Select a template and click summarize to generate a summary
								</div>
							{/if}
						</div>
					</ScrollArea>
				</Tabs.Content>
			</Tabs.Root>
		</div>
	{/if}
	<Dialog.Root bind:open={titleDialogOpen}>
		<Dialog.Content class="w-[90svw] rounded-md p-2">
			<Dialog.Header>
				<Dialog.Title>Rename File</Dialog.Title>
				<Dialog.Description>Enter a new title for this file</Dialog.Description>
			</Dialog.Header>
			<div class="py-4">
				<Input
					bind:value={newTitle}
					placeholder="Enter new title"
					class={error ? 'border-red-500' : ''}
				/>
				{#if error}
					<p class="mt-1 text-xs text-red-500">{error}</p>
				{/if}
			</div>
			<Dialog.Footer>
				<div class="flex items-center justify-between">
					<Button variant="outline" onclick={() => (titleDialogOpen = false)}>Cancel</Button>
					<Button onclick={handleTitleUpdate}>Save</Button>
				</div>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>
	{#if file.diarization}
		<AlertDialog.Root bind:open={isDialogOpen}>
			<AlertDialog.Content class="w-[90svw] rounded-md p-2">
				<AlertDialog.Header>
					<AlertDialog.Title>Label Speakers</AlertDialog.Title>
					<AlertDialog.Description>
						Assign custom names to speakers in the transcript
					</AlertDialog.Description>
				</AlertDialog.Header>
				<SpeakerLabels
					fileId={file.id}
					transcript={file.transcript}
					onSave={handleSpeakerLabelsClose}
				/>
			</AlertDialog.Content>
		</AlertDialog.Root>
	{/if}
{/if}