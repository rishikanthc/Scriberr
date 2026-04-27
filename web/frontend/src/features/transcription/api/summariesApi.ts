export type SummaryStatus = "pending" | "processing" | "completed" | "failed";

export type TranscriptionSummary = {
  id: string;
  transcription_id: string;
  content: string;
  model: string;
  provider: string;
  status: SummaryStatus;
  error: string | null;
  transcript_truncated: boolean;
  context_window: number;
  input_characters: number;
  created_at: string;
  updated_at: string;
  completed_at: string | null;
};

export async function getTranscriptionSummary(
  transcriptionId: string,
  headers: Record<string, string>
): Promise<TranscriptionSummary | null> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/summary`, {
    headers,
  });
  if (response.status === 404) return null;
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<TranscriptionSummary>;
}

async function readError(response: Response) {
  try {
    const data = await response.json();
    return data?.error?.message || data?.message || response.statusText;
  } catch {
    return response.statusText;
  }
}
