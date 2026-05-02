import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import {
  createTranscription,
  getTranscriptionTranscript,
  listTranscriptions,
  stopTranscription,
  type CreateTranscriptionPayload,
  type ListTranscriptionsOptions,
  type Transcription,
  type TranscriptionsResponse,
} from "@/features/transcription/api/transcriptionsApi";

export const transcriptionsQueryKey = ["transcriptions"] as const;
export const transcriptionTranscriptQueryKey = (transcriptionId: string) => ["transcription-transcript", transcriptionId] as const;

export function useTranscriptions() {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: transcriptionsQueryKey,
    queryFn: () => listTranscriptions(getAuthHeaders()),
    enabled: isAuthenticated,
  });
}

export function useTaggedTranscriptions(tagRef: string | undefined) {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: [...transcriptionsQueryKey, "tag", tagRef || "missing"],
    queryFn: () => listTranscriptions(getAuthHeaders(), { tagRefs: tagRef ? [tagRef] : [], tagMatch: "any" } satisfies ListTranscriptionsOptions),
    enabled: isAuthenticated && Boolean(tagRef),
  });
}

export function useTranscriptionTranscript(transcriptionId: string | undefined, enabled: boolean) {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: transcriptionId ? transcriptionTranscriptQueryKey(transcriptionId) : ["transcription-transcript", "missing"],
    queryFn: () => getTranscriptionTranscript(transcriptionId || "", getAuthHeaders()),
    enabled: isAuthenticated && Boolean(transcriptionId) && enabled,
    staleTime: 30_000,
  });
}

export function useCreateTranscription() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: CreateTranscriptionPayload) => createTranscription(payload, getAuthHeaders()),
    onSuccess: (transcription) => {
      queryClient.setQueryData<TranscriptionsResponse>(transcriptionsQueryKey, (current) => {
        if (!current) return { items: [transcription], next_cursor: null };
        const existing = current.items.some((item) => item.id === transcription.id);
        const items = existing
          ? current.items.map((item) => item.id === transcription.id ? transcription : item)
          : [transcription, ...current.items];
        return { ...current, items };
      });
      queryClient.invalidateQueries({ queryKey: transcriptionsQueryKey });
    },
  });
}

export function preferVisibleTranscription(candidate: Transcription, current: Transcription | undefined) {
  if (!current) return candidate;
  const candidateUpdatedAt = new Date(candidate.updated_at).getTime();
  const currentUpdatedAt = new Date(current.updated_at).getTime();
  if (candidateUpdatedAt !== currentUpdatedAt) {
    return candidateUpdatedAt > currentUpdatedAt ? candidate : current;
  }
  const candidateActive = isActiveTranscription(candidate);
  const currentActive = isActiveTranscription(current);
  if (candidateActive !== currentActive) return candidateActive ? candidate : current;
  return candidate;
}

function isActiveTranscription(transcription: Transcription) {
  return transcription.status === "queued" || transcription.status === "processing";
}

export function useStopTranscription() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (transcriptionId: string) => stopTranscription(transcriptionId, getAuthHeaders()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: transcriptionsQueryKey });
      queryClient.invalidateQueries({ queryKey: ["audioFiles"] });
    },
  });
}
