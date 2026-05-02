import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { transcriptionsQueryKey } from "@/features/transcription/hooks/useTranscriptions";
import {
  addTranscriptionTag,
  createTag,
  deleteTag,
  listTags,
  listTranscriptionTags,
  removeTranscriptionTag,
  replaceTranscriptionTags,
  updateTag,
  type SaveTagPayload,
  type TagsResponse,
} from "@/features/tags/api/tagsApi";

export const tagsQueryKey = ["tags"] as const;
export const transcriptionTagsQueryKey = (transcriptionId: string) => ["transcription-tags", transcriptionId] as const;

export function useTags() {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: tagsQueryKey,
    queryFn: () => listTags(getAuthHeaders()),
    enabled: isAuthenticated,
  });
}

export function useSaveTag() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: SaveTagPayload) => (
      payload.id
        ? updateTag({ ...payload, id: payload.id }, getAuthHeaders())
        : createTag(payload, getAuthHeaders())
    ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tagsQueryKey });
    },
  });
}

export function useDeleteTag() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (tagId: string) => deleteTag(tagId, getAuthHeaders()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tagsQueryKey });
      queryClient.invalidateQueries({ queryKey: ["transcription-tags"] });
      queryClient.invalidateQueries({ queryKey: transcriptionsQueryKey });
    },
  });
}

export function useTranscriptionTags(transcriptionId: string | undefined, enabled = true) {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: transcriptionId ? transcriptionTagsQueryKey(transcriptionId) : ["transcription-tags", "missing"],
    queryFn: () => listTranscriptionTags(transcriptionId || "", getAuthHeaders()),
    enabled: isAuthenticated && Boolean(transcriptionId) && enabled,
    staleTime: 30_000,
  });
}

export function useReplaceTranscriptionTags(transcriptionId: string) {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (tagIds: string[]) => replaceTranscriptionTags(transcriptionId, tagIds, getAuthHeaders()),
    onSuccess: (response) => {
      updateTranscriptionTagsCache(queryClient, transcriptionId, response);
      queryClient.invalidateQueries({ queryKey: transcriptionsQueryKey });
    },
  });
}

export function useAddTranscriptionTag(transcriptionId: string) {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (tagId: string) => addTranscriptionTag(transcriptionId, tagId, getAuthHeaders()),
    onSuccess: (response) => {
      updateTranscriptionTagsCache(queryClient, transcriptionId, response);
      queryClient.invalidateQueries({ queryKey: transcriptionsQueryKey });
    },
  });
}

export function useRemoveTranscriptionTag(transcriptionId: string) {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (tagId: string) => removeTranscriptionTag(transcriptionId, tagId, getAuthHeaders()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: transcriptionTagsQueryKey(transcriptionId) });
      queryClient.invalidateQueries({ queryKey: transcriptionsQueryKey });
    },
  });
}

function updateTranscriptionTagsCache(
  queryClient: ReturnType<typeof useQueryClient>,
  transcriptionId: string,
  response: TagsResponse
) {
  queryClient.setQueryData(transcriptionTagsQueryKey(transcriptionId), response);
  queryClient.invalidateQueries({ queryKey: transcriptionTagsQueryKey(transcriptionId) });
}
