<script lang="ts">
	import {
		ContextMenu,
		ContextMenuContent,
		ContextMenuItem,
		ContextMenuTrigger
	} from '$lib/components/ui/context-menu/index.js';
	import { CheckCircle2, LoaderCircle, Download, MessageCircle } from 'lucide-svelte';

	// --- TYPES ---
	type AudioRecord = {
		id: string;
		title: string;
		created_at: string;
		transcript: string;
		downloading?: boolean;
	};

	type JobStatus = 'processing' | 'completed' | 'failed';

	type Props = {
		records: AudioRecord[];
		transcriptionStatus: Record<string, JobStatus>;
		summarizationStatus: Record<string, JobStatus>;
		isUploading: boolean;
		onOpenDetail: (record: AudioRecord) => void;
		onOpenModelSelect: (record: AudioRecord) => void;
		onOpenSummarizeDialog: (record: AudioRecord) => void;
		onOpenChatDialog: (record: AudioRecord) => void;
		onDeleteRecord: (id: string) => void;
	};

	let {
		records,
		transcriptionStatus,
		summarizationStatus,
		isUploading,
		onOpenDetail,
		onOpenModelSelect,
		onOpenSummarizeDialog,
		onOpenChatDialog,
		onDeleteRecord
	}: Props = $props();

	function formatDate(dateString: string) {
		return new Date(dateString).toLocaleString();
	}
</script>

{#if records.length === 0 && !isUploading}
	<div class="py-10 text-center text-gray-500">
		<p>No recordings yet. Click "New Recording" to upload your first file.</p>
	</div>
{:else}
	<div class="mt-4 space-y-3">
		{#each records as record (record.id)}
			<ContextMenu>
				<ContextMenuTrigger
					class={`flex items-center justify-between gap-4 rounded-lg bg-gray-700/50 p-4 transition-colors hover:bg-gray-700 ${
						record.downloading 
							? 'opacity-50 cursor-not-allowed' 
							: 'cursor-pointer'
					}`}
					onclick={() => !record.downloading && onOpenDetail(record)}
				>
					<div class="flex min-w-0 flex-1 items-center gap-4">
						{#if record.downloading}
							<Download class="h-5 w-5 flex-shrink-0 animate-pulse text-red-400" />
						{:else if transcriptionStatus[record.id] === 'processing'}
							<LoaderCircle class="h-5 w-5 flex-shrink-0 animate-spin text-yellow-400" />
						{:else if summarizationStatus[record.id] === 'processing'}
							<LoaderCircle class="h-5 w-5 flex-shrink-0 animate-spin text-blue-400" />
						{:else if transcriptionStatus[record.id] === 'completed'}
							<CheckCircle2 class="h-5 w-5 flex-shrink-0 text-green-400" />
						{:else}
							<div class="h-5 w-5 flex-shrink-0"></div>
						{/if}
						<span class="truncate font-medium" title={record.title}>
							{record.title}
							{#if record.downloading}
								<span class="ml-2 text-sm text-red-400">(Downloading...)</span>
							{/if}
						</span>
					</div>
					<span class="flex-shrink-0 text-sm text-gray-400">
						{formatDate(record.created_at)}
					</span>
				</ContextMenuTrigger>
				<ContextMenuContent class="border-gray-700 bg-gray-800 shadow-lg text-gray-100">
					<ContextMenuItem
						onclick={(e) => {
							e.stopPropagation();
							onOpenModelSelect(record);
						}}
						class="data-[highlighted]:bg-gray-700 data-[highlighted]:text-gray-50"
						disabled={record.downloading || transcriptionStatus[record.id] === 'processing' ||
							summarizationStatus[record.id] === 'processing'}
					>
						Transcribe...
					</ContextMenuItem>
					<ContextMenuItem
						onclick={(e) => {
							e.stopPropagation();
							onOpenSummarizeDialog(record);
						}}
						class="data-[highlighted]:bg-gray-700 data-[highlighted]:text-gray-50"
						disabled={record.downloading || transcriptionStatus[record.id] !== 'completed' ||
							summarizationStatus[record.id] === 'processing'}
					>
						Summarize...
					</ContextMenuItem>
					<ContextMenuItem
						onclick={(e) => {
							e.stopPropagation();
							onOpenChatDialog(record);
						}}
						class="data-[highlighted]:bg-gray-700 data-[highlighted]:text-gray-50"
						disabled={record.downloading || transcriptionStatus[record.id] !== 'completed'}
					>
						<MessageCircle class="h-4 w-4 mr-2" />
						Chat with Transcript...
					</ContextMenuItem>
					<ContextMenuItem
						onclick={(e) => {
							e.stopPropagation();
							onDeleteRecord(record.id);
						}}
						class="text-magenta-400 data-[highlighted]:bg-gray-700 data-[highlighted]:text-red-500"
						disabled={record.downloading}
					>
						Delete
					</ContextMenuItem>
				</ContextMenuContent>
			</ContextMenu>
		{/each}
	</div>
{/if}
