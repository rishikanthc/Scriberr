import { useEffect, useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog";
import { Trash2, FileText } from "lucide-react";
import type { SummaryTemplate } from "./SummaryTemplateDialog";
import { useAuth } from "@/features/auth/hooks/useAuth";

interface SummaryTemplatesTableProps {
  onEdit: (tpl: SummaryTemplate) => void;
  refreshTrigger?: number;
  disabled?: boolean;
}

export function SummaryTemplatesTable({ onEdit, refreshTrigger = 0, disabled = false }: SummaryTemplatesTableProps) {
  const { getAuthHeaders } = useAuth();
  const [items, setItems] = useState<SummaryTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [openPop, setOpenPop] = useState<Record<string, boolean>>({});
  const [deleting, setDeleting] = useState<Set<string>>(new Set());

  const fetchItems = useCallback(async () => {
    try {
      setLoading(true);
      const res = await fetch('/api/v1/summaries', { headers: { ...getAuthHeaders() } });
      if (res.ok) {
        const data: SummaryTemplate[] = await res.json();
        setItems(data);
      }
    } finally {
      setLoading(false);
    }
  }, [getAuthHeaders]);

  useEffect(() => { fetchItems(); }, [fetchItems, refreshTrigger]);

  const handleDelete = async (id: string) => {
    setOpenPop(prev => ({ ...prev, [id]: false }));
    try {
      setDeleting(prev => new Set(prev).add(id));
      const res = await fetch(`/api/v1/summaries/${id}`, { method: 'DELETE', headers: { ...getAuthHeaders() } });
      if (res.ok) {
        setItems(prev => prev.filter(i => i.id !== id));
      } else {
        alert('Failed to delete');
      }
    } finally {
      setDeleting(prev => { const s = new Set(prev); s.delete(id); return s; });
    }
  };

  const formatDate = (d?: string) => d ? new Date(d).toLocaleString() : '';

  if (loading) {
    return (
      <div className="space-y-2">
        {[...Array(3)].map((_, i) => (
          <div key={i} className="bg-carbon-100 dark:bg-carbon-800 rounded-lg p-4 animate-pulse h-16" />
        ))}
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className={`text-center py-16 ${disabled ? 'opacity-60 pointer-events-none' : ''}`}>
        <div className="bg-[var(--bg-main)] rounded-full w-16 h-16 mx-auto mb-4 flex items-center justify-center border border-[var(--border-subtle)]">
          <FileText className="h-8 w-8 text-[var(--text-tertiary)]" />
        </div>
        <h3 className="text-lg font-medium text-[var(--text-primary)] mb-2">No summary templates</h3>
        <p className="text-[var(--text-secondary)] mb-6 max-w-sm mx-auto">Create your first summarization template to reuse your prompt.</p>
      </div>
    );
  }

  return (
    <div className={`space-y-2 ${disabled ? 'opacity-60 pointer-events-none' : ''}`}>
      {items.map(tpl => (
        <div key={tpl.id} className="group bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-lg p-4 hover:border-[var(--brand-solid)] transition-all duration-200 cursor-pointer shadow-sm" onClick={() => !disabled && onEdit(tpl)}>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3 flex-1 min-w-0">
              <div className="bg-[var(--bg-main)] rounded-md p-1.5 text-[var(--text-tertiary)]">
                <FileText className="h-3.5 w-3.5" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-3">
                  <h3 className="text-sm font-medium text-[var(--text-primary)] truncate">{tpl.name}</h3>
                  <span className="text-xs text-[var(--text-tertiary)] whitespace-nowrap">{formatDate(tpl.created_at)}</span>
                </div>
                {tpl.description && (
                  <p className="text-xs text-[var(--text-secondary)] truncate mt-1">{tpl.description}</p>
                )}
              </div>
            </div>
            <div className="opacity-0 group-hover:opacity-100 transition-opacity duration-200" onClick={(e) => e.stopPropagation()}>
              {!disabled && (
                <Popover open={openPop[tpl.id!] || false} onOpenChange={(open) => setOpenPop(prev => ({ ...prev, [tpl.id!]: open }))}>
                  <PopoverTrigger asChild>
                    <Button variant="ghost" size="sm" className="h-7 w-7 p-0 hover:bg-carbon-300 dark:hover:bg-carbon-600">
                      ⋮
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-32 bg-[var(--bg-card)] border-[var(--border-subtle)] p-1 text-[var(--text-primary)]">
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button variant="ghost" size="sm" className="w-full justify-start h-7 text-xs hover:bg-[var(--error)]/10 text-[var(--error)] hover:text-[var(--error)]" disabled={deleting.has(tpl.id!)}>
                          <Trash2 className="mr-2 h-3 w-3" /> Delete
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent className="bg-[var(--bg-card)] border-[var(--border-subtle)]">
                        <AlertDialogHeader>
                          <AlertDialogTitle className="text-[var(--text-primary)]">Delete Template</AlertDialogTitle>
                          <AlertDialogDescription className="text-[var(--text-secondary)]">Are you sure you want to delete "{tpl.name}"?</AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel className="bg-[var(--bg-secondary)] border-[var(--border-subtle)] text-[var(--text-primary)] hover:bg-[var(--bg-main)]">Cancel</AlertDialogCancel>
                          <AlertDialogAction className="bg-[var(--error)] text-white hover:bg-[var(--error)]/90" onClick={() => handleDelete(tpl.id!)}>Delete</AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </PopoverContent>
                </Popover>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
