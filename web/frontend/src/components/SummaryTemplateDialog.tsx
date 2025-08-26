import { useEffect, useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "./ui/dialog";
import { Label } from "./ui/label";
import { Input } from "./ui/input";
import { Textarea } from "./ui/textarea";
import { Button } from "./ui/button";

export interface SummaryTemplate {
  id?: string;
  name: string;
  description?: string;
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
  const [prompt, setPrompt] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open) {
      setName(initial?.name || "");
      setDescription(initial?.description || "");
      setPrompt(initial?.prompt || "");
    }
  }, [open, initial]);

  const handleSave = async () => {
    if (!name.trim() || !prompt.trim()) return;
    try {
      setSaving(true);
      await onSave({ id: initial?.id, name: name.trim(), description: description.trim() || undefined, prompt: prompt.trim() });
      onOpenChange(false);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700">
        <DialogHeader>
          <DialogTitle className="text-gray-900 dark:text-gray-100">{initial ? 'Edit Summary Template' : 'New Summary Template'}</DialogTitle>
          <DialogDescription className="text-gray-600 dark:text-gray-400">
            Define a reusable prompt to summarize transcripts.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label className="text-gray-700 dark:text-gray-300">Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g., Concise Bullet Summary" />
          </div>
          <div className="space-y-2">
            <Label className="text-gray-700 dark:text-gray-300">Description (optional)</Label>
            <Input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Short description" />
          </div>
          <div className="space-y-2">
            <Label className="text-gray-700 dark:text-gray-300">Prompt</Label>
            <Textarea rows={8} value={prompt} onChange={(e) => setPrompt(e.target.value)} placeholder="Write the summarization instructions..." />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={saving || !name.trim() || !prompt.trim()}
            className="bg-blue-600 hover:bg-blue-700 text-white dark:bg-blue-600 dark:hover:bg-blue-500"
          >
            {saving ? 'Saving...' : 'Save'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
