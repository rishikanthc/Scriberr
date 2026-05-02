export type RecordingStatus =
  | "recording"
  | "stopping"
  | "finalizing"
  | "ready"
  | "failed"
  | "canceled"
  | "expired";

export type RecordingSourceKind = "microphone" | "tab" | "system";

export type RecordingSession = {
  id: string;
  title: string;
  status: RecordingStatus;
  source_kind: RecordingSourceKind;
  mime_type: string;
  received_chunks: number;
  received_bytes: number;
  duration_seconds: number | null;
  file_id: string | null;
  transcription_id: string | null;
  progress: number;
  progress_stage: string;
  started_at: string | null;
  stopped_at: string | null;
  completed_at: string | null;
  failed_at: string | null;
  created_at: string;
  updated_at: string;
};

export type RecordingsResponse = {
  items: RecordingSession[];
  next_cursor: string | null;
};

export type CreateRecordingPayload = {
  title: string;
  source_kind: RecordingSourceKind;
  mime_type: string;
  codec?: string;
  chunk_duration_ms?: number;
  auto_transcribe?: boolean;
  profile_id?: string;
  options?: {
    language?: string;
    diarization?: boolean;
  };
};

export type RecordingChunkResponse = {
  recording_id: string;
  chunk_index: number;
  status: "stored" | "already_stored";
  received_chunks: number;
  received_bytes: number;
};

export type UploadRecordingChunkPayload = {
  recordingId: string;
  chunkIndex: number;
  chunk: Blob;
  mimeType: string;
  sha256?: string;
  durationMs?: number;
};

export type StopRecordingPayload = {
  final_chunk_index: number;
  duration_ms?: number;
  auto_transcribe?: boolean;
};

export type RecordingCommandResponse = {
  id: string;
  status: RecordingStatus;
};

export type ListRecordingsOptions = {
  limit?: number;
};

export async function listRecordings(
  headers: Record<string, string>,
  options: ListRecordingsOptions = {}
): Promise<RecordingsResponse> {
  const params = new URLSearchParams();
  if (options.limit) params.set("limit", options.limit.toString());

  const query = params.toString();
  const response = await fetch(`/api/v1/recordings${query ? `?${query}` : ""}`, {
    headers,
  });

  if (!response.ok) {
    throw new Error(await readErrorMessage(response, "Failed to load recordings"));
  }

  return response.json() as Promise<RecordingsResponse>;
}

export async function getRecording(
  recordingId: string,
  headers: Record<string, string>
): Promise<RecordingSession> {
  const response = await fetch(`/api/v1/recordings/${recordingId}`, {
    headers,
  });

  if (!response.ok) {
    throw new Error(await readErrorMessage(response, "Failed to load recording"));
  }

  return response.json() as Promise<RecordingSession>;
}

export async function createRecording(
  payload: CreateRecordingPayload,
  headers: Record<string, string>
): Promise<RecordingSession> {
  const response = await fetch("/api/v1/recordings", {
    method: "POST",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      ...payload,
      auto_transcribe: payload.auto_transcribe ?? false,
      options: payload.options ?? {},
    }),
  });

  if (!response.ok) {
    throw new Error(await readErrorMessage(response, "Failed to create recording"));
  }

  return response.json() as Promise<RecordingSession>;
}

export async function uploadRecordingChunk(
  payload: UploadRecordingChunkPayload,
  headers: Record<string, string>
): Promise<RecordingChunkResponse> {
  const chunkHeaders: Record<string, string> = {
    ...headers,
    "Content-Type": payload.mimeType || payload.chunk.type || "application/octet-stream",
  };
  if (payload.sha256) chunkHeaders["X-Chunk-SHA256"] = payload.sha256;
  if (typeof payload.durationMs === "number") {
    chunkHeaders["X-Chunk-Duration-Ms"] = Math.max(0, Math.round(payload.durationMs)).toString();
  }

  const response = await fetch(
    `/api/v1/recordings/${payload.recordingId}/chunks/${payload.chunkIndex}`,
    {
      method: "PUT",
      headers: chunkHeaders,
      body: payload.chunk,
    }
  );

  if (!response.ok) {
    throw new Error(await readErrorMessage(response, "Failed to upload recording chunk"));
  }

  return response.json() as Promise<RecordingChunkResponse>;
}

export async function stopRecording(
  recordingId: string,
  payload: StopRecordingPayload,
  headers: Record<string, string>
): Promise<RecordingSession> {
  const response = await fetch(`/api/v1/recordings/${recordingId}:stop`, {
    method: "POST",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error(await readErrorMessage(response, "Failed to stop recording"));
  }

  return response.json() as Promise<RecordingSession>;
}

export async function cancelRecording(
  recordingId: string,
  headers: Record<string, string>
): Promise<RecordingCommandResponse> {
  const response = await fetch(`/api/v1/recordings/${recordingId}:cancel`, {
    method: "POST",
    headers,
  });

  if (!response.ok) {
    throw new Error(await readErrorMessage(response, "Failed to cancel recording"));
  }

  return response.json() as Promise<RecordingCommandResponse>;
}

export async function retryFinalizeRecording(
  recordingId: string,
  headers: Record<string, string>
): Promise<RecordingSession> {
  const response = await fetch(`/api/v1/recordings/${recordingId}:retry-finalize`, {
    method: "POST",
    headers,
  });

  if (!response.ok) {
    throw new Error(await readErrorMessage(response, "Failed to retry recording finalization"));
  }

  return response.json() as Promise<RecordingSession>;
}

async function readErrorMessage(response: Response, fallback: string) {
  try {
    const body = await response.json() as { error?: { message?: string } };
    return body.error?.message || fallback;
  } catch {
    return fallback;
  }
}
