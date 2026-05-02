import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { filesQueryKey } from "@/features/files/hooks/useFiles";
import { useAuth } from "@/features/auth/hooks/useAuth";
import {
  cancelRecording,
  createRecording,
  getRecording,
  listRecordings,
  retryFinalizeRecording,
  stopRecording,
  uploadRecordingChunk,
  type CreateRecordingPayload,
  type RecordingSession,
  type RecordingsResponse,
  type StopRecordingPayload,
  type UploadRecordingChunkPayload,
} from "@/features/recording/api/recordingsApi";

export const recordingsQueryKey = ["recordings"] as const;
export const recordingQueryKey = (recordingId: string) => [...recordingsQueryKey, recordingId] as const;

export function useRecordings() {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: recordingsQueryKey,
    queryFn: () => listRecordings(getAuthHeaders()),
    enabled: isAuthenticated,
    refetchInterval: (query) => {
      const data = query.state.data as RecordingsResponse | undefined;
      return data?.items.some(isActiveRecordingSession) ? 1500 : false;
    },
  });
}

export function useRecording(recordingId: string | undefined) {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: recordingId ? recordingQueryKey(recordingId) : [...recordingsQueryKey, "missing"],
    queryFn: () => getRecording(recordingId || "", getAuthHeaders()),
    enabled: isAuthenticated && Boolean(recordingId),
    refetchInterval: (query) => {
      const data = query.state.data as RecordingSession | undefined;
      return data && isActiveRecordingSession(data) ? 1500 : false;
    },
  });
}

export function useCreateRecording() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: CreateRecordingPayload) => createRecording(payload, getAuthHeaders()),
    onSuccess: (recording) => {
      upsertRecordingSession(queryClient, recording);
    },
  });
}

export function useUploadRecordingChunk() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: UploadRecordingChunkPayload) => uploadRecordingChunk(payload, getAuthHeaders()),
    onSuccess: (chunk) => {
      queryClient.setQueryData<RecordingSession>(recordingQueryKey(chunk.recording_id), (current) => (
        current
          ? {
            ...current,
            received_chunks: chunk.received_chunks,
            received_bytes: chunk.received_bytes,
            updated_at: new Date().toISOString(),
          }
          : current
      ));
      queryClient.invalidateQueries({ queryKey: recordingsQueryKey });
    },
  });
}

export function useStopRecording() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ recordingId, payload }: { recordingId: string; payload: StopRecordingPayload }) => (
      stopRecording(recordingId, payload, getAuthHeaders())
    ),
    onSuccess: (recording) => {
      upsertRecordingSession(queryClient, recording);
      queryClient.invalidateQueries({ queryKey: filesQueryKey });
      queryClient.invalidateQueries({ queryKey: ["audioFiles"] });
    },
  });
}

export function useCancelRecording() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (recordingId: string) => cancelRecording(recordingId, getAuthHeaders()),
    onSuccess: (recording) => {
      queryClient.setQueryData<RecordingSession>(recordingQueryKey(recording.id), (current) => (
        current
          ? {
            ...current,
            status: recording.status,
            updated_at: new Date().toISOString(),
          }
          : current
      ));
      queryClient.invalidateQueries({ queryKey: recordingsQueryKey });
    },
  });
}

export function useRetryFinalizeRecording() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (recordingId: string) => retryFinalizeRecording(recordingId, getAuthHeaders()),
    onSuccess: (recording) => {
      upsertRecordingSession(queryClient, recording);
      queryClient.invalidateQueries({ queryKey: filesQueryKey });
      queryClient.invalidateQueries({ queryKey: ["audioFiles"] });
    },
  });
}

function upsertRecordingSession(
  queryClient: ReturnType<typeof useQueryClient>,
  recording: RecordingSession
) {
  queryClient.setQueryData<RecordingSession>(recordingQueryKey(recording.id), recording);
  queryClient.setQueryData<RecordingsResponse>(recordingsQueryKey, (current) => {
    if (!current) return { items: [recording], next_cursor: null };

    const existing = current.items.some((item) => item.id === recording.id);
    const items = existing
      ? current.items.map((item) => item.id === recording.id ? recording : item)
      : [recording, ...current.items];
    return { ...current, items };
  });
  queryClient.invalidateQueries({ queryKey: recordingsQueryKey });
}

function isActiveRecordingSession(recording: RecordingSession) {
  return recording.status === "recording" || recording.status === "stopping" || recording.status === "finalizing";
}
