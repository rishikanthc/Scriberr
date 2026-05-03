import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import {
  changePassword,
  getGeneralSettings,
  updateGeneralSettings,
  type ChangePasswordPayload,
  type UpdateGeneralSettingsPayload,
} from "@/features/settings/api/generalSettingsApi";

export const generalSettingsQueryKey = ["settings", "general"] as const;

export function useGeneralSettings() {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: generalSettingsQueryKey,
    queryFn: () => getGeneralSettings(getAuthHeaders()),
    enabled: isAuthenticated,
  });
}

export function useUpdateGeneralSettings() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: UpdateGeneralSettingsPayload) => updateGeneralSettings(payload, getAuthHeaders()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: generalSettingsQueryKey });
    },
  });
}

export function useChangePassword() {
  const { getAuthHeaders } = useAuth();

  return useMutation({
    mutationFn: (payload: ChangePasswordPayload) => changePassword(payload, getAuthHeaders()),
  });
}
