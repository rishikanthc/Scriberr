import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { transcriptionsQueryKey } from "@/features/transcription/hooks/useTranscriptions";

type TranscriptionEvent = {
  name: string;
  data: {
    id?: string;
    file_id?: string;
    status?: string;
    progress?: number;
  };
};

export function useTranscriptionListEvents() {
  const { token } = useAuth();
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!token) return;

    const abortController = new AbortController();

    const connect = async () => {
      try {
        const response = await fetch("/api/v1/events", {
          headers: { Authorization: `Bearer ${token}` },
          signal: abortController.signal,
        });

        if (!response.ok || !response.body) return;

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
            if (!parsed || !parsed.name.startsWith("transcription.")) continue;
            queryClient.invalidateQueries({ queryKey: transcriptionsQueryKey });
          }
        }
      } catch (error) {
        if ((error as Error).name === "AbortError") return;
      }
    };

    connect();

    return () => abortController.abort();
  }, [token, queryClient]);
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
