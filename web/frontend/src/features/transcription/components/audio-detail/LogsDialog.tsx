import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { FileText } from "lucide-react";
import { useLogs } from "@/features/transcription/hooks/useAudioDetail";

interface LogsDialogProps {
    audioId: string;
    isOpen: boolean;
    onClose: (open: boolean) => void;
}

export function LogsDialog({ audioId, isOpen, onClose }: LogsDialogProps) {
    const { data: logsContent, isLoading } = useLogs(audioId);

    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-4xl w-[95vw] bg-white dark:bg-carbon-900 border-carbon-200 dark:border-carbon-800 max-h-[90vh] overflow-y-auto">
                <DialogHeader>
                    <DialogTitle className="text-carbon-900 dark:text-carbon-100 flex items-center gap-2">
                        <FileText className="h-5 w-5 text-carbon-600 dark:text-carbon-400" />
                        Transcription Logs
                    </DialogTitle>
                    <DialogDescription className="text-carbon-600 dark:text-carbon-400">
                        Raw output logs from the transcription process
                    </DialogDescription>
                </DialogHeader>

                <div className="mt-4">
                    {isLoading ? (
                        <div className="h-64 flex items-center justify-center">
                            <div className="animate-pulse flex flex-col items-center">
                                <div className="h-4 bg-carbon-200 dark:bg-carbon-800 rounded w-48 mb-2"></div>
                                <div className="h-4 bg-carbon-200 dark:bg-carbon-800 rounded w-32"></div>
                            </div>
                        </div>
                    ) : (
                        <pre className="bg-carbon-950 text-carbon-50 p-4 rounded-lg overflow-x-auto text-xs sm:text-sm font-mono leading-relaxed whitespace-pre-wrap max-h-[60vh] overflow-y-auto border border-carbon-800">
                            {logsContent || "No logs available."}
                        </pre>
                    )}
                </div>
            </DialogContent>
        </Dialog>
    );
}
