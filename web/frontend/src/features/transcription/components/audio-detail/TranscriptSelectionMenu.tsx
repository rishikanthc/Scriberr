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
            <div className="glass shadow-2xl rounded-full px-5 py-2.5 flex items-center gap-3 pointer-events-auto transform hover:scale-105 duration-200 border border-white/20 dark:border-white/10">
                <button
                    type="button"
                    className="flex items-center gap-2 text-carbon-900 dark:text-carbon-100 hover:text-brand-500 dark:hover:text-brand-400 transition-colors font-medium text-sm"
                    onClick={onAddNote}
                >
                    <Plus className="h-4 w-4" /> <span>Add note</span>
                </button>
                <div className="w-px h-4 bg-carbon-200 dark:bg-carbon-700"></div>
                <button
                    type="button"
                    className="flex items-center gap-2 text-carbon-900 dark:text-carbon-100 hover:text-brand-500 dark:hover:text-brand-400 transition-colors font-medium text-sm"
                    onClick={onListenFromHere}
                >
                    <Ear className="h-4 w-4" /> <span>Listen</span>
                </button>
            </div>
        </div>
    );
}
