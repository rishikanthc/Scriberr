import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Sparkles, Download, Check } from "lucide-react";
import { useState } from "react";
import { useTranscript, useAudioDetail } from "@/features/transcription/hooks/useAudioDetail";
import { useSpeakerMappings } from "@/features/transcription/hooks/useTranscriptionSpeakers";
import { useTranscriptDownload } from "@/features/transcription/hooks/useTranscriptDownload";

interface DownloadDialogProps {
    audioId: string;
    isOpen: boolean;
    onClose: (open: boolean) => void;
    initialFormat?: 'txt' | 'json';
}

export function DownloadDialog({ audioId, isOpen, onClose, initialFormat = 'txt' }: DownloadDialogProps) {
    const { data: transcript } = useTranscript(audioId, true);
    const { data: audioFile } = useAudioDetail(audioId);
    const { data: speakerMappings = {} } = useSpeakerMappings(audioId, true);
    const { downloadTXT, downloadJSON } = useTranscriptDownload();

    const [includeSpeakerLabels, setIncludeSpeakerLabels] = useState(true);
    const [includeTimestamps, setIncludeTimestamps] = useState(true);

    const handleDownloadConfirm = () => {
        if (!transcript || !audioFile) return;

        // Helper to get filename base
        const getFileNameWithoutExt = () => {
            const name = audioFile.title || audioFile.audio_path.split("/").pop() || "transcript";
            return name.replace(/\.[^/.]+$/, '');
        };
        const filenameBase = getFileNameWithoutExt();

        if (initialFormat === 'txt') {
            downloadTXT(transcript, filenameBase, speakerMappings, { includeSpeakerLabels, includeTimestamps });
        } else {
            downloadJSON(transcript, filenameBase, speakerMappings, { includeSpeakerLabels, includeTimestamps });
        }
        onClose(false);
    };

    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-md bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2 text-xl">
                        <Sparkles className="h-5 w-5 text-primary" />
                        Download Transcript
                    </DialogTitle>
                    <DialogDescription>
                        Configure download options for {initialFormat.toUpperCase()} format.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    <div className="flex items-center justify-between">
                        <Label htmlFor="speaker-labels" className="text-carbon-700 dark:text-carbon-300">
                            Include Speaker Labels
                        </Label>
                        <Switch
                            id="speaker-labels"
                            checked={includeSpeakerLabels}
                            onCheckedChange={setIncludeSpeakerLabels}
                            disabled={!transcript?.segments?.some(s => s.speaker)}
                        />
                    </div>

                    <div className="flex items-center justify-between">
                        <Label htmlFor="timestamps" className="text-carbon-700 dark:text-carbon-300">
                            Include Timestamps
                        </Label>
                        <Switch
                            id="timestamps"
                            checked={includeTimestamps}
                            onCheckedChange={setIncludeTimestamps}
                            disabled={!transcript?.segments}
                        />
                    </div>

                    {(!includeSpeakerLabels && !includeTimestamps) && (
                        <div className="text-sm text-carbon-500 dark:text-carbon-400 bg-carbon-50 dark:bg-carbon-800 p-3 rounded-md">
                            <div className="flex items-center gap-2">
                                <Check className="h-4 w-4 text-carbon-900 dark:text-carbon-100" />
                                Transcript will be formatted as a single paragraph
                            </div>
                        </div>
                    )}

                    {(includeSpeakerLabels || includeTimestamps) && (
                        <div className="text-sm text-carbon-500 dark:text-carbon-400 bg-carbon-50 dark:bg-carbon-800 p-3 rounded-md">
                            <div className="flex items-center gap-2">
                                <Check className="h-4 w-4 text-carbon-900 dark:text-carbon-100" />
                                Transcript will be formatted in segments with selected labels
                            </div>
                        </div>
                    )}
                </div>

                <DialogFooter className="gap-2">
                    <Button
                        variant="outline"
                        onClick={() => onClose(false)}
                        className="bg-white dark:bg-carbon-800 border-carbon-300 dark:border-carbon-600 text-carbon-700 dark:text-carbon-200 hover:bg-carbon-50 dark:hover:bg-carbon-700"
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleDownloadConfirm}
                        className="bg-carbon-900 dark:bg-carbon-700 hover:bg-carbon-950 dark:hover:bg-carbon-600 text-white"
                    >
                        <Download className="mr-2 h-4 w-4" />
                        Download {initialFormat.toUpperCase()}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
