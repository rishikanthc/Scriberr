import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import type { TranscriptionStatus, TranscriptionsResponse } from "@/features/transcription/api/transcriptionsApi";
import { transcriptionTranscriptQueryKey, transcriptionsQueryKey } from "@/features/transcription/hooks/useTranscriptions";

type TranscriptionEvent = {
  name: string;
  data: {
    id?: string;
    status?: string;
    progress?: number;
    stage?: string;
  };
};

export function useTranscriptionDetailEvents(transcriptionId: string | undefined) {
  const { token } = useAuth();
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!token || !transcriptionId) return;

    const abortController = new AbortController();
    let reconnectTimer: number | undefined;

    const invalidateDetail = () => {
      queryClient.invalidateQueries({ queryKey: transcriptionsQueryKey });
      queryClient.invalidateQueries({ queryKey: transcriptionTranscriptQueryKey(transcriptionId) });
    };

    const scheduleReconnect = () => {
      if (abortController.signal.aborted) return;
      reconnectTimer = window.setTimeout(() => {
        void connect();
      }, 1500);
    };

    const connect = async () => {
      try {
        const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/events`, {
          headers: { Authorization: `Bearer ${token}` },
          signal: abortController.signal,
        });

        if (!response.ok || !response.body) {
          scheduleReconnect();
          return;
        }

        invalidateDetail();

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        while (!abortController.signal.aborted) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const chunks = buffer.split("\n\n");
          buffer = chunks.pop() || "";

          for (const chunk of chunks) {
            const parsed = parseSSEChunk(chunk);
            if (!parsed || parsed.data.id !== transcriptionId) continue;

            queryClient.setQueryData<TranscriptionsResponse>(transcriptionsQueryKey, (current) => {
              if (!current) return current;
              return {
                ...current,
                items: current.items.map((transcription) => {
                  if (transcription.id !== transcriptionId) return transcription;
                  return {
                    ...transcription,
                    status: normalizeEventStatus(parsed.data.status) || transcription.status,
                    progress: parsed.data.progress ?? transcription.progress,
                    progress_stage: parsed.data.stage || transcription.progress_stage,
                    updated_at: new Date().toISOString(),
                  };
                }),
              };
            });

            if (parsed.name === "transcription.completed" || parsed.data.status === "completed") {
              queryClient.invalidateQueries({ queryKey: transcriptionTranscriptQueryKey(transcriptionId) });
            }
          }
        }

        scheduleReconnect();
      } catch (error) {
        if ((error as Error).name === "AbortError") return;
        scheduleReconnect();
      }
    };

    void connect();

    return () => {
      abortController.abort();
      if (reconnectTimer) window.clearTimeout(reconnectTimer);
    };
  }, [token, transcriptionId, queryClient]);
}

function parseSSEChunk(chunk: string): TranscriptionEvent | null {
  let name = "";
  let data = "";

  for (const line of chunk.split("\n")) {
    if (line.startsWith("event:")) {
      name = line.slice("event:".length).trim();
    } else if (line.startsWith("data:")) {
      data += line.slice("data:".length).trim();
    }
  }

  if (!name || !data) return null;

  try {
    return { name, data: JSON.parse(data) };
  } catch {
    return null;
  }
}

function normalizeEventStatus(status?: string): TranscriptionStatus | undefined {
  switch (status) {
    case "queued":
    case "processing":
    case "completed":
    case "failed":
    case "canceled":
      return status;
    default:
      return undefined;
  }
}
