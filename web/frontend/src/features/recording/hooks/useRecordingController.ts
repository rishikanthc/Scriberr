import { createContext, useContext } from "react";

export type OptimisticRecordingSummary = {
  id: string;
  title: string;
  status: "recording" | "paused" | "finalizing" | "failed";
  fileId: string | null;
  progress?: number;
};

export type MinimizedRecordingSummary = {
  title: string;
  status: "recording" | "paused" | "stopping" | "finalizing" | "failed";
  elapsedMs: number;
};

export type RecordingContextValue = {
  openDialog: () => void;
  closeDialog: () => void;
  dialogOpen: boolean;
  optimisticRecording: OptimisticRecordingSummary | null;
  minimizedRecording: MinimizedRecordingSummary | null;
};

export const RecordingContext = createContext<RecordingContextValue | null>(null);

export function useRecordingController() {
  const context = useContext(RecordingContext);
  if (!context) {
    throw new Error("useRecordingController must be used within RecordingProvider");
  }
  return context;
}
