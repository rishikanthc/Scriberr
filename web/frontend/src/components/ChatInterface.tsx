import { useState, useEffect, useRef, useCallback, memo } from "react";
import { Send, User, MessageCircle, Copy, Check, Sparkles, Brain, ChevronDown } from "lucide-react";
import ReactMarkdown from 'react-markdown'
import remarkMath from 'remark-math'
import rehypeKatex from 'rehype-katex'
import rehypeRaw from 'rehype-raw'
import rehypeHighlight from 'rehype-highlight'
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useChatEvents } from "../contexts/ChatEventsContext";
import { useToast } from "./ui/toast";
import { cn } from "@/lib/utils";

// Helper function to parse thinking content from model responses
function parseThinkingContent(content: string): { thinking: string | null; response: string } {
  // Match <think>...</think> tags (common in reasoning models)
  const thinkMatch = content.match(/<think>([\s\S]*?)<\/think>/);
  if (thinkMatch) {
    return {
      thinking: thinkMatch[1].trim(),
      response: content.replace(/<think>[\s\S]*?<\/think>/, '').trim()
    };
  }

  // Detect Qwen3's internal thinking pattern
  // Patterns like "Okay, the user..." or "thinking:" prefixes
  const thinkingPatterns = [
    /^(Okay,\s+(?:the user|I need to|let me)[\s\S]*?)(?=\n\n(?:[A-Z]|This|The|Here|Based|In|To))/i,
    /^(thinking:\s*[\s\S]*?)(?=\n\n)/i,
    /^(Let me (?:think|analyze|read|check|consider)[\s\S]*?)(?=\n\n(?:[A-Z]|This|The|Here|Based|In|To))/i,
  ];

  for (const pattern of thinkingPatterns) {
    const match = content.match(pattern);
    if (match && match[1].length > 50) { // Only capture substantial thinking blocks
      return {
        thinking: match[1].trim(),
        response: content.slice(match[0].length).trim()
      };
    }
  }

  return { thinking: null, response: content };
}

// Collapsible thinking block component with streaming support
function ThinkingBlock({ content, isStreaming = false }: { content: string; isStreaming?: boolean }) {
  const [expanded, setExpanded] = useState(isStreaming); // Auto-expand when streaming

  // Auto-expand when streaming starts
  useEffect(() => {
    if (isStreaming) setExpanded(true);
  }, [isStreaming]);

  return (
    <div className="mb-3 border border-purple-500/20 rounded-xl bg-purple-500/5 dark:bg-purple-500/10 overflow-hidden">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-purple-600 dark:text-purple-400 hover:bg-purple-500/5 transition-colors"
      >
        <Brain className={cn("h-4 w-4", isStreaming && "animate-pulse")} />
        <span className="font-medium">
          {isStreaming ? "Thinking..." : "Thinking"}
        </span>
        {isStreaming && (
          <span className="flex gap-1 ml-2">
            <span className="w-1.5 h-1.5 bg-purple-500 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
            <span className="w-1.5 h-1.5 bg-purple-500 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
            <span className="w-1.5 h-1.5 bg-purple-500 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
          </span>
        )}
        <ChevronDown className={cn("h-4 w-4 ml-auto transition-transform duration-200", expanded && "rotate-180")} />
      </button>
      {expanded && (
        <div className="px-4 pb-3 text-sm text-muted-foreground italic border-t border-purple-500/10 pt-3 whitespace-pre-wrap">
          {content}
          {isStreaming && <span className="inline-block w-2 h-4 bg-purple-500 ml-0.5 animate-pulse" />}
        </div>
      )}
    </div>
  );
}

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
  isStreaming?: boolean; // Track if this message is currently streaming
}

interface ChatInterfaceProps {
  transcriptionId: string;
  activeSessionId?: string;
  onSessionChange?: (sessionId: string | null) => void;
  onClose?: () => void;
  hideSidebar?: boolean;
}

export const ChatInterface = memo(function ChatInterface({ transcriptionId, activeSessionId, onSessionChange }: ChatInterfaceProps) {
  const { getAuthHeaders } = useAuth();
  const { emitSessionTitleUpdated, emitTitleGenerating } = useChatEvents();
  const { toast } = useToast();
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [activeSession, setActiveSession] = useState<ChatSession | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputMessage, setInputMessage] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [streamingMessage, setStreamingMessage] = useState("");
  const [selectedModel, setSelectedModel] = useState("gpt-3.5-turbo");
  const [error, setError] = useState<string | null>(null);
  const [contextInfo, setContextInfo] = useState<{ used: number; limit: number; trimmed: number } | null>(null);


  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const scrollToBottom = useCallback(() => {
    const el = messagesContainerRef.current
    if (el) {
      el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' })
    } else {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, []);

  useEffect(() => {
    const el = messagesContainerRef.current
    if (!el) return
    const distanceFromBottom = el.scrollHeight - (el.scrollTop + el.clientHeight)
    const nearBottom = distanceFromBottom < 120
    if (nearBottom) {
      scrollToBottom()
    }
  }, [messages, streamingMessage, scrollToBottom])

  useEffect(() => {
    if (transcriptionId) {
      loadChatModels();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [transcriptionId]); // loadChatModels ref loop avoidance

  // Memoize load functions to prevent recreating on every render
  const loadChatSession = useCallback(async (sessionId: string) => {
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
    } catch (err: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
      console.error("Error loading chat session:", err);
      setError(err.message);
      setMessages([]);
    }
  }, [getAuthHeaders]);

  const loadChatSessions = useCallback(async () => {
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
    } catch (err: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
      console.error("Error loading chat sessions:", err);
      // Don't set error message for sessions if the main issue is OpenAI config
      if (!err.message.includes("OpenAI")) {
        setError(err.message);
      }
      setSessions([]);
    }
  }, [transcriptionId, getAuthHeaders, activeSessionId, activeSession, onSessionChange, loadChatSession]);

  // Respond to external sessionId changes (via router) - optimize to avoid unnecessary re-runs
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
  }, [activeSessionId, activeSession?.id, loadChatSession, loadChatSessions, sessions]);

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
    } catch (err: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
      console.error("Error loading chat models:", err);
      setError(err.message);
      setSessions([]);
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
          content: messageContent,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to send message");
      }

      // Parse context info from response headers
      const contextUsed = parseInt(response.headers.get('X-Context-Used') || '0');
      const contextLimit = parseInt(response.headers.get('X-Context-Limit') || '0');
      const messagesTrimmed = parseInt(response.headers.get('X-Messages-Trimmed') || '0');
      if (contextUsed && contextLimit) {
        setContextInfo({ used: contextUsed, limit: contextLimit, trimmed: messagesTrimmed });
      }

      // Handle streaming response
      const reader = response.body?.getReader();
      if (!reader) throw new Error("No response body");

      let assistantContent = "";
      setStreamingMessage("");

      const messageId = Date.now() + 1;
      const assistantMessage: ChatMessage = {
        id: messageId,
        role: "assistant",
        content: "",
        created_at: new Date().toISOString(),
        isStreaming: true, // Mark as streaming
      };



      // Use ref to track assistant message index to avoid recreating array
      let assistantMessageIndex = -1;
      setMessages(prev => {
        assistantMessageIndex = prev.length;
        return [...prev, assistantMessage];
      });

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = new TextDecoder().decode(value);
        assistantContent += chunk;

        // Update message content while streaming
        setMessages(prev => {
          const newMessages = [...prev];
          if (assistantMessageIndex >= 0 && assistantMessageIndex < newMessages.length) {
            newMessages[assistantMessageIndex] = {
              ...newMessages[assistantMessageIndex],
              content: assistantContent,
              isStreaming: true
            };
          }
          return newMessages;
        });
      }

      // Mark streaming as complete
      // setStreamingMessageId(null);
      setMessages(prev => {
        const newMessages = [...prev];
        if (assistantMessageIndex >= 0 && assistantMessageIndex < newMessages.length) {
          newMessages[assistantMessageIndex] = {
            ...newMessages[assistantMessageIndex],
            content: assistantContent,
            isStreaming: false
          };
        }
        return newMessages;
      });

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
          const sid = activeSession?.id || activeSessionId;
          if (sid) {
            emitTitleGenerating({ sessionId: sid, isGenerating: true });
            try {
              const res = await fetch(`/api/v1/chat/sessions/${sid}/title/auto`, {
                method: 'POST',
                headers: { ...getAuthHeaders() }
              });

              if (res.ok) {
                const updated = await res.json();
                setSessions(prev => prev.map(s => s.id === updated.id ? { ...s, title: updated.title } : s));
                if ((activeSession && activeSession.id === updated.id) || (!activeSession && sid === updated.id)) {
                  setActiveSession(prev => prev ? { ...prev, title: updated.title } as any : prev); // eslint-disable-line @typescript-eslint/no-explicit-any
                }
                toast({
                  title: '✨ Chat Renamed',
                  description: `Renamed to "${updated.title}"`
                });
                emitSessionTitleUpdated({ sessionId: updated.id, title: updated.title });
              }
            } catch (error) {
              console.error('Error generating title:', error);
              toast({
                title: 'Failed to generate title',
                description: 'Could not auto-generate chat title'
              });
            } finally {
              emitTitleGenerating({ sessionId: sid, isGenerating: false });
            }
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
    } catch (err: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
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
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const PreBlock = (props: any) => {
    const preRef = useRef<HTMLPreElement>(null)
    const [copied, setCopied] = useState(false)
    const handleCopy = async () => {
      try {
        const text = preRef.current?.innerText || ''
        await navigator.clipboard.writeText(text)
        setCopied(true)
        setTimeout(() => setCopied(false), 1200)
      } catch {
        // clipboard write failed - ignore
      }
    }
    return (
      <div className="relative group">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleCopy}
          className="absolute right-2 top-2 h-auto px-2 py-1 text-xs opacity-0 group-hover:opacity-100 transition-opacity"
          aria-label="Copy code"
        >
          {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
          {copied ? 'Copied' : 'Copy'}
        </Button>
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
    <div className="h-full flex flex-col bg-transparent">
      {activeSession || activeSessionId ? (
        <>
          {/* Messages Container */}
          <div
            ref={messagesContainerRef}
            className="flex-1 overflow-y-auto pb-2.5 flex flex-col justify-between w-full flex-auto max-w-full z-10"
            id="messages-container"
          >
            <div className="h-full w-full flex flex-col px-3 py-4 space-y-5">
              {(messages || []).map(message => (
                <div key={message.id} className="group w-full">
                  {message.role === "user" ? (
                    /* User Message - Scriberr Design System */
                    <div className="flex justify-end">
                      <div className="flex w-full px-2 mx-auto">
                        <div className="w-full flex justify-end">
                          <div className="flex gap-3 max-w-3xl">
                            <div className="flex-1 overflow-hidden">
                              {/* User card with brand gradient accent */}
                              <div className="relative bg-gradient-to-br from-[#FFAB40]/8 to-[#FF6D20]/5 dark:from-[#FFAB40]/10 dark:to-[#FF6D20]/5 text-[var(--text-primary)] rounded-2xl rounded-tr-sm px-4 py-3 border border-[rgba(0,0,0,0.06)] dark:border-[rgba(255,255,255,0.08)] shadow-[0_2px_4px_rgba(0,0,0,0.04),0_8px_16px_rgba(0,0,0,0.04)] dark:shadow-[0_2px_4px_rgba(0,0,0,0.2),0_8px_16px_rgba(0,0,0,0.1)] hover:shadow-[0_4px_8px_rgba(0,0,0,0.06),0_12px_24px_rgba(0,0,0,0.06)] hover:-translate-y-0.5 transition-all duration-200">
                                {/* Copy button */}
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  onClick={async () => { try { await navigator.clipboard.writeText(message.content || ''); } catch { /* ignore */ } }}
                                  className="absolute right-2 top-2 h-7 w-7 p-0 opacity-0 group-hover:opacity-100 transition-opacity text-[var(--brand-solid)] hover:bg-[var(--brand-solid)]/10 rounded-full"
                                  title="Copy message"
                                >
                                  <Copy className="h-3.5 w-3.5" />
                                </Button>
                                <div className="text-sm leading-relaxed pr-8 font-reading text-[var(--text-primary)]">
                                  {message.content}
                                </div>
                              </div>
                            </div>
                            {/* User avatar with brand accent */}
                            <div className="h-9 w-9 rounded-full bg-gradient-to-br from-[#FFAB40] to-[#FF6D20] flex items-center justify-center flex-shrink-0 shadow-md">
                              <User className="h-4 w-4 text-white" />
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  ) : (
                    /* Assistant Message - Scriberr Design System */
                    <div className="flex justify-start">
                      <div className="flex w-full px-2 mx-auto">
                        <div className="w-full flex justify-start">
                          <div className="flex gap-3 max-w-5xl w-full">
                            {/* AI avatar with subtle glow */}
                            <div className="h-9 w-9 rounded-full bg-[var(--bg-card)] dark:bg-[#1F1F1F] flex items-center justify-center flex-shrink-0 border border-[rgba(0,0,0,0.06)] dark:border-[rgba(255,255,255,0.08)] shadow-[0_2px_8px_rgba(0,0,0,0.06)]">
                              <Sparkles className="h-4 w-4 text-[var(--brand-solid)]" />
                            </div>
                            <div className="flex-1 space-y-2 overflow-hidden">
                              {/* Assistant card with floating design */}
                              <div className="relative bg-[var(--bg-card)] dark:bg-[#141414] border border-[rgba(0,0,0,0.06)] dark:border-[rgba(255,255,255,0.08)] rounded-2xl rounded-tl-sm px-4 py-4 shadow-[0_2px_4px_rgba(0,0,0,0.04),0_10px_20px_rgba(0,0,0,0.04)] dark:shadow-[0_2px_4px_rgba(0,0,0,0.3),0_10px_20px_rgba(0,0,0,0.15)] hover:shadow-[0_4px_8px_rgba(0,0,0,0.06),0_14px_28px_rgba(0,0,0,0.06)] hover:-translate-y-0.5 transition-all duration-200">
                                {/* Copy button for assistant message */}
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  onClick={async () => { try { await navigator.clipboard.writeText(message.content || ''); } catch { /* ignore */ } }}
                                  className="absolute right-2 top-2 h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
                                  title="Copy message"
                                >
                                  <Copy className="h-3 w-3" />
                                </Button>
                                {(() => {
                                  const { thinking, response } = parseThinkingContent(message.content);
                                  const isCurrentlyStreaming = message.isStreaming === true;

                                  // During streaming, if we detect thinking pattern but no response yet,
                                  // show thinking in real-time
                                  const hasResponse = response && response.length > 0;
                                  const showThinkingStream = isCurrentlyStreaming && !hasResponse && message.content.length > 0;

                                  return (
                                    <>
                                      {/* Show thinking block - either detected thinking or streaming content that looks like thinking */}
                                      {(thinking || showThinkingStream) && (
                                        <ThinkingBlock
                                          content={thinking || message.content}
                                          isStreaming={isCurrentlyStreaming && !hasResponse}
                                        />
                                      )}

                                      {/* Show response content with streaming cursor */}
                                      {hasResponse && (
                                        <div className="prose prose-sm dark:prose-invert max-w-none text-foreground leading-relaxed font-reading">
                                          <ReactMarkdown
                                            remarkPlugins={[remarkMath]}
                                            // eslint-disable-next-line @typescript-eslint/no-explicit-any
                                            rehypePlugins={[rehypeRaw as any, rehypeKatex as any, rehypeHighlight as any]}
                                            // eslint-disable-next-line @typescript-eslint/no-explicit-any
                                            components={{ pre: PreBlock as any }}
                                          >
                                            {response}
                                          </ReactMarkdown>
                                          {isCurrentlyStreaming && (
                                            <span className="inline-block w-2 h-4 bg-[var(--brand-solid)] ml-0.5 animate-pulse align-middle" />
                                          )}
                                        </div>
                                      )}

                                      {/* If no thinking detected and streaming, show content directly with cursor */}
                                      {!thinking && !showThinkingStream && !hasResponse && isCurrentlyStreaming && message.content.length > 0 && (
                                        <div className="prose prose-sm dark:prose-invert max-w-none text-foreground leading-relaxed font-reading">
                                          <ReactMarkdown
                                            remarkPlugins={[remarkMath]}
                                            // eslint-disable-next-line @typescript-eslint/no-explicit-any
                                            rehypePlugins={[rehypeRaw as any, rehypeKatex as any, rehypeHighlight as any]}
                                            // eslint-disable-next-line @typescript-eslint/no-explicit-any
                                            components={{ pre: PreBlock as any }}
                                          >
                                            {message.content}
                                          </ReactMarkdown>
                                          <span className="inline-block w-2 h-4 bg-[var(--brand-solid)] ml-0.5 animate-pulse align-middle" />
                                        </div>
                                      )}
                                    </>
                                  );
                                })()}
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
                    <div className="flex w-full max-w-5xl px-2 mx-auto">
                      <div className="w-full flex justify-start">
                        <div className="flex space-x-3 max-w-5xl w-full">
                          <div className="h-8 w-8 rounded-full bg-indigo-500/10 dark:bg-indigo-500/20 flex items-center justify-center flex-shrink-0 border border-indigo-500/20 animate-pulse">
                            <Sparkles className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
                          </div>
                          <div className="flex-1 space-y-2 overflow-hidden">
                            <div className="flex items-center space-x-2 text-muted-foreground">
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
          <div className="pb-4 pt-2 bg-gradient-to-t from-background via-background to-transparent sticky bottom-0 z-20 pb-[env(safe-area-inset-bottom)]">
            <div className="flex w-full px-3 mx-auto">
              <div className="w-full">
                <div className="flex items-center gap-2 bg-[#F9FAFB] dark:bg-zinc-900 rounded-full p-2 mx-auto shadow-sm border border-transparent focus-within:border-[#FF6D20] focus-within:ring-1 focus-within:ring-[#FF6D20]/20 transition-all duration-300">
                  <Input
                    ref={inputRef}
                    value={inputMessage}
                    onChange={(e) => setInputMessage(e.target.value)}
                    onKeyDown={handleKeyPress}
                    placeholder="Type your message..."
                    disabled={isLoading}
                    className="flex-1 border-0 bg-transparent focus-visible:ring-0 focus:ring-0 outline-none resize-none text-sm placeholder:text-muted-foreground font-reading px-4 h-9"
                  />
                  <Button
                    onClick={sendMessage}
                    disabled={isLoading || !inputMessage.trim()}
                    size="icon"
                    className={cn(
                      "h-9 w-9 p-0 rounded-full shadow-sm transition-all duration-300 hover:scale-105 active:scale-95",
                      !inputMessage.trim() || isLoading
                        ? "bg-gray-200 text-gray-400 dark:bg-zinc-800 dark:text-zinc-600"
                        : "bg-gradient-to-br from-[#FFAB40] to-[#FF3D00] text-white shadow-orange-500/20"
                    )}
                  >
                    <Send className="h-4 w-4" />
                  </Button>
                </div>
                {/* Context usage and disclaimer */}
                <div className="flex items-center justify-center gap-3 mt-2 px-2 text-xs">
                  {contextInfo && (
                    <div className="flex items-center gap-2">
                      <span className={cn(
                        "px-2 py-0.5 rounded-full font-medium",
                        contextInfo.used / contextInfo.limit > 0.8
                          ? "bg-orange-500/10 text-orange-600 dark:text-orange-400"
                          : "bg-muted text-muted-foreground"
                      )}>
                        {Math.round((contextInfo.used / contextInfo.limit) * 100)}% context
                      </span>
                      {contextInfo.trimmed > 0 && (
                        <span className="text-amber-600 dark:text-amber-400" title={`${contextInfo.trimmed} older messages removed to fit context window`}>
                          ({contextInfo.trimmed} trimmed)
                        </span>
                      )}
                    </div>
                  )}
                  <span className="text-carbon-500">
                    AI can make mistakes. Verify important information.
                  </span>
                </div>
              </div>
            </div>
          </div>
        </>
      ) : (
        <div className="flex items-center h-full">
          <div className="flex flex-col items-center justify-center w-full max-w-md mx-auto p-6 text-center">
            <div className="h-16 w-16 rounded-full bg-muted flex items-center justify-center mb-4">
              <MessageCircle className="h-8 w-8 text-muted-foreground" />
            </div>
            <h3 className="text-lg font-bold text-foreground mb-2">How can I help you today?</h3>
            <p className="text-sm text-muted-foreground max-w-sm">
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
});