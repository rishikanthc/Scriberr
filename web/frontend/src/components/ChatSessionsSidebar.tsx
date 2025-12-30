import { useEffect, useState } from 'react'
import { Plus, Trash2, Edit2, MessageSquare, Search, Sparkles } from 'lucide-react'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from './ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select'
import { Label } from './ui/label'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "./ui/alert-dialog"
import { useAuth } from '@/features/auth/hooks/useAuth'
import { useChatEvents } from '../contexts/ChatEventsContext'

interface ChatSession {
  id: string
  transcription_id: string
  title: string
  model: string
  message_count: number
}

export function ChatSessionsSidebar({
  transcriptionId,
  activeSessionId,
  onSessionChange,
}: {
  transcriptionId: string
  activeSessionId?: string
  onSessionChange: (id: string | null) => void
}) {
  const { getAuthHeaders } = useAuth()
  const { subscribeSessionTitleUpdated, subscribeTitleGenerating } = useChatEvents()
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const [availableModels, setAvailableModels] = useState<string[]>([])
  const [selectedModel, setSelectedModel] = useState<string>('')
  const [showNewSessionDialog, setShowNewSessionDialog] = useState(false)
  const [newSessionTitle, setNewSessionTitle] = useState('')
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editTitle, setEditTitle] = useState('')
  const [generatingTitleIds, setGeneratingTitleIds] = useState<Set<string>>(new Set())
  const [deleteId, setDeleteId] = useState<string | null>(null)

  useEffect(() => {
    loadModels()
    loadSessions()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [transcriptionId])

  // Reactively apply title updates emitted elsewhere
  useEffect(() => {
    const unsubscribe = subscribeSessionTitleUpdated(({ sessionId, title }) => {
      setSessions(prev => prev.map(s => (s.id === sessionId ? { ...s, title } : s)))
    })
    return unsubscribe
  }, [subscribeSessionTitleUpdated])

  // Listen for title generation status
  useEffect(() => {
    const unsubscribe = subscribeTitleGenerating(({ sessionId, isGenerating }) => {
      setGeneratingTitleIds(prev => {
        const newSet = new Set(prev)
        if (isGenerating) {
          newSet.add(sessionId)
        } else {
          newSet.delete(sessionId)
        }
        return newSet
      })
    })
    return unsubscribe
  }, [subscribeTitleGenerating])

  async function loadModels() {
    try {
      const res = await fetch('/api/v1/chat/models', { headers: getAuthHeaders() })
      if (!res.ok) return
      const data = await res.json()
      setAvailableModels(data.models || [])
      if (!selectedModel && data.models?.length) setSelectedModel(data.models[0])
    } catch { /* ignore */ }
  }

  async function loadSessions() {
    try {
      const res = await fetch(`/api/v1/chat/transcriptions/${transcriptionId}/sessions`, { headers: getAuthHeaders() })
      if (!res.ok) return
      const data = await res.json()
      setSessions(data || [])
    } catch { /* ignore */ }
  }

  async function createSession() {
    if (!selectedModel) return
    try {
      const res = await fetch('/api/v1/chat/sessions', {
        method: 'POST',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ transcription_id: transcriptionId, model: selectedModel, title: newSessionTitle || undefined }),
      })
      if (!res.ok) return
      const created = await res.json()
      setSessions(prev => [created, ...prev])
      onSessionChange(created.id)
      setShowNewSessionDialog(false)
      setNewSessionTitle('')
    } catch { /* ignore */ }
  }

  async function updateTitle(id: string, title: string) {
    try {
      const res = await fetch(`/api/v1/chat/sessions/${id}/title`, {
        method: 'PUT',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ title }),
      })
      if (!res.ok) return
      const updated = await res.json()
      setSessions(prev => prev.map(s => (s.id === id ? updated : s)))
      setEditingId(null)
    } catch { /* ignore */ }
  }

  async function initiateDelete(id: string) {
    setDeleteId(id)
  }

  async function confirmDelete() {
    if (!deleteId) return
    const id = deleteId
    try {
      const res = await fetch(`/api/v1/chat/sessions/${id}`, { method: 'DELETE', headers: getAuthHeaders() })
      if (!res.ok) return

      // Update sessions list first
      const updatedSessions = sessions.filter(s => s.id !== id)
      setSessions(updatedSessions)

      // If we deleted the active session, switch to the next available one
      if (activeSessionId === id) {
        if (updatedSessions.length > 0) {
          // Switch to the first available session (topmost)
          onSessionChange(updatedSessions[0].id)
        } else {
          // No sessions left, stay on chat page but with null session
          onSessionChange(null)
        }
      }
    } catch { /* ignore */ } finally {
      setDeleteId(null)
    }
  }

  return (
    <div className="h-full flex flex-col bg-background/50">
      {/* Header */}
      <div className="flex-shrink-0 p-4">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-foreground">Chats</h2>
          <Dialog open={showNewSessionDialog} onOpenChange={setShowNewSessionDialog}>
            <DialogTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="h-8 w-8 p-0 text-muted-foreground hover:text-[#FF6D20] hover:bg-orange-500/10 transition-colors"
                title="New Chat"
              >
                <Plus className="h-5 w-5" />
              </Button>
            </DialogTrigger>
            <DialogContent className="w-[calc(100%-2rem)] max-w-md mx-auto bg-[var(--bg-card)] dark:bg-[#0A0A0A] border border-[rgba(0,0,0,0.06)] dark:border-[rgba(255,255,255,0.08)] shadow-[0_2px_4px_rgba(0,0,0,0.04),0_24px_48px_rgba(0,0,0,0.08)] dark:shadow-[0_2px_4px_rgba(0,0,0,0.3),0_24px_48px_rgba(0,0,0,0.3)] p-0 rounded-2xl overflow-hidden">
              <DialogHeader className="p-5 pb-0">
                <DialogTitle className="text-xl font-bold text-[var(--text-primary)] flex items-center gap-2">
                  <div className="h-9 w-9 rounded-full bg-gradient-to-br from-[#FFAB40] to-[#FF6D20] flex items-center justify-center shadow-md">
                    <Sparkles className="h-4 w-4 text-white" />
                  </div>
                  New Chat Session
                </DialogTitle>
                <p className="text-sm text-[var(--text-tertiary)] mt-1">
                  Start a conversation about this transcript
                </p>
              </DialogHeader>
              <div className="p-5 space-y-5">
                <div className="space-y-2">
                  <Label htmlFor="model" className="text-sm font-medium text-[var(--text-secondary)]">Model</Label>
                  <Select value={selectedModel} onValueChange={setSelectedModel}>
                    <SelectTrigger className="w-full h-11 bg-[var(--bg-main)] border-[var(--border-subtle)] text-[var(--text-primary)] focus:ring-2 focus:ring-[var(--brand-solid)]/20 focus:border-[var(--brand-solid)] hover:border-[var(--brand-solid)]/50 transition-all rounded-xl">
                      <SelectValue placeholder="Select a model" />
                    </SelectTrigger>
                    <SelectContent className="bg-[var(--bg-card)] border-[var(--border-subtle)] rounded-xl shadow-lg">
                      {(availableModels || []).map(m => (
                        <SelectItem
                          key={m}
                          value={m}
                          className="focus:bg-[var(--brand-light)] focus:text-[var(--brand-solid)] cursor-pointer py-2.5"
                        >
                          <span className="truncate">{m}</span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="title" className="text-sm font-medium text-[var(--text-secondary)]">
                    Title <span className="text-[var(--text-tertiary)] font-normal">(optional)</span>
                  </Label>
                  <Input
                    id="title"
                    value={newSessionTitle}
                    onChange={e => setNewSessionTitle(e.target.value)}
                    placeholder="Auto-generated after 2 exchanges..."
                    className="h-11 bg-[var(--bg-main)] border-[var(--border-subtle)] focus-visible:ring-2 focus-visible:ring-[var(--brand-solid)]/20 focus-visible:border-[var(--brand-solid)] transition-all rounded-xl"
                  />
                </div>
              </div>
              <div className="p-5 pt-0 flex flex-col-reverse sm:flex-row gap-3 sm:justify-end">
                <Button
                  variant="ghost"
                  onClick={() => setShowNewSessionDialog(false)}
                  className="h-11 px-6 text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-main)] rounded-full w-full sm:w-auto"
                >
                  Cancel
                </Button>
                <Button
                  onClick={createSession}
                  disabled={!selectedModel}
                  className="h-11 px-6 bg-gradient-to-br from-[#FFAB40] to-[#FF3D00] text-white hover:scale-[1.02] active:scale-[0.98] transition-transform shadow-md disabled:opacity-50 disabled:cursor-not-allowed rounded-full w-full sm:w-auto"
                >
                  <MessageSquare className="h-4 w-4 mr-2" />
                  Start Chat
                </Button>
              </div>
            </DialogContent>
          </Dialog>
        </div>

        {/* Search bar placeholder - similar to Open-webui */}
        <div className="relative mb-4">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-[var(--text-tertiary)]" />
          <Input
            placeholder="Search conversations..."
            className="pl-10 bg-[var(--bg-card)] border-[var(--border-subtle)] shadow-sm text-sm focus-visible:ring-[var(--brand-solid)] transition-all"
            disabled
          />
        </div>
      </div>

      {/* Chat Sessions List */}
      <div className="flex-1 overflow-y-auto px-2 pb-4">
        {sessions.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-carbon-500 dark:text-carbon-400">
            <MessageSquare className="h-12 w-12 mb-4 text-carbon-300 dark:text-carbon-600" />
            <div className="text-sm text-center">
              <p className="font-medium">No conversations yet</p>
              <p className="mt-1 text-xs">Start a new chat to begin!</p>
            </div>
          </div>
        ) : (
          <div className="space-y-2">
            {sessions.map(session => (
              <div
                key={session.id}
                onClick={() => onSessionChange(session.id)}
                className={`
                  group relative p-3 rounded-xl border cursor-pointer transition-all duration-200 pr-10 min-h-[64px]
                  ${session.id === activeSessionId
                    ? 'bg-[var(--bg-card)] dark:bg-[#1F1F1F] border-[#FF6D20] shadow-[0_2px_4px_rgba(0,0,0,0.04),0_8px_16px_rgba(0,0,0,0.06)] dark:shadow-[0_2px_4px_rgba(0,0,0,0.3),0_8px_16px_rgba(0,0,0,0.2)] ring-1 ring-[#FF6D20]/20 z-10'
                    : 'bg-[var(--bg-card)] dark:bg-[#141414] border-[rgba(0,0,0,0.06)] dark:border-[rgba(255,255,255,0.08)] shadow-[0_2px_4px_rgba(0,0,0,0.04),0_8px_16px_rgba(0,0,0,0.04)] dark:shadow-[0_2px_4px_rgba(0,0,0,0.2),0_8px_16px_rgba(0,0,0,0.1)] hover:shadow-[0_4px_8px_rgba(0,0,0,0.06),0_12px_24px_rgba(0,0,0,0.06)] hover:-translate-y-0.5 hover:border-[var(--brand-solid)]/30'
                  }
                `}
              >
                <div className="flex items-start justify-between gap-2 overflow-hidden">
                  <div className="min-w-0 flex-1">
                    {editingId === session.id ? (
                      <Input
                        value={editTitle}
                        onChange={e => setEditTitle(e.target.value)}
                        onKeyDown={e => {
                          if (e.key === 'Enter') updateTitle(session.id, editTitle)
                          if (e.key === 'Escape') setEditingId(null)
                        }}
                        onBlur={() => updateTitle(session.id, editTitle)}
                        className="h-6 text-sm bg-background border-border p-0 focus-visible:ring-0"
                        autoFocus
                      />
                    ) : (
                      <h3 className={`text-sm font-medium truncate leading-tight ${session.id === activeSessionId ? 'text-[#FF6D20]' : 'text-foreground group-hover:text-foreground'}`}>
                        {session.title || 'Untitled Chat'}
                        {generatingTitleIds.has(session.id) && (
                          <span className="inline-flex items-center ml-2 text-brand-500 dark:text-brand-400" title="Generating title...">
                            <Sparkles className="h-3 w-3 animate-pulse" />
                          </span>
                        )}
                      </h3>
                    )}
                    <div className="flex items-center gap-2 mt-2">
                      <span className="text-[10px] uppercase tracking-wider text-muted-foreground bg-muted/50 px-1.5 py-0.5 rounded-md font-medium">
                        {session.model}
                      </span>
                    </div>
                  </div>

                  {/* Overlay Actions */}
                  <div className={`absolute top-2 right-2 flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity bg-card/80 backdrop-blur-sm rounded-lg p-0.5 shadow-sm border border-border/50 ${session.id === activeSessionId ? 'opacity-0 hover:opacity-100' : ''}`}>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 text-muted-foreground hover:text-foreground hover:bg-accent"
                      onClick={(e) => {
                        e.stopPropagation();
                        setEditingId(session.id);
                        setEditTitle(session.title);
                      }}
                      title="Rename"
                    >
                      <Edit2 className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 text-muted-foreground hover:text-red-500 hover:bg-red-500/10"
                      onClick={(e) => {
                        e.stopPropagation();
                        initiateDelete(session.id);
                      }}
                      title="Delete"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Delete Confirmation Alert */}
      <AlertDialog open={!!deleteId} onOpenChange={(open) => !open && setDeleteId(null)}>
        <AlertDialogContent className="bg-[#FFFFFF] dark:bg-[#0A0A0A] border-[var(--border-subtle)] shadow-[var(--shadow-float)] rounded-[var(--radius-card)]">
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Chat Session?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete the chat history for this session.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="rounded-full border-[var(--border-subtle)] hover:bg-[var(--bg-main)]">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmDelete}
              className="rounded-full bg-red-500 text-white hover:bg-red-600 shadow-sm"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
