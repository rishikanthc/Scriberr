import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import {
  getLLMProviderSettings,
  saveLLMProviderSettings,
  type SaveLLMProviderPayload,
} from "@/features/settings/api/llmProviderApi";

export const llmProviderQueryKey = ["settings", "llm-provider"] as const;

export function useLLMProviderSettings() {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: llmProviderQueryKey,
    queryFn: () => getLLMProviderSettings(getAuthHeaders()),
    enabled: isAuthenticated,
  });
}

export function useSaveLLMProviderSettings() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: SaveLLMProviderPayload) => saveLLMProviderSettings(payload, getAuthHeaders()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: llmProviderQueryKey });
    },
  });
}
