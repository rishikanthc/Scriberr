import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { filesQueryKey } from "@/features/files/hooks/useFiles";

export type FileEvent = {
  name: string;
  data: {
    id?: string;
    kind?: string;
    status?: string;
    progress?: number;
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
          headers: { Authorization: `Bearer ${token}` },
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
          const chunks = buffer.split("\n\n");
          buffer = chunks.pop() || "";

          for (const chunk of chunks) {
            const parsed = parseSSEChunk(chunk);
            if (!parsed || !parsed.name.startsWith("file.")) continue;
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

function parseSSEChunk(chunk: string): FileEvent | null {
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
