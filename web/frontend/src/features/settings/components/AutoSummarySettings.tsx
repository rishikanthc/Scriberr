import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { useAuth } from "@/features/auth/hooks/useAuth";

type Template = { id: string; name: string; model: string };
type Settings = { auto_summarize: boolean; default_template_id?: string; default_model?: string };

export function AutoSummarySettings() {
  const { getAuthHeaders } = useAuth();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [enabled, setEnabled] = useState(false);
  const [templateID, setTemplateID] = useState("");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");

  useEffect(() => {
    (async () => {
      const headers = { ...getAuthHeaders() };
      const [settingsResponse, templatesResponse] = await Promise.all([
        fetch("/api/v1/summaries/settings", { headers }),
        fetch("/api/v1/summaries", { headers }),
      ]);
      if (settingsResponse.ok) {
        const settings: Settings = await settingsResponse.json();
        setEnabled(settings.auto_summarize);
        setTemplateID(settings.default_template_id || "");
      }
      if (templatesResponse.ok) setTemplates(await templatesResponse.json());
    })();
  }, [getAuthHeaders]);

  const save = async () => {
    setMessage("");
    if (enabled && !templateID) {
      setMessage("Choose a default template before enabling auto-summary.");
      return;
    }
    setSaving(true);
    try {
      const response = await fetch("/api/v1/summaries/settings", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...getAuthHeaders() },
        body: JSON.stringify({ auto_summarize: enabled, default_template_id: templateID }),
      });
      if (!response.ok) {
        setMessage((await response.json()).error || "Failed to save auto-summary settings.");
        return;
      }
      setMessage("Auto-summary settings saved.");
    } catch {
      setMessage("Failed to save auto-summary settings.");
    } finally { setSaving(false); }
  };

  return <div className="bg-[var(--bg-main)]/50 border border-[var(--border-subtle)] rounded-[var(--radius-card)] p-4 sm:p-6 shadow-sm">
    <div className="flex items-start justify-between gap-4">
      <div><h3 className="text-lg font-medium text-[var(--text-primary)]">Auto-Summary</h3><p className="text-sm text-[var(--text-secondary)] mt-1">Generate a summary in the background when a transcription finishes successfully.</p></div>
      <Switch checked={enabled} onCheckedChange={setEnabled} aria-label="Enable automatic summaries" />
    </div>
    <div className="mt-4 max-w-xl">
      <label htmlFor="auto-summary-template" className="block text-sm font-medium text-[var(--text-primary)] mb-2">Default summary template</label>
      <select id="auto-summary-template" value={templateID} onChange={(e) => setTemplateID(e.target.value)} className="flex h-9 w-full rounded-[var(--radius-btn)] border border-[var(--border-subtle)] bg-[var(--bg-card)] px-3 py-1 text-sm text-[var(--text-primary)]" disabled={templates.length === 0}>
        <option value="">Select a template</option>{templates.map((template) => <option key={template.id} value={template.id}>{template.name} ({template.model})</option>)}
      </select>
      {templates.length === 0 && <p className="text-xs text-[var(--text-tertiary)] mt-2">Create a summary template below first.</p>}
    </div>
    <div className="flex items-center gap-3 mt-4"><Button onClick={save} disabled={saving}>{saving ? "Saving..." : "Save settings"}</Button>{message && <span className="text-sm text-[var(--text-secondary)]">{message}</span>}</div>
  </div>;
}
