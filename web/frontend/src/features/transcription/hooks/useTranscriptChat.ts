import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useFiles } from "@/features/files/hooks/useFiles";
import { useTranscriptions } from "@/features/transcription/hooks/useTranscriptions";
import {
  addChatContextTranscript,
  createChatSession,
  deleteChatContextTranscript,
  listChatContext,
  listChatMessages,
  listChatModels,
  listChatSessions,
  streamChatMessage,
  updateChatSession,
  type ChatMessage,
  type ChatStreamEvent,
  type CreateChatSessionPayload,
  type StreamChatMessagePayload,
  type UpdateChatSessionPayload,
} from "@/features/transcription/api/chatApi";

export const chatModelsQueryKey = ["chat", "models"] as const;
export const chatSessionsQueryKey = (parentTranscriptionId: string) => ["chat", "sessions", parentTranscriptionId] as const;
export const chatMessagesQueryKey = (sessionId: string) => ["chat", "messages", sessionId] as const;
export const chatContextQueryKey = (sessionId: string) => ["chat", "context", sessionId] as const;

export function useChatModels(enabled = true) {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: chatModelsQueryKey,
    queryFn: () => listChatModels(getAuthHeaders()),
    enabled: isAuthenticated && enabled,
    retry: false,
  });
}

export function useChatSessions(parentTranscriptionId: string | undefined, enabled = true) {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: parentTranscriptionId ? chatSessionsQueryKey(parentTranscriptionId) : ["chat", "sessions", "missing"],
    queryFn: () => listChatSessions(parentTranscriptionId || "", getAuthHeaders()),
    enabled: isAuthenticated && enabled && Boolean(parentTranscriptionId),
  });
}

export function useChatMessages(sessionId: string | null | undefined, enabled = true) {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: sessionId ? chatMessagesQueryKey(sessionId) : ["chat", "messages", "missing"],
    queryFn: () => listChatMessages(sessionId || "", getAuthHeaders()),
    enabled: isAuthenticated && enabled && Boolean(sessionId),
  });
}

export function useChatContext(sessionId: string | null | undefined, enabled = true) {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: sessionId ? chatContextQueryKey(sessionId) : ["chat", "context", "missing"],
    queryFn: () => listChatContext(sessionId || "", getAuthHeaders()),
    enabled: isAuthenticated && enabled && Boolean(sessionId),
  });
}

export function useCreateChatSession(parentTranscriptionId: string | undefined) {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: CreateChatSessionPayload) => createChatSession(payload, getAuthHeaders()),
    onSuccess: (session) => {
      queryClient.invalidateQueries({ queryKey: chatSessionsQueryKey(parentTranscriptionId || session.parent_transcription_id) });
      queryClient.invalidateQueries({ queryKey: chatContextQueryKey(session.id) });
    },
  });
}

export function useUpdateChatSession(parentTranscriptionId: string | undefined) {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ sessionId, payload }: { sessionId: string; payload: UpdateChatSessionPayload }) => updateChatSession(sessionId, payload, getAuthHeaders()),
    onSuccess: (session) => {
      queryClient.invalidateQueries({ queryKey: chatSessionsQueryKey(parentTranscriptionId || session.parent_transcription_id) });
    },
  });
}

export function useAddChatContextTranscript() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ sessionId, transcriptionId }: { sessionId: string; transcriptionId: string }) => addChatContextTranscript(sessionId, transcriptionId, getAuthHeaders()),
    onSuccess: (_source, variables) => {
      const sessionId = variables?.sessionId;
      if (sessionId) queryClient.invalidateQueries({ queryKey: chatContextQueryKey(sessionId) });
    },
  });
}

export function useDeleteChatContextTranscript() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ sessionId, contextSourceId }: { sessionId: string; contextSourceId: string }) => deleteChatContextTranscript(sessionId, contextSourceId, getAuthHeaders()),
    onSuccess: (_source, variables) => {
      const sessionId = variables?.sessionId;
      if (sessionId) queryClient.invalidateQueries({ queryKey: chatContextQueryKey(sessionId) });
    },
  });
}

export function useStreamChatMessage() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ sessionId, payload, onEvent }: { sessionId: string; payload: StreamChatMessagePayload; onEvent: (event: ChatStreamEvent) => void }) => streamChatMessage(sessionId, payload, getAuthHeaders(), onEvent),
    onSettled: (_data, _error, variables) => {
      const sessionId = variables?.sessionId;
      if (sessionId) {
        queryClient.invalidateQueries({ queryKey: chatMessagesQueryKey(sessionId) });
        queryClient.invalidateQueries({ queryKey: chatContextQueryKey(sessionId) });
      }
    },
  });
}

export type CompletedTranscriptChoice = {
  transcriptionId: string;
  fileId: string;
  title: string;
  createdAt: string;
};

export function useCompletedTranscriptChoices() {
  const filesQuery = useFiles();
  const transcriptionsQuery = useTranscriptions();

  const fileById = new Map((filesQuery.data?.items || []).map((file) => [file.id, file]));
  const seenFileIds = new Set<string>();
  const choices: CompletedTranscriptChoice[] = [];

  for (const transcription of transcriptionsQuery.data?.items || []) {
    if (transcription.status !== "completed" || seenFileIds.has(transcription.file_id)) continue;
    seenFileIds.add(transcription.file_id);
    const file = fileById.get(transcription.file_id);
    choices.push({
      transcriptionId: transcription.id,
      fileId: transcription.file_id,
      title: file?.title?.trim() || transcription.title || "Untitled recording",
      createdAt: transcription.created_at,
    });
  }

  return {
    choices,
    isLoading: filesQuery.isLoading || transcriptionsQuery.isLoading,
    isError: filesQuery.isError || transcriptionsQuery.isError,
  };
}

export function mergeStreamMessages(current: ChatMessage[], event: ChatStreamEvent): ChatMessage[] {
  if (event.type !== "chat.message.created" && event.type !== "chat.delta.content" && event.type !== "chat.delta.reasoning" && event.type !== "chat.run.completed" && event.type !== "chat.run.failed") {
    return current;
  }
  if (event.type === "chat.message.created") {
    return upsertMessages(current, [event.user_message, event.assistant_message]);
  }
  return current.map((message) => {
    if (message.id !== event.message_id) return message;
    if (event.type === "chat.delta.content") {
      return { ...message, content: message.content + event.delta, status: "streaming" };
    }
    if (event.type === "chat.delta.reasoning") {
      return { ...message, reasoning_content: message.reasoning_content + event.delta, status: "streaming" };
    }
    if (event.type === "chat.run.completed") {
      return event.assistant_message;
    }
    return { ...message, status: "failed" };
  });
}

function upsertMessages(current: ChatMessage[], incoming: ChatMessage[]) {
  const next = [...current];
  for (const message of incoming) {
    const index = next.findIndex((item) => item.id === message.id);
    if (index >= 0) {
      next[index] = message;
    } else {
      next.push(message);
    }
  }
  return next;
}
