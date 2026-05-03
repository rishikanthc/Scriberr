import { createContext, useCallback, useContext, useEffect, useMemo, useRef, type PropsWithChildren } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { filesQueryKey } from "@/features/files/hooks/useFiles";
import type { FilesResponse, ScriberrFile } from "@/features/files/api/filesApi";
import { recordingQueryKey, recordingsQueryKey } from "@/features/recording/hooks/useRecordingSession";
import type { RecordingSession, RecordingStatus, RecordingsResponse } from "@/features/recording/api/recordingsApi";
import { transcriptionsQueryKey } from "@/features/transcription/hooks/useTranscriptions";
import type { TranscriptionStatus, TranscriptionsResponse } from "@/features/transcription/api/transcriptionsApi";

type AppEvent = {
  name: string;
  data: Record<string, unknown>;
};

export type FileEvent = AppEvent & {
  data: {
    id?: string;
    kind?: string;
    status?: string;
    progress?: number;
    title?: string;
    description?: string;
    deleted?: boolean;
  };
};

export type RecordingEvent = AppEvent & {
  data: {
    id?: string;
    status?: string;
    stage?: string;
    progress?: number;
    file_id?: string;
    transcription_id?: string;
  };
};

type TranscriptionEvent = AppEvent & {
  data: {
    id?: string;
    file_id?: string;
    status?: string;
    progress?: number;
    stage?: string;
  };
};

type Subscriber = {
  prefix: string;
  callback: (event: AppEvent) => void;
};

type AppEventsContextValue = {
  subscribe: (prefix: string, callback: (event: AppEvent) => void) => () => void;
};

const AppEventsContext = createContext<AppEventsContextValue | null>(null);

export function AppEventsProvider({ children }: PropsWithChildren) {
  const { token } = useAuth();
  const queryClient = useQueryClient();
  const subscribers = useRef<Subscriber[]>([]);

  const subscribe = useCallback((prefix: string, callback: (event: AppEvent) => void) => {
    const subscriber = { prefix, callback };
    subscribers.current = [...subscribers.current, subscriber];
    return () => {
      subscribers.current = subscribers.current.filter((item) => item !== subscriber);
    };
  }, []);

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
            const event = parseSSEChunk(chunk);
            if (!event) continue;
            applyAppEventToCache(queryClient, event);
            for (const subscriber of subscribers.current) {
              if (event.name.startsWith(subscriber.prefix)) subscriber.callback(event);
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
  }, [token, queryClient]);

  const value = useMemo(() => ({ subscribe }), [subscribe]);
  return <AppEventsContext.Provider value={value}>{children}</AppEventsContext.Provider>;
}

export function useAppEventSubscription<TEvent extends AppEvent>(prefix: string, callback: (event: TEvent) => void) {
  const context = useContext(AppEventsContext);

  useEffect(() => {
    if (!context) return;
    return context.subscribe(prefix, (event) => callback(event as TEvent));
  }, [callback, context, prefix]);
}

function applyAppEventToCache(queryClient: ReturnType<typeof useQueryClient>, event: AppEvent) {
  if (event.name.startsWith("file.")) {
    applyFileEventToCache(queryClient, event as FileEvent);
  } else if (event.name.startsWith("transcription.")) {
    applyTranscriptionEventToCache(queryClient, event as TranscriptionEvent);
  } else if (event.name.startsWith("recording.")) {
    applyRecordingEventToCache(queryClient, event as RecordingEvent);
  }
}

function applyFileEventToCache(queryClient: ReturnType<typeof useQueryClient>, event: FileEvent) {
  const fileId = event.data.id;
  if (!fileId) return;

  let updatedKnownFile = false;
  queryClient.setQueryData<FilesResponse>(filesQueryKey, (current) => {
    if (!current) return current;
    return {
      ...current,
      items: current.items.map((file) => {
        if (file.id !== fileId) return file;
        updatedKnownFile = true;
        return applyFileEvent(file, event);
      }),
    };
  });
  queryClient.setQueryData<ScriberrFile>([...filesQueryKey, fileId], (current) => {
    if (!current) return current;
    updatedKnownFile = true;
    return applyFileEvent(current, event);
  });
  if (!updatedKnownFile) queryClient.invalidateQueries({ queryKey: filesQueryKey });
}

function applyFileEvent(file: ScriberrFile, event: FileEvent): ScriberrFile {
  return {
    ...file,
    title: event.data.title ?? file.title,
    description: event.data.description ?? file.description,
    status: normalizeFileStatus(event.data.status) ?? file.status,
    updated_at: new Date().toISOString(),
  };
}

function applyTranscriptionEventToCache(queryClient: ReturnType<typeof useQueryClient>, event: TranscriptionEvent) {
  const transcriptionId = event.data.id;
  if (!transcriptionId) return;

  let updatedKnownTranscription = false;
  queryClient.setQueriesData<TranscriptionsResponse>({ queryKey: transcriptionsQueryKey }, (current) => {
    if (!current) return current;
    return {
      ...current,
      items: current.items.map((transcription) => {
        if (transcription.id !== transcriptionId) return transcription;
        updatedKnownTranscription = true;
        return {
          ...transcription,
          status: normalizeTranscriptionStatus(event.data.status) || transcription.status,
          progress: event.data.progress ?? transcription.progress,
          progress_stage: event.data.stage || transcription.progress_stage,
          updated_at: new Date().toISOString(),
        };
      }),
    };
  });
  if (!updatedKnownTranscription) queryClient.invalidateQueries({ queryKey: transcriptionsQueryKey });
}

function applyRecordingEventToCache(queryClient: ReturnType<typeof useQueryClient>, event: RecordingEvent) {
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
  if (!updatedKnownRecording) queryClient.invalidateQueries({ queryKey: recordingsQueryKey });
  if (event.name === "recording.ready" || event.data.file_id) queryClient.invalidateQueries({ queryKey: filesQueryKey });
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

function normalizeFileStatus(status: string | undefined): ScriberrFile["status"] | undefined {
  switch (status) {
    case "ready":
    case "uploaded":
    case "processing":
    case "failed":
    case "stopped":
    case "canceled":
      return status;
    default:
      return undefined;
  }
}

function normalizeTranscriptionStatus(status: string | undefined): TranscriptionStatus | undefined {
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

function parseSSEChunk(chunk: string): AppEvent | null {
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
    return { name, data: JSON.parse(data) as Record<string, unknown> };
  } catch {
    return null;
  }
}
