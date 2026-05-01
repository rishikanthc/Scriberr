import { Check, Loader2, X } from "lucide-react";
import type { UploadItem } from "@/features/files/hooks/useFileImport";

type UploadProgressShelfProps = {
  items: UploadItem[];
  onDismiss: (id: string) => void;
};

export function UploadProgressShelf({ items, onDismiss }: UploadProgressShelfProps) {
  if (items.length === 0) return null;

  const active = items.filter((item) => item.status === "uploading" || item.status === "processing").length;
  const completed = items.filter((item) => item.status === "ready").length;
  const failed = items.filter((item) => item.status === "failed").length;
  const averageProgress = Math.round(items.reduce((sum, item) => sum + item.progress, 0) / items.length);
  const primary = items[0];

  return (
    <aside className="scr-upload-shelf" aria-live="polite">
      <div className="scr-upload-summary">
        <div className="scr-upload-status-icon" data-status={primary.status}>
          {primary.status === "ready" ? <Check size={14} aria-hidden="true" /> : <Loader2 size={14} aria-hidden="true" />}
        </div>
        <div className="scr-upload-copy">
          <p className="scr-upload-title">{uploadTitle(primary, active, completed, failed, items.length)}</p>
          <p className="scr-upload-meta">{primary.fileName}</p>
        </div>
      </div>
      <div className="scr-upload-bar" aria-hidden="true">
        <div className="scr-upload-bar-fill" style={{ width: `${averageProgress}%` }} />
      </div>
      <div className="scr-upload-items">
        {items.slice(0, 4).map((item) => (
          <div className="scr-upload-item" key={item.id}>
            <span className="scr-upload-item-name">{item.fileName}</span>
            <span className="scr-upload-item-state">{itemLabel(item)}</span>
            {(item.status === "ready" || item.status === "failed") && (
              <button type="button" className="scr-upload-dismiss" aria-label={`Dismiss ${item.fileName}`} onClick={() => onDismiss(item.id)}>
                <X size={12} aria-hidden="true" />
              </button>
            )}
          </div>
        ))}
      </div>
    </aside>
  );
}

function uploadTitle(primary: UploadItem, active: number, completed: number, failed: number, total: number) {
  if (active > 0 && primary.source === "youtube") return "Importing from YouTube";
  if (active > 0) return total === 1 ? "Importing file" : `Importing ${active} of ${total}`;
  if (failed > 0 && completed === 0) return total === 1 ? "Import failed" : `${failed} imports failed`;
  if (failed > 0) return `${completed} imported, ${failed} failed`;
  return total === 1 ? "Import complete" : `${completed} files imported`;
}

function itemLabel(item: UploadItem) {
  switch (item.status) {
    case "uploading":
      return `${item.progress}%`;
    case "processing":
      return item.source === "youtube" ? "Preparing audio" : "Extracting";
    case "ready":
      return "Ready";
    case "failed":
      return "Failed";
  }
}
