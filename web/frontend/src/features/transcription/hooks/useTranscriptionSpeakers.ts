import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";

export interface SpeakerMapping {
    id?: number;
    original_speaker: string;
    custom_name: string;
}

export function useSpeakerMappings(audioId: string, enabled: boolean) {
    const { getAuthHeaders } = useAuth();

    return useQuery({
        queryKey: ["speakerMappings", audioId],
        queryFn: async () => {
            const response = await fetch(`/api/v1/transcription/${audioId}/speakers`, {
                headers: getAuthHeaders(),
            });
            if (!response.ok) throw new Error("Failed to fetch speaker mappings");
            const mappings: SpeakerMapping[] = await response.json();

            // Convert to lookup object for easier consumption
            const mappingObj: Record<string, string> = {};
            mappings.forEach(mapping => {
                mappingObj[mapping.original_speaker] = mapping.custom_name;
            });
            return mappingObj;
        },
        enabled: enabled,
    });
}

export function useUpdateSpeaker(audioId: string) {
    const { getAuthHeaders } = useAuth();
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async ({ originalSpeaker, customName }: { originalSpeaker: string, customName: string }) => {
            const response = await fetch(`/api/v1/transcription/${audioId}/speakers`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    ...getAuthHeaders(),
                },
                body: JSON.stringify({
                    original_speaker: originalSpeaker,
                    custom_name: customName,
                }),
            });
            if (!response.ok) throw new Error("Failed to update speaker name");
            return response.json();
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["speakerMappings", audioId] });
            // Also invalidate transcript if it embeds speaker names (though currently we derive it)
        },
    });
}
