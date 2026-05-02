import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Circle, Loader2, Pause, Play, RefreshCw, RotateCcw, Square, X } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { AppButton } from "@/shared/ui/Button";
import { RecordingPulseField } from "@/features/recording/components/RecordingPulseField";
import { useBrowserRecorder, type BrowserRecorderStatus } from "@/features/recording/hooks/useBrowserRecorder";

type RecordingDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  recorder: ReturnType<typeof useBrowserRecorder>;
};

const activeStatuses: BrowserRecorderStatus[] = ["recording", "paused", "stopping", "finalizing", "failed"];
const defaultDeviceValue = "__default_microphone__";

export function RecordingDialog({ open, onOpenChange, recorder }: RecordingDialogProps) {
  const [title, setTitle] = useState(() => defaultRecordingTitle());
  const permissionPromptedForOpenRef = useRef(false);
  const { state } = recorder;
  const { getAudioStream, requestMicrophonePermission } = recorder;
  const active = activeStatuses.includes(state.status);
  const canEditTitle = state.status === "idle" || state.status === "unsupported" || state.status === "permission-denied" || state.status === "permission-ready" || state.status === "canceled" || state.status === "ready";
  const canStart = state.status === "idle" || state.status === "permission-denied" || state.status === "permission-ready" || state.status === "canceled" || state.status === "ready";
  const canPause = state.status === "recording";
  const canResume = state.status === "paused";
  const canStop = state.status === "recording" || state.status === "paused" || state.status === "failed";
  const stopping = state.status === "stopping" || state.status === "finalizing";
  const selectableDevices = state.availableDevices.filter((device) => device.deviceId);
  const canChangeDevice = !active && !state.devicesLoading;
  const selectedDeviceValue = state.selectedDeviceId || defaultDeviceValue;
  const audioStream = getAudioStream();

  const statusLabel = useMemo(() => recorderStatusLabel(state.status), [state.status]);

  useEffect(() => {
    if (!open) {
      permissionPromptedForOpenRef.current = false;
      return;
    }
    if (permissionPromptedForOpenRef.current) return;
    if (state.status !== "idle" && state.status !== "permission-denied") return;
    permissionPromptedForOpenRef.current = true;
    void requestMicrophonePermission();
  }, [open, requestMicrophonePermission, state.status]);

  const handleOpenChange = useCallback((nextOpen: boolean) => {
    if (nextOpen) {
      if (!state.session && (state.status === "idle" || state.status === "ready" || state.status === "canceled")) {
        setTitle(defaultRecordingTitle());
      }
      onOpenChange(true);
      return;
    }

    if (active) {
      onOpenChange(false);
      return;
    }

    if (state.status === "ready" || state.status === "canceled") {
      recorder.reset();
      setTitle(defaultRecordingTitle());
    }
    onOpenChange(false);
  }, [active, onOpenChange, recorder, state.session, state.status]);

  const handleStart = useCallback(() => {
    const trimmedTitle = title.trim() || defaultRecordingTitle();
    setTitle(trimmedTitle);
    void recorder.start({
      title: trimmedTitle,
      deviceId: state.selectedDeviceId || undefined,
    });
  }, [recorder, state.selectedDeviceId, title]);

  const handleDeviceChange = useCallback((deviceId: string) => {
    recorder.setSelectedDeviceId(deviceId === defaultDeviceValue ? "" : deviceId);
  }, [recorder]);

  const handleStop = useCallback(() => {
    void recorder.stop();
  }, [recorder]);

  const handleCancel = useCallback(() => {
    void recorder.cancel();
  }, [recorder]);

  const handleDone = useCallback(() => {
    recorder.reset();
    setTitle(defaultRecordingTitle());
    onOpenChange(false);
  }, [onOpenChange, recorder]);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        className="w-[min(100%,520px)] gap-0 overflow-hidden border border-[var(--scr-border-strong)] bg-[var(--scr-surface-raised)] p-0 text-[var(--scr-text-primary)] shadow-[var(--scr-shadow-float)]"
        onEscapeKeyDown={(event) => {
          if (active) {
            event.preventDefault();
            onOpenChange(false);
          }
        }}
        onPointerDownOutside={(event) => {
          if (active) {
            event.preventDefault();
            onOpenChange(false);
          }
        }}
      >
        <DialogHeader className="border-b border-[var(--scr-border-subtle)] px-5 py-4 text-left">
          <DialogTitle className="text-base font-semibold text-[var(--scr-text-strong)]">
            Record audio
          </DialogTitle>
          <DialogDescription className="text-xs text-[var(--scr-text-secondary)]">
            Microphone input
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-5 px-5 py-5">
          <div className="grid gap-2">
            <Label htmlFor="recording-title" className="text-xs font-medium text-[var(--scr-text-secondary)]">
              Title
            </Label>
            <Input
              id="recording-title"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              disabled={!canEditTitle}
              className="h-10 rounded-[var(--scr-radius-sm)] border-[var(--scr-border-subtle)] bg-[var(--scr-surface-panel)] text-sm"
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="recording-input-device" className="text-xs font-medium text-[var(--scr-text-secondary)]">
              Input device
            </Label>
            <div className="flex gap-2">
              <Select
                value={selectedDeviceValue}
                onValueChange={handleDeviceChange}
                disabled={!canChangeDevice}
              >
                <SelectTrigger
                  id="recording-input-device"
                  className="h-10 min-w-0 flex-1 rounded-[var(--scr-radius-sm)] border-[var(--scr-border-subtle)] bg-[var(--scr-surface-panel)] text-sm"
                >
                  <SelectValue placeholder={state.devicesLoading ? "Loading microphones" : "Default microphone"} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={defaultDeviceValue}>Default microphone</SelectItem>
                  {selectableDevices.map((device) => (
                    <SelectItem key={device.deviceId} value={device.deviceId}>
                      {device.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <AppButton
                type="button"
                variant="secondary"
                onClick={() => void recorder.refreshInputDevices()}
                disabled={active || state.devicesLoading}
                aria-label="Refresh input devices"
              >
                {state.devicesLoading ? <Loader2 className="animate-spin" size={14} aria-hidden="true" /> : <RefreshCw size={14} aria-hidden="true" />}
                Refresh
              </AppButton>
            </div>
            {state.devicesError ? (
              <p className="text-xs text-[var(--error)]">{state.devicesError}</p>
            ) : null}
          </div>

          <div className="scr-recorder-visual-shell">
            <RecordingPulseField
              stream={audioStream}
              active={state.status === "recording"}
              paused={state.status === "paused"}
            />
            <div className="scr-recorder-visual-meta">
              <div className="scr-recorder-status-copy">
                <span className="scr-recorder-status-dot" data-status={state.status}>
                  <Circle className={state.status === "recording" ? "h-2.5 w-2.5 fill-current" : "h-2.5 w-2.5"} aria-hidden="true" />
                </span>
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium text-[var(--scr-text-strong)]">{statusLabel}</p>
                  <p className="truncate text-xs text-[var(--scr-text-secondary)]">
                    {state.selectedMimeType || "Waiting for microphone"}
                  </p>
                </div>
              </div>
              <time className="scr-recorder-compact-timer">
                {formatDuration(state.elapsedMs)}
              </time>
            </div>
          </div>

          {state.error ? (
            <div className="rounded-[var(--scr-radius-sm)] border border-[color-mix(in_srgb,var(--error)_28%,transparent)] bg-[color-mix(in_srgb,var(--error)_8%,transparent)] px-3 py-2 text-sm text-[var(--error)]">
              {state.error}
            </div>
          ) : null}

          {state.appliedSettings ? (
            <div className="grid grid-cols-2 gap-2 text-xs text-[var(--scr-text-secondary)] sm:grid-cols-4">
              <span>Echo {settingLabel(state.appliedSettings.echoCancellation)}</span>
              <span>Noise {settingLabel(state.appliedSettings.noiseSuppression)}</span>
              <span>AGC {settingLabel(state.appliedSettings.autoGainControl)}</span>
              <span>{state.appliedSettings.sampleRate ? `${state.appliedSettings.sampleRate} Hz` : "Auto rate"}</span>
            </div>
          ) : null}
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-[var(--scr-border-subtle)] px-5 py-4">
          <AppButton
            variant="secondary"
            onClick={handleCancel}
            disabled={!active || stopping}
            aria-label="Cancel recording"
          >
            <X size={14} aria-hidden="true" />
            Cancel
          </AppButton>

          <div className="flex flex-wrap items-center justify-end gap-2">
            {state.status === "failed" && state.failedChunkIndex !== null ? (
              <AppButton variant="secondary" onClick={recorder.retryPendingChunk}>
                <RotateCcw size={14} aria-hidden="true" />
                Retry
              </AppButton>
            ) : null}

            {canStart ? (
              <AppButton onClick={handleStart}>
                <Play size={14} aria-hidden="true" />
                Start
              </AppButton>
            ) : null}

            {canPause ? (
              <AppButton variant="secondary" onClick={recorder.pause}>
                <Pause size={14} aria-hidden="true" />
                Pause
              </AppButton>
            ) : null}

            {canResume ? (
              <AppButton onClick={recorder.resume}>
                <Play size={14} aria-hidden="true" />
                Resume
              </AppButton>
            ) : null}

            {canStop ? (
              <AppButton onClick={handleStop} disabled={stopping}>
                {stopping ? <Loader2 className="animate-spin" size={14} aria-hidden="true" /> : <Square size={14} aria-hidden="true" />}
                Stop
              </AppButton>
            ) : null}

            {state.status === "ready" || state.status === "canceled" ? (
              <AppButton onClick={handleDone}>Done</AppButton>
            ) : null}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function defaultRecordingTitle() {
  const now = new Date();
  const date = [
    now.getFullYear(),
    String(now.getMonth() + 1).padStart(2, "0"),
    String(now.getDate()).padStart(2, "0"),
  ].join("");
  const time = [
    String(now.getHours()).padStart(2, "0"),
    String(now.getMinutes()).padStart(2, "0"),
    String(now.getSeconds()).padStart(2, "0"),
  ].join("");
  return `recording-${date}-${time}`;
}

function formatDuration(ms: number) {
  const totalSeconds = Math.floor(ms / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
  }
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

function recorderStatusLabel(status: BrowserRecorderStatus) {
  switch (status) {
    case "idle":
      return "Ready to record";
    case "unsupported":
      return "Recording unavailable";
    case "permission-denied":
      return "Microphone permission needed";
    case "permission-ready":
      return "Microphone ready";
    case "recording":
      return "Recording";
    case "paused":
      return "Paused";
    case "stopping":
      return "Saving chunks";
    case "finalizing":
      return "Finalizing audio";
    case "ready":
      return "Recording saved";
    case "failed":
      return "Recording needs attention";
    case "canceled":
      return "Recording canceled";
  }
}

function settingLabel(value: unknown) {
  if (value === true) return "on";
  if (value === false) return "off";
  return "auto";
}
