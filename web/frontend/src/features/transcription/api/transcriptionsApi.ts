export type TranscriptionStatus = "queued" | "processing" | "completed" | "failed" | "stopped" | "canceled";

export type Transcription = {
  id: string;
  file_id: string;
  title: string;
  status: TranscriptionStatus;
  progress: number;
  progress_stage: string;
  duration_seconds: number | null;
  created_at: string;
  updated_at: string;
};

export type TranscriptSegment = {
  id?: string;
  start: number;
  end: number;
  speaker?: string;
  text: string;
};

export type TranscriptWord = {
  start: number;
  end: number;
  word: string;
  speaker?: string;
};

export type TranscriptionTranscript = {
  transcription_id: string;
  text: string;
  segments: TranscriptSegment[];
  words: TranscriptWord[];
};

export type TranscriptionsResponse = {
  items: Transcription[];
  next_cursor: string | null;
};

export type CreateTranscriptionPayload = {
  fileId: string;
  profileId: string;
  title?: string;
};

export type ListTranscriptionsOptions = {
  tagRefs?: string[];
  tagMatch?: "any" | "all";
};

export async function listTranscriptions(headers: Record<string, string>, options: ListTranscriptionsOptions = {}): Promise<TranscriptionsResponse> {
  const items: Transcription[] = [];
  let nextCursor: string | null = null;

  do {
    const params = new URLSearchParams({ limit: "100", sort: "-created_at" });
    if (nextCursor) params.set("cursor", nextCursor);
    for (const tagRef of options.tagRefs || []) {
      params.append("tag", tagRef);
    }
    if (options.tagRefs?.length && options.tagMatch) {
      params.set("tag_match", options.tagMatch);
    }

    const response = await fetch(`/api/v1/transcriptions?${params.toString()}`, {
      headers,
    });
    if (!response.ok) throw new Error(await readError(response));

    const page = await response.json() as TranscriptionsResponse;
    items.push(...page.items);
    nextCursor = page.next_cursor;
  } while (nextCursor);

  return { items, next_cursor: null };
}

export async function getTranscriptionTranscript(
  transcriptionId: string,
  headers: Record<string, string>
): Promise<TranscriptionTranscript> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/transcript`, {
    headers,
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<TranscriptionTranscript>;
}

export async function createTranscription(
  payload: CreateTranscriptionPayload,
  headers: Record<string, string>
): Promise<Transcription> {
  const response = await fetch("/api/v1/transcriptions", {
    method: "POST",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      file_id: payload.fileId,
      profile_id: payload.profileId,
      title: payload.title,
    }),
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<Transcription>;
}

export async function stopTranscription(
  transcriptionId: string,
  headers: Record<string, string>
): Promise<Pick<Transcription, "id" | "status">> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}:stop`, {
    method: "POST",
    headers,
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<Pick<Transcription, "id" | "status">>;
}

async function readError(response: Response) {
  try {
    const data = await response.json();
    return data?.error?.message || data?.message || response.statusText;
  } catch {
    return response.statusText;
  }
}
