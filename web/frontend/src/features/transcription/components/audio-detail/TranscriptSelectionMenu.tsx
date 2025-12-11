import { Plus, Ear, StickyNote } from "lucide-react";

interface TranscriptSelectionMenuProps {
    isOpen: boolean;
    isMobile: boolean;
    position: { x: number; y: number };
    onAddNote: () => void;
    onListenFromHere: () => void;
}

export function TranscriptSelectionMenu({
    isOpen,
    isMobile,
    position,
    onAddNote,
    onListenFromHere
}: TranscriptSelectionMenuProps) {
    if (!isOpen) return null;

    // Mobile View
    if (isMobile) {
        return (
            <div className="fixed bottom-6 right-6 flex flex-col gap-3 z-[10000]" onMouseDown={(e) => e.stopPropagation()}>
                <button
                    type="button"
                    className="h-14 w-14 rounded-full bg-carbon-900 text-white shadow-xl flex items-center justify-center hover:bg-carbon-950 transition-all active:scale-95"
                    onClick={onListenFromHere}
                    title="Listen from here"
                >
                    <Ear className="h-6 w-6" />
                </button>
                <button
                    type="button"
                    className="h-14 w-14 rounded-full bg-carbon-900 text-white shadow-xl flex items-center justify-center hover:bg-carbon-950 transition-all active:scale-95"
                    onClick={onAddNote}
                    title="Add note"
                >
                    <StickyNote className="h-6 w-6" />
                </button>
            </div>
        );
    }

    // Desktop View
    return (
        <div
            style={{
                position: 'fixed',
                left: position.x,
                top: position.y,
                transform: 'translate(-50%, -100%)',
                zIndex: 10000
            }}
            onMouseDown={(e) => e.stopPropagation()}
        >
            <div className="bg-[var(--bg-popover)] text-[var(--text-primary)] border border-[var(--border-subtle)] text-base font-medium rounded-full shadow-[var(--shadow-float)] px-5 py-2.5 flex items-center gap-3 pointer-events-auto hover:bg-[var(--bg-card)] transition-colors ring-0 transform hover:scale-105 duration-200">
                <button type="button" className="flex items-center gap-2 hover:text-[var(--brand-solid)] transition-colors" onClick={onAddNote}>
                    <Plus className="h-4 w-4" /> <span className="font-semibold text-sm">Add note</span>
                </button>
                <div className="w-px h-4 bg-[var(--border-subtle)]"></div>
                <button type="button" className="flex items-center gap-2 hover:text-[var(--brand-solid)] transition-colors" onClick={onListenFromHere}>
                    <Ear className="h-4 w-4" /> <span className="text-sm font-semibold">Listen</span>
                </button>
            </div>
        </div>
    );
}
