import { AlertCircle, Loader2, Mic, Pause, Radio } from "lucide-react";
import { cn } from "@/lib/utils";
import type { BrowserRecorderState } from "@/features/recording/hooks/useBrowserRecorder";

type RecordingSidebarItemProps = {
  state: BrowserRecorderState;
  onOpen: () => void;
};

export function RecordingSidebarItem({ state, onOpen }: RecordingSidebarItemProps) {
  const title = state.session?.title || "Recording";

  return (
    <button
      type="button"
      className="fixed left-3 top-24 z-40 flex w-[172px] items-center gap-2 rounded-[var(--scr-radius-md)] border border-[var(--scr-border-subtle)] bg-[var(--scr-surface-raised)] px-3 py-2 text-left text-[var(--scr-text-primary)] shadow-[var(--scr-shadow-card)] transition hover:border-[var(--scr-brand-border)] hover:shadow-[var(--scr-shadow-float)] focus-visible:border-[var(--scr-brand-border)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--scr-brand-muted)] max-[760px]:left-2 max-[760px]:w-10 max-[760px]:justify-center max-[760px]:px-0"
      onClick={onOpen}
      aria-label={`Open recorder, ${recordingStatusLabel(state.status)}, ${formatDuration(state.elapsedMs)}`}
      title={`${title} - ${formatDuration(state.elapsedMs)}`}
    >
      <span
        className={cn(
          "grid h-7 w-7 shrink-0 place-items-center rounded-full",
          state.status === "failed"
            ? "bg-[color-mix(in_srgb,var(--error)_12%,transparent)] text-[var(--error)]"
            : "bg-[var(--scr-brand-muted)] text-[var(--scr-brand-solid)]"
        )}
      >
        {recordingIcon(state.status)}
      </span>
      <span className="min-w-0 max-[760px]:hidden">
        <span className="block truncate text-xs font-semibold text-[var(--scr-text-strong)]">{title}</span>
        <span className="block truncate font-mono text-xs tabular-nums text-[var(--scr-text-secondary)]">
          {formatDuration(state.elapsedMs)}
        </span>
      </span>
    </button>
  );
}

function recordingIcon(status: BrowserRecorderState["status"]) {
  switch (status) {
    case "paused":
      return <Pause size={14} aria-hidden="true" />;
    case "stopping":
    case "finalizing":
      return <Loader2 className="animate-spin" size={14} aria-hidden="true" />;
    case "failed":
      return <AlertCircle size={14} aria-hidden="true" />;
    case "recording":
      return <Radio size={14} aria-hidden="true" />;
    default:
      return <Mic size={14} aria-hidden="true" />;
  }
}

function recordingStatusLabel(status: BrowserRecorderState["status"]) {
  switch (status) {
    case "recording":
      return "recording";
    case "paused":
      return "paused";
    case "stopping":
      return "saving";
    case "finalizing":
      return "finalizing";
    case "failed":
      return "needs attention";
    default:
      return status;
  }
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
