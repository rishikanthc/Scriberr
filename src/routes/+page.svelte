<script lang="ts">
	import RecordPanel from './components/RecordPanel.svelte';
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
		Settings
	} from 'lucide-svelte';
	import { Preferences } from '@capacitor/preferences';
	import { sineInOut } from 'svelte/easing';
	import { quintOut } from 'svelte/easing';
	import { clearServerConfig } from '$lib/stores/config';
	import UploadPanel from './components/UploadPanel.svelte';
	import StatusPanel from './components/StatusPanel.svelte';
	import TemplatesList from './components/TemplatesList.svelte';

	// Active panel state management
	let activePanel = $state('files'); // Default panel
	let showAudioRec: boolean = $state(false);
	let selectedFileId;

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

	function handleTemplateMenu() {
		activePanel = 'templates';
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
			class="mx-auto mb-6 mt-4 flex h-[40px] w-fit items-center justify-center gap-8 rounded-md border border-neutral-700/30 bg-neutral-600/35 p-1 shadow-sm backdrop-blur-md"
		>
			<div class="mb-[-5px] ml-2 font-['Lombok'] text-gray-100">SCRIBERR</div>
			<div class="flex items-center justify-evenly gap-2">
				<button
					class="flex items-center justify-center rounded-md p-2 transition-all duration-200 {showAudioRec ===
					true
						? 'bg-gray-900 text-white'
						: 'text-neutral-400'} backdrop-blur-lg hover:bg-neutral-900"
					on:click={handleRecMenu}
				>
					<Mic size={18} />
				</button>
				<button
					class="flex items-center justify-center rounded-md p-2 transition-all duration-200 {activePanel ===
					'files'
						? 'bg-gray-900 text-white'
						: 'text-neutral-400'} backdrop-blur-lg hover:bg-gray-900"
					on:click={() => {
						showAudioRec = false;
						activePanel = 'files';
					}}
				>
					<AudioLines size={18} />
				</button>
				<button
					class="flex h-[35px] w-[35px] items-center justify-center rounded-md p-2 transition-all duration-200 {activePanel ===
					'status'
						? 'bg-gray-900 text-white'
						: 'text-neutral-400'} backdrop-blur-lg hover:bg-gray-900"
					on:click={() => {
						showAudioRec = false;
						activePanel = 'status';
					}}
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="100%"
						height="100%"
						viewBox="0 0 24 24"
						fill="none"
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
						<line
							x1="8"
							y1="12"
							x2="8"
							y2="12"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
						>
							<animate attributeName="y1" values="14;10;14" dur="1s" repeatCount="indefinite" />
							<animate attributeName="y2" values="10;14;10" dur="1s" repeatCount="indefinite" />
						</line>
						<line
							x1="12"
							y1="12"
							x2="12"
							y2="12"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
						>
							<animate attributeName="y1" values="10;14;10" dur="1s" repeatCount="indefinite" />
							<animate attributeName="y2" values="14;10;14" dur="1s" repeatCount="indefinite" />
						</line>
						<line
							x1="16"
							y1="12"
							x2="16"
							y2="12"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
						>
							<animate attributeName="y1" values="12;8;12" dur="1s" repeatCount="indefinite" />
							<animate attributeName="y2" values="16;12;16" dur="1s" repeatCount="indefinite" />
						</line>
					</svg>
				</button>

				<button
					class="flex items-center justify-center rounded-md p-2 transition-all duration-200 {activePanel ===
					'templates'
						? 'bg-gray-900 text-white'
						: 'text-neutral-400'} backdrop-blur-lg hover:bg-neutral-900"
					on:click={() => {
						showAudioRec = false;
						activePanel = 'templates';
					}}
				>
					<NotepadTextDashed size={18} />
				</button>

				<div class="flex items-center justify-center">
					<DropdownMenu.Root class="text-white">
						<DropdownMenu.Trigger asChild>
							<User size={18} class="text-neutral-400" />
						</DropdownMenu.Trigger>
						<DropdownMenu.Content class="w-48">
							<DropdownMenu.Label>My Account</DropdownMenu.Label>
							<DropdownMenu.Separator />
							<DropdownMenu.Item on:click={handleLogout} class="cursor-pointer">
								<LogOut class="mr-2 h-4 w-4 text-gray-700" />
								<span>Log out</span>
							</DropdownMenu.Item>
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				</div>
			</div>
		</div>

		<!-- Content Area -->
		<div class="mx-auto p-2 lg:w-[500px] 2xl:mt-8">
			{#if activePanel === 'files'}
				<FilesSidebar bind:showAudioRec />
			{:else if activePanel === 'status'}
				<StatusPanel />
			{:else if activePanel === 'templates'}
				<TemplatesList />
			{/if}
		</div>
	</div>
</div>