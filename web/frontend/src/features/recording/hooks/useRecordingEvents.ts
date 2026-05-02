import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { filesQueryKey } from "@/features/files/hooks/useFiles";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { recordingQueryKey, recordingsQueryKey } from "@/features/recording/hooks/useRecordingSession";
import type { RecordingSession, RecordingStatus, RecordingsResponse } from "@/features/recording/api/recordingsApi";

export type RecordingEvent = {
  name: string;
  data: {
    id?: string;
    status?: string;
    stage?: string;
    progress?: number;
    file_id?: string;
    transcription_id?: string;
  };
};

export function useRecordingEvents(onEvent?: (event: RecordingEvent) => void) {
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
          cache: "no-store",
          headers: {
            Accept: "text/event-stream",
            Authorization: `Bearer ${token}`,
            "Cache-Control": "no-cache",
          },
          signal: abortController.signal,
        });

        if (!response.ok || !response.body) {
          scheduleReconnect();
          return;
        }

        queryClient.invalidateQueries({ queryKey: recordingsQueryKey });

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
            if (!parsed || !parsed.name.startsWith("recording.")) continue;

            applyRecordingEventToCache(queryClient, parsed);
            onEvent?.(parsed);
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
  }, [token, queryClient, onEvent]);
}

function applyRecordingEventToCache(
  queryClient: ReturnType<typeof useQueryClient>,
  event: RecordingEvent
) {
  const recordingId = event.data.id;
  if (!recordingId) return;

  let updatedKnownRecording = false;
  queryClient.setQueryData<RecordingsResponse>(recordingsQueryKey, (current) => {
    if (!current) return current;
    return {
      ...current,
      items: current.items.map((recording) => {
        if (recording.id !== recordingId) return recording;
        updatedKnownRecording = true;
        return applyRecordingEvent(recording, event);
      }),
    };
  });

  queryClient.setQueryData<RecordingSession>(recordingQueryKey(recordingId), (current) => {
    if (!current) return current;
    updatedKnownRecording = true;
    return applyRecordingEvent(current, event);
  });

  if (!updatedKnownRecording) {
    queryClient.invalidateQueries({ queryKey: recordingsQueryKey });
  }

  if (event.name === "recording.ready" || event.data.file_id) {
    queryClient.invalidateQueries({ queryKey: filesQueryKey });
    queryClient.invalidateQueries({ queryKey: ["audioFiles"] });
  }
}

function applyRecordingEvent(recording: RecordingSession, event: RecordingEvent): RecordingSession {
  return {
    ...recording,
    status: normalizeRecordingStatus(event.data.status) || recording.status,
    progress: event.data.progress ?? recording.progress,
    progress_stage: event.data.stage || recording.progress_stage,
    file_id: event.data.file_id ?? recording.file_id,
    transcription_id: event.data.transcription_id ?? recording.transcription_id,
    updated_at: new Date().toISOString(),
  };
}

function normalizeRecordingStatus(status: string | undefined): RecordingStatus | undefined {
  switch (status) {
    case "recording":
    case "stopping":
    case "finalizing":
    case "ready":
    case "failed":
    case "canceled":
    case "expired":
      return status;
    default:
      return undefined;
  }
}

function parseSSEChunk(chunk: string): RecordingEvent | null {
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
