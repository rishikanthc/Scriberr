export type TranscriptionStatus = "queued" | "processing" | "completed" | "failed" | "canceled";

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

export type TranscriptionsResponse = {
  items: Transcription[];
  next_cursor: string | null;
};

export type CreateTranscriptionPayload = {
  fileId: string;
  profileId: string;
  title?: string;
};

export async function listTranscriptions(headers: Record<string, string>): Promise<TranscriptionsResponse> {
  const response = await fetch("/api/v1/transcriptions?limit=100&sort=-created_at", {
    headers,
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<TranscriptionsResponse>;
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

async function readError(response: Response) {
  try {
    const data = await response.json();
    return data?.error?.message || data?.message || response.statusText;
  } catch {
    return response.statusText;
  }
}
