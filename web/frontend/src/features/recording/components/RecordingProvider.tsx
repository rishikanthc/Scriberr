import { createContext, useCallback, useContext, useMemo, useState, type PropsWithChildren } from "react";
import { RecordingDialog } from "@/features/recording/components/RecordingDialog";
import { RecordingSidebarItem } from "@/features/recording/components/RecordingSidebarItem";
import { useBrowserRecorder, type BrowserRecorderState } from "@/features/recording/hooks/useBrowserRecorder";

type RecordingContextValue = {
  openDialog: () => void;
  closeDialog: () => void;
  dialogOpen: boolean;
};

const RecordingContext = createContext<RecordingContextValue | null>(null);

const minimizedStatuses: BrowserRecorderState["status"][] = ["recording", "paused", "stopping", "finalizing", "failed"];

export function RecordingProvider({ children }: PropsWithChildren) {
  const recorder = useBrowserRecorder();
  const [dialogOpen, setDialogOpen] = useState(false);
  const minimized = !dialogOpen && minimizedStatuses.includes(recorder.state.status);

  const openDialog = useCallback(() => {
    setDialogOpen(true);
  }, []);

  const closeDialog = useCallback(() => {
    setDialogOpen(false);
  }, []);

  const value = useMemo(() => ({
    openDialog,
    closeDialog,
    dialogOpen,
  }), [closeDialog, dialogOpen, openDialog]);

  return (
    <RecordingContext.Provider value={value}>
      {children}
      <RecordingDialog open={dialogOpen} onOpenChange={setDialogOpen} recorder={recorder} />
      {minimized ? <RecordingSidebarItem state={recorder.state} onOpen={openDialog} /> : null}
    </RecordingContext.Provider>
  );
}

export function useRecordingController() {
  const context = useContext(RecordingContext);
  if (!context) {
    throw new Error("useRecordingController must be used within RecordingProvider");
  }
  return context;
}
