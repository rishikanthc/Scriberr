import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { listFiles } from "@/features/files/api/filesApi";

export const filesQueryKey = ["files"] as const;

export function useFiles() {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: filesQueryKey,
    queryFn: () => listFiles(getAuthHeaders()),
    enabled: isAuthenticated,
  });
}
