import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { transcriptionsQueryKey } from "@/features/transcription/hooks/useTranscriptions";
import type { TranscriptionStatus, TranscriptionsResponse } from "@/features/transcription/api/transcriptionsApi";

type TranscriptionEvent = {
  name: string;
  data: {
    id?: string;
    file_id?: string;
    status?: string;
    progress?: number;
    stage?: string;
  };
};

export function useTranscriptionListEvents() {
  const { token } = useAuth();
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!token) return;

    const abortController = new AbortController();
    let reconnectTimer: number | undefined;

    const scheduleReconnect = () => {
      if (abortController.signal.aborted) return;
      reconnectTimer = window.setTimeout(() => {
        void connect();
      }, 1500);
    };

    const connect = async () => {
      try {
        const response = await fetch("/api/v1/events", {
          headers: { Authorization: `Bearer ${token}` },
          signal: abortController.signal,
        });

        if (!response.ok || !response.body) {
          scheduleReconnect();
          return;
        }

        queryClient.invalidateQueries({ queryKey: transcriptionsQueryKey });

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        while (!abortController.signal.aborted) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const chunks = buffer.split(/\r?\n\r?\n/);
          buffer = chunks.pop() || "";

          for (const chunk of chunks) {
            const parsed = parseSSEChunk(chunk);
            if (!parsed || !parsed.name.startsWith("transcription.")) continue;
            let updatedKnownTranscription = false;
            queryClient.setQueryData<TranscriptionsResponse>(transcriptionsQueryKey, (current) => {
              if (!current || !parsed.data.id) return current;

              return {
                ...current,
                items: current.items.map((transcription) => {
                  if (transcription.id !== parsed.data.id) return transcription;
                  updatedKnownTranscription = true;
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
            queryClient.setQueriesData({ queryKey: ["audioFiles"] }, (current: unknown) => {
              if (!current || !parsed.data.id) return current;
              return updateAudioFilesQueryData(current, parsed.data.file_id || parsed.data.id, parsed.data);
            });
            if (!updatedKnownTranscription) {
              queryClient.invalidateQueries({ queryKey: transcriptionsQueryKey });
              queryClient.invalidateQueries({ queryKey: ["audioFiles"] });
            }
          }
        }

        scheduleReconnect();
      } catch (error) {
        if ((error as Error).name === "AbortError") return;
        scheduleReconnect();
      }
    };

    connect();

    return () => {
      abortController.abort();
      if (reconnectTimer) window.clearTimeout(reconnectTimer);
    };
  }, [token, queryClient]);
}

function updateAudioFilesQueryData(current: unknown, id: string, data: TranscriptionEvent["data"]) {
  if (!current || typeof current !== "object") return current;
  const value = current as {
    pages?: Array<{ jobs?: Array<Record<string, unknown>> }>;
    jobs?: Array<Record<string, unknown>>;
  };
  const updateJob = (job: Record<string, unknown>) => {
    if (job.id !== id) return job;
    return {
      ...job,
      status: data.status || job.status,
      progress: data.progress ?? job.progress,
      progress_stage: data.stage || job.progress_stage,
      updated_at: new Date().toISOString(),
    };
  };
  if (Array.isArray(value.pages)) {
    return {
      ...value,
      pages: value.pages.map((page) => ({
        ...page,
        jobs: Array.isArray(page.jobs) ? page.jobs.map(updateJob) : page.jobs,
      })),
    };
  }
  if (Array.isArray(value.jobs)) {
    return {
      ...value,
      jobs: value.jobs.map(updateJob),
    };
  }
  return current;
}

function parseSSEChunk(chunk: string): TranscriptionEvent | null {
  let name = "";
  let data = "";

  for (const line of chunk.split("\n")) {
    const trimmed = line.trimEnd();
    if (trimmed.startsWith("event:")) {
      name = trimmed.slice("event:".length).trim();
    } else if (trimmed.startsWith("data:")) {
      data += trimmed.slice("data:".length).trim();
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
    case "stopped":
    case "canceled":
      return status;
    default:
      return undefined;
  }
}
