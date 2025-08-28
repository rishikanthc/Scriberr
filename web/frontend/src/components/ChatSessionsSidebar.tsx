import { useEffect, useState } from 'react'
import { Plus, Trash2, Edit2, MessageSquare, Search, Sparkles } from 'lucide-react'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from './ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select'
import { Label } from './ui/label'
import { useAuth } from '../contexts/AuthContext'
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
    } catch {}
  }

  return (
    <div className="h-full flex flex-col bg-gray-50 dark:bg-gray-850">
      {/* Header */}
      <div className="flex-shrink-0 p-4">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Chats</h2>
          <Dialog open={showNewSessionDialog} onOpenChange={setShowNewSessionDialog}>
            <DialogTrigger asChild>
              <Button 
                variant="ghost" 
                size="sm" 
                className="h-8 w-8 p-0 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 hover:bg-gray-200 dark:hover:bg-gray-800"
                title="New Chat"
              >
                <Plus className="h-4 w-4" />
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[425px] bg-white dark:bg-gray-850 shadow-2xl">
              <DialogHeader><DialogTitle>New Chat Session</DialogTitle></DialogHeader>
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="model">Model</Label>
                  <Select value={selectedModel} onValueChange={setSelectedModel}>
                    <SelectTrigger className="w-full bg-white dark:bg-gray-800 border-0 text-foreground">
                      <SelectValue placeholder="Select a model" />
                    </SelectTrigger>
                    <SelectContent className="bg-white dark:bg-gray-850 border-0">
                      {(availableModels || []).map(m => <SelectItem key={m} value={m}>{m}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="title">Title (optional)</Label>
                  <Input id="title" value={newSessionTitle} onChange={e => setNewSessionTitle(e.target.value)} placeholder="Enter a title..." />
                </div>
                <Button onClick={createSession} className="w-full">Create Session</Button>
              </div>
            </DialogContent>
          </Dialog>
        </div>
        
        {/* Search bar placeholder - similar to Open-webui */}
        <div className="relative mb-4">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
          <Input 
            placeholder="Search conversations..." 
            className="pl-10 bg-white dark:bg-gray-800 border-0 text-sm"
            disabled
          />
        </div>
      </div>

      {/* Chat Sessions List */}
      <div className="flex-1 overflow-y-auto px-2 pb-4">
        {sessions.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-gray-500 dark:text-gray-400">
            <MessageSquare className="h-12 w-12 mb-4 text-gray-300 dark:text-gray-600" />
            <div className="text-sm text-center">
              <p className="font-medium">No conversations yet</p>
              <p className="mt-1 text-xs">Start a new chat to begin!</p>
            </div>
          </div>
        ) : (
          <div className="space-y-1">
            {sessions.map(s => (
              <div
                key={s.id}
                className={`group relative flex items-center p-2 mx-2 rounded-lg cursor-pointer transition-all duration-150 ${
                  activeSessionId === s.id 
                    ? 'bg-gray-200 dark:bg-gray-800' 
                    : 'hover:bg-gray-100 dark:hover:bg-gray-800'
                }`}
                onClick={() => onSessionChange(s.id)}
              >
                {/* Chat icon */}
                <div className="flex-shrink-0 mr-3">
                  <MessageSquare className="h-4 w-4 text-gray-400 dark:text-gray-500" />
                </div>
                
                <div className="flex-1 min-w-0 flex items-center">
                  {editingId === s.id ? (
                    <Input
                      value={editTitle}
                      onChange={e => setEditTitle(e.target.value)}
                      onKeyDown={e => {
                        if (e.key === 'Enter') updateTitle(s.id, editTitle)
                        if (e.key === 'Escape') setEditingId(null)
                      }}
                      onBlur={() => updateTitle(s.id, editTitle)}
                      className="h-6 text-sm bg-white dark:bg-gray-900 border-0 p-0 focus-visible:ring-0"
                      autoFocus
                    />
                  ) : (
                    <div className="flex items-center gap-2 min-w-0 flex-1">
                      <div className="truncate text-sm text-gray-900 dark:text-gray-100 font-medium">
                        {s.title || 'New Chat'}
                      </div>
                      {generatingTitleIds.has(s.id) && (
                        <div className="flex-shrink-0 text-blue-500 dark:text-blue-400" title="Generating title...">
                          <Sparkles className="h-3 w-3 animate-pulse" />
                        </div>
                      )}
                    </div>
                  )}
                </div>
                
                {/* Action buttons */}
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <Button 
                    size="sm" 
                    variant="ghost" 
                    className="h-6 w-6 p-0 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded" 
                    onClick={(e) => { e.stopPropagation(); setEditingId(s.id); setEditTitle(s.title) }}
                    title="Rename chat"
                  >
                    <Edit2 className="h-3 w-3" />
                  </Button>
                  <Button 
                    size="sm" 
                    variant="ghost" 
                    className="h-6 w-6 p-0 text-gray-400 hover:text-red-500 rounded" 
                    onClick={(e) => { e.stopPropagation(); deleteSession(s.id) }}
                    title="Delete chat"
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
