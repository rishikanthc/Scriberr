import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { filesQueryKey } from "@/features/files/hooks/useFiles";
import type { FilesResponse, ScriberrFile } from "@/features/files/api/filesApi";

export type FileEvent = {
  name: string;
  data: {
    id?: string;
    kind?: string;
    status?: string;
    progress?: number;
    title?: string;
    deleted?: boolean;
  };
};

export function useFileEvents(onEvent?: (event: FileEvent) => void) {
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

        queryClient.invalidateQueries({ queryKey: filesQueryKey });

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
            if (!parsed || !parsed.name.startsWith("file.")) continue;
            applyFileEventToCache(queryClient, parsed);
            queryClient.invalidateQueries({ queryKey: filesQueryKey });
            queryClient.invalidateQueries({ queryKey: ["audioFiles"] });
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

function applyFileEventToCache(queryClient: ReturnType<typeof useQueryClient>, event: FileEvent) {
  const fileId = event.data.id;
  if (!fileId) return;

  queryClient.setQueryData<FilesResponse>(filesQueryKey, (current) => {
    if (!current) return current;
    return {
      ...current,
      items: current.items.map((file) => file.id === fileId ? applyFileEvent(file, event) : file),
    };
  });
  queryClient.setQueryData<ScriberrFile>([...filesQueryKey, fileId], (current) => (
    current ? applyFileEvent(current, event) : current
  ));
}

function applyFileEvent(file: ScriberrFile, event: FileEvent): ScriberrFile {
  return {
    ...file,
    title: event.data.title ?? file.title,
    status: normalizeFileStatus(event.data.status) ?? file.status,
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

function parseSSEChunk(chunk: string): FileEvent | null {
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
