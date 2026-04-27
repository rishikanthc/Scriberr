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

export type SummaryWidgetRunStatus = "pending" | "processing" | "completed" | "failed";

export type SummaryWidgetRun = {
  id: string;
  summary_id: string;
  transcription_id: string;
  widget_id: string;
  widget_name: string;
  display_title: string;
  context_source: "summary" | "transcript";
  render_markdown: boolean;
  model: string;
  provider: string;
  status: SummaryWidgetRunStatus;
  output: string;
  error: string | null;
  context_truncated: boolean;
  context_window: number;
  input_characters: number;
  created_at: string;
  updated_at: string;
  completed_at: string | null;
};

type SummaryWidgetRunsResponse = {
  items?: SummaryWidgetRun[];
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

export async function listTranscriptionSummaryWidgets(
  transcriptionId: string,
  headers: Record<string, string>
): Promise<SummaryWidgetRun[]> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/summary/widgets`, {
    headers,
  });
  if (!response.ok) throw new Error(await readError(response));
  const data = (await response.json()) as SummaryWidgetRunsResponse;
  return data.items || [];
}

async function readError(response: Response) {
  try {
    const data = await response.json();
    return data?.error?.message || data?.message || response.statusText;
  } catch {
    return response.statusText;
  }
}
