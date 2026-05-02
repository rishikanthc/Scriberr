import { useCallback, useEffect, useMemo, useState, type PropsWithChildren } from "react";
import { RecordingDialog } from "@/features/recording/components/RecordingDialog";
import { RecordingSidebarItem } from "@/features/recording/components/RecordingSidebarItem";
import { useBrowserRecorder, type BrowserRecorderState } from "@/features/recording/hooks/useBrowserRecorder";
import { RecordingContext, type OptimisticRecordingSummary } from "@/features/recording/hooks/useRecordingController";
import { useRecordingEvents, type RecordingEvent } from "@/features/recording/hooks/useRecordingEvents";

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
  const optimisticRecording = useMemo(() => optimisticRecordingFromParts({
    eventOverlay,
    fileId: recorder.state.session?.file_id ?? null,
    id: recorder.state.session?.id,
    progress: recorder.state.session?.progress,
    status: recorder.state.status,
    title: recorder.state.session?.title,
  }), [
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

type OptimisticRecordingParts = {
  eventOverlay: RecordingEventOverlay;
  fileId: string | null;
  id?: string;
  progress?: number;
  status: BrowserRecorderState["status"];
  title?: string;
};

function optimisticRecordingFromParts(parts: OptimisticRecordingParts): OptimisticRecordingSummary | null {
  if (!parts.id) return null;
  const title = parts.title || "Recording";
  const fileId = parts.eventOverlay.fileId ?? parts.fileId;
  const progress = parts.eventOverlay.progress ?? parts.progress;

  switch (parts.status) {
    case "recording":
    case "paused":
      return {
        id: parts.id,
        title,
        status: parts.eventOverlay.status || parts.status,
        fileId,
      };
    case "stopping":
    case "finalizing":
      return {
        id: parts.id,
        title,
        status: parts.eventOverlay.status || "finalizing",
        fileId,
        progress,
      };
    case "failed":
      return {
        id: parts.id,
        title,
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
