<script lang="ts">
	import { page } from '$app/stores';
	import { dev } from '$app/environment';

	// Log the error details
	console.error('Error details:', {
		status: $page.status,
		error: $page.error
	});
</script>

<div class="p-4">
	<h1 class="text-2xl font-bold text-red-600">Error Occurred</h1>
	<pre class="mt-4 rounded bg-gray-100 p-4">
        Status: {$page.status}
        Message: {($page.error as any)?.message || 'Unknown error'}
        {#if dev}
			Stack: {($page.error as any)?.stack || 'No stack trace'}
		{/if}
    </pre>
</div>
