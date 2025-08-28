import { useState, useEffect, useRef } from "react";
import { Send, Bot, User, MessageCircle, Copy, Check } from "lucide-react";
import ReactMarkdown from 'react-markdown'
import remarkMath from 'remark-math'
import rehypeKatex from 'rehype-katex'
import rehypeRaw from 'rehype-raw'
import rehypeHighlight from 'rehype-highlight'
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { useAuth } from "../contexts/AuthContext";
import { useChatEvents } from "../contexts/ChatEventsContext";
import { useToast } from "./ui/toast";

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

export function ChatInterface({ transcriptionId, activeSessionId, onSessionChange }: ChatInterfaceProps) {
  const { getAuthHeaders } = useAuth();
  const { emitSessionTitleUpdated } = useChatEvents();
  const { toast } = useToast();
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [activeSession, setActiveSession] = useState<ChatSession | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputMessage, setInputMessage] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [streamingMessage, setStreamingMessage] = useState("");
  const [selectedModel, setSelectedModel] = useState("gpt-3.5-turbo");
  const [error, setError] = useState<string | null>(null);
  
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const scrollToBottom = () => {
    const el = messagesContainerRef.current
    if (el) {
      el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' })
    } else {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
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
    if (!activeSessionId) return;
    if (activeSession?.id === activeSessionId) return;

    const found = sessions.find(s => s.id === activeSessionId);
    if (found) {
      setActiveSession(found);
      loadChatSession(found.id);
    } else {
      // Fallback: load the session directly and refresh sessions list
      setActiveSession(null);
      setMessages([]);
      loadChatSession(activeSessionId);
      loadChatSessions();
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
      if (data.models && data.models.length > 0 && !selectedModel) {
        setSelectedModel(data.models[0]);
      }
      setError(null);
      
      // Only load chat sessions if models loaded successfully
      loadChatSessions();
    } catch (err: any) {
      console.error("Error loading chat models:", err);
      setError(err.message);
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
      setMessages([])
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
          content: `${messageContent}\n\nTypeset all your answers in markdown and provide the markdown formatted string. Write equations in latex. Your response should contain only the markdown formatted string - nothing else. DO NOT wrap your response in code blocks, backticks, or any other formatting - return the raw markdown content directly.`,
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

      // Store the complete response before any potential session updates
      const finalAssistantContent = assistantContent;
      const finalMessages = [...messages, userMessage, { ...assistantMessage, content: finalAssistantContent }];
      
      // Auto-generate title after 2nd exchange (when we have 2 user messages and 2 assistant responses)
      const userMessageCount = finalMessages.filter(msg => msg.role === 'user').length;
      const assistantMessageCount = finalMessages.filter(msg => msg.role === 'assistant').length;
      
      // Only generate title after the 2nd complete exchange
      if (userMessageCount === 2 && assistantMessageCount === 2) {
        // Wait a moment to ensure UI is updated, then generate title
        setTimeout(async () => {
          try {
            const sid = activeSession?.id || activeSessionId;
            if (sid) {
              const res = await fetch(`/api/v1/chat/sessions/${sid}/title/auto`, {
                method: 'POST',
                headers: { ...getAuthHeaders() }
              });
              
              if (res.ok) {
                const updated = await res.json();
                setSessions(prev => prev.map(s => s.id === updated.id ? { ...s, title: updated.title } : s));
                if ((activeSession && activeSession.id === updated.id) || (!activeSession && sid === updated.id)) {
                  setActiveSession(prev => prev ? { ...prev, title: updated.title } as any : prev);
                }
                toast({ title: 'Chat Renamed', description: updated.title });
                emitSessionTitleUpdated({ sessionId: updated.id, title: updated.title });
              }
            }
          } catch (error) {
            console.error('Error generating title:', error);
          }
        }, 500); // Small delay to ensure message is fully processed
      }

      // Update session metadata without full reload to prevent message loss
      try {
        const sid = activeSession?.id || activeSessionId;
        if (sid) {
          setSessions(prev => prev.map(s => 
            s.id === sid 
              ? { ...s, message_count: finalMessages.length, updated_at: new Date().toISOString() }
              : s
          ));
        }
      } catch (error) {
        console.error('Error updating session metadata:', error);
      }
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
    <div className="h-full flex flex-col bg-white dark:bg-gray-900">
      {activeSession || activeSessionId ? (
        <>
          {/* Messages Container */}
          <div 
            ref={messagesContainerRef} 
            className="flex-1 overflow-y-auto pb-2.5 flex flex-col justify-between w-full flex-auto max-w-full z-10"
            id="messages-container"
          >
            <div className="h-full w-full flex flex-col px-6 py-6 space-y-6">
              {(messages || []).map(message => (
                <div key={message.id} className="group w-full">
                  {message.role === "user" ? (
                    /* User Message */
                    <div className="flex justify-end">
                      <div className="flex w-full max-w-5xl px-6 mx-auto">
                        <div className="w-full flex justify-end">
                          <div className="flex space-x-3 max-w-3xl">
                            <div className="flex-1 overflow-hidden">
                              <div className="bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-white rounded-2xl px-4 py-3 relative">
                                {/* Copy button */}
                                <button
                                  onClick={async () => { try { await navigator.clipboard.writeText(message.content || ''); } catch {} }}
                                  className="absolute right-2 top-2 opacity-0 group-hover:opacity-100 transition-opacity p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-600"
                                  title="Copy message"
                                >
                                  <Copy className="h-3 w-3" />
                                </button>
                                <div className="text-sm leading-relaxed pr-6">
                                  {message.content}
                                </div>
                              </div>
                            </div>
                            <div className="h-8 w-8 rounded-full bg-gray-300 dark:bg-gray-700 flex items-center justify-center flex-shrink-0">
                              <User className="h-4 w-4 text-gray-700 dark:text-white" />
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  ) : (
                    /* Assistant Message */
                    <div className="flex justify-start">
                      <div className="flex w-full max-w-5xl px-6 mx-auto">
                        <div className="w-full flex justify-start">
                          <div className="flex space-x-3 max-w-5xl w-full">
                            <div className="h-8 w-8 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center flex-shrink-0">
                              <Bot className="h-4 w-4 text-gray-700 dark:text-gray-200" />
                            </div>
                            <div className="flex-1 space-y-2 overflow-hidden">
                              <div className="flex items-center space-x-2">
                                <div className="font-medium text-gray-800 dark:text-gray-100 text-sm">Assistant</div>
                              </div>
                              <div className="prose prose-sm dark:prose-invert max-w-none text-gray-700 dark:text-gray-200 leading-relaxed">
                                {/* Copy button for assistant message */}
                                <button
                                  onClick={async () => { try { await navigator.clipboard.writeText(message.content || ''); } catch {} }}
                                  className="float-right opacity-0 group-hover:opacity-100 transition-opacity p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800 ml-2"
                                  title="Copy message"
                                >
                                  <Copy className="h-3 w-3" />
                                </button>
                                <ReactMarkdown
                                  remarkPlugins={[remarkMath]}
                                  rehypePlugins={[rehypeRaw as any, rehypeKatex as any, rehypeHighlight as any]}
                                  components={{ pre: PreBlock as any }}
                                >
                                  {message.content}
                                </ReactMarkdown>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              ))}
              
              {/* Loading Indicator */}
              {isLoading && (
                <div className="group w-full">
                  <div className="flex justify-start">
                    <div className="flex w-full max-w-5xl px-6 mx-auto">
                      <div className="w-full flex justify-start">
                        <div className="flex space-x-3 max-w-5xl w-full">
                          <div className="h-8 w-8 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center flex-shrink-0">
                            <Bot className="h-4 w-4 text-gray-700 dark:text-gray-200" />
                          </div>
                          <div className="flex-1 space-y-2 overflow-hidden">
                            <div className="flex items-center space-x-2">
                              <div className="font-medium text-gray-800 dark:text-gray-100 text-sm">Assistant</div>
                            </div>
                            <div className="flex items-center space-x-2 text-gray-500 dark:text-gray-400">
                              <div className="flex space-x-1">
                                <div className="w-2 h-2 bg-current rounded-full animate-bounce" style={{ animationDelay: '0ms' }}></div>
                                <div className="w-2 h-2 bg-current rounded-full animate-bounce" style={{ animationDelay: '150ms' }}></div>
                                <div className="w-2 h-2 bg-current rounded-full animate-bounce" style={{ animationDelay: '300ms' }}></div>
                              </div>
                              <span className="text-sm">Generating response...</span>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}
              
              <div ref={messagesEndRef} />
            </div>
          </div>

          {/* Input Area */}
          <div className="pb-2">
            <div className="flex w-full max-w-5xl px-6 mx-auto">
              <div className="w-full">
                <div className="flex items-end gap-3 bg-gray-50 dark:bg-gray-800 rounded-2xl p-3 mx-auto">
                  <Input
                    ref={inputRef}
                    value={inputMessage}
                    onChange={(e) => setInputMessage(e.target.value)}
                    onKeyDown={handleKeyPress}
                    placeholder="Send a message..."
                    disabled={isLoading}
                    className="flex-1 border-0 bg-transparent focus-visible:ring-0 focus:ring-0 outline-none resize-none text-sm placeholder:text-gray-500 dark:placeholder:text-gray-400"
                  />
                  <Button
                    onClick={sendMessage}
                    disabled={isLoading || !inputMessage.trim()}
                    size="sm"
                    className="h-8 w-8 p-0 rounded-full bg-gray-900 hover:bg-gray-800 dark:bg-gray-100 dark:hover:bg-gray-200 text-white dark:text-gray-900"
                  >
                    <Send className="h-4 w-4" />
                  </Button>
                </div>
                {/* Bottom disclaimer */}
                <div className="text-xs text-gray-500 text-center mt-2 px-2">
                  AI can make mistakes. Verify important information.
                </div>
              </div>
            </div>
          </div>
        </>
      ) : (
        <div className="flex items-center h-full">
          <div className="flex flex-col items-center justify-center w-full max-w-md mx-auto p-6 text-center">
            <div className="h-16 w-16 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center mb-4">
              <MessageCircle className="h-8 w-8 text-gray-400" />
            </div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">How can I help you today?</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400 max-w-sm">
              Start a conversation about this transcript or ask any questions you have.
            </p>
          </div>
        </div>
      )}

      {error && (
        <div className="absolute bottom-4 right-4 bg-red-500 text-white p-3 rounded-lg shadow-lg max-w-sm">
          <p className="text-sm">{error}</p>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setError(null)}
            className="mt-2 text-white hover:bg-red-600"
          >
            Dismiss
          </Button>
        </div>
      )}
    </div>
  );
}