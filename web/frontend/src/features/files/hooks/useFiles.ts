import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { getFile, listFiles, updateFile, type FilesResponse, type UpdateFilePayload } from "@/features/files/api/filesApi";

export const filesQueryKey = ["files"] as const;

export function useFiles() {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: filesQueryKey,
    queryFn: () => listFiles(getAuthHeaders()),
    enabled: isAuthenticated,
    refetchInterval: (query) => {
      const data = query.state.data as FilesResponse | undefined;
      return data?.items.some((file) => file.status === "processing") ? 1500 : false;
    },
  });
}

export function useFile(fileId: string) {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: [...filesQueryKey, fileId],
    queryFn: () => getFile(fileId, getAuthHeaders()),
    enabled: isAuthenticated && Boolean(fileId),
  });
}

export function useUpdateFile(fileId: string) {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: UpdateFilePayload) => updateFile(fileId, payload, getAuthHeaders()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: filesQueryKey });
      queryClient.invalidateQueries({ queryKey: [...filesQueryKey, fileId] });
    },
  });
}
