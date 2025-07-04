<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { ScrollArea } from '$lib/components/ui/scroll-area/index.js';
	import { Send, Plus, Trash2, MessageCircle } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';

	// --- TYPES ---
	type AudioRecord = {
		id: string;
		title: string;
		created_at: string;
		transcript: string;
	};

	type ChatSession = {
		id: string;
		audio_id: string;
		title: string;
		model: string;
		created_at: string;
		updated_at: string;
	};

	type ChatMessage = {
		id: string;
		session_id: string;
		role: 'user' | 'assistant' | 'system';
		content: string;
		created_at: string;
	};

	// --- PROPS ---
	let {
		open = $bindable(),
		record,
		modelOptions
	}: {
		open: boolean;
		record: AudioRecord | null;
		modelOptions: string[];
	} = $props();

	// --- STATE ---
	let sessions = $state<ChatSession[]>([]);
	let currentSession = $state<ChatSession | null>(null);
	let messages = $state<ChatMessage[]>([]);
	let newMessage = $state('');
	let isLoading = $state(false);
	let isCreatingSession = $state(false);
	let selectedModel = $state('');
	let newSessionTitle = $state('');

	// Group models by provider
	const openaiModels = $derived(modelOptions.filter(model => !model.startsWith('ollama:')));
	const ollamaModels = $derived(modelOptions.filter(model => model.startsWith('ollama:')));

	// --- EFFECTS ---
	$effect(() => {
		if (open && record) {
			fetchChatSessions();
			if (modelOptions.length > 0 && !selectedModel) {
				selectedModel = modelOptions[0];
			}
		}
	});

	$effect(() => {
		if (currentSession) {
			fetchChatMessages();
		}
	});

	// --- FUNCTIONS ---
	async function fetchChatSessions() {
		if (!record) return;

		try {
			const response = await fetch(`/api/chat/sessions?audio_id=${record.id}`, {
				credentials: 'include'
			});
			if (!response.ok) throw new Error('Failed to fetch chat sessions');
			sessions = await response.json();
		} catch (error) {
			console.error('Error fetching chat sessions:', error);
			toast.error('Failed to load chat sessions');
		}
	}

	async function fetchChatMessages() {
		if (!currentSession) return;

		try {
			const response = await fetch(`/api/chat/messages?session_id=${currentSession.id}`, {
				credentials: 'include'
			});
			if (!response.ok) throw new Error('Failed to fetch chat messages');
			messages = await response.json();
		} catch (error) {
			console.error('Error fetching chat messages:', error);
			toast.error('Failed to load chat messages');
		}
	}

	async function createNewSession() {
		if (!record || !selectedModel || !newSessionTitle.trim()) return;

		isCreatingSession = true;
		try {
			const response = await fetch('/api/chat/sessions', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					audio_id: record.id,
					title: newSessionTitle.trim(),
					model: selectedModel
				}),
				credentials: 'include'
			});

			if (!response.ok) {
				const error = await response.json();
				throw new Error(error.error || 'Failed to create chat session');
			}

			const session: ChatSession = await response.json();
			sessions = [session, ...sessions];
			currentSession = session;
			newSessionTitle = '';
			toast.success('Chat session created');
		} catch (error) {
			const message = error instanceof Error ? error.message : 'Failed to create chat session';
			toast.error('Error', { description: message });
		} finally {
			isCreatingSession = false;
		}
	}

	async function sendMessage() {
		if (!currentSession || !newMessage.trim() || isLoading) return;

		const userMessage = newMessage.trim();
		newMessage = '';
		isLoading = true;

		// Add user message to UI immediately
		const tempUserMessage: ChatMessage = {
			id: `temp-${Date.now()}`,
			session_id: currentSession.id,
			role: 'user',
			content: userMessage,
			created_at: new Date().toISOString()
		};
		messages = [...messages, tempUserMessage];

		try {
			const response = await fetch('/api/chat/messages', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					session_id: currentSession.id,
					message: userMessage
				}),
				credentials: 'include'
			});

			if (!response.ok) {
				const error = await response.json();
				throw new Error(error.error || 'Failed to send message');
			}

			const result = await response.json();
			
			// Add assistant response
			const assistantMessage: ChatMessage = {
				id: result.message_id,
				session_id: currentSession.id,
				role: 'assistant',
				content: result.content,
				created_at: new Date().toISOString()
			};
			messages = [...messages, assistantMessage];

			// Update session timestamp
			currentSession.updated_at = new Date().toISOString();
		} catch (error) {
			const message = error instanceof Error ? error.message : 'Failed to send message';
			toast.error('Error', { description: message });
			// Remove the temporary user message on error
			messages = messages.filter(m => m.id !== tempUserMessage.id);
		} finally {
			isLoading = false;
		}
	}

	async function deleteSession(sessionId: string) {
		try {
			const response = await fetch(`/api/chat/sessions/${sessionId}`, {
				method: 'DELETE',
				credentials: 'include'
			});

			if (!response.ok) {
				const error = await response.json();
				throw new Error(error.error || 'Failed to delete session');
			}

			sessions = sessions.filter(s => s.id !== sessionId);
			if (currentSession?.id === sessionId) {
				currentSession = null;
				messages = [];
			}
			toast.success('Chat session deleted');
		} catch (error) {
			const message = error instanceof Error ? error.message : 'Failed to delete session';
			toast.error('Error', { description: message });
		}
	}

	function formatDate(dateString: string) {
		return new Date(dateString).toLocaleString();
	}

	function getModelDisplayName(model: string) {
		if (model.startsWith('ollama:')) {
			return model.replace('ollama:', '');
		}
		return model;
	}

	function handleKeyPress(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			sendMessage();
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="border-none bg-gray-700 text-gray-200 sm:max-w-4xl max-h-[90vh] overflow-hidden">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<MessageCircle class="h-5 w-5" />
				Chat with Transcript
				{#if record}
					<span class="text-gray-400">- {record.title}</span>
				{/if}
			</Dialog.Title>
		</Dialog.Header>

		<div class="flex h-[70vh] gap-4">
			<!-- Sidebar with sessions -->
			<div class="w-80 border-r border-gray-600 flex flex-col">
				<!-- Create new session -->
				<div class="p-4 border-b border-gray-600">
					<div class="space-y-3">
						<Input
							placeholder="Session title"
							bind:value={newSessionTitle}
							class="bg-gray-600 border-gray-500 text-gray-200"
						/>
						<Select.Root bind:value={selectedModel} type="single">
							<Select.Trigger class="w-full bg-gray-600 border-gray-500 text-gray-200">
								{selectedModel ? getModelDisplayName(selectedModel) : 'Select model'}
							</Select.Trigger>
							<Select.Content>
								{#if openaiModels.length > 0}
									<Select.Group>
										<Select.Label>OpenAI Models</Select.Label>
										{#each openaiModels as model}
											<Select.Item value={model}>{model}</Select.Item>
										{/each}
									</Select.Group>
								{/if}
								{#if ollamaModels.length > 0}
									<Select.Group>
										<Select.Label>Ollama Models</Select.Label>
										{#each ollamaModels as model}
											<Select.Item value={model}>{getModelDisplayName(model)}</Select.Item>
										{/each}
									</Select.Group>
								{/if}
							</Select.Content>
						</Select.Root>
						<Button
							onclick={createNewSession}
							disabled={!newSessionTitle.trim() || !selectedModel || isCreatingSession}
							class="w-full bg-blue-500 text-gray-100 hover:bg-blue-600"
						>
							{#if isCreatingSession}
								Creating...
							{:else}
								<Plus class="h-4 w-4 mr-2" />
								New Chat
							{/if}
						</Button>
					</div>
				</div>

				<!-- Sessions list -->
				<ScrollArea class="flex-1">
					<div class="p-2 space-y-2">
						{#each sessions as session (session.id)}
							<button
								type="button"
								class="w-full p-3 rounded-lg cursor-pointer transition-colors text-left {currentSession?.id === session.id
									? 'bg-blue-500 text-white'
									: 'bg-gray-600 hover:bg-gray-500 text-gray-200'}"
								onclick={() => (currentSession = session)}
							>
								<div class="flex items-start justify-between">
									<div class="flex-1 min-w-0">
										<div class="font-medium truncate">{session.title}</div>
										<div class="text-xs opacity-75 truncate">
											{getModelDisplayName(session.model)}
										</div>
										<div class="text-xs opacity-50">
											{formatDate(session.updated_at)}
										</div>
									</div>
									<Button
										variant="ghost"
										size="sm"
										class="h-6 w-6 p-0 text-red-400 hover:text-red-300 hover:bg-red-500/20"
										onclick={(e) => {
											e.stopPropagation();
											deleteSession(session.id);
										}}
									>
										<Trash2 class="h-3 w-3" />
									</Button>
								</div>
							</button>
						{/each}
					</div>
				</ScrollArea>
			</div>

			<!-- Chat area -->
			<div class="flex-1 flex flex-col">
				{#if currentSession}
					<!-- Messages -->
					<ScrollArea class="flex-1 p-4">
						<div class="space-y-4">
							{#each messages as message (message.id)}
								{#if message.role !== 'system'}
									<div class="flex {message.role === 'user' ? 'justify-end' : 'justify-start'}">
										<div
											class="max-w-[80%] p-3 rounded-lg {message.role === 'user'
												? 'bg-blue-500 text-white'
												: 'bg-gray-600 text-gray-200'}"
										>
											<div class="whitespace-pre-wrap">{message.content}</div>
											<div class="text-xs opacity-50 mt-1">
												{formatDate(message.created_at)}
											</div>
										</div>
									</div>
								{/if}
							{/each}
							{#if isLoading}
								<div class="flex justify-start">
									<div class="bg-gray-600 text-gray-200 p-3 rounded-lg">
										<div class="flex items-center gap-2">
											<div class="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
											Thinking...
										</div>
									</div>
								</div>
							{/if}
						</div>
					</ScrollArea>

					<!-- Input area -->
					<div class="p-4 border-t border-gray-600">
						<div class="flex gap-2">
							<Input
								placeholder="Type your message..."
								bind:value={newMessage}
								onkeypress={handleKeyPress}
								disabled={isLoading}
								class="flex-1 bg-gray-600 border-gray-500 text-gray-200"
							/>
							<Button
								onclick={sendMessage}
								disabled={!newMessage.trim() || isLoading}
								class="bg-blue-500 text-gray-100 hover:bg-blue-600"
							>
								<Send class="h-4 w-4" />
							</Button>
						</div>
					</div>
				{:else}
					<!-- No session selected -->
					<div class="flex-1 flex items-center justify-center text-gray-400">
						<div class="text-center">
							<MessageCircle class="h-12 w-12 mx-auto mb-4 opacity-50" />
							<p>Select a chat session or create a new one to start chatting</p>
						</div>
					</div>
				{/if}
			</div>
		</div>
	</Dialog.Content>
</Dialog.Root> 