export type ChatModel = {
  id: string;
  display_name: string;
  context_window: number;
  context_window_source: string;
  supports_streaming: boolean;
  supports_reasoning: boolean;
};

export type ChatModelsResponse = {
  provider: string;
  configured: boolean;
  models: ChatModel[];
};

export type ChatSession = {
  id: string;
  parent_transcription_id: string;
  title: string;
  provider: string;
  model: string;
  system_prompt: string | null;
  status: "active" | "archived";
  last_message_at: string | null;
  created_at: string;
  updated_at: string;
};

export type ChatMessageRole = "user" | "assistant" | "system" | "tool";
export type ChatMessageStatus = "pending" | "streaming" | "completed" | "failed" | "canceled";

export type ChatMessage = {
  id: string;
  session_id: string;
  role: ChatMessageRole;
  content: string;
  reasoning_content: string;
  status: ChatMessageStatus;
  provider: string | null;
  model: string | null;
  run_id: string | null;
  prompt_tokens: number | null;
  completion_tokens: number | null;
  reasoning_tokens: number | null;
  total_tokens: number | null;
  created_at: string;
  updated_at: string;
};

export type ChatContextSource = {
  id: string;
  session_id: string;
  transcription_id: string;
  kind: "parent_transcript" | "transcript";
  enabled: boolean;
  status: "active" | "disabled" | "compacting" | "compacted" | "failed";
  position: number;
  compaction_status: "none" | "compacting" | "compacted" | "failed";
  has_plain_text_snapshot: boolean;
  has_compacted_snapshot: boolean;
  snapshot_hash: string | null;
  source_version: string | null;
  tokens_estimated: number;
  created_at: string;
  updated_at: string;
};

export type ChatRunUsage = {
  prompt_tokens: number;
  completion_tokens: number;
  reasoning_tokens: number;
  total_tokens: number;
};

export type ChatStreamEvent =
  | { type: "chat.run.started"; session_id: string; run_id: string; message_id: string; status?: string }
  | {
      type: "chat.message.created";
      session_id: string;
      run_id: string;
      message_id: string;
      assistant_message_id: string;
      user_message: ChatMessage;
      assistant_message: ChatMessage;
    }
  | { type: "chat.delta.reasoning"; session_id: string; run_id: string; message_id: string; delta: string }
  | { type: "chat.delta.content"; session_id: string; run_id: string; message_id: string; delta: string }
  | {
      type: "chat.run.completed";
      session_id: string;
      run_id: string;
      message_id: string;
      status: "completed";
      assistant_message: ChatMessage;
      usage?: ChatRunUsage;
    }
  | { type: "chat.run.failed"; session_id: string; run_id: string; message_id: string; error: string };

export type CreateChatSessionPayload = {
  parent_transcription_id: string;
  title?: string;
  model: string;
};

export type UpdateChatSessionPayload = {
  title?: string;
  status?: "active" | "archived";
  system_prompt?: string | null;
};

export type StreamChatMessagePayload = {
  content: string;
  model: string;
  temperature?: number;
};

type Collection<T> = {
  items: T[];
  next_cursor: string | null;
};

export async function listChatModels(headers: Record<string, string>): Promise<ChatModelsResponse> {
  const response = await fetch("/api/v1/chat/models", { headers });
  if (!response.ok) throw new Error(await readError(response));
  const data = await response.json() as ChatModelsResponse;
  return { ...data, models: data.models || [] };
}

export async function listChatSessions(parentTranscriptionId: string, headers: Record<string, string>): Promise<Collection<ChatSession>> {
  const params = new URLSearchParams({ parent_transcription_id: parentTranscriptionId });
  const response = await fetch(`/api/v1/chat/sessions?${params.toString()}`, { headers });
  if (!response.ok) throw new Error(await readError(response));
  const data = await response.json() as Collection<ChatSession>;
  return { items: data.items || [], next_cursor: data.next_cursor || null };
}

export async function createChatSession(payload: CreateChatSessionPayload, headers: Record<string, string>): Promise<ChatSession> {
  const response = await fetch("/api/v1/chat/sessions", {
    method: "POST",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<ChatSession>;
}

export async function updateChatSession(sessionId: string, payload: UpdateChatSessionPayload, headers: Record<string, string>): Promise<ChatSession> {
  const response = await fetch(`/api/v1/chat/sessions/${sessionId}`, {
    method: "PATCH",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<ChatSession>;
}

export async function listChatMessages(sessionId: string, headers: Record<string, string>): Promise<Collection<ChatMessage>> {
  const response = await fetch(`/api/v1/chat/sessions/${sessionId}/messages`, { headers });
  if (!response.ok) throw new Error(await readError(response));
  const data = await response.json() as Collection<ChatMessage>;
  return { items: data.items || [], next_cursor: data.next_cursor || null };
}

export async function listChatContext(sessionId: string, headers: Record<string, string>): Promise<Collection<ChatContextSource>> {
  const response = await fetch(`/api/v1/chat/sessions/${sessionId}/context`, { headers });
  if (!response.ok) throw new Error(await readError(response));
  const data = await response.json() as Collection<ChatContextSource>;
  return { items: data.items || [], next_cursor: data.next_cursor || null };
}

export async function addChatContextTranscript(sessionId: string, transcriptionId: string, headers: Record<string, string>): Promise<ChatContextSource> {
  const response = await fetch(`/api/v1/chat/sessions/${sessionId}/context/transcripts`, {
    method: "POST",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify({ transcription_id: transcriptionId }),
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<ChatContextSource>;
}

export async function deleteChatContextTranscript(sessionId: string, contextSourceId: string, headers: Record<string, string>): Promise<void> {
  const response = await fetch(`/api/v1/chat/sessions/${sessionId}/context/transcripts/${contextSourceId}`, {
    method: "DELETE",
    headers,
  });
  if (!response.ok) throw new Error(await readError(response));
}

export async function streamChatMessage(
  sessionId: string,
  payload: StreamChatMessagePayload,
  headers: Record<string, string>,
  onEvent: (event: ChatStreamEvent) => void
): Promise<void> {
  const response = await fetch(`/api/v1/chat/sessions/${sessionId}/messages:stream`, {
    method: "POST",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify({
      content: payload.content,
      model: payload.model,
      temperature: payload.temperature ?? 0,
    }),
  });
  if (!response.ok || !response.body) throw new Error(await readError(response));

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split(/\n\n/);
    buffer = parts.pop() || "";
    for (const part of parts) {
      const event = parseSSEEvent(part);
      if (event) onEvent(event);
    }
  }

  if (buffer.trim()) {
    const event = parseSSEEvent(buffer);
    if (event) onEvent(event);
  }
}

function parseSSEEvent(raw: string): ChatStreamEvent | null {
  const lines = raw.split(/\r?\n/);
  const type = lines.find((line) => line.startsWith("event:"))?.slice(6).trim();
  const data = lines
    .filter((line) => line.startsWith("data:"))
    .map((line) => line.slice(5).trimStart())
    .join("\n");
  if (!type || !data) return null;
  try {
    return { type, ...JSON.parse(data) } as ChatStreamEvent;
  } catch {
    return null;
  }
}

async function readError(response: Response) {
  try {
    const data = await response.json();
    return data?.error?.message || data?.message || response.statusText;
  } catch {
    return response.statusText;
  }
}
