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
                ) : executionData ? (
                    <div className="space-y-6 py-4">
                        {/* Overall Processing Time */}
                        <div className="bg-[var(--bg-main)] rounded-[var(--radius-card)] border border-[var(--border-subtle)] p-6 shadow-sm">
                            <h3 className="text-base font-semibold text-[var(--text-primary)] mb-4 flex items-center gap-2">
                                <Clock className="h-4 w-4 text-[var(--text-secondary)]" />
                                Processing Timeline
                            </h3>
                            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                                <MetricCard
                                    label="Started"
                                    value={new Date(executionData.started_at).toLocaleString()}
                                />
                                <MetricCard
                                    label="Completed"
                                    value={executionData.completed_at ? new Date(executionData.completed_at).toLocaleString() : 'In Progress'}
                                />
                                <MetricCard
                                    label="Total Duration"
                                    value={executionData.processing_duration ? `${(executionData.processing_duration / 1000).toFixed(1)}s` : '...'}
                                    highlight
                                />
                            </div>
                        </div>

                        {/* Individual Track Processing */}
                        {executionData.is_multi_track && executionData.multi_track_timings && executionData.multi_track_timings.length > 0 && (
                            <div className="bg-[var(--bg-main)] rounded-[var(--radius-card)] border border-[var(--border-subtle)] p-6 shadow-sm">
                                <h3 className="text-base font-semibold text-[var(--text-primary)] mb-4 flex items-center gap-2">
                                    <UsersRound className="h-4 w-4 text-[var(--text-secondary)]" />
                                    Track Processing
                                </h3>
                                <div className="space-y-3">
                                    {executionData.multi_track_timings.map((timing, index) => (
                                        <div key={index} className="flex gap-4 p-4 bg-[var(--bg-card)] rounded-[var(--radius-card)] border border-[var(--border-subtle)] items-center justify-between">
                                            <div>
                                                <span className="block font-medium text-[var(--text-primary)] text-sm mb-1">{timing.track_name}</span>
                                                <div className="flex gap-3 text-xs text-[var(--text-tertiary)]">
                                                    <span>Start: {new Date(timing.start_time).toLocaleTimeString()}</span>
                                                    <span>End: {new Date(timing.end_time).toLocaleTimeString()}</span>
                                                </div>
                                            </div>
                                            <span className="font-mono text-lg font-bold text-[var(--brand-solid)]">
                                                {(timing.duration / 1000).toFixed(1)}s
                                            </span>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}

                        {/* Parameters Display */}
                        {executionData.actual_parameters && (
                            <div className="bg-[var(--bg-main)] rounded-[var(--radius-card)] border border-[var(--border-subtle)] p-6 shadow-sm">
                                <h3 className="text-base font-semibold text-[var(--text-primary)] mb-4">
                                    Configuration Parameters
                                </h3>
                                <div className="bg-[var(--bg-card)] p-4 rounded-[var(--radius-card)] border border-[var(--border-subtle)] font-mono text-xs text-[var(--text-secondary)] overflow-x-auto">
                                    <pre>{JSON.stringify(executionData.actual_parameters, null, 2)}</pre>
                                </div>
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

function MetricCard({ label, value, highlight = false }: { label: string; value: string; highlight?: boolean }) {
    return (
        <div className="bg-[var(--bg-card)] p-3 rounded-[var(--radius-card)] border border-[var(--border-subtle)]">
            <span className="block text-xs font-medium text-[var(--text-tertiary)] uppercase tracking-wider mb-1">{label}</span>
            <span className={`block font-mono text-sm sm:text-base ${highlight ? 'text-[var(--brand-solid)] font-bold' : 'text-[var(--text-primary)]'}`}>
                {value}
            </span>
        </div>
    );
}
