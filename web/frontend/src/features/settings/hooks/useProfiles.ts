import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { listProfiles } from "@/features/settings/api/profilesApi";

export const profilesQueryKey = ["profiles"] as const;

export function useProfiles() {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: profilesQueryKey,
    queryFn: () => listProfiles(getAuthHeaders()),
    enabled: isAuthenticated,
  });
}
