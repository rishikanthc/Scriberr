import { createContext, useCallback, useContext, useEffect, useMemo, useState, type PropsWithChildren } from "react";
import { RecordingDialog } from "@/features/recording/components/RecordingDialog";
import { RecordingSidebarItem } from "@/features/recording/components/RecordingSidebarItem";
import { useBrowserRecorder, type BrowserRecorderState } from "@/features/recording/hooks/useBrowserRecorder";
import { useRecordingEvents, type RecordingEvent } from "@/features/recording/hooks/useRecordingEvents";

export type OptimisticRecordingSummary = {
  id: string;
  title: string;
  status: "recording" | "paused" | "finalizing" | "failed";
  fileId: string | null;
  progress?: number;
};

type RecordingContextValue = {
  openDialog: () => void;
  closeDialog: () => void;
  dialogOpen: boolean;
  optimisticRecording: OptimisticRecordingSummary | null;
};

const RecordingContext = createContext<RecordingContextValue | null>(null);

const minimizedStatuses: BrowserRecorderState["status"][] = ["recording", "paused", "stopping", "finalizing", "failed"];

type RecordingEventOverlay = {
  status?: OptimisticRecordingSummary["status"];
  fileId?: string | null;
  progress?: number;
};

export function RecordingProvider({ children }: PropsWithChildren) {
  const recorder = useBrowserRecorder();
  const [eventOverlay, setEventOverlay] = useState<RecordingEventOverlay>({});
  const sessionId = recorder.state.session?.id;

  const handleRecordingEvent = useCallback((event: RecordingEvent) => {
    if (!sessionId || event.data.id !== sessionId) return;
    setEventOverlay({
      status: optimisticStatusFromEvent(event),
      fileId: event.data.file_id,
      progress: event.data.progress,
    });
  }, [sessionId]);

  useRecordingEvents(handleRecordingEvent);
  const [dialogOpen, setDialogOpen] = useState(false);
  const minimized = !dialogOpen && minimizedStatuses.includes(recorder.state.status);
  const optimisticRecording = useMemo(() => optimisticRecordingFromState(recorder.state, eventOverlay), [
    eventOverlay,
    recorder.state.session?.file_id,
    recorder.state.session?.id,
    recorder.state.session?.progress,
    recorder.state.session?.title,
    recorder.state.status,
  ]);

  const openDialog = useCallback(() => {
    setDialogOpen(true);
  }, []);

  const closeDialog = useCallback(() => {
    setDialogOpen(false);
  }, []);

  useEffect(() => {
    setEventOverlay({});
  }, [sessionId]);

  const value = useMemo(() => ({
    openDialog,
    closeDialog,
    dialogOpen,
    optimisticRecording,
  }), [closeDialog, dialogOpen, openDialog, optimisticRecording]);

  return (
    <RecordingContext.Provider value={value}>
      {children}
      <RecordingDialog open={dialogOpen} onOpenChange={setDialogOpen} recorder={recorder} />
      {minimized ? <RecordingSidebarItem state={recorder.state} onOpen={openDialog} /> : null}
    </RecordingContext.Provider>
  );
}

function optimisticRecordingFromState(state: BrowserRecorderState, eventOverlay: RecordingEventOverlay): OptimisticRecordingSummary | null {
  if (!state.session) return null;
  const fileId = eventOverlay.fileId ?? state.session.file_id;
  const progress = eventOverlay.progress ?? state.session.progress;

  switch (state.status) {
    case "recording":
    case "paused":
      return {
        id: state.session.id,
        title: state.session.title || "Recording",
        status: eventOverlay.status || state.status,
        fileId,
      };
    case "stopping":
    case "finalizing":
      return {
        id: state.session.id,
        title: state.session.title || "Recording",
        status: eventOverlay.status || "finalizing",
        fileId,
        progress,
      };
    case "failed":
      return {
        id: state.session.id,
        title: state.session.title || "Recording",
        status: "failed",
        fileId,
      };
    default:
      return null;
  }
}

function optimisticStatusFromEvent(event: RecordingEvent): OptimisticRecordingSummary["status"] | undefined {
  switch (event.data.status) {
    case "recording":
      return "recording";
    case "stopping":
    case "finalizing":
    case "ready":
      return "finalizing";
    case "failed":
      return "failed";
    default:
      return undefined;
  }
}

export function useRecordingController() {
  const context = useContext(RecordingContext);
  if (!context) {
    throw new Error("useRecordingController must be used within RecordingProvider");
  }
  return context;
}
