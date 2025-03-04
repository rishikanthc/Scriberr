<script lang="ts">
	import AudioRec from './AudioRec.svelte';
	import { isRecording } from '$lib/stores/recstate';

	import { setContext } from 'svelte';

	let state = $state({ recording: false });
	setContext('recstate', state);

	function handleRecordingComplete({ url, duration }) {
		console.log('Recording completed:', { url, duration });
		// Handle the recording as needed
	}
</script>

<AudioRec />
