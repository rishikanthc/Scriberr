<script lang="ts">
	import { Input } from '$lib/components/ui/input';
	import { speakerLabels } from '$lib/stores/speakerLabels';
	import { Button } from '$lib/components/ui/button';
	import { audioFiles } from '$lib/stores/audioFiles';
	import { toast } from 'svelte-sonner';
	import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
	import { apiFetch, createEventSource } from '$lib/api';

	export let fileId: number;
	export let transcript: TranscriptSegment[];
	export let onSave: () => void;

	let labels: Record<string, string> = {};
	let uniqueSpeakers: string[] = [];

	$: {
		uniqueSpeakers = [...new Set(transcript.map((segment) => segment.speaker))].filter(Boolean);
		labels = $speakerLabels[fileId] || {};
	}

	async function updateTranscript(
		fileId: number,
		transcript: TranscriptSegment[],
		labels: Record<string, string>
	) {
		const response = await apiFetch(`/api/audio/${fileId}/transcript`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ transcript, speakerLabels: labels })
		});

		if (!response.ok) throw new Error('Failed to update transcript');
		return response.json();
	}

	async function handleSaveLabels() {
		try {
			const updatedTranscript = transcript.map((segment) => ({
				...segment,
				speaker: segment.speaker ? labels[segment.speaker] || segment.speaker : undefined
			}));

			await updateTranscript(fileId, updatedTranscript, labels);
			console.log('POS 1');
			speakerLabels.updateLabels((store) => ({ ...store, [fileId]: labels }));
			console.log('POS 2');
			audioFiles.updateFile(fileId, { transcript: updatedTranscript });

			toast.success('Speaker labels updated successfully');
			onSave();
		} catch (error) {
			console.error('Failed to save speaker labels:', error);
			toast.error('Failed to update speaker labels');
		}
	}
</script>

<div class="space-y-4 py-4">
	{#each uniqueSpeakers as speaker}
		<div class="flex items-center gap-2">
			<Input
				value={labels[speaker] || ''}
				placeholder={speaker}
				oninput={(e) => {
					labels[speaker] = e.currentTarget.value;
				}}
			/>
		</div>
	{/each}
</div>

<div class="mt-4 flex w-full items-center justify-center gap-2 lg:justify-end">
	<div><AlertDialog.Cancel>Cancel</AlertDialog.Cancel></div>
	<div><Button onclick={handleSaveLabels}>Save Changes</Button></div>
</div>
