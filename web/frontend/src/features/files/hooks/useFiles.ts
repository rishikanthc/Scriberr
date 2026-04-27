import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { listFiles, type FilesResponse } from "@/features/files/api/filesApi";

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
