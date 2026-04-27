import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { getTranscriptionSummary } from "@/features/transcription/api/summariesApi";

export const transcriptionSummaryQueryKey = (transcriptionId: string) => ["transcription-summary", transcriptionId] as const;

export function useTranscriptionSummary(transcriptionId: string | undefined, enabled: boolean) {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: transcriptionId ? transcriptionSummaryQueryKey(transcriptionId) : ["transcription-summary", "missing"],
    queryFn: () => getTranscriptionSummary(transcriptionId || "", getAuthHeaders()),
    enabled: isAuthenticated && Boolean(transcriptionId) && enabled,
    retry: false,
    staleTime: 30_000,
  });
}
