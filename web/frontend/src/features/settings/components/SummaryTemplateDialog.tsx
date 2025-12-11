import { useEffect, useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useAuth } from "@/features/auth/hooks/useAuth";

export interface SummaryTemplate {
  id?: string;
  name: string;
  description?: string;
  model?: string;
  prompt: string;
  created_at?: string;
  updated_at?: string;
}

interface SummaryTemplateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (tpl: Omit<SummaryTemplate, 'created_at' | 'updated_at'>) => Promise<void> | void;
  initial?: SummaryTemplate | null;
}

export function SummaryTemplateDialog({ open, onOpenChange, onSave, initial }: SummaryTemplateDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [model, setModel] = useState("");
  const [prompt, setPrompt] = useState("");
  const [saving, setSaving] = useState(false);
  const [models, setModels] = useState<string[]>([]);
  const { getAuthHeaders } = useAuth();

  useEffect(() => {
    if (open) {
      setName(initial?.name || "");
      setDescription(initial?.description || "");
      setModel(initial?.model || "");
      setPrompt(initial?.prompt || "");
      // Load models when dialog opens
      (async () => {
        try {
          const res = await fetch('/api/v1/chat/models', { headers: { ...getAuthHeaders() } });
          if (res.ok) {
            const data = await res.json();
            setModels(data.models || []);
            if (!initial?.model && (data.models || []).length) {
              setModel(data.models[0]);
            }
          }
        } catch { }
      })();
    }
  }, [open, initial]);

  const handleSave = async () => {
    if (!name.trim() || !prompt.trim() || !model.trim()) return;
    try {
      setSaving(true);
      await onSave({ id: initial?.id, name: name.trim(), description: description.trim() || undefined, model: model.trim(), prompt: prompt.trim() });
      onOpenChange(false);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl md:max-w-2xl lg:max-w-3xl xl:max-w-4xl max-h-[90vh] md:max-h-[92vh] overflow-y-auto bg-[var(--bg-card)] border-[var(--border-subtle)]">
        <DialogHeader>
          <DialogTitle className="text-[var(--text-primary)]">{initial ? 'Edit Summary Template' : 'New Summary Template'}</DialogTitle>
          <DialogDescription className="text-[var(--text-secondary)]">
            Define a reusable prompt to summarize transcripts.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label className="text-[var(--text-primary)]">Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g., Concise Bullet Summary" className="bg-[var(--bg-main)] border-[var(--border-subtle)] text-[var(--text-primary)] focus:border-[var(--brand-solid)]" />
          </div>
          <div className="space-y-2">
            <Label className="text-[var(--text-primary)]">Model</Label>
            <Select value={model} onValueChange={setModel}>
              <SelectTrigger className="bg-[var(--bg-main)] border-[var(--border-subtle)] text-[var(--text-primary)] focus:border-[var(--brand-solid)]">
                <SelectValue placeholder={models.length ? 'Select model' : 'No models available'} />
              </SelectTrigger>
              <SelectContent className="bg-[var(--bg-card)] border-[var(--border-subtle)] text-[var(--text-primary)] max-h-60">
                {models.map((m) => (
                  <SelectItem key={m} value={m} className="text-[var(--text-primary)] focus:bg-[var(--bg-secondary)]">{m}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label className="text-[var(--text-primary)]">Description (optional)</Label>
            <Input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Short description" className="bg-[var(--bg-main)] border-[var(--border-subtle)] text-[var(--text-primary)] focus:border-[var(--brand-solid)]" />
          </div>
          <div className="space-y-2">
            <Label className="text-[var(--text-primary)]">Prompt</Label>
            <Textarea
              rows={10}
              className="resize-y min-h-40 sm:min-h-60 md:min-h-72 lg:min-h-[50vh] max-h-[65vh] bg-[var(--bg-main)] border-[var(--border-subtle)] text-[var(--text-primary)] focus:border-[var(--brand-solid)]"
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Write the summarization instructions..."
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} className="bg-[var(--bg-secondary)] border-[var(--border-subtle)] text-[var(--text-primary)]">
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={saving || !name.trim() || !prompt.trim() || !model.trim()}
            className="!bg-[var(--brand-gradient)] hover:!opacity-90 text-white border-none"
          >
            {saving ? 'Saving...' : 'Save'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
