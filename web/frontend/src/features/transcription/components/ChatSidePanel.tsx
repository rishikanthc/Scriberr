import { useState, useEffect } from "react";
import { ChatSessionsSidebar } from "@/components/ChatSessionsSidebar";
import { ChatInterface } from "@/components/ChatInterface";
import { Button } from "@/components/ui/button";
import { X, ArrowLeft } from "lucide-react";
import { cn } from "@/lib/utils";

interface ChatSidePanelProps {
    transcriptionId: string;
    isOpen: boolean;
    onClose: () => void;
    isMobile: boolean;
}

export function ChatSidePanel({ transcriptionId, isOpen, onClose, isMobile }: ChatSidePanelProps) {
    // view state: 'list' (sessions) or 'chat' (active session)
    const [view, setView] = useState<'list' | 'chat'>('list');
    const [activeSessionId, setActiveSessionId] = useState<string | null>(null);

    // Reset view when closed or id changes
    useEffect(() => {
        if (!isOpen) {
            // Optional: reset state? User might want persistence.
            // Let's keep state for now so they don't lose place if they toggle quickly.
        }
    }, [isOpen]);

    const handleSessionSelect = (sessionId: string | null) => {
        if (sessionId) {
            setActiveSessionId(sessionId);
            setView('chat');
        } else {
            setActiveSessionId(null);
            setView('list'); // Or keep at chat with "new chat" state?
        }
    };

    const handleBackToList = () => {
        setView('list');
        setActiveSessionId(null);
    };

    if (!isOpen && !isMobile) return null; // For desktop split, we might handle visibility via parent layout, but helpful here too.

    return (
        <div className={cn(
            "flex flex-col h-full bg-[var(--bg-card)] border-l border-[var(--border-subtle)] shadow-[var(--shadow-float)] transition-all duration-300",
            // Mobile Specifics handled by parent usually, but here for internal structure
            "w-full h-full"
        )}>
            {/* Header */}
            <div className="flex-shrink-0 h-14 border-b border-[var(--border-subtle)] flex items-center justify-between px-4 bg-[var(--bg-card)]/50 backdrop-blur-sm z-10">
                <div className="flex items-center gap-2">
                    {view === 'chat' && (
                        <Button variant="ghost" size="icon" onClick={handleBackToList} className="h-8 w-8 -ml-2 text-[var(--text-secondary)]">
                            <ArrowLeft className="h-4 w-4" />
                        </Button>
                    )}
                    <div className="flex items-center gap-2 font-bold text-[var(--text-primary)]">
                        <span>{view === 'chat' ? 'Chat' : 'Sessions'}</span>
                    </div>
                </div>
                <Button variant="ghost" size="icon" onClick={onClose} className="h-8 w-8 text-[var(--text-tertiary)] hover:text-[var(--text-primary)]">
                    <X className="h-4 w-4" />
                </Button>
            </div>

            {/* Content */}
            <div className="flex-1 overflow-hidden relative">
                <div className={cn(
                    "absolute inset-0 transition-transform duration-300 ease-in-out",
                    view === 'list' ? 'translate-x-0' : '-translate-x-full'
                )}>
                    <ChatSessionsSidebar
                        transcriptionId={transcriptionId}
                        onSessionChange={handleSessionSelect}
                    />
                </div>
                <div className={cn(
                    "absolute inset-0 transition-transform duration-300 ease-in-out bg-[var(--bg-card)]",
                    view === 'chat' ? 'translate-x-0' : 'translate-x-full'
                )}>
                    {activeSessionId && (
                        <ChatInterface
                            transcriptionId={transcriptionId}
                            activeSessionId={activeSessionId}
                            onSessionChange={setActiveSessionId}
                            hideSidebar={true}
                        />
                    )}
                </div>
            </div>
        </div>
    );
}
