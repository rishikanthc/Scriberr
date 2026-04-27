import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import {
  deleteProfile,
  listProfiles,
  listTranscriptionModels,
  saveProfile,
  type TranscriptionProfileOptions,
} from "@/features/settings/api/profilesApi";

export const profilesQueryKey = ["profiles"] as const;
export const transcriptionModelsQueryKey = ["transcription-models"] as const;

export function useProfiles() {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: profilesQueryKey,
    queryFn: () => listProfiles(getAuthHeaders()),
    enabled: isAuthenticated,
  });
}

export function useTranscriptionModels() {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: transcriptionModelsQueryKey,
    queryFn: () => listTranscriptionModels(getAuthHeaders()),
    enabled: isAuthenticated,
  });
}

export function useSaveProfile() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (profile: {
      id?: string;
      name: string;
      description: string;
      is_default: boolean;
      options: TranscriptionProfileOptions;
    }) => saveProfile(profile, getAuthHeaders()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: profilesQueryKey });
    },
  });
}

export function useDeleteProfile() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (profileId: string) => deleteProfile(profileId, getAuthHeaders()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: profilesQueryKey });
    },
  });
}
