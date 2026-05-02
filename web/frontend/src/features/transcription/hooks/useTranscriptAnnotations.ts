import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import {
  createTranscriptAnnotation,
  createTranscriptNote,
  createTranscriptNoteEntry,
  deleteTranscriptNoteEntry,
  deleteTranscriptAnnotation,
  listTranscriptAnnotations,
  updateTranscriptNoteEntry,
  type CreateTranscriptHighlightRequest,
  type CreateTranscriptNoteEntryRequest,
  type CreateTranscriptNoteRequest,
  type TranscriptAnnotation,
  type TranscriptNoteAnnotation,
} from "@/features/transcription/api/annotationsApi";

export const transcriptAnnotationsQueryKey = (transcriptionId: string) => ["transcript-annotations", transcriptionId] as const;

export function useTranscriptAnnotations(transcriptionId: string | undefined, enabled: boolean) {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: transcriptionId ? transcriptAnnotationsQueryKey(transcriptionId) : ["transcript-annotations", "missing"],
    queryFn: () => listTranscriptAnnotations(transcriptionId || "", getAuthHeaders()),
    enabled: isAuthenticated && Boolean(transcriptionId) && enabled,
    staleTime: 30_000,
  });
}

export function useCreateTranscriptHighlight(transcriptionId: string) {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: CreateTranscriptHighlightRequest) => createTranscriptAnnotation(
      transcriptionId,
      { ...payload, kind: "highlight", content: null },
      getAuthHeaders()
    ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: transcriptAnnotationsQueryKey(transcriptionId) });
    },
  });
}

export function useCreateTranscriptNote(transcriptionId: string) {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: CreateTranscriptNoteRequest) => createTranscriptNote(
      transcriptionId,
      payload,
      getAuthHeaders()
    ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: transcriptAnnotationsQueryKey(transcriptionId) });
    },
  });
}

export function useDeleteTranscriptAnnotation(transcriptionId: string) {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (annotationId: string) => deleteTranscriptAnnotation(transcriptionId, annotationId, getAuthHeaders()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: transcriptAnnotationsQueryKey(transcriptionId) });
    },
  });
}

export const useDeleteTranscriptHighlight = useDeleteTranscriptAnnotation;
export const useDeleteTranscriptNote = useDeleteTranscriptAnnotation;

export function useCreateTranscriptNoteEntry(transcriptionId: string) {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ annotationId, content }: CreateTranscriptNoteEntryRequest & { annotationId: string }) => createTranscriptNoteEntry(
      transcriptionId,
      annotationId,
      { content },
      getAuthHeaders()
    ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: transcriptAnnotationsQueryKey(transcriptionId) });
    },
  });
}

export function useUpdateTranscriptNoteEntry(transcriptionId: string) {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ annotationId, entryId, content }: { annotationId: string; entryId: string; content: string }) => updateTranscriptNoteEntry(
      transcriptionId,
      annotationId,
      entryId,
      { content },
      getAuthHeaders()
    ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: transcriptAnnotationsQueryKey(transcriptionId) });
    },
  });
}

export function useDeleteTranscriptNoteEntry(transcriptionId: string) {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ annotationId, entryId }: { annotationId: string; entryId: string }) => deleteTranscriptNoteEntry(
      transcriptionId,
      annotationId,
      entryId,
      getAuthHeaders()
    ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: transcriptAnnotationsQueryKey(transcriptionId) });
    },
  });
}

export function selectTranscriptNotes(annotations: TranscriptAnnotation[] | undefined): TranscriptNoteAnnotation[] {
  return (annotations || [])
    .filter(isTranscriptNote)
    .sort(compareTranscriptNotes);
}

function isTranscriptNote(annotation: TranscriptAnnotation): annotation is TranscriptNoteAnnotation {
  return annotation.kind === "note" && Array.isArray(annotation.entries) && annotation.entries.length > 0;
}

function compareTranscriptNotes(first: TranscriptNoteAnnotation, second: TranscriptNoteAnnotation) {
  const created = second.created_at.localeCompare(first.created_at);
  if (created !== 0) return created;
  return second.anchor.start_ms - first.anchor.start_ms;
}
