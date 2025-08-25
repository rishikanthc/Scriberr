import { useEffect, useState } from 'react'
import { Plus, Trash2, Edit2 } from 'lucide-react'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from './ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select'
import { Label } from './ui/label'
import { useAuth } from '../contexts/AuthContext'

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
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const [availableModels, setAvailableModels] = useState<string[]>([])
  const [selectedModel, setSelectedModel] = useState<string>('')
  const [showNewSessionDialog, setShowNewSessionDialog] = useState(false)
  const [newSessionTitle, setNewSessionTitle] = useState('')
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editTitle, setEditTitle] = useState('')

  useEffect(() => {
    loadModels()
    loadSessions()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [transcriptionId])

  async function loadModels() {
    try {
      const res = await fetch('/api/v1/chat/models', { headers: getAuthHeaders() })
      if (!res.ok) return
      const data = await res.json()
      setAvailableModels(data.models || [])
      if (!selectedModel && data.models?.length) setSelectedModel(data.models[0])
    } catch {}
  }

  async function loadSessions() {
    try {
      const res = await fetch(`/api/v1/chat/transcriptions/${transcriptionId}/sessions`, { headers: getAuthHeaders() })
      if (!res.ok) return
      const data = await res.json()
      setSessions(data || [])
    } catch {}
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
    } catch {}
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
    } catch {}
  }

  async function deleteSession(id: string) {
    if (!confirm('Delete this chat session?')) return
    try {
      const res = await fetch(`/api/v1/chat/sessions/${id}`, { method: 'DELETE', headers: getAuthHeaders() })
      if (!res.ok) return
      setSessions(prev => prev.filter(s => s.id !== id))
      if (activeSessionId === id) onSessionChange(null)
    } catch {}
  }

  return (
    <div className="h-full w-full flex flex-col bg-gray-50 dark:bg-gray-800 chat-sidebar">
      <div className="p-4 border-b border-sidebar-border/50">
        <div className="flex items-center justify-between">
          <h3 className="font-medium">Chat Sessions</h3>
          <Dialog open={showNewSessionDialog} onOpenChange={setShowNewSessionDialog}>
            <DialogTrigger asChild>
              <Button size="sm" variant="outline"><Plus className="h-4 w-4" /></Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[425px] bg-background">
              <DialogHeader><DialogTitle>New Chat Session</DialogTitle></DialogHeader>
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="model">Model</Label>
                  <Select value={selectedModel} onValueChange={setSelectedModel}>
                    <SelectTrigger className="w-full"><SelectValue placeholder="Select a model" /></SelectTrigger>
                    <SelectContent>
                      {(availableModels || []).map(m => <SelectItem key={m} value={m}>{m}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="title">Title (optional)</Label>
                  <Input id="title" value={newSessionTitle} onChange={e => setNewSessionTitle(e.target.value)} />
                </div>
                <Button onClick={createSession} className="w-full">Create</Button>
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </div>
      <div className="flex-1 overflow-y-auto chat-scroll p-2 space-y-1">
        {(sessions || []).map(s => (
          <div
            key={s.id}
            className={`p-3 rounded-lg cursor-pointer group ${activeSessionId === s.id ? 'bg-gray-100 dark:bg-gray-700' : 'hover:bg-gray-100 dark:hover:bg-gray-700'}`}
            onClick={() => onSessionChange(s.id)}
          >
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0 flex-1">
                {editingId === s.id ? (
                  <Input
                    value={editTitle}
                    onChange={e => setEditTitle(e.target.value)}
                    onKeyDown={e => {
                      if (e.key === 'Enter') updateTitle(s.id, editTitle)
                      if (e.key === 'Escape') setEditingId(null)
                    }}
                    onBlur={() => updateTitle(s.id, editTitle)}
                    className="h-7 text-sm"
                    autoFocus
                  />
                ) : (
                  <h4 className="text-sm font-medium truncate">{s.title}</h4>
                )}
                <div className="text-xs text-muted-foreground mt-1">{s.model} • {s.message_count} messages</div>
              </div>
              <div className="opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
                <Button size="sm" variant="ghost" className="h-6 w-6 p-0" onClick={(e) => { e.stopPropagation(); setEditingId(s.id); setEditTitle(s.title) }}>
                  <Edit2 className="h-3 w-3" />
                </Button>
                <Button size="sm" variant="ghost" className="h-6 w-6 p-0 text-destructive" onClick={(e) => { e.stopPropagation(); deleteSession(s.id) }}>
                  <Trash2 className="h-3 w-3" />
                </Button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

