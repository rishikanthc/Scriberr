import { useEffect, useMemo, useRef, useState, type FormEvent, type KeyboardEvent } from "react";
import { Check, ChevronDown, Copy, FileText, ListFilter, MessageSquarePlus, Plus, Search, Send, Sparkles, Trash2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useToast } from "@/components/ui/toast";
import { ReadOnlyMarkdown } from "@/features/transcription/components/ReadOnlyMarkdown";
import {
  mergeStreamMessages,
  useAddChatContextTranscript,
  useChatContext,
  useChatMessages,
  useChatModels,
  useChatSessions,
  useCompletedTranscriptChoices,
  useCreateChatSession,
  useDeleteChatContextTranscript,
  useStreamChatMessage,
} from "@/features/transcription/hooks/useTranscriptChat";
import type { ChatContextSource, ChatMessage, ChatSession } from "@/features/transcription/api/chatApi";

type TranscriptChatPanelProps = {
  parentTranscriptionId?: string;
};

export function TranscriptChatPanel({ parentTranscriptionId }: TranscriptChatPanelProps) {
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);
  const [selectedModel, setSelectedModel] = useState("");
  const [composerValue, setComposerValue] = useState("");
  const [displayMessages, setDisplayMessages] = useState<ChatMessage[]>([]);
  const [sessionPickerOpen, setSessionPickerOpen] = useState(false);
  const [contextPickerOpen, setContextPickerOpen] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);
  const messagesScrollRef = useRef<HTMLDivElement | null>(null);
  const modelsQuery = useChatModels(Boolean(parentTranscriptionId));
  const sessionsQuery = useChatSessions(parentTranscriptionId, Boolean(parentTranscriptionId));
  const activeSession = useMemo(
    () => sessionsQuery.data?.items.find((session) => session.id === activeSessionId) || null,
    [activeSessionId, sessionsQuery.data?.items]
  );
  const messagesQuery = useChatMessages(activeSessionId, Boolean(activeSessionId));
  const contextQuery = useChatContext(activeSessionId, Boolean(activeSessionId));
  const transcriptChoices = useCompletedTranscriptChoices();
  const createSessionMutation = useCreateChatSession(parentTranscriptionId);
  const addContextMutation = useAddChatContextTranscript();
  const deleteContextMutation = useDeleteChatContextTranscript();
  const streamMutation = useStreamChatMessage();
  const { toast } = useToast();

  const models = modelsQuery.data?.models || [];
  const providerUnavailable = modelsQuery.isError;
  const canChat = Boolean(parentTranscriptionId && !providerUnavailable && models.length > 0);
  const contextSources = contextQuery.data?.items || [];
  const activeContextSources = contextSources.filter((source) => source.enabled);
  const contextTitleByTranscriptionId = useMemo(() => {
    const map = new Map<string, string>();
    for (const choice of transcriptChoices.choices) {
      map.set(choice.transcriptionId, choice.title);
    }
    return map;
  }, [transcriptChoices.choices]);
  const selectableContexts = transcriptChoices.choices.filter((choice) => (
    !activeContextSources.some((source) => source.transcription_id === choice.transcriptionId)
  ));

  useEffect(() => {
    if (activeSessionId && sessionsQuery.data?.items.some((session) => session.id === activeSessionId)) return;
    setActiveSessionId(sessionsQuery.data?.items[0]?.id || null);
  }, [activeSessionId, sessionsQuery.data?.items]);

  useEffect(() => {
    if (selectedModel && models.some((model) => model.id === selectedModel)) return;
    setSelectedModel(activeSession?.model || models[0]?.id || "");
  }, [activeSession?.model, models, selectedModel]);

  useEffect(() => {
    setDisplayMessages(messagesQuery.data?.items || []);
  }, [messagesQuery.data?.items]);

  useEffect(() => {
    const element = messagesScrollRef.current;
    if (!element) return;
    const distanceFromBottom = element.scrollHeight - (element.scrollTop + element.clientHeight);
    if (distanceFromBottom < 180) {
      messagesEndRef.current?.scrollIntoView({ block: "end" });
    }
  }, [displayMessages]);

  const handleCreateSession = async () => {
    if (!parentTranscriptionId || !selectedModel || createSessionMutation.isPending) return;
    try {
      const session = await createSessionMutation.mutateAsync({
        parent_transcription_id: parentTranscriptionId,
        model: selectedModel,
      });
      setActiveSessionId(session.id);
      setSessionPickerOpen(false);
      setDisplayMessages([]);
    } catch (error) {
      toast({
        title: "Chat session was not created",
        description: error instanceof Error ? error.message : "Check your model provider settings.",
      });
    }
  };

  const handleAddContext = async (transcriptionId: string) => {
    if (!activeSessionId || addContextMutation.isPending) return;
    try {
      await addContextMutation.mutateAsync({ sessionId: activeSessionId, transcriptionId });
      setContextPickerOpen(false);
    } catch (error) {
      toast({
        title: "Context was not added",
        description: error instanceof Error ? error.message : "Try selecting the transcript again.",
      });
    }
  };

  const handleRemoveContext = async (source: ChatContextSource) => {
    if (!activeSessionId || deleteContextMutation.isPending) return;
    try {
      await deleteContextMutation.mutateAsync({ sessionId: activeSessionId, contextSourceId: source.id });
    } catch (error) {
      toast({
        title: "Context was not removed",
        description: error instanceof Error ? error.message : "Try again.",
      });
    }
  };

  const handleSubmit = async (event?: FormEvent<HTMLFormElement>) => {
    event?.preventDefault();
    const content = composerValue.trim();
    if (!activeSessionId || !selectedModel || !content || streamMutation.isPending) return;
    setComposerValue("");
    try {
      await streamMutation.mutateAsync({
        sessionId: activeSessionId,
        payload: { content, model: selectedModel },
        onEvent: (streamEvent) => {
          setDisplayMessages((current) => mergeStreamMessages(current, streamEvent));
          if (streamEvent.type === "chat.run.failed") {
            toast({ title: "Chat response failed", description: streamEvent.error });
          }
        },
      });
    } catch (error) {
      setComposerValue(content);
      toast({
        title: "Message was not sent",
        description: error instanceof Error ? error.message : "Try again.",
      });
    }
  };

  const handleComposerKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      event.currentTarget.form?.requestSubmit();
    }
  };

  if (!parentTranscriptionId) {
    return (
      <div className="scr-chat-disabled">
        <MessageSquarePlus size={34} aria-hidden="true" />
        <h2>Transcript required</h2>
        <p>Chat becomes available after this recording has a completed transcript.</p>
      </div>
    );
  }

  if (providerUnavailable) {
    return (
      <div className="scr-chat-disabled">
        <MessageSquarePlus size={34} aria-hidden="true" />
        <h2>LLM provider required</h2>
        <p>Configure an LLM provider in Settings before using chat.</p>
      </div>
    );
  }

  return (
    <section className="scr-chat-panel" aria-label="Transcript chat">
      <header className="scr-chat-header">
        <Popover open={sessionPickerOpen} onOpenChange={setSessionPickerOpen}>
          <PopoverTrigger asChild>
            <Button className="scr-chat-session-trigger" type="button" variant="ghost" aria-label="Choose chat session">
              <span>{activeSession?.title || "New chat"}</span>
              <ChevronDown size={16} aria-hidden="true" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="scr-chat-session-popover" align="start" side="bottom">
            <div className="scr-chat-session-popover-header">
              <Search size={20} aria-hidden="true" />
              <span>Search</span>
              <Button type="button" variant="ghost" size="icon" aria-label="Filter sessions" disabled>
                <ListFilter size={17} aria-hidden="true" />
              </Button>
            </div>
            <div className="scr-chat-session-list">
              {sessionsQuery.isLoading ? <p className="scr-chat-menu-status">Loading conversations.</p> : null}
              {!sessionsQuery.isLoading && (sessionsQuery.data?.items.length || 0) === 0 ? (
                <p className="scr-chat-menu-status">No conversations yet.</p>
              ) : null}
              {groupSessions(sessionsQuery.data?.items || []).map((group) => (
                <div className="scr-chat-session-group" key={group.label}>
                  <h3>{group.label}</h3>
                  {group.sessions.map((session) => (
                    <button
                      key={session.id}
                      className="scr-chat-session-item"
                      type="button"
                      data-active={session.id === activeSessionId}
                      onClick={() => {
                        setActiveSessionId(session.id);
                        setSelectedModel(session.model);
                        setSessionPickerOpen(false);
                      }}
                    >
                      <span>{session.title || "Transcript chat"}</span>
                      <small>{formatRelativeSessionTime(session.last_message_at || session.updated_at)}</small>
                    </button>
                  ))}
                </div>
              ))}
            </div>
          </PopoverContent>
        </Popover>

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              className="scr-chat-header-action"
              type="button"
              variant="ghost"
              size="icon"
              aria-label="New chat session"
              disabled={!canChat || createSessionMutation.isPending}
              onClick={handleCreateSession}
            >
              <MessageSquarePlus size={18} aria-hidden="true" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>New chat</TooltipContent>
        </Tooltip>
      </header>

      <div className="scr-chat-messages" ref={messagesScrollRef}>
        {messagesQuery.isLoading ? <p className="scr-chat-status">Loading chat.</p> : null}
        {!messagesQuery.isLoading && !activeSessionId ? (
          <div className="scr-chat-empty">
            <Sparkles size={28} aria-hidden="true" />
            <p>Start a new chat to ask about your transcripts.</p>
          </div>
        ) : null}
        {!messagesQuery.isLoading && activeSessionId && displayMessages.length === 0 ? (
          <div className="scr-chat-empty">
            <Sparkles size={28} aria-hidden="true" />
            <p>Ask anything about your conversations.</p>
          </div>
        ) : null}
        {displayMessages.map((message) => (
          <ChatMessageItem key={message.id} message={message} />
        ))}
        <div ref={messagesEndRef} />
      </div>

      <form className="scr-chat-composer" onSubmit={handleSubmit}>
        <div className="scr-chat-context-row">
          <Popover open={contextPickerOpen} onOpenChange={setContextPickerOpen}>
            <PopoverTrigger asChild>
              <Button
                className="scr-chat-context-add"
                type="button"
                variant="ghost"
                size="icon"
                aria-label="Add transcript context"
                disabled={!activeSessionId || !canChat}
              >
                <Plus size={18} aria-hidden="true" />
              </Button>
            </PopoverTrigger>
            <PopoverContent className="scr-chat-context-popover" align="start" side="top">
              <Command>
                <CommandInput placeholder="Search" />
                <CommandList>
                  <CommandEmpty>No completed transcripts.</CommandEmpty>
                  <CommandGroup heading="Conversations">
                    {transcriptChoices.isLoading ? (
                      <p className="scr-chat-menu-status">Loading transcripts.</p>
                    ) : selectableContexts.map((choice) => (
                      <CommandItem
                        key={choice.transcriptionId}
                        value={`${choice.title} ${choice.createdAt}`}
                        onSelect={() => handleAddContext(choice.transcriptionId)}
                      >
                        <span className="scr-chat-context-option">
                          <span>{choice.title}</span>
                          <small>{formatContextDate(choice.createdAt)}</small>
                        </span>
                      </CommandItem>
                    ))}
                  </CommandGroup>
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>

          {activeContextSources.map((source) => (
            <button
              className="scr-chat-context-chip"
              key={source.id}
              type="button"
              title="Remove context"
              onClick={() => handleRemoveContext(source)}
            >
              <FileText size={16} aria-hidden="true" />
              <span>{contextTitleByTranscriptionId.get(source.transcription_id) || "Transcript"}</span>
              <X className="scr-chat-context-chip-remove" size={15} aria-hidden="true" />
            </button>
          ))}
        </div>

        <textarea
          className="scr-chat-input"
          value={composerValue}
          aria-label="Chat message"
          placeholder="Ask anything about your conversations"
          rows={2}
          disabled={!activeSessionId || !canChat || streamMutation.isPending}
          onChange={(event) => setComposerValue(event.currentTarget.value)}
          onKeyDown={handleComposerKeyDown}
        />

        <div className="scr-chat-composer-footer">
          <Select value={selectedModel} onValueChange={setSelectedModel} disabled={!canChat || streamMutation.isPending}>
            <SelectTrigger className="scr-chat-model-trigger" aria-label="Model">
              <SelectValue placeholder="Model" />
            </SelectTrigger>
            <SelectContent className="scr-chat-model-menu" align="start">
              {models.map((model) => (
                <SelectItem key={model.id} value={model.id}>{model.display_name || model.id}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            className="scr-chat-send"
            type="submit"
            variant="ghost"
            size="icon"
            aria-label="Send message"
            disabled={!activeSessionId || !selectedModel || composerValue.trim().length === 0 || streamMutation.isPending}
          >
            <Send size={18} aria-hidden="true" />
          </Button>
        </div>
      </form>
    </section>
  );
}

function ChatMessageItem({ message }: { message: ChatMessage }) {
  const [copied, setCopied] = useState(false);
  const isAssistant = message.role === "assistant";
  const canCopy = message.content.trim().length > 0;

  const handleCopy = async () => {
    if (!canCopy) return;
    try {
      await navigator.clipboard.writeText(message.content);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1100);
    } catch {
      setCopied(false);
    }
  };

  if (message.role !== "user" && message.role !== "assistant") return null;

  return (
    <article className="scr-chat-message" data-role={message.role}>
      {isAssistant && message.reasoning_content.trim() ? (
        <details className="scr-chat-reasoning">
          <summary>Show thinking</summary>
          <p>{message.reasoning_content}</p>
        </details>
      ) : null}
      <div className="scr-chat-message-body">
        {isAssistant ? (
          <div className="scr-chat-message-markdown">
            <ReadOnlyMarkdown content={message.content || (message.status === "streaming" ? " " : "")} />
          </div>
        ) : (
          <p>{message.content}</p>
        )}
      </div>
      <div className="scr-chat-message-actions">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button type="button" variant="ghost" size="icon" aria-label="Copy message" disabled={!canCopy} onClick={handleCopy}>
              {copied ? <Check size={16} aria-hidden="true" /> : <Copy size={16} aria-hidden="true" />}
            </Button>
          </TooltipTrigger>
          <TooltipContent>{copied ? "Copied" : "Copy"}</TooltipContent>
        </Tooltip>
        {message.role === "user" ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button type="button" variant="ghost" size="icon" aria-label="Delete message" disabled>
                <Trash2 size={16} aria-hidden="true" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Delete</TooltipContent>
          </Tooltip>
        ) : null}
      </div>
    </article>
  );
}

function groupSessions(sessions: ChatSession[]) {
  const now = Date.now();
  const groups = [
    { label: "Today", sessions: [] as ChatSession[] },
    { label: "Past week", sessions: [] as ChatSession[] },
    { label: "Older", sessions: [] as ChatSession[] },
  ];
  for (const session of sessions) {
    const timestamp = new Date(session.last_message_at || session.updated_at || session.created_at).getTime();
    const age = now - timestamp;
    if (age < 24 * 60 * 60 * 1000) groups[0].sessions.push(session);
    else if (age < 7 * 24 * 60 * 60 * 1000) groups[1].sessions.push(session);
    else groups[2].sessions.push(session);
  }
  return groups.filter((group) => group.sessions.length > 0);
}

function formatRelativeSessionTime(value: string) {
  const timestamp = new Date(value).getTime();
  const diffMs = Date.now() - timestamp;
  const minutes = Math.max(1, Math.round(diffMs / 60_000));
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.round(hours / 24);
  return `${days}d ago`;
}

function formatContextDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(new Date(value));
}
