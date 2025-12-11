import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { Info, Clock } from "lucide-react";
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
            <DialogContent className="sm:max-w-4xl w-[95vw] bg-white dark:bg-carbon-900 border-carbon-200 dark:border-carbon-800 max-h-[90vh] overflow-y-auto">
                <DialogHeader>
                    <DialogTitle className="text-carbon-900 dark:text-carbon-100 flex items-center gap-2">
                        <Info className="h-5 w-5 text-carbon-600 dark:text-carbon-400" />
                        Transcription Execution Details
                    </DialogTitle>
                    <DialogDescription className="text-carbon-600 dark:text-carbon-400">
                        Parameters used and processing time for this transcription
                    </DialogDescription>
                </DialogHeader>

                {isLoading ? (
                    <div className="py-8 text-center">
                        <div className="animate-pulse">
                            <div className="h-4 bg-carbon-200 dark:bg-carbon-600 rounded w-3/4 mx-auto mb-4"></div>
                            <div className="h-4 bg-carbon-200 dark:bg-carbon-600 rounded w-1/2 mx-auto"></div>
                        </div>
                    </div>
                ) : executionData ? (
                    <div className="space-y-4 sm:space-y-6 py-2 sm:py-4">
                        {/* Overall Processing Time */}
                        <div className="bg-carbon-100/50 dark:bg-carbon-800/50 backdrop-blur-md border border-carbon-200/50 dark:border-carbon-700/50 rounded-lg p-4 sm:p-6 shadow-sm">
                            <h3 className="text-lg font-semibold text-carbon-900 dark:text-carbon-100 mb-3 sm:mb-4 flex items-center gap-2">
                                <Clock className="h-5 w-5 text-carbon-600 dark:text-carbon-400" />
                                Overall Processing Time
                            </h3>
                            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4 text-sm">
                                <div className="bg-white/60 dark:bg-carbon-950/40 backdrop-blur-sm rounded-md p-3 border border-carbon-200/50 dark:border-carbon-700/50 shadow-sm">
                                    <span className="text-carbon-600 dark:text-carbon-400 font-medium">Started:</span>
                                    <p className="font-mono text-carbon-900 dark:text-carbon-100 mt-1 text-xs sm:text-sm">
                                        {new Date(executionData.started_at).toLocaleString()}
                                    </p>
                                </div>
                                <div className="bg-white/60 dark:bg-carbon-950/40 backdrop-blur-sm rounded-md p-3 border border-carbon-200/50 dark:border-carbon-700/50 shadow-sm">
                                    <span className="text-carbon-600 dark:text-carbon-400 font-medium">Completed:</span>
                                    <p className="font-mono text-carbon-900 dark:text-carbon-100 mt-1 text-xs sm:text-sm">
                                        {executionData.completed_at
                                            ? new Date(executionData.completed_at).toLocaleString()
                                            : 'N/A'
                                        }
                                    </p>
                                </div>
                                <div className="bg-white/60 dark:bg-carbon-950/40 backdrop-blur-sm rounded-md p-3 border border-carbon-200/50 dark:border-carbon-700/50 shadow-sm">
                                    <span className="text-carbon-600 dark:text-carbon-400 font-medium">Total Duration:</span>
                                    <p className="font-mono text-xl sm:text-2xl font-bold text-carbon-900 dark:text-carbon-100 mt-1">
                                        {executionData.processing_duration
                                            ? `${(executionData.processing_duration / 1000).toFixed(1)}s`
                                            : 'N/A'
                                        }
                                    </p>
                                </div>
                            </div>
                        </div>

                        {/* Individual Track Processing */}
                        {executionData.is_multi_track && executionData.multi_track_timings && executionData.multi_track_timings.length > 0 && (
                            <div className="bg-carbon-100/50 dark:bg-carbon-800/50 backdrop-blur-md border border-carbon-200/50 dark:border-carbon-700/50 rounded-lg p-4 sm:p-6 shadow-sm">
                                <h3 className="text-lg font-semibold text-carbon-900 dark:text-carbon-100 mb-3 sm:mb-4 flex items-center gap-2">
                                    <UsersIcon />
                                    Individual Track Processing
                                </h3>
                                <div className="space-y-3">
                                    {executionData.multi_track_timings.map((timing, index) => (
                                        <div key={index} className="flex items-center gap-2 bg-carbon-200/50 dark:bg-carbon-800/50 backdrop-blur-sm p-1 rounded-lg border border-carbon-200/50 dark:border-carbon-700/50">
                                            <div className="bg-white/60 dark:bg-carbon-950/40 backdrop-blur-sm rounded-md p-3 border border-carbon-200/50 dark:border-carbon-700/50 flex-grow">
                                                <div className="flex justify-between items-center mb-2">
                                                    <span className="font-medium text-carbon-800 dark:text-carbon-200">
                                                        {timing.track_name}
                                                    </span>
                                                    <span className="font-mono text-lg font-bold text-carbon-600 dark:text-carbon-400">
                                                        {(timing.duration / 1000).toFixed(1)}s
                                                    </span>
                                                </div>
                                                <div className="grid grid-cols-2 gap-2 text-xs text-carbon-600 dark:text-carbon-400">
                                                    <div>
                                                        <span className="font-medium">Started:</span>
                                                        <p className="font-mono mt-0.5">{new Date(timing.start_time).toLocaleTimeString()}</p>
                                                    </div>
                                                    <div>
                                                        <span className="font-medium">Ended:</span>
                                                        <p className="font-mono mt-0.5">{new Date(timing.end_time).toLocaleTimeString()}</p>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}

                        {/* Parameters Display */}
                        {executionData.actual_parameters && (
                            <div className="bg-carbon-100/50 dark:bg-carbon-800/50 backdrop-blur-md border border-carbon-200/50 dark:border-carbon-700/50 rounded-lg p-4 sm:p-6 shadow-sm">
                                <h3 className="text-lg font-semibold text-carbon-900 dark:text-carbon-100 mb-3 sm:mb-4">
                                    Execution Parameters
                                </h3>
                                <pre className="text-xs sm:text-sm font-mono bg-white/60 dark:bg-carbon-950/40 backdrop-blur-sm p-4 rounded-md border border-carbon-200/50 dark:border-carbon-700/50 overflow-x-auto text-carbon-800 dark:text-carbon-200">
                                    {JSON.stringify(executionData.actual_parameters, null, 2)}
                                </pre>
                            </div>
                        )}
                    </div>
                ) : (
                    <div className="py-8 text-center text-carbon-500">
                        No execution data available.
                    </div>
                )}
            </DialogContent>
        </Dialog>
    );
}

function UsersIcon() {
    return (
        <svg className="h-5 w-5 text-carbon-600 dark:text-carbon-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
        </svg>
    )
}
