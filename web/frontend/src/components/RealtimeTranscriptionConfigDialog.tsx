import { useState } from "react";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Zap, Cpu, CircuitBoard } from "lucide-react";
import { useRouter } from "../contexts/RouterContext";

interface RealtimeTranscriptionConfigDialogProps {
    isOpen: boolean;
    onClose: () => void;
}

export function RealtimeTranscriptionConfigDialog({
    isOpen,
    onClose,
}: RealtimeTranscriptionConfigDialogProps) {
    const { navigate } = useRouter();
    const [model, setModel] = useState("base");
    const [device, setDevice] = useState("cpu");

    const handleStart = () => {
        navigate({
            path: "realtime-transcription",
            params: { model, device },
        });
        onClose();
    };

    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-[425px] bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2 text-carbon-900 dark:text-carbon-100">
                        <Zap className="h-5 w-5 text-amber-500" />
                        Real-time Transcription
                    </DialogTitle>
                    <DialogDescription className="text-carbon-600 dark:text-carbon-400">
                        Configure your session. Ensure the local transcription server is running.
                    </DialogDescription>
                </DialogHeader>

                <div className="grid gap-6 py-4">
                    {/* Model Selection */}
                    <div className="space-y-2">
                        <Label className="text-carbon-700 dark:text-carbon-300">Model Size</Label>
                        <Select value={model} onValueChange={setModel}>
                            <SelectTrigger className="bg-white dark:bg-carbon-900 border-carbon-300 dark:border-carbon-600">
                                <SelectValue placeholder="Select model" />
                            </SelectTrigger>
                            <SelectContent className="bg-white dark:bg-carbon-900 border-carbon-200 dark:border-carbon-700">
                                <SelectItem value="tiny">Tiny (Fastest, Lower Accuracy)</SelectItem>
                                <SelectItem value="base">Base (Balanced)</SelectItem>
                                <SelectItem value="small">Small (Good Accuracy)</SelectItem>
                                <SelectItem value="medium">Medium (Better Accuracy, Slower)</SelectItem>
                                <SelectItem value="large-v3">Large v3 (Best Accuracy, Slowest)</SelectItem>
                            </SelectContent>
                        </Select>
                        <p className="text-xs text-carbon-500 dark:text-carbon-400">
                            Larger models require more RAM/VRAM and may be slower.
                        </p>
                    </div>

                    {/* Device Selection */}
                    <div className="space-y-2">
                        <Label className="text-carbon-700 dark:text-carbon-300">Compute Device</Label>
                        <div className="grid grid-cols-2 gap-4">
                            <div
                                onClick={() => setDevice("cpu")}
                                className={`cursor-pointer rounded-lg border-2 p-4 flex flex-col items-center gap-2 transition-all ${device === "cpu"
                                        ? "border-blue-500 bg-blue-50 dark:bg-blue-900/20"
                                        : "border-carbon-200 dark:border-carbon-700 hover:border-carbon-300 dark:hover:border-carbon-600"
                                    }`}
                            >
                                <Cpu className={`h-6 w-6 ${device === "cpu" ? "text-blue-500" : "text-carbon-500"}`} />
                                <span className={`text-sm font-medium ${device === "cpu" ? "text-blue-700 dark:text-blue-300" : "text-carbon-600 dark:text-carbon-400"}`}>
                                    CPU
                                </span>
                            </div>
                            <div
                                onClick={() => setDevice("cuda")}
                                className={`cursor-pointer rounded-lg border-2 p-4 flex flex-col items-center gap-2 transition-all ${device === "cuda"
                                        ? "border-green-500 bg-green-50 dark:bg-green-900/20"
                                        : "border-carbon-200 dark:border-carbon-700 hover:border-carbon-300 dark:hover:border-carbon-600"
                                    }`}
                            >
                                <CircuitBoard className={`h-6 w-6 ${device === "cuda" ? "text-green-500" : "text-carbon-500"}`} />
                                <span className={`text-sm font-medium ${device === "cuda" ? "text-green-700 dark:text-green-300" : "text-carbon-600 dark:text-carbon-400"}`}>
                                    NVIDIA GPU
                                </span>
                            </div>
                        </div>
                    </div>
                </div>

                <DialogFooter>
                    <Button variant="outline" onClick={onClose} className="border-carbon-300 dark:border-carbon-600">
                        Cancel
                    </Button>
                    <Button onClick={handleStart} className="bg-carbon-900 dark:bg-carbon-100 text-white dark:text-carbon-900 hover:bg-carbon-800 dark:hover:bg-carbon-200">
                        Start Session
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
