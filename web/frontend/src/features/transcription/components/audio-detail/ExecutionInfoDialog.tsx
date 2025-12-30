import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { Info, Clock, UsersRound } from "lucide-react";
import { useExecutionData } from "@/features/transcription/hooks/useAudioDetail";

interface ExecutionInfoDialogProps {
    audioId: string;
    isOpen: boolean;
    onClose: (open: boolean) => void;
}

export function ExecutionInfoDialog({ audioId, isOpen, onClose }: ExecutionInfoDialogProps) {
    const { data: executionData, isLoading } = useExecutionData(audioId);

    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-4xl w-[95vw] bg-[var(--bg-card)] border-[var(--border-subtle)] shadow-[var(--shadow-float)] max-h-[90vh] overflow-y-auto">
                <DialogHeader className="border-b border-[var(--border-subtle)] pb-4">
                    <DialogTitle className="text-[var(--text-primary)] flex items-center gap-2 text-xl font-bold tracking-tight">
                        <Info className="h-5 w-5 text-[var(--brand-solid)]" />
                        Transcription Details
                    </DialogTitle>
                    <DialogDescription className="text-[var(--text-secondary)]">
                        Technical execution metrics and processing parameters.
                    </DialogDescription>
                </DialogHeader>

                {isLoading ? (
                    <div className="py-12 flex flex-col items-center justify-center gap-4">
                        <div className="h-8 w-8 border-4 border-[var(--brand-solid)] border-t-transparent rounded-full animate-spin" />
                        <span className="text-[var(--text-tertiary)] animate-pulse">Loading execution data...</span>
                    </div>
                ) : executionData && executionData.available !== false ? (
                    <div className="space-y-6 py-4">
                        {/* Overall Processing Time */}
                        <div className="bg-[var(--bg-main)] rounded-[var(--radius-card)] border border-[var(--border-subtle)] p-4 sm:p-6 shadow-sm">
                            <h3 className="text-base font-bold text-[var(--text-primary)] mb-4 flex items-center gap-2">
                                <Clock className="h-4 w-4 text-[var(--text-secondary)]" />
                                Processing Timeline
                            </h3>
                            <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 sm:gap-4">
                                <MetricCard
                                    label="Started"
                                    value={executionData.started_at ? new Date(executionData.started_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : 'N/A'}
                                    subtext={executionData.started_at ? new Date(executionData.started_at).toLocaleDateString() : ''}
                                />
                                <MetricCard
                                    label="Completed"
                                    value={executionData.completed_at ? new Date(executionData.completed_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : 'In Progress'}
                                    subtext={executionData.completed_at ? new Date(executionData.completed_at).toLocaleDateString() : ''}
                                />
                                <MetricCard
                                    label="Duration"
                                    value={executionData.processing_duration ? `${(executionData.processing_duration / 1000).toFixed(1)}s` : '...'}
                                    highlight
                                    className="col-span-2 sm:col-span-1"
                                />
                            </div>
                        </div>

                        {/* Individual Track Processing */}
                        {executionData.is_multi_track && executionData.multi_track_timings && executionData.multi_track_timings.length > 0 && (
                            <div className="bg-[var(--bg-main)] rounded-[var(--radius-card)] border border-[var(--border-subtle)] p-4 sm:p-6 shadow-sm">
                                <h3 className="text-base font-bold text-[var(--text-primary)] mb-4 flex items-center gap-2">
                                    <UsersRound className="h-4 w-4 text-[var(--text-secondary)]" />
                                    Track Processing
                                </h3>
                                <div className="space-y-3">
                                    {executionData.multi_track_timings.map((timing, index) => (
                                        <div key={index} className="flex flex-col gap-2 p-3 bg-[var(--bg-card)] rounded-[var(--radius-card)] border border-[var(--border-subtle)]">
                                            <div className="flex justify-between items-start gap-2">
                                                <span className="font-medium text-[var(--text-primary)] text-sm break-all leading-tight">{timing.track_name}</span>
                                                <span className="font-mono text-sm font-bold text-[var(--brand-solid)] flex-shrink-0">
                                                    {(timing.duration / 1000).toFixed(1)}s
                                                </span>
                                            </div>
                                            <div className="flex justify-between text-[11px] text-[var(--text-tertiary)] bg-[var(--bg-main)]/50 p-1.5 rounded-[var(--radius-sm)]">
                                                <span>{new Date(timing.start_time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })}</span>
                                                <span>→</span>
                                                <span>{new Date(timing.end_time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })}</span>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}

                        {/* Parameters Display - Curated Snapshot */}
                        {executionData.actual_parameters && (
                            <div className="bg-[var(--bg-main)] rounded-[var(--radius-card)] border border-[var(--border-subtle)] p-4 sm:p-6 shadow-sm">
                                <h3 className="text-base font-bold text-[var(--text-primary)] mb-4">
                                    Configuration Parameters
                                </h3>
                                <CuratedParamsDisplay params={executionData.actual_parameters} />
                            </div>
                        )}
                    </div>
                ) : (
                    <div className="py-12 text-center text-[var(--text-tertiary)]">
                        No execution metrics available for this file.
                    </div>
                )}
            </DialogContent>
        </Dialog>
    );
}

function MetricCard({ label, value, subtext, highlight = false, className = "" }: { label: string; value: string; subtext?: string; highlight?: boolean; className?: string }) {
    return (
        <div className={`bg-[var(--bg-card)] p-3 rounded-[var(--radius-card)] border border-[var(--border-subtle)] flex flex-col justify-center ${className}`}>
            <span className="block text-[10px] sm:text-xs font-medium text-[var(--text-tertiary)] uppercase tracking-wider mb-1">{label}</span>
            <span className={`block font-mono text-sm sm:text-base ${highlight ? 'text-[var(--brand-solid)] font-bold' : 'text-[var(--text-primary)]'}`}>
                {value}
            </span>
            {subtext && <span className="block text-[10px] text-[var(--text-secondary)] mt-0.5">{subtext}</span>}
        </div>
    );
}

// Helper to display curated params based on model type
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function CuratedParamsDisplay({ params }: { params: any }) {
    // Determine keys to show based on model_family
    // Common keys for all
    const commonKeys = [
        'model_family',
        'task',
        'language',
        'output_format',
        'device',
        'compute_type',
        'batch_size',
        'diarize'
    ];

    let specificKeys: string[] = [];

    if (params.model_family === 'whisper') {
        specificKeys = [
            'model',
            'no_align',
            'vad_method',
            ...(params.diarize ? ['diarize_model', 'min_speakers', 'max_speakers', 'hf_token'] : [])
        ];
    } else if (params.model_family === 'nvidia_parakeet') {
        specificKeys = [
            'attention_context_left',
            'attention_context_right',
            ...(params.diarize ? ['diarize_model'] : [])
        ];
    } else if (params.model_family === 'openai') {
        specificKeys = ['model', 'api_key']; // api_key should ideally be masked or hidden
    } else if (params.model_family === 'nvidia_canary') {
        specificKeys = ['model']; // Canary usually simpler
    }

    const keysToShow = [...commonKeys, ...specificKeys];

    // Filter and map
    const displayEntries = keysToShow.map(key => {
        let value = params[key];
        if (value === undefined || value === null) return null;

        // Formatting
        if (typeof value === 'boolean') value = value ? 'Yes' : 'No';
        if (key === 'hf_token' || key === 'api_key') value = '******'; // Mask secrets

        return { key: formatParamKey(key), value };
    }).filter(entry => entry !== null);

    return (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-4 gap-y-2 text-sm">
            {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
            {displayEntries.map((entry: any) => (
                <div key={entry.key} className="flex justify-between items-center py-1 border-b border-[var(--border-subtle)] last:border-0 sm:last:border-b">
                    <span className="text-[var(--text-secondary)]">{entry.key}</span>
                    <span className="font-mono text-[var(--text-primary)] font-medium text-xs break-all text-right ml-4">{entry.value}</span>
                </div>
            ))}
        </div>
    );
}

function formatParamKey(key: string): string {
    return key.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ');
}
