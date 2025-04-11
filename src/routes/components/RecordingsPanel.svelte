<script lang="ts">
</script>

<div
	class="relative mx-auto flex h-[80%] w-[40%] rounded-3xl border border-white/20 bg-gray-500/10 shadow-xl backdrop-blur-lg"
>
	<!-- Sidebar -->
	<div
		class="flex w-16 flex-col items-center rounded-l-3xl border-r border-white/10 bg-gray-500/40 py-5 backdrop-blur-md"
	>
		<button
			class="mb-4 flex h-10 w-10 items-center justify-center rounded-full bg-white/20 text-white hover:bg-white/40"
		>
			🏠
		</button>
		<button
			class="mb-4 flex h-10 w-10 items-center justify-center rounded-full bg-white/20 text-white hover:bg-white/40"
		>
			📷
		</button>
		<button
			class="mb-4 flex h-10 w-10 items-center justify-center rounded-full bg-white/20 text-white hover:bg-white/40"
		>
			⚙️
		</button>
	</div>

	<!-- Main Content -->
	<div class="flex flex-grow flex-col p-8">
		<!-- Search Bar -->
		<div
			class="mb-6 flex items-center rounded-lg border border-white/20 bg-white/10 px-4 py-2 shadow-md backdrop-blur-md"
		>
			<input
				type="text"
				placeholder="Search..."
				class="flex-grow bg-transparent text-white placeholder-gray-400 outline-none"
			/>
			<button class="text-white"> 🔍 </button>
		</div>

		<!-- Content Cards -->
		<div class="grid grid-cols-1 gap-4 overflow-y-auto">
			<!-- Card -->
			<div
				class="flex items-center rounded-lg border border-white/20 bg-white/10 p-4 shadow-md backdrop-blur-lg"
			>
				<img
					src="https://via.placeholder.com/80"
					alt="Thumbnail"
					class="mr-4 h-16 w-16 rounded-lg"
				/>
				<div>
					<h3 class="text-lg font-medium text-white">Sample Video Title</h3>
					<p class="text-sm text-gray-400">Channel Name · 1 hour ago</p>
				</div>
			</div>

			<!-- Duplicate Card Example -->
			<div
				class="flex items-center rounded-lg border border-white/20 bg-white/10 p-4 shadow-md backdrop-blur-lg"
			>
				<img
					src="https://via.placeholder.com/80"
					alt="Thumbnail"
					class="mr-4 h-16 w-16 rounded-lg"
				/>
				<div>
					<h3 class="text-lg font-medium text-white">Another Video Title</h3>
					<p class="text-sm text-gray-400">Another Channel · 2 hours ago</p>
				</div>
			</div>
		</div>
	</div>
</div>
