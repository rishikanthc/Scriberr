import { useState, useEffect, useRef } from "react";
import { Send, Bot, User, MessageCircle, Plus, Trash2, Edit2, Copy, Check } from "lucide-react";
import ReactMarkdown from 'react-markdown'
import remarkMath from 'remark-math'
import rehypeKatex from 'rehype-katex'
import rehypeRaw from 'rehype-raw'
import rehypeHighlight from 'rehype-highlight'
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { useAuth } from "../contexts/AuthContext";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./ui/select";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "./ui/dialog";
import { Label } from "./ui/label";

interface ChatSession {
  id: string;
  transcription_id: string;
  title: string;
  model: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  message_count: number;
  last_message?: {
    id: number;
    role: string;
    content: string;
    created_at: string;
  };
}

interface ChatMessage {
  id: number;
  role: "user" | "assistant";
  content: string;
  created_at: string;
}

interface ChatInterfaceProps {
  transcriptionId: string;
  activeSessionId?: string;
  onSessionChange?: (sessionId: string | null) => void;
  onClose?: () => void;
  hideSidebar?: boolean;
}

export function ChatInterface({ transcriptionId, activeSessionId, onSessionChange, hideSidebar = false }: ChatInterfaceProps) {
  const { getAuthHeaders } = useAuth();
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [activeSession, setActiveSession] = useState<ChatSession | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputMessage, setInputMessage] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [streamingMessage, setStreamingMessage] = useState("");
  const [availableModels, setAvailableModels] = useState<string[]>([]);
  const [selectedModel, setSelectedModel] = useState("gpt-3.5-turbo");
  const [showNewSessionDialog, setShowNewSessionDialog] = useState(false);
  const [newSessionTitle, setNewSessionTitle] = useState("");
  const [editingTitle, setEditingTitle] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [error, setError] = useState<string | null>(null);
  
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    const el = messagesContainerRef.current
    if (!el) return
    const distanceFromBottom = el.scrollHeight - (el.scrollTop + el.clientHeight)
    const nearBottom = distanceFromBottom < 120
    if (nearBottom) {
      scrollToBottom()
    }
  }, [messages, streamingMessage])

  useEffect(() => {
    if (transcriptionId) {
      loadChatModels();
    }
  }, [transcriptionId]);

  // Respond to external sessionId changes (via router)
  useEffect(() => {
    if (!sessions || sessions.length === 0) return;
    if (!activeSessionId) return;
    if (activeSession?.id === activeSessionId) return;

    const found = sessions.find(s => s.id === activeSessionId);
    if (found) {
      setActiveSession(found);
      loadChatSession(found.id);
    }
  }, [activeSessionId, sessions]);

  const loadChatModels = async () => {
    try {
      const response = await fetch("/api/v1/chat/models", {
        headers: getAuthHeaders(),
      });
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to load models");
      }
      
      const data = await response.json();
      setAvailableModels(data.models || []);
      if (data.models && data.models.length > 0 && !selectedModel) {
        setSelectedModel(data.models[0]);
      }
      setError(null);
      
      // Only load chat sessions if models loaded successfully
      loadChatSessions();
    } catch (err: any) {
      console.error("Error loading chat models:", err);
      setError(err.message);
      setAvailableModels([]);
      setSessions([]);
    }
  };

  const loadChatSessions = async () => {
    try {
      const response = await fetch(`/api/v1/chat/transcriptions/${transcriptionId}/sessions`, {
        headers: getAuthHeaders(),
      });
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to load chat sessions");
      }
      
      const data = await response.json();
      setSessions(data || []);

      // Determine active session: prefer prop, else most recent
      if (data && data.length > 0) {
        if (activeSessionId) {
          const fromProp = data.find((s: ChatSession) => s.id === activeSessionId);
          if (fromProp) {
            setActiveSession(fromProp);
            loadChatSession(fromProp.id);
          }
        } else if (!activeSession) {
          setActiveSession(data[0]);
          loadChatSession(data[0].id);
          // If no sessionId in URL and consumer wants routing, inform
          onSessionChange?.(data[0].id);
        }
      }
    } catch (err: any) {
      console.error("Error loading chat sessions:", err);
      // Don't set error message for sessions if the main issue is OpenAI config
      if (!err.message.includes("OpenAI")) {
        setError(err.message);
      }
      setSessions([]);
    }
  };

  const loadChatSession = async (sessionId: string) => {
    try {
      const response = await fetch(`/api/v1/chat/sessions/${sessionId}`, {
        headers: getAuthHeaders(),
      });
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to load chat session");
      }
      
      const data = await response.json();
      setMessages(data.messages || []);
    } catch (err: any) {
      console.error("Error loading chat session:", err);
      setError(err.message);
      setMessages([]);
    }
  };

  const createNewSession = async () => {
    if (!selectedModel || !availableModels || availableModels.length === 0) {
      setError("Please wait for models to load or configure OpenAI API key");
      return;
    }

    try {
      const response = await fetch("/api/v1/chat/sessions", {
        method: "POST",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          transcription_id: transcriptionId,
          model: selectedModel,
          title: newSessionTitle.trim() || undefined,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to create chat session");
      }

      const newSession = await response.json();
      setSessions(prev => [newSession, ...prev]);
      setActiveSession(newSession);
      setMessages([]);
      setShowNewSessionDialog(false);
      setNewSessionTitle("");
      setError(null);
      onSessionChange?.(newSession.id);
    } catch (err: any) {
      console.error("Error creating chat session:", err);
      setError(err.message);
    }
  };

  const updateSessionTitle = async (sessionId: string, newTitle: string) => {
    try {
      const response = await fetch(`/api/v1/chat/sessions/${sessionId}/title`, {
        method: "PUT",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ title: newTitle }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to update title");
      }

      const updatedSession = await response.json();
      setSessions(prev => prev.map(s => s.id === sessionId ? updatedSession : s));
      if (activeSession?.id === sessionId) {
        setActiveSession(updatedSession);
      }
      setEditingTitle(null);
      setError(null);
    } catch (err: any) {
      console.error("Error updating session title:", err);
      setError(err.message);
    }
  };

  const deleteSession = async (sessionId: string) => {
    if (!confirm("Are you sure you want to delete this chat session?")) return;

    try {
      const response = await fetch(`/api/v1/chat/sessions/${sessionId}`, {
        method: "DELETE",
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to delete session");
      }

      setSessions(prev => prev.filter(s => s.id !== sessionId));
      if (activeSession?.id === sessionId) {
        const remainingSessions = sessions.filter(s => s.id !== sessionId);
        if (remainingSessions.length > 0) {
          setActiveSession(remainingSessions[0]);
          loadChatSession(remainingSessions[0].id);
          onSessionChange?.(remainingSessions[0].id);
        } else {
          setActiveSession(null);
          setMessages([]);
          onSessionChange?.(null);
        }
      }
      setError(null);
    } catch (err: any) {
      console.error("Error deleting session:", err);
      setError(err.message);
    }
  };

  const sendMessage = async () => {
    if (!activeSession || !inputMessage.trim() || isLoading) return;

    const messageContent = inputMessage.trim();
    setInputMessage("");
    setIsLoading(true);
    setError(null);

    try {
      // Add user message to UI immediately
      const userMessage: ChatMessage = {
        id: Date.now(),
        role: "user",
        content: messageContent,
        created_at: new Date().toISOString(),
      };
      setMessages(prev => [...prev, userMessage]);

      const response = await fetch(`/api/v1/chat/sessions/${activeSession.id}/messages`, {
        method: "POST",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          content: `${messageContent}\n\nTypeset all your answers in markdown and provide the markdown formatted string (Make sure not to wrap the markdown string within fenced codeblocks)-write equations in latex.`,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to send message");
      }

      // Handle streaming response
      const reader = response.body?.getReader();
      if (!reader) throw new Error("No response body");

      let assistantContent = "";
      setStreamingMessage("");

      const assistantMessage: ChatMessage = {
        id: Date.now() + 1,
        role: "assistant",
        content: "",
        created_at: new Date().toISOString(),
      };
      setMessages(prev => [...prev, assistantMessage]);

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = new TextDecoder().decode(value);
        assistantContent += chunk;
        
        setMessages(prev => prev.map((msg, index) => 
          index === prev.length - 1 ? { ...msg, content: assistantContent } : msg
        ));
      }

      // Reload sessions to update message count and last message
      loadChatSessions();
    } catch (err: any) {
      console.error("Error sending message:", err);
      setError(err.message);
      // Remove the user message from UI if there was an error
      setMessages(prev => prev.slice(0, -1));
    } finally {
      setIsLoading(false);
      setStreamingMessage("");
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  // Removed timestamp formatting helper as timestamps are no longer displayed in the sidebar

  // Code block with copy button
  const PreBlock = (props: any) => {
    const preRef = useRef<HTMLPreElement>(null)
    const [copied, setCopied] = useState(false)
    const handleCopy = async () => {
      try {
        const text = preRef.current?.innerText || ''
        await navigator.clipboard.writeText(text)
        setCopied(true)
        setTimeout(() => setCopied(false), 1200)
      } catch {}
    }
    return (
      <div className="relative group">
        <button
          onClick={handleCopy}
          className="absolute right-2 top-2 inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs bg-black/60 dark:bg-white/10 text-white dark:text-gray-200 hover:bg-black/70 dark:hover:bg-white/20 transition-opacity opacity-0 group-hover:opacity-100"
          aria-label="Copy code"
        >
          {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
          {copied ? 'Copied' : 'Copy'}
        </button>
        <pre ref={preRef} className={props.className}>{props.children}</pre>
      </div>
    )
  }

  if (error && error.includes("OpenAI")) {
    return (
      <div className="h-full flex flex-col items-center justify-center p-6">
        <MessageCircle className="h-16 w-16 text-muted-foreground mb-4" />
        <h3 className="text-lg font-medium mb-2">OpenAI Configuration Required</h3>
        <p className="text-sm text-muted-foreground text-center mb-4">
          To use the chat feature, please configure your OpenAI API key in Settings.
        </p>
        <Button onClick={() => window.location.href = "/settings"}>
          Go to Settings
        </Button>
      </div>
    );
  }

  return (
      <div className="h-full flex chat-shell overflow-hidden">
        {/* Session Sidebar (optional) */}
      {!hideSidebar && (
      <div className="w-80 bg-gray-50 dark:bg-gray-800 flex flex-col shadow-sm chat-sidebar shrink-0 overflow-hidden">
        <div className="p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-medium">Chat Sessions</h3>
            <Dialog open={showNewSessionDialog} onOpenChange={setShowNewSessionDialog}>
              <DialogTrigger asChild>
                <Button size="sm" variant="outline">
                  <Plus className="h-4 w-4" />
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-[425px] bg-background shadow-lg">
                <DialogHeader className="pb-4">
                  <DialogTitle className="text-lg font-semibold text-foreground">New Chat Session</DialogTitle>
                </DialogHeader>
                <div className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="model" className="text-sm font-medium text-foreground">Model</Label>
                    <Select value={selectedModel} onValueChange={setSelectedModel}>
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder="Select a model" />
                      </SelectTrigger>
                      <SelectContent>
                        {(availableModels || []).map(model => (
                          <SelectItem key={model} value={model}>{model}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="title" className="text-sm font-medium text-foreground">Title (optional)</Label>
                    <Input
                      id="title"
                      value={newSessionTitle}
                      onChange={(e) => setNewSessionTitle(e.target.value)}
                      placeholder="Enter a title for this session"
                      className="w-full"
                    />
                  </div>
                  <Button onClick={createNewSession} className="w-full">
                    Create Session
                  </Button>
                </div>
              </DialogContent>
            </Dialog>
          </div>
        </div>

        {/* Sessions List */}
        <div className="flex-1 overflow-y-auto chat-scroll">
          {!sessions || sessions.length === 0 ? (
            <div className="p-4 text-center text-sm text-muted-foreground">
              No chat sessions yet. Create your first one!
            </div>
          ) : (
            <div className="space-y-1 p-2">
              {(sessions || []).map(session => (
                <div
                  key={session.id}
                  className={`p-3 rounded-lg cursor-pointer transition-colors group ${
                    activeSession?.id === session.id
                      ? "bg-gray-100 dark:bg-gray-700 shadow-sm"
                      : "hover:bg-gray-100 dark:hover:bg-gray-700"
                  }`}
                  onClick={() => {
                    setActiveSession(session);
                    loadChatSession(session.id);
                    onSessionChange?.(session.id);
                  }}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                      {editingTitle === session.id ? (
                        <Input
                          value={editTitle}
                          onChange={(e) => setEditTitle(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === "Enter") {
                              updateSessionTitle(session.id, editTitle);
                            } else if (e.key === "Escape") {
                              setEditingTitle(null);
                            }
                          }}
                          onBlur={() => updateSessionTitle(session.id, editTitle)}
                          className="h-6 text-sm"
                          autoFocus
                        />
                      ) : (
                        <h4 className="text-sm font-medium truncate">{session.title}</h4>
                      )}
                      <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
                        <span>{session.model}</span>
                        <span>•</span>
                        <span>{session.message_count} messages</span>
                      </div>
                      {/* Removed timestamp per design */}
                    </div>
                    <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-6 w-6 p-0"
                        onClick={(e) => {
                          e.stopPropagation();
                          setEditingTitle(session.id);
                          setEditTitle(session.title);
                        }}
                      >
                        <Edit2 className="h-3 w-3" />
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-6 w-6 p-0 text-destructive hover:text-destructive"
                        onClick={(e) => {
                          e.stopPropagation();
                          deleteSession(session.id);
                        }}
                      >
                        <Trash2 className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
      )}

      {/* Chat Area */}
      <div className="flex-1 flex flex-col overflow-hidden relative">
        {activeSession ? (
          <>
            {/* Chat Header */}
            <div className="p-4 bg-background shadow-sm">
              <h2 className="font-medium">{activeSession.title}</h2>
              <p className="text-sm text-muted-foreground">
                Model: {activeSession.model}
              </p>
            </div>

            {/* Messages: only scrollable area; padded for bottom input */}
            <div ref={messagesContainerRef} className="flex-1 overflow-y-auto p-4 pt-6 pb-28 md:pb-32 space-y-4 chat-scroll">
              {(messages || []).map(message => (
                <div
                  key={message.id}
                  className={`flex gap-3 ${
                    message.role === "user" ? "justify-end" : "justify-start"
                  }`}
                >
                  {message.role === "assistant" && (
                    <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                      <Bot className="h-4 w-4 text-primary" />
                    </div>
                  )}
                  <div
                    className={`max-w-[70%] rounded-lg p-3 ${
                      message.role === "user"
                        ? "bg-gray-100 dark:bg-gray-700"
                        : "bg-muted"
                    }`}
                  >
                    <div className="prose prose-gray dark:prose-invert max-w-none text-base">
                      <ReactMarkdown
                        remarkPlugins={[remarkMath]}
                        rehypePlugins={[rehypeRaw as any, rehypeKatex as any, rehypeHighlight as any]}
                        components={{ pre: PreBlock as any }}
                      >
                        {message.content}
                      </ReactMarkdown>
                    </div>
                  </div>
                  {message.role === "user" && (
                    <div className="h-8 w-8 rounded-full bg-muted flex items-center justify-center flex-shrink-0">
                      <User className="h-4 w-4" />
                    </div>
                  )}
                </div>
              ))}
              <div ref={messagesEndRef} />
            </div>

            {/* Input: absolute at bottom with margin */}
            <div className="absolute bottom-0 left-0 right-0 p-4">
              <div className="flex gap-2 rounded-lg border bg-background p-2 shadow-sm">
                <Input
                  ref={inputRef}
                  value={inputMessage}
                  onChange={(e) => setInputMessage(e.target.value)}
                  onKeyDown={handleKeyPress}
                  placeholder="Type your message..."
                  disabled={isLoading}
                  className="flex-1"
                />
                <Button
                  onClick={sendMessage}
                  disabled={isLoading || !inputMessage.trim()}
                  size="sm"
                >
                  <Send className="h-4 w-4" />
                </Button>
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 flex flex-col items-center justify-center p-6">
            <MessageCircle className="h-16 w-16 text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium mb-2">No Chat Session Selected</h3>
            <p className="text-sm text-muted-foreground text-center">
              Create a new chat session to start discussing this transcript with AI.
            </p>
          </div>
        )}
      </div>

      {error && (
        <div className="absolute bottom-4 right-4 bg-destructive text-destructive-foreground p-3 rounded-lg shadow-lg">
          <p className="text-sm">{error}</p>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setError(null)}
            className="mt-2 text-destructive-foreground hover:bg-destructive/20"
          >
            Dismiss
          </Button>
        </div>
      )}
    </div>
  );
}
