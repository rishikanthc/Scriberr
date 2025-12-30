import { useEffect, useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { FormField } from "@/components/transcription/FormHelpers";
import { Loader2 } from "lucide-react";

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

// Styled class constants
const inputClassName = `
  h-11 bg-[var(--bg-main)] border border-[var(--border-subtle)] rounded-xl
  text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)]
  focus:border-[var(--brand-solid)] focus:ring-2 focus:ring-[var(--brand-solid)]/20
  transition-all duration-200
  [color-scheme:light] dark:[color-scheme:dark]
`;

const selectTriggerClassName = `
  h-11 bg-[var(--bg-main)] border border-[var(--border-subtle)] rounded-xl
  text-[var(--text-primary)] shadow-none
  focus:border-[var(--brand-solid)] focus:ring-2 focus:ring-[var(--brand-solid)]/20
`;

const selectContentClassName = `
  bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-xl
`;

const selectItemClassName = `
  text-[var(--text-primary)] rounded-lg mx-1 cursor-pointer
  focus:bg-[var(--brand-light)] focus:text-[var(--brand-solid)]
`;

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
        } catch { /* ignore */ }
      })();
    }
  }, [open, initial, getAuthHeaders]);

  const handleSave = async () => {
    if (!name.trim() || !prompt.trim() || !model.trim()) return;
    try {
      setSaving(true);
      await onSave({
        id: initial?.id,
        name: name.trim(),
        description: description.trim() || undefined,
        model: model.trim(),
        prompt: prompt.trim()
      });
      onOpenChange(false);
    } finally {
      setSaving(false);
    }
  };

  const isFormValid = name.trim() && prompt.trim() && model.trim();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-full sm:max-w-2xl w-[calc(100vw-1rem)] max-h-[90vh] overflow-hidden flex flex-col p-0 gap-0 bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-2xl"
        style={{ boxShadow: 'var(--shadow-float)' }}
      >
        {/* Header */}
        <DialogHeader className="px-6 pt-6 pb-4 border-b border-[var(--border-subtle)]">
          <DialogTitle className="text-xl font-semibold text-[var(--text-primary)]">
            {initial ? 'Edit Summary Template' : 'New Summary Template'}
          </DialogTitle>
          <DialogDescription className="text-[var(--text-secondary)] text-sm mt-1">
            Create a reusable prompt to generate summaries from transcripts.
          </DialogDescription>
        </DialogHeader>

        {/* Scrollable Content */}
        <div className="flex-1 overflow-y-auto px-6 py-6 space-y-5">
          {/* Name Field */}
          <FormField label="Template Name" htmlFor="templateName">
            <Input
              id="templateName"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., Concise Bullet Summary"
              className={inputClassName}
            />
          </FormField>

          {/* Model Selection */}
          <FormField
            label="Model"
            description="Choose the LLM model to use for generating summaries."
          >
            <Select value={model} onValueChange={setModel}>
              <SelectTrigger className={selectTriggerClassName}>
                <SelectValue placeholder={models.length ? 'Select a model' : 'No models available'} />
              </SelectTrigger>
              <SelectContent className={selectContentClassName}>
                {models.map((m) => (
                  <SelectItem key={m} value={m} className={selectItemClassName}>
                    {m}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>

          {/* Description Field */}
          <FormField label="Description" htmlFor="templateDesc" optional>
            <Input
              id="templateDesc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Brief description of this template"
              className={inputClassName}
            />
          </FormField>

          {/* Prompt Field */}
          <FormField
            label="Prompt"
            htmlFor="templatePrompt"
            description="Instructions for the model. The transcript will be provided as context."
          >
            <Textarea
              id="templatePrompt"
              rows={12}
              className={`${inputClassName} resize-y min-h-[200px] max-h-[50vh]`}
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Write the summarization instructions...

Example:
Summarize the following transcript into concise bullet points. Focus on key decisions, action items, and important discussion topics."
            />
          </FormField>
        </div>

        {/* Footer */}
        <DialogFooter className="px-6 py-4 border-t border-[var(--border-subtle)] gap-3 sm:gap-2">
          <Button
            variant="ghost"
            onClick={() => onOpenChange(false)}
            className="rounded-xl text-[var(--text-secondary)] hover:bg-[var(--bg-main)] cursor-pointer"
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={saving || !isFormValid}
            className="rounded-xl text-white cursor-pointer bg-gradient-to-r from-[#FFAB40] to-[#FF3D00] hover:opacity-90 active:scale-[0.98] transition-all shadow-lg shadow-orange-500/20"
          >
            {saving ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Saving...
              </>
            ) : (
              initial ? 'Update Template' : 'Create Template'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
