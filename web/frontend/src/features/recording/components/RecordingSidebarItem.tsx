import { AlertCircle, Loader2, Mic, Pause, Radio } from "lucide-react";
import { cn } from "@/lib/utils";
import type { MinimizedRecordingSummary } from "@/features/recording/hooks/useRecordingController";

type RecordingSidebarItemProps = {
  recording: MinimizedRecordingSummary;
  onOpen: () => void;
};

export function RecordingSidebarItem({ recording, onOpen }: RecordingSidebarItemProps) {
  return (
    <button
      type="button"
      className="scr-nav-recording"
      onClick={onOpen}
      aria-label={`Open recorder, ${recordingStatusLabel(recording.status)}, ${formatDuration(recording.elapsedMs)}`}
      title={`${recording.title} - ${formatDuration(recording.elapsedMs)}`}
    >
      <span
        className={cn(
          "scr-nav-recording-icon",
          recording.status === "failed" && "scr-nav-recording-icon-error"
        )}
      >
        {recordingIcon(recording.status)}
      </span>
      <span className="scr-nav-recording-copy">
        <span className="scr-nav-recording-title">{recording.title}</span>
        <span className="scr-nav-recording-meta">
          <span>{recordingStatusLabel(recording.status)}</span>
          <span>{formatDuration(recording.elapsedMs)}</span>
        </span>
      </span>
    </button>
  );
}

function recordingIcon(status: MinimizedRecordingSummary["status"]) {
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

function recordingStatusLabel(status: MinimizedRecordingSummary["status"]) {
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
