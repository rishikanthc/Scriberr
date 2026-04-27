import { useEffect, useState } from "react";
import { Check, X } from "lucide-react";
import { AppButton, IconButton } from "@/shared/ui/Button";
import { Select } from "@/shared/ui/Select";
import type { SummaryWidget, SummaryWidgetContextSource, SummaryWidgetPayload } from "@/features/settings/api/summaryWidgetsApi";

type SummaryWidgetDialogProps = {
  open: boolean;
  widget: SummaryWidget | null;
  onClose: () => void;
  onSave: (widget: SummaryWidgetPayload) => Promise<void>;
};

const contextOptions = [
  { value: "summary", label: "Summary", description: "Use the generated overview" },
  { value: "transcript", label: "Transcript", description: "Use transcript text" },
];

export function SummaryWidgetDialog({ open, widget, onClose, onSave }: SummaryWidgetDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [alwaysEnabled, setAlwaysEnabled] = useState(true);
  const [whenToUse, setWhenToUse] = useState("");
  const [contextSource, setContextSource] = useState<SummaryWidgetContextSource>("summary");
  const [prompt, setPrompt] = useState("");
  const [renderMarkdown, setRenderMarkdown] = useState(false);
  const [displayTitle, setDisplayTitle] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) return;
    setName(widget?.name || "");
    setDescription(widget?.description || "");
    setAlwaysEnabled(widget?.always_enabled ?? true);
    setWhenToUse(widget?.when_to_use || "");
    setContextSource(widget?.context_source || "summary");
    setPrompt(widget?.prompt || "");
    setRenderMarkdown(widget?.render_markdown || false);
    setDisplayTitle(widget?.display_title || widget?.name || "");
    setEnabled(widget?.enabled ?? true);
    setError("");
  }, [open, widget]);

  if (!open) return null;

  const submit = async () => {
    const cleanName = name.trim();
    const cleanTitle = displayTitle.trim();
    const cleanPrompt = prompt.trim();
    if (!cleanName) {
      setError("Widget name is required.");
      return;
    }
    if (!cleanTitle) {
      setError("Display title is required.");
      return;
    }
    if (!cleanPrompt) {
      setError("Prompt is required.");
      return;
    }
    if (!alwaysEnabled && !whenToUse.trim()) {
      setError("Describe when this widget should run.");
      return;
    }
    setSaving(true);
    setError("");
    try {
      await onSave({
        id: widget?.id,
        name: cleanName,
        description: description.trim(),
        always_enabled: alwaysEnabled,
        when_to_use: whenToUse.trim(),
        context_source: contextSource,
        prompt: cleanPrompt,
        render_markdown: renderMarkdown,
        display_title: cleanTitle,
        enabled,
      });
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save widget.");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="scr-modal-backdrop" role="presentation">
      <section className="scr-profile-modal scr-profile-modal-compact" role="dialog" aria-modal="true" aria-labelledby="summary-widget-title">
        <header className="scr-modal-header">
          <div>
            <h2 id="summary-widget-title" className="scr-modal-title">{widget ? "Edit widget" : "New widget"}</h2>
            <p className="scr-modal-copy">Define extracted information to show below transcript summaries.</p>
          </div>
          <IconButton label="Close widget dialog" onClick={onClose}>
            <X size={18} aria-hidden="true" />
          </IconButton>
        </header>

        <div className="scr-modal-body">
          {error ? <div className="scr-alert">{error}</div> : null}

          <section className="scr-settings-section">
            <h3 className="scr-settings-section-title">Widget</h3>
            <div className="scr-form-grid">
              <TextField label="Name" value={name} onChange={setName} placeholder="Action Items" />
              <TextField label="Display title" value={displayTitle} onChange={setDisplayTitle} placeholder="Action Items" />
              <TextField className="scr-control-wide" label="Description" value={description} onChange={setDescription} placeholder="Optional note for this widget" />
            </div>
            <CheckRow label="Enabled" checked={enabled} onChange={setEnabled} />
            <CheckRow label="Always enabled" checked={alwaysEnabled} onChange={setAlwaysEnabled} />
            {!alwaysEnabled ? (
              <label className="scr-control">
                <span>When to use</span>
                <textarea
                  className="scr-textarea"
                  value={whenToUse}
                  placeholder="Use this for meetings where people assign follow-up work."
                  onChange={(event) => setWhenToUse(event.target.value)}
                />
              </label>
            ) : null}
          </section>

          <section className="scr-settings-section">
            <h3 className="scr-settings-section-title">Generation</h3>
            <div className="scr-form-grid">
              <Select label="Context" value={contextSource} options={contextOptions} onChange={(value) => setContextSource(value as SummaryWidgetContextSource)} />
            </div>
            <CheckRow label="Render as markdown" checked={renderMarkdown} onChange={setRenderMarkdown} />
            <label className="scr-control">
              <span>Prompt</span>
              <textarea
                className="scr-textarea scr-textarea-tall"
                value={prompt}
                placeholder="Extract clear action items. Include owner and task when available."
                onChange={(event) => setPrompt(event.target.value)}
              />
            </label>
          </section>
        </div>

        <footer className="scr-modal-footer">
          <AppButton variant="secondary" onClick={onClose}>Cancel</AppButton>
          <AppButton onClick={submit} disabled={saving || !name.trim() || !displayTitle.trim() || !prompt.trim()}>
            {saving ? "Saving..." : "Save widget"}
          </AppButton>
        </footer>
      </section>
    </div>
  );
}

function TextField({ label, value, onChange, placeholder, className }: { label: string; value: string; onChange: (value: string) => void; placeholder?: string; className?: string }) {
  return (
    <label className={["scr-control", className].filter(Boolean).join(" ")}>
      <span>{label}</span>
      <input className="scr-input" value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

function CheckRow({ label, checked, onChange }: { label: string; checked: boolean; onChange: (checked: boolean) => void }) {
  return (
    <label className="scr-check-row">
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      <span className="scr-check-box" aria-hidden="true">{checked ? <Check size={13} /> : null}</span>
      <span>{label}</span>
    </label>
  );
}
