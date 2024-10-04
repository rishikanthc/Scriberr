<script>
	export let size = 10;
	export let color = '#007bff';
	export let duration = 1.5;
	export let message;
</script>

<div class="flex items-center justify-start gap-2">
	<div class="dot" style="--size: {size}px; --color: {color}; --duration: {duration}s;"></div>
	<div class="text-2xl">{message}</div>
</div>

<style>
	.dot-container {
		display: inline-flex;
		gap: 5px;
		align-items: center;
		justify-content: center;
	}

	.dot {
		width: var(--size);
		height: var(--size);
		background-color: var(--color);
		border-radius: 50%;
		animation: pulse var(--duration) ease-in-out infinite;
	}

	@keyframes pulse {
		0%,
		100% {
			transform: scale(0.5);
			opacity: 0.5;
		}
		50% {
			transform: scale(1);
			opacity: 1;
		}
	}
</style>
