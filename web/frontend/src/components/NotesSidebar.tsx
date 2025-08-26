import { useState } from "react";
import type { Note } from "../types/note";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Textarea } from "./ui/textarea";
import { Trash2, Pencil, Save, X, ExternalLink } from "lucide-react";

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
    <div className="h-full overflow-y-auto space-y-3">
      {notes.length === 0 && (
        <p className="text-sm text-gray-600 dark:text-gray-300">No notes yet. Select transcript text to add one.</p>
      )}
      {notes.map((n) => (
        <Card key={n.id} className="p-3 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
          <div className="flex items-start justify-between gap-2">
            <div className="text-xs text-blue-600 dark:text-blue-300 font-mono">
              <button className="hover:underline" onClick={() => onJumpTo(n.start_time)} title="Jump to time">
                <ExternalLink className="inline h-3 w-3 mr-1" /> {formatTime(n.start_time)} - {formatTime(n.end_time)}
              </button>
            </div>
            <div className="flex items-center gap-1">
              {editingId === n.id ? (
                <>
                  <Button size="icon" variant="ghost" className="h-7 w-7" disabled={saving} onClick={() => saveEdit(n.id)} title="Save">
                    <Save className="h-4 w-4" />
                  </Button>
                  <Button size="icon" variant="ghost" className="h-7 w-7" onClick={cancelEdit} title="Cancel">
                    <X className="h-4 w-4" />
                  </Button>
                </>
              ) : (
                <Button size="icon" variant="ghost" className="h-7 w-7" onClick={() => startEdit(n)} title="Edit">
                  <Pencil className="h-4 w-4" />
                </Button>
              )}
              <Button size="icon" variant="ghost" className="h-7 w-7" disabled={deletingId === n.id} onClick={() => del(n.id)} title="Delete">
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          </div>
          <blockquote className="text-xs text-gray-500 dark:text-gray-400 border-l-2 border-gray-300 dark:border-gray-600 pl-2 mt-2 italic select-text">
            {n.quote}
          </blockquote>
          {editingId === n.id ? (
            <div className="mt-2">
              <Textarea value={draft} onChange={(e) => setDraft(e.target.value)} rows={4} />
            </div>
          ) : (
            <p className="mt-2 text-sm text-gray-800 dark:text-gray-100 whitespace-pre-wrap">
              {n.content}
            </p>
          )}
        </Card>
      ))}
    </div>
  );
}
