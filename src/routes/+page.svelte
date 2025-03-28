<script lang="ts">
	import RecordPanel from './components/RecordPanel.svelte';
	import FilePanel from './components/FilePanel.svelte';
	import Transcribe from './components/Transcribe.svelte';
	import { initialize } from '@capacitor-community/safe-area';
	import { SafeArea } from '@capacitor-community/safe-area';
	import Files from './components/Files.svelte';
	import { onMount } from 'svelte';
	import AudioRec from './components/AudioRec.svelte';
	import { slide, fade } from 'svelte/transition';
	import { Button } from '$lib/components/ui/button';
	import FilesSidebar from './components/files-sidebar.svelte';
	import { Toaster } from '$lib/components/ui/sonner/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import {
		NotepadTextDashed,
		AudioLines,
		User,
		Upload,
		LogOut,
		FileText,
		Mic,
		Settings,
		ChevronLeft
	} from 'lucide-svelte';
	import { Preferences } from '@capacitor/preferences';
	import { sineInOut } from 'svelte/easing';
	import { quintOut } from 'svelte/easing';
	import { clearServerConfig } from '$lib/stores/config';
	import { audioFiles } from '$lib/stores/audioFiles';
	import UploadPanel from './components/UploadPanel.svelte';
	import StatusPanel from './components/StatusPanel.svelte';
	import TemplatesList from './components/TemplatesList.svelte';

	// Active panel state management
	let activePanel = $state('files'); // Default panel
	let showAudioRec: boolean = $state(false);
	let showUpload: boolean = $state(false);
	let selectedFileId = $state<number | null>(null);

	const handleLogout = async () => {
		try {
			const isCapacitor = window.Capacitor && window.Capacitor.isNative;
			const response = await fetch('/api/auth/logout', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				}
			});
			if (!response.ok) {
				throw new Error('Logout failed');
			}
			if (isCapacitor) {
				await Preferences.clear();
				clearServerConfig();
			}
			window.location.href = '/login';
		} catch (error) {
			console.error('Logout failed:', error);
		}
	};

	function handleRecMenu() {
		activePanel = 'files';
		showAudioRec = !showAudioRec;
	}

	function handleFileMenu() {
		showAudioRec = false;
		activePanel = 'files';
	}

	function handleStatusMenu() {
		showAudioRec = false;
		activePanel = 'status';
	}

	function handleTemplateMenu() {
		showAudioRec = false;
		activePanel = 'templates';
	}

	function handleUploadClick() {
		showUpload = true;
	}

	onMount(() => {
		initialize();
		SafeArea.enable({
			config: {
				customColorsForSystemBars: true,
				statusBarColor: '#00000000', // transparent
				statusBarContent: 'light',
				navigationBarColor: '#00000000', // transparent
				navigationBarContent: 'light'
			}
		});
	});
</script>

<div
	class="fixed inset-0 w-screen overflow-hidden bg-[#1e1a17] bg-cover bg-center bg-no-repeat"
	style="background-image: url('/background.svg')"
>
	<Toaster />
	<div
		class="bg-white/2 h-screen pb-[var(--safe-area-inset-bottom)] pt-[var(--safe-area-inset-top)] shadow-xl backdrop-blur-lg"
	>
		<!-- Control Panel -->
		<div
			class="mx-auto mb-6 mt-4 flex h-[50px] w-fit items-center justify-center gap-6 rounded-md border border-neutral-700/30 bg-neutral-600/35 p-1 shadow-sm backdrop-blur-md"
		>
			<div class="mb-[-5px] ml-2 font-['Lombok'] text-gray-100">SCRIBERR</div>
			<div class="flex items-center justify-evenly gap-4">
				<button
					class="flex flex-col items-center justify-center rounded-md p-2 transition-all duration-200 {showAudioRec ? 'bg-gray-900 text-white' : 'text-neutral-400'} backdrop-blur-lg hover:bg-neutral-900"
					onclick={handleRecMenu}
					aria-label="Record Audio"
				>
					<Mic size={16} />
					<span class="text-[10px] mt-1">Record</span>
				</button>
				
				<button
					class="flex flex-col items-center justify-center rounded-md p-2 transition-all duration-200 {activePanel === 'files' ? 'bg-gray-900 text-white' : 'text-neutral-400'} backdrop-blur-lg hover:bg-neutral-900"
					onclick={handleFileMenu}
					aria-label="Files"
				>
					<AudioLines size={16} />
					<span class="text-[10px] mt-1">Files</span>
				</button>
				
				<button
					class="flex flex-col items-center justify-center rounded-md p-2 transition-all duration-200 {showUpload ? 'bg-gray-900 text-white' : 'text-neutral-400'} backdrop-blur-lg hover:bg-neutral-900"
					onclick={handleUploadClick}
					aria-label="Upload Files"
				>
					<Upload size={16} />
					<span class="text-[10px] mt-1">Upload</span>
				</button>
				
				<button
					class="flex flex-col items-center justify-center rounded-md p-2 transition-all duration-200 {activePanel === 'status' ? 'bg-gray-900 text-white' : 'text-neutral-400'} backdrop-blur-lg hover:bg-neutral-900"
					onclick={handleStatusMenu}
					aria-label="Processing Status"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="16"
						height="16"
						viewBox="0 0 24 24"
						fill="none"
						class="mb-[2px]"
					>
						<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" opacity="0.2" />
						<path
							d="M12 2A10 10 0 0 1 22 12"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
						>
							<animateTransform
								attributeName="transform"
								type="rotate"
								from="0 12 12"
								to="360 12 12"
								dur="1s"
								repeatCount="indefinite"
							/>
						</path>
					</svg>
					<span class="text-[10px] mt-1">Status</span>
				</button>

				<button
					class="flex flex-col items-center justify-center rounded-md p-2 transition-all duration-200 {activePanel === 'templates' ? 'bg-gray-900 text-white' : 'text-neutral-400'} backdrop-blur-lg hover:bg-neutral-900"
					onclick={handleTemplateMenu}
					aria-label="Templates"
				>
					<NotepadTextDashed size={16} />
					<span class="text-[10px] mt-1">Templates</span>
				</button>

				<div class="flex flex-col items-center justify-center ml-2">
					<DropdownMenu.Root class="text-white">
						<DropdownMenu.Trigger asChild>
							<button class="flex flex-col items-center text-neutral-400 hover:text-white" aria-label="User Menu">
								<User size={16} />
								<span class="text-[10px] mt-1">Account</span>
							</button>
						</DropdownMenu.Trigger>
						<DropdownMenu.Content class="w-48">
							<DropdownMenu.Label>My Account</DropdownMenu.Label>
							<DropdownMenu.Separator />
							<DropdownMenu.Item onclick={handleLogout} class="cursor-pointer">
								<LogOut class="mr-2 h-4 w-4 text-gray-700" />
								<span>Log out</span>
							</DropdownMenu.Item>
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				</div>
			</div>
		</div>

		<!-- Content Area -->
		<div class="mx-auto p-2 {activePanel === 'files' ? 'w-full max-w-[1600px] xl:max-w-[1800px] 2xl:max-w-[2000px]' : 'lg:w-[500px]'} 2xl:mt-8">
			{#if activePanel === 'files'}
				<div class="flex flex-col lg:flex-row lg:gap-6">
					<div class="lg:w-[350px] xl:w-[400px] {selectedFileId ? 'hidden lg:block' : ''}">
						<FilesSidebar bind:showAudioRec bind:showUpload bind:selectedFileId />
					</div>
					<div id="file-detail-container" class="flex-1 {!selectedFileId ? 'hidden lg:block' : ''}">
						{#if selectedFileId}
							<div class="rounded-xl border-none bg-neutral-400/15 p-4 shadow-lg backdrop-blur-xl h-full overflow-hidden">
								<!-- Mobile Back Button -->
								<div class="mb-4 flex items-center lg:hidden">
									<Button 
										variant="ghost" 
										class="text-gray-300"
										onclick={() => {selectedFileId = null}}
									>
										<ChevronLeft size={18} class="mr-1" />
										Back to Files
									</Button>
								</div>
								<!-- File Panel -->
								{#key selectedFileId}
									{#if $audioFiles.find(f => f.id === selectedFileId)}
										<FilePanel file={$audioFiles.find(f => f.id === selectedFileId)} isOpen={true} />
									{/if}
								{/key}
							</div>
						{:else}
							<div class="flex h-full items-center justify-center rounded-xl border-none bg-neutral-400/15 p-8 shadow-lg backdrop-blur-xl text-gray-300">
								<p>Select a recording from the list to view details</p>
							</div>
						{/if}
					</div>
				</div>
			{:else if activePanel === 'status'}
				<StatusPanel />
			{:else if activePanel === 'templates'}
				<TemplatesList />
			{/if}
		</div>
	</div>

	{#if showUpload}
		<UploadPanel bind:showUpload />
	{/if}
</div>