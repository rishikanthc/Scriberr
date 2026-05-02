export type TranscriptAnnotationKind = "highlight" | "note";
export type TranscriptAnnotationStatus = "active" | "stale";

export type TranscriptAnnotationAnchor = {
  start_ms: number;
  end_ms: number;
  start_word?: number;
  end_word?: number;
  start_char?: number;
  end_char?: number;
  text_hash?: string;
};

export type TranscriptAnnotationEntry = {
  id: string;
  annotation_id: string;
  transcription_id?: string;
  content: string;
  created_at: string;
  updated_at: string;
};

export type TranscriptAnnotation = {
  id: string;
  transcription_id: string;
  kind: TranscriptAnnotationKind;
  content: string | null;
  color: string | null;
  quote: string;
  anchor: TranscriptAnnotationAnchor;
  status: TranscriptAnnotationStatus;
  created_at: string;
  updated_at: string;
  entries?: TranscriptAnnotationEntry[];
};

export type TranscriptNoteAnnotation = TranscriptAnnotation & {
  kind: "note";
  content: null;
  entries: TranscriptAnnotationEntry[];
};

export type TranscriptAnnotationsResponse = {
  items: TranscriptAnnotation[];
  next_cursor: string | null;
};

export type ListTranscriptAnnotationsOptions = {
  kind?: TranscriptAnnotationKind;
  updated_after?: string;
};

export type CreateTranscriptAnnotationRequest = {
  kind: TranscriptAnnotationKind;
  content?: string | null;
  color?: string | null;
  quote: string;
  anchor: TranscriptAnnotationAnchor;
};

export type CreateTranscriptHighlightRequest = Omit<CreateTranscriptAnnotationRequest, "kind" | "content"> & {
  content?: null;
};

export type CreateTranscriptNoteRequest = Omit<CreateTranscriptAnnotationRequest, "kind" | "content"> & {
  content: string;
};

export type CreateTranscriptNoteEntryRequest = {
  content: string;
};

export type UpdateTranscriptNoteEntryRequest = {
  content: string;
};

export async function listTranscriptAnnotations(
  transcriptionId: string,
  headers: Record<string, string>,
  options: ListTranscriptAnnotationsOptions = {}
): Promise<TranscriptAnnotationsResponse> {
  const items: TranscriptAnnotation[] = [];
  let nextCursor: string | null = null;

  do {
    const params = new URLSearchParams({ limit: "100" });
    if (options.kind) params.set("kind", options.kind);
    if (options.updated_after) params.set("updated_after", options.updated_after);
    if (nextCursor) params.set("cursor", nextCursor);

    const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/annotations?${params.toString()}`, {
      headers,
    });
    if (!response.ok) throw new Error(await readError(response));

    const page = await response.json() as TranscriptAnnotationsResponse;
    items.push(...page.items);
    nextCursor = page.next_cursor;
  } while (nextCursor);

  return { items, next_cursor: null };
}

export async function createTranscriptAnnotation(
  transcriptionId: string,
  payload: CreateTranscriptAnnotationRequest,
  headers: Record<string, string>
): Promise<TranscriptAnnotation> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/annotations`, {
    method: "POST",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<TranscriptAnnotation>;
}

export async function createTranscriptNote(
  transcriptionId: string,
  payload: CreateTranscriptNoteRequest,
  headers: Record<string, string>
): Promise<TranscriptNoteAnnotation> {
  const annotation = await createTranscriptAnnotation(
    transcriptionId,
    { ...payload, kind: "note" },
    headers
  );
  return annotation as TranscriptNoteAnnotation;
}

export async function deleteTranscriptAnnotation(
  transcriptionId: string,
  annotationId: string,
  headers: Record<string, string>
): Promise<void> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/annotations/${annotationId}`, {
    method: "DELETE",
    headers,
  });
  if (!response.ok) throw new Error(await readError(response));
}

export async function createTranscriptNoteEntry(
  transcriptionId: string,
  annotationId: string,
  payload: CreateTranscriptNoteEntryRequest,
  headers: Record<string, string>
): Promise<TranscriptAnnotationEntry> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/annotations/${annotationId}/entries`, {
    method: "POST",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<TranscriptAnnotationEntry>;
}

export async function updateTranscriptNoteEntry(
  transcriptionId: string,
  annotationId: string,
  entryId: string,
  payload: UpdateTranscriptNoteEntryRequest,
  headers: Record<string, string>
): Promise<TranscriptAnnotationEntry> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/annotations/${annotationId}/entries/${entryId}`, {
    method: "PATCH",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<TranscriptAnnotationEntry>;
}

export async function deleteTranscriptNoteEntry(
  transcriptionId: string,
  annotationId: string,
  entryId: string,
  headers: Record<string, string>
): Promise<void> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/annotations/${annotationId}/entries/${entryId}`, {
    method: "DELETE",
    headers,
  });
  if (!response.ok) throw new Error(await readError(response));
}

async function readError(response: Response) {
  try {
    const data = await response.json();
    return data?.error?.message || data?.message || response.statusText;
  } catch {
    return response.statusText;
  }
}
