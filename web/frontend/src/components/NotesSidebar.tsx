import { useState } from "react";
import type { Note } from "../types/note";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Textarea } from "./ui/textarea";
import { Trash2, Pencil, Save, X, ExternalLink, Copy, Check } from "lucide-react";

interface NotesSidebarProps {
  notes: Note[];
  onEdit: (id: string, newContent: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onJumpTo: (time: number) => void;
}

export function NotesSidebar({ notes, onEdit, onDelete, onJumpTo }: NotesSidebarProps) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState<string>("");
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const startEdit = (n: Note) => {
    setEditingId(n.id);
    setDraft(n.content);
  };

  const cancelEdit = () => {
    setEditingId(null);
    setDraft("");
  };

  const saveEdit = async (id: string) => {
    try {
      setSaving(true);
      await onEdit(id, draft.trim());
      setEditingId(null);
      setDraft("");
    } finally {
      setSaving(false);
    }
  };

  const del = async (id: string) => {
    setDeletingId(id);
    try {
      await onDelete(id);
    } finally {
      setDeletingId(null);
    }
  };

  const formatTime = (s: number) => {
    const m = Math.floor(s / 60);
    const ss = Math.floor(s % 60).toString().padStart(2, "0");
    return `${m}:${ss}`;
  };

  return (
    <div className="h-full overflow-y-auto space-y-2">
      {notes.length === 0 && (
        <p className="text-sm text-carbon-600 dark:text-carbon-300">No notes yet. Select transcript text to add one.</p>
      )}
      {notes.map((n) => (
        <Card key={n.id} className="p-2 bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700">
          <div className="flex items-start justify-between gap-1.5">
            <div className="text-xs text-blue-600 dark:text-blue-300 font-mono">
              <button className="hover:underline" onClick={() => onJumpTo(n.start_time)} title="Jump to time">
                <ExternalLink className="inline h-3 w-3 mr-1" /> {formatTime(n.start_time)} - {formatTime(n.end_time)}
              </button>
            </div>
            <div className="flex items-center gap-0.5">
              {/* Copy note content */}
              <Button
                size="icon"
                variant="ghost"
                className="h-6 w-6"
                title={copiedId === n.id ? "Copied" : "Copy note"}
                onClick={async () => {
                  try {
                    await navigator.clipboard.writeText(n.content || "");
                    setCopiedId(n.id);
                    setTimeout(() => setCopiedId((prev) => (prev === n.id ? null : prev)), 1200);
                  } catch {}
                }}
              >
                {copiedId === n.id ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
              </Button>
              {editingId === n.id ? (
                <>
                  <Button size="icon" variant="ghost" className="h-6 w-6" disabled={saving} onClick={() => saveEdit(n.id)} title="Save">
                    <Save className="h-3.5 w-3.5" />
                  </Button>
                  <Button size="icon" variant="ghost" className="h-6 w-6" onClick={cancelEdit} title="Cancel">
                    <X className="h-3.5 w-3.5" />
                  </Button>
                </>
              ) : (
                <Button size="icon" variant="ghost" className="h-6 w-6" onClick={() => startEdit(n)} title="Edit">
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
              )}
              <Button size="icon" variant="ghost" className="h-6 w-6" disabled={deletingId === n.id} onClick={() => del(n.id)} title="Delete">
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
          <blockquote className="text-xs text-carbon-500 dark:text-carbon-400 border-l-2 border-carbon-300 dark:border-carbon-600 pl-2 mt-1 italic select-text">
            {n.quote}
          </blockquote>
          {editingId === n.id ? (
            <div className="mt-1">
              <Textarea value={draft} onChange={(e) => setDraft(e.target.value)} rows={3} />
            </div>
          ) : (
            <p className="mt-1 text-sm text-carbon-800 dark:text-carbon-100 whitespace-pre-wrap">
              {n.content}
            </p>
          )}
        </Card>
      ))}
    </div>
  );
}
