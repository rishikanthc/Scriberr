import { useEffect, useRef } from 'react';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { useQueryClient } from '@tanstack/react-query';
import type { AudioFile } from '@/features/transcription/hooks/useAudioFiles';

interface JobUpdateEvent {
    type: string;
    payload: {
        job_id: string;
        status: AudioFile['status'];
        error?: string;
        progress?: number;
    };
}

type PipelineStage =
    | 'queued'
    | 'preprocessing'
    | 'transcribing'
    | 'diarizing'
    | 'merging'
    | 'persisting'
    | 'completed'
    | 'failed';

interface PipelineUpdateEvent {
    type: string;
    payload: {
        job_id: string;
        execution_id?: string;
        stage: PipelineStage;
        progress: number;
        step_message?: string;
        error?: string;
        started_at: string;
        updated_at: string;
    };
}

export const useTranscriptionEvents = (jobId: string | null) => {
    const { token, getAuthHeaders } = useAuth();
    const queryClient = useQueryClient();
    const abortControllerRef = useRef<AbortController | null>(null);

    useEffect(() => {
        if (!token || !jobId) return;

        // Cleanup previous connection if any
        if (abortControllerRef.current) {
            abortControllerRef.current.abort();
        }

        const abortController = new AbortController();
        abortControllerRef.current = abortController;

        const upsertJobState = (
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            oldData: any,
            jobID: string,
            patch: Partial<AudioFile & { pipeline_stage?: string; pipeline_progress?: number; pipeline_message?: string }>
        ) => {
            if (!oldData) return oldData;

            if (oldData.pages) {
                return {
                    ...oldData,
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    pages: oldData.pages.map((page: any) => ({
                        ...page,
                        jobs: page.jobs.map((job: AudioFile) =>
                            job.id === jobID ? { ...job, ...patch } : job
                        ),
                    })),
                };
            }

            if (oldData.jobs) {
                return {
                    ...oldData,
                    jobs: oldData.jobs.map((job: AudioFile) =>
                        job.id === jobID ? { ...job, ...patch } : job
                    ),
                };
            }

            return oldData;
        };

        const hydratePipelineSnapshot = async () => {
            try {
                const response = await fetch(`/api/v1/transcription/${jobId}/pipeline-status`, {
                    headers: getAuthHeaders(),
                    signal: abortController.signal,
                });

                if (!response.ok) return;
                const data = await response.json();
                if (!data?.available || !data?.status) return;

                queryClient.setQueriesData(
                    { queryKey: ['audioFiles'] },
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    (oldData: any) => upsertJobState(oldData, jobId, {
                        pipeline_stage: data.status.stage,
                        pipeline_progress: data.status.progress,
                        pipeline_message: data.status.step_message || undefined,
                    })
                );
            } catch {
                // Snapshot is best-effort; SSE remains the source of truth.
            }
        };

        const connect = async () => {
            try {
                const response = await fetch(`/api/v1/events?job_id=${jobId}`, {
                    headers: getAuthHeaders(),
                    signal: abortController.signal,
                });

                if (!response.ok) {
                    throw new Error(`SSE connection failed: ${response.status}`);
                }

                if (!response.body) {
                    throw new Error('No response body');
                }

                const reader = response.body.getReader();
                const decoder = new TextDecoder();
                let buffer = '';

                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;

                    const chunk = decoder.decode(value, { stream: true });
                    buffer += chunk;

                    const lines = buffer.split('\n\n');
                    // Keep the last partial line in buffer
                    buffer = lines.pop() || '';

                    for (const line of lines) {
                        const trimmed = line.trim();
                        if (!trimmed || trimmed.startsWith(':')) continue; // Skip comments/keepalives

                        if (trimmed.startsWith('data: ')) {
                            const data = trimmed.slice(6);
                            try {
                                const event = JSON.parse(data);
                                handleEvent(event);
                            } catch (e) {
                                console.error('Failed to parse SSE data:', e);
                            }
                        }
                    }
                }
            } catch (error) {
                if ((error as Error).name !== 'AbortError') {
                    // Ignore "Error in input stream" which happens on abort/close in some browsers
                    const errorMsg = (error as Error).message;
                    if (!errorMsg.includes('Error in input stream')) {
                        console.error('SSE connection error, reconnecting in 5s...', error);
                        setTimeout(() => {
                            if (!abortController.signal.aborted) {
                                connect();
                            }
                        }, 5000);
                    }
                }
            }
        };

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const handleEvent = (event: any) => {
            if (event.type === 'job_update') {
                const payload = event.payload as JobUpdateEvent['payload'];

                queryClient.setQueriesData(
                    { queryKey: ['audioFiles'] },
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    (oldData: any) => upsertJobState(oldData, payload.job_id, {
                        status: payload.status,
                        error_message: payload.error || undefined,
                    })
                );
            }

            if (event.type === 'pipeline_update') {
                const payload = event.payload as PipelineUpdateEvent['payload'];
                queryClient.setQueriesData(
                    { queryKey: ['audioFiles'] },
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    (oldData: any) => upsertJobState(oldData, payload.job_id, {
                        pipeline_stage: payload.stage,
                        pipeline_progress: payload.progress,
                        pipeline_message: payload.step_message || undefined,
                        error_message: payload.error || undefined,
                    })
                );
            }
        };

        hydratePipelineSnapshot();
        connect();

        return () => {
            abortController.abort();
        };
    }, [token, queryClient, jobId, getAuthHeaders]);
};
