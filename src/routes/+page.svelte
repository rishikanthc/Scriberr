<script lang="ts">
	import { Separator } from 'bits-ui';
	import Processing from '$lib/components/Processing.svelte';
	import FilePanel from '$lib/components/FilePanel.svelte';
	export let data;
	$: records = data?.records;
	$: fileUrls = data?.fileUrls;
	$: templates = data?.templates;

	async function onUpload() {
		console.log('Files uploaded from page');
		const response = await fetch('/api/records');
		const jres = await response.json();
		records = jres.records;
		fileUrls = jres.fileUrls;
	}
	async function refreshTemplates() {
		const response = await fetch('/api/templates');
		templates = await response.json();
	}
</script>

<div>
	<div class="">
		<div class="flex items-center justify-between gap-2"></div>
		{#if records}
			<FilePanel
				bind:data={records}
				{fileUrls}
				{templates}
				on:onUpload={onUpload}
				on:finishedProcessing={onUpload}
				on:templatesModified={refreshTemplates}
				on:recordsModified={onUpload}
			/>
		{/if}
	</div>
</div>
