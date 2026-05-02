import { useEffect, useState } from "react";
import { X } from "lucide-react";
import { IconButton, AppButton } from "@/shared/ui/Button";
import type { AudioTag, SaveTagPayload } from "@/features/tags/api/tagsApi";
import { hasDuplicateTagName } from "@/features/tags/api/tagsApi";

type TagDialogProps = {
  open: boolean;
  tag: AudioTag | null;
  tags: AudioTag[];
  saving: boolean;
  onClose: () => void;
  onSave: (payload: SaveTagPayload) => Promise<void>;
};

export function TagDialog({ open, tag, tags, saving, onClose, onSave }: TagDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [whenToUse, setWhenToUse] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) return;
    setName(tag?.name || "");
    setDescription(tag?.description || "");
    setWhenToUse(tag?.when_to_use || "");
    setError("");
  }, [open, tag]);

  if (!open) return null;

  const submit = async () => {
    const cleanName = name.trim();
    if (!cleanName) {
      setError("Tag name is required.");
      return;
    }
    if (hasDuplicateTagName(tags, cleanName, tag?.id)) {
      setError("A tag with this name already exists.");
      return;
    }
    setError("");
    try {
      await onSave({
        id: tag?.id,
        name: cleanName,
        description: description.trim() || null,
        when_to_use: whenToUse.trim() || null,
      });
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save tag.");
    }
  };

  return (
    <div className="scr-modal-backdrop" role="presentation">
      <section className="scr-profile-modal scr-tag-modal" role="dialog" aria-modal="true" aria-labelledby="tag-dialog-title">
        <header className="scr-modal-header">
          <div>
            <h2 id="tag-dialog-title" className="scr-modal-title">{tag ? "Edit tag" : "New tag"}</h2>
            <p className="scr-modal-copy">Create a reusable label for organizing audio.</p>
          </div>
          <IconButton label="Close tag dialog" onClick={onClose}>
            <X size={18} aria-hidden="true" />
          </IconButton>
        </header>

        <div className="scr-modal-body">
          {error ? <div className="scr-alert">{error}</div> : null}
          <label className="scr-control">
            <span>Name</span>
            <input className="scr-input" value={name} placeholder="Client call" onChange={(event) => setName(event.target.value)} />
          </label>
          <label className="scr-control">
            <span>Description</span>
            <textarea className="scr-textarea" value={description} placeholder="Calls with customers and prospects" onChange={(event) => setDescription(event.target.value)} />
          </label>
          <label className="scr-control">
            <span>When to use</span>
            <textarea className="scr-textarea" value={whenToUse} placeholder="Use when the conversation includes customer commitments." onChange={(event) => setWhenToUse(event.target.value)} />
          </label>
        </div>

        <footer className="scr-modal-footer">
          <AppButton variant="secondary" onClick={onClose}>Cancel</AppButton>
          <AppButton onClick={submit} disabled={saving || !name.trim()}>
            {saving ? "Saving..." : "Save tag"}
          </AppButton>
        </footer>
      </section>
    </div>
  );
}
