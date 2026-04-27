import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import {
  createTranscription,
  listTranscriptions,
  type CreateTranscriptionPayload,
  type TranscriptionsResponse,
} from "@/features/transcription/api/transcriptionsApi";

export const transcriptionsQueryKey = ["transcriptions"] as const;

export function useTranscriptions() {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: transcriptionsQueryKey,
    queryFn: () => listTranscriptions(getAuthHeaders()),
    enabled: isAuthenticated,
    refetchInterval: (query) => {
      const data = query.state.data as TranscriptionsResponse | undefined;
      return data?.items.some((transcription) => transcription.status === "queued" || transcription.status === "processing")
        ? 1500
        : false;
    },
  });
}

export function useCreateTranscription() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: CreateTranscriptionPayload) => createTranscription(payload, getAuthHeaders()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: transcriptionsQueryKey });
    },
  });
}
