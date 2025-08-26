import { useEffect, useState, useCallback } from "react";
import { Button } from "./ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "./ui/alert-dialog";
import { Trash2, FileText } from "lucide-react";
import type { SummaryTemplate } from "./SummaryTemplateDialog";
import { useAuth } from "../contexts/AuthContext";

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
      const res = await fetch('/api/v1/summaries', { headers: { ...getAuthHeaders() }});
      if (res.ok) {
        const data: SummaryTemplate[] = await res.json();
        setItems(data);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchItems(); }, [fetchItems, refreshTrigger]);

  const handleDelete = async (id: string) => {
    setOpenPop(prev => ({ ...prev, [id]: false }));
    try {
      setDeleting(prev => new Set(prev).add(id));
      const res = await fetch(`/api/v1/summaries/${id}`, { method: 'DELETE', headers: { ...getAuthHeaders() }});
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
          <div key={i} className="bg-gray-100 dark:bg-gray-800 rounded-lg p-4 animate-pulse h-16" />
        ))}
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className={`text-center py-16 ${disabled ? 'opacity-60 pointer-events-none' : ''}`}>
        <div className="bg-gray-100 dark:bg-gray-700 rounded-full w-16 h-16 mx-auto mb-4 flex items-center justify-center">
          <FileText className="h-8 w-8 text-gray-400 dark:text-gray-500" />
        </div>
        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">No summary templates</h3>
        <p className="text-gray-600 dark:text-gray-400 mb-6 max-w-sm mx-auto">Create your first summarization template to reuse your prompt.</p>
      </div>
    );
  }

  return (
    <div className={`space-y-2 ${disabled ? 'opacity-60 pointer-events-none' : ''}`}>
      {items.map(tpl => (
        <div key={tpl.id} className="group bg-gray-100 dark:bg-gray-700 rounded-lg p-4 hover:bg-gray-200 dark:hover:bg-gray-600 transition-all duration-200 cursor-pointer" onClick={() => !disabled && onEdit(tpl)}>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3 flex-1 min-w-0">
              <div className="bg-gray-200 dark:bg-gray-800 rounded-md p-1.5">
                <FileText className="h-3.5 w-3.5 text-gray-500 dark:text-gray-400" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-3">
                  <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{tpl.name}</h3>
                  <span className="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">{formatDate(tpl.created_at)}</span>
                </div>
                {tpl.description && (
                  <p className="text-xs text-gray-500 dark:text-gray-400 truncate mt-1">{tpl.description}</p>
                )}
              </div>
            </div>
            <div className="opacity-0 group-hover:opacity-100 transition-opacity duration-200" onClick={(e) => e.stopPropagation()}>
              {!disabled && (
              <Popover open={openPop[tpl.id!] || false} onOpenChange={(open) => setOpenPop(prev => ({ ...prev, [tpl.id!]: open }))}>
                <PopoverTrigger asChild>
                  <Button variant="ghost" size="sm" className="h-7 w-7 p-0 hover:bg-gray-300 dark:hover:bg-gray-600">
                    ⋮
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-32 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-600 p-1">
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button variant="ghost" size="sm" className="w-full justify-start h-7 text-xs hover:bg-gray-100 dark:hover:bg-gray-700 text-red-500 dark:text-red-400 hover:text-red-600 dark:hover:text-red-300" disabled={deleting.has(tpl.id!)}>
                        <Trash2 className="mr-2 h-3 w-3" /> Delete
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                      <AlertDialogHeader>
                        <AlertDialogTitle className="text-gray-900 dark:text-gray-100">Delete Template</AlertDialogTitle>
                        <AlertDialogDescription className="text-gray-600 dark:text-gray-400">Are you sure you want to delete "{tpl.name}"?</AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel className="bg-gray-100 dark:bg-gray-700 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-600">Cancel</AlertDialogCancel>
                        <AlertDialogAction className="bg-red-600 text-white hover:bg-red-700" onClick={() => handleDelete(tpl.id!)}>Delete</AlertDialogAction>
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
