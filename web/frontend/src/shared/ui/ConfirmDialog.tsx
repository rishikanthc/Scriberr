import { AlertTriangle, X } from "lucide-react";
import { AppButton, IconButton } from "./Button";

type ConfirmDialogProps = {
  open: boolean;
  title: string;
  description: string;
  confirmLabel: string;
  busy?: boolean;
  onCancel: () => void;
  onConfirm: () => void;
};

export function ConfirmDialog({ open, title, description, confirmLabel, busy, onCancel, onConfirm }: ConfirmDialogProps) {
  if (!open) return null;

  return (
    <div className="scr-modal-backdrop" role="presentation">
      <section className="scr-confirm-modal" role="alertdialog" aria-modal="true" aria-labelledby="scr-confirm-title" aria-describedby="scr-confirm-copy">
        <header className="scr-confirm-header">
          <span className="scr-confirm-mark" aria-hidden="true">
            <AlertTriangle size={18} />
          </span>
          <IconButton label="Close confirmation" onClick={onCancel}>
            <X size={17} aria-hidden="true" />
          </IconButton>
        </header>
        <div className="scr-confirm-body">
          <h2 id="scr-confirm-title" className="scr-confirm-title">{title}</h2>
          <p id="scr-confirm-copy" className="scr-confirm-copy">{description}</p>
        </div>
        <footer className="scr-confirm-footer">
          <AppButton variant="secondary" onClick={onCancel}>Cancel</AppButton>
          <AppButton className="scr-button-danger" onClick={onConfirm} disabled={busy}>
            {busy ? "Deleting..." : confirmLabel}
          </AppButton>
        </footer>
      </section>
    </div>
  );
}
