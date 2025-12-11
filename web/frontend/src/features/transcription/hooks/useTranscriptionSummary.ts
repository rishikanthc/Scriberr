import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useState } from "react";

export interface SummaryTemplate {
    id: string;
    name: string;
    model: string;
    prompt: string;
}

export function useSummaryTemplates() {
    const { getAuthHeaders } = useAuth();
    return useQuery({
        queryKey: ["summaryTemplates"],
        queryFn: async () => {
            const response = await fetch("/api/v1/summaries", {
                headers: getAuthHeaders(),
            });
            if (!response.ok) throw new Error("Failed to load summary templates");
            return response.json() as Promise<SummaryTemplate[]>;
        },
        staleTime: 5 * 60 * 1000, // Templates don't change often
    });
}

export function useExistingSummary(audioId: string) {
    const { getAuthHeaders } = useAuth();
    return useQuery({
        queryKey: ["summary", audioId],
        queryFn: async () => {
            const response = await fetch(`/api/v1/transcription/${audioId}/summary`, {
                headers: getAuthHeaders(),
            });
            if (!response.ok) return null; // No summary exists
            return response.json() as Promise<{ content: string }>;
        },
        retry: false,
    });
}

export function useSummarizer(audioId: string) {
    const { getAuthHeaders } = useAuth();
    const queryClient = useQueryClient();
    const [isStreaming, setIsStreaming] = useState(false);
    const [streamContent, setStreamContent] = useState("");
    const [error, setError] = useState<string | null>(null);

    const generateSummary = async (templateId: string, model: string, prompt: string, transcriptText: string) => {
        setIsStreaming(true);
        setStreamContent("");
        setError(null);

        const combinedContent = `Transcript:\n${transcriptText}\n\nInstructions:\n${prompt}`;

        try {
            const res = await fetch('/api/v1/summarize', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
                body: JSON.stringify({
                    model: model,
                    content: combinedContent,
                    transcription_id: audioId,
                    template_id: templateId
                }),
            });

            if (!res.body) {
                throw new Error('Failed to start summary stream.');
            }

            const reader = res.body.getReader();
            const decoder = new TextDecoder();

            while (true) {
                const { done, value } = await reader.read();
                if (done) {
                    const tail = decoder.decode();
                    if (tail) setStreamContent(prev => prev + tail);
                    break;
                }
                const chunk = decoder.decode(value, { stream: true });
                if (chunk) setStreamContent(prev => prev + chunk);
            }

            // Invalidate summary query after successful generation
            queryClient.invalidateQueries({ queryKey: ["summary", audioId] });

        } catch (e) {
            setError(e instanceof Error ? e.message : "Summary generation failed");
        } finally {
            setIsStreaming(false);
        }
    };

    return { generateSummary, isStreaming, streamContent, error };
}
