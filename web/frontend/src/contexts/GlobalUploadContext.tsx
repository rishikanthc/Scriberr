import {
    createContext,
    useContext,
    useState,
    useCallback,
    type PropsWithChildren,
} from "react";
import { useLocation } from "react-router-dom";
import { useAudioUpload, useMultiTrackUpload } from "@/features/transcription/hooks/useAudioFiles";
import { useToast } from "@/components/ui/toast";
import { MultiTrackUploadDialog } from "@/features/transcription/components/MultiTrackUploadDialog";

// Types
interface FileWithType {
    file: File;
    isVideo: boolean;
}

interface UploadProgress {
    fileName: string;
    status: "uploading" | "success" | "error";
    error?: string;
}

interface GlobalUploadContextValue {
    // File upload
    handleFileSelect: (
        files: File | File[] | FileWithType | FileWithType[]
    ) => Promise<void>;
    // Multi-track
    handleMultiTrackUpload: (
        files: File[],
        aupFile: File,
        title: string
    ) => Promise<void>;
    openMultiTrackDialog: () => void;
    // Recording completion
    handleRecordingComplete: (blob: Blob, title: string) => Promise<void>;
    // State
    isUploading: boolean;
    uploadProgress: UploadProgress[];
    // For Dashboard to render its own progress bar
    isOnDashboard: boolean;
}

const GlobalUploadContext = createContext<GlobalUploadContextValue | null>(
    null
);

export function GlobalUploadProvider({ children }: PropsWithChildren) {
    const { mutateAsync: uploadFile } = useAudioUpload();
    const { mutateAsync: uploadMultiTrack } = useMultiTrackUpload();
    const { toast } = useToast();
    const location = useLocation();

    // Check if we're on the dashboard (home page)
    const isOnDashboard = location.pathname === "/" || location.pathname === "";

    // Upload state
    const [uploadProgress, setUploadProgress] = useState<UploadProgress[]>([]);
    const [isUploading, setIsUploading] = useState(false);

    // Multi-track dialog state
    const [isMultiTrackDialogOpen, setIsMultiTrackDialogOpen] = useState(false);
    const [multiTrackPreview, setMultiTrackPreview] = useState<{
        audioFiles: File[];
        aupFile: File;
        title: string;
    } | null>(null);

    const handleFileSelect = useCallback(
        async (files: File | File[] | FileWithType | FileWithType[]) => {
            // Normalize input to an array of FileWithType objects
            const fileArray = Array.isArray(files) ? files : [files];
            const processedFiles = fileArray.map((item) => {
                if ("file" in item && "isVideo" in item) {
                    return item;
                } else {
                    return { file: item as File, isVideo: false };
                }
            });

            if (processedFiles.length === 0) return;

            setIsUploading(true);

            // If on dashboard, use progress bar; otherwise use toasts
            if (isOnDashboard) {
                setUploadProgress(
                    processedFiles.map((item) => ({
                        fileName: item.file.name,
                        status: "uploading",
                    }))
                );
            } else {
                toast({
                    title: "Uploading...",
                    description: `Uploading ${processedFiles.length} file(s)`,
                });
            }

            let successCount = 0;

            // Upload files sequentially
            for (let i = 0; i < processedFiles.length; i++) {
                const fileItem = processedFiles[i];
                const file = fileItem.file;
                const isVideo = fileItem.isVideo;

                try {
                    await uploadFile({ file, isVideo });

                    if (isOnDashboard) {
                        setUploadProgress((prev) =>
                            prev.map((item, index) =>
                                index === i
                                    ? { ...item, status: "success", error: undefined }
                                    : item
                            )
                        );
                    }
                    successCount++;
                } catch (error) {
                    if (isOnDashboard) {
                        setUploadProgress((prev) =>
                            prev.map((item, index) =>
                                index === i
                                    ? {
                                        ...item,
                                        status: "error",
                                        error:
                                            error instanceof Error
                                                ? error.message
                                                : "Upload failed",
                                    }
                                    : item
                            )
                        );
                    } else {
                        toast({
                            title: "Upload Failed",
                            description: `Failed to upload ${file.name}`,
                        });
                    }
                }
            }

            setIsUploading(false);

            // Show success toast if not on dashboard
            if (!isOnDashboard && successCount > 0) {
                toast({
                    title: "Upload Complete",
                    description: `Successfully uploaded ${successCount} file(s)`,
                });
            }

            // Auto-hide progress after 3 seconds if all succeeded (for dashboard)
            if (isOnDashboard && successCount === fileArray.length) {
                setTimeout(() => setUploadProgress([]), 3000);
            }
        },
        [isOnDashboard, uploadFile, toast]
    );

    const handleMultiTrackUpload = useCallback(
        async (files: File[], aupFile: File, title: string) => {
            setIsUploading(true);

            if (isOnDashboard) {
                setUploadProgress([
                    {
                        fileName: `${title} (${files.length} tracks)`,
                        status: "uploading",
                    },
                ]);
            } else {
                toast({
                    title: "Uploading Multi-Track...",
                    description: `Uploading ${title} with ${files.length} tracks`,
                });
            }

            try {
                await uploadMultiTrack({ files, aupFile, title });

                if (isOnDashboard) {
                    setUploadProgress([
                        {
                            fileName: `${title} (${files.length} tracks)`,
                            status: "success",
                        },
                    ]);
                    setTimeout(() => setUploadProgress([]), 3000);
                } else {
                    toast({
                        title: "Upload Complete",
                        description: `Successfully uploaded ${title}`,
                    });
                }
            } catch (error) {
                if (isOnDashboard) {
                    setUploadProgress([
                        {
                            fileName: `${title} (${files.length} tracks)`,
                            status: "error",
                            error:
                                error instanceof Error ? error.message : "Upload failed",
                        },
                    ]);
                } else {
                    toast({
                        title: "Upload Failed",
                        description: `Failed to upload ${title}`,
                    });
                }
            } finally {
                setIsUploading(false);
            }
        },
        [isOnDashboard, uploadMultiTrack, toast]
    );

    const openMultiTrackDialog = useCallback(() => {
        setMultiTrackPreview(null);
        setIsMultiTrackDialogOpen(true);
    }, []);

    const handleRecordingComplete = useCallback(
        async (blob: Blob, title: string) => {
            const file = new File([blob], `${title}.webm`, { type: blob.type });
            await handleFileSelect(file);
        },
        [handleFileSelect]
    );

    const handleMultiTrackDialogClose = useCallback(() => {
        setIsMultiTrackDialogOpen(false);
        setMultiTrackPreview(null);
    }, []);

    const handleMultiTrackConfirm = useCallback(
        async (files: File[], aupFile: File, title: string) => {
            await handleMultiTrackUpload(files, aupFile, title);
            handleMultiTrackDialogClose();
        },
        [handleMultiTrackUpload, handleMultiTrackDialogClose]
    );


    const value: GlobalUploadContextValue = {
        handleFileSelect,
        handleMultiTrackUpload,
        openMultiTrackDialog,
        handleRecordingComplete,
        isUploading,
        uploadProgress,
        isOnDashboard,
    };

    return (
        <GlobalUploadContext.Provider value={value}>
            {children}

            {/* Multi-track Upload Dialog (global) */}
            <MultiTrackUploadDialog
                open={isMultiTrackDialogOpen}
                onOpenChange={handleMultiTrackDialogClose}
                onMultiTrackUpload={handleMultiTrackConfirm}
                prePopulatedFiles={multiTrackPreview?.audioFiles}
                prePopulatedAupFile={multiTrackPreview?.aupFile}
                prePopulatedTitle={multiTrackPreview?.title}
            />
        </GlobalUploadContext.Provider>
    );
}

// eslint-disable-next-line react-refresh/only-export-components
export function useGlobalUpload() {
    const ctx = useContext(GlobalUploadContext);
    if (!ctx) {
        throw new Error(
            "useGlobalUpload must be used within GlobalUploadProvider"
        );
    }
    return ctx;
}
