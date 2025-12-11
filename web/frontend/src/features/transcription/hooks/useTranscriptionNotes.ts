import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";

export interface Note {
    id: string;
    transcription_id: string;
    content: string;
    start_time: number;
    end_time: number;
    start_word_index: number;
    end_word_index: number; // inclusive
    quote: string;
    created_at?: string;
    updated_at?: string;
}

export function useNotes(audioId: string) {
    const { getAuthHeaders } = useAuth();

    return useQuery({
        queryKey: ["notes", audioId],
        queryFn: async () => {
            const response = await fetch(`/api/v1/transcription/${audioId}/notes`, {
                headers: getAuthHeaders(),
            });
            if (!response.ok) throw new Error("Failed to fetch notes");
            const data = await response.json();
            // Sort notes
            return data.sort((a: Note, b: Note) => {
                if (a.start_time !== b.start_time) return a.start_time - b.start_time;
                if (a.start_word_index !== b.start_word_index) return a.start_word_index - b.start_word_index;
                return (a.created_at || '').localeCompare(b.created_at || '');
            });
        },
        enabled: !!audioId,
    });
}

export function useCreateNote(audioId: string) {
    const { getAuthHeaders } = useAuth();
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (note: Omit<Note, "id" | "transcription_id" | "created_at" | "updated_at">) => {
            const response = await fetch(`/api/v1/transcription/${audioId}/notes`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    ...getAuthHeaders(),
                },
                body: JSON.stringify(note),
            });
            if (!response.ok) throw new Error("Failed to create note");
            return response.json();
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["notes", audioId] });
        },
    });
}

export function useUpdateNote(audioId: string) {
    const { getAuthHeaders } = useAuth();
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async ({ id, content }: { id: string; content: string }) => {
            const response = await fetch(`/api/v1/notes/${id}`, {
                method: "PUT",
                headers: {
                    "Content-Type": "application/json",
                    ...getAuthHeaders(),
                },
                body: JSON.stringify({ content }),
            });
            if (!response.ok) throw new Error("Failed to update note");
            return response.json();
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["notes", audioId] });
        },
    });
}

export function useDeleteNote(audioId: string) {
    const { getAuthHeaders } = useAuth();
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (id: string) => {
            const response = await fetch(`/api/v1/notes/${id}`, {
                method: "DELETE",
                headers: getAuthHeaders(),
            });
            if (!response.ok) throw new Error("Failed to delete note");
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["notes", audioId] });
        },
    });
}
