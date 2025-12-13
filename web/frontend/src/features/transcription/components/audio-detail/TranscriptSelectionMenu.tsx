import { Plus, Ear } from "lucide-react";
import type { SelectionMenuState } from "@/features/transcription/hooks/useSelectionMenu";

interface TranscriptSelectionMenuProps {
    menuState: SelectionMenuState | null;
    onAddNote: () => void;
    onListenFromHere: () => void;
}

export function TranscriptSelectionMenu({
    menuState,
    onAddNote,
    onListenFromHere
}: TranscriptSelectionMenuProps) {
    if (!menuState || !menuState.visible) return null;

    // Unified Floating Pill Menu (same design for desktop and mobile)
    return (
        <div
            style={{
                position: 'fixed',
                left: menuState.x,
                top: menuState.y,
                transform: 'translate(-50%, -100%)',
                zIndex: 10000,
                // Ensure native selection handles are not blocked on touch devices
                pointerEvents: 'auto'
            }}
            // Prevent touches/clicks on the menu from clearing the selection
            onMouseDown={(e) => e.stopPropagation()}
            onTouchStart={(e) => e.stopPropagation()}
        >
            {/* Menu Bubble */}
            <div className="glass shadow-2xl rounded-full px-4 py-2 flex items-center gap-2 pointer-events-auto transform hover:scale-105 duration-200 border border-white/20 dark:border-white/10">
                <button
                    type="button"
                    className="flex items-center gap-1.5 text-carbon-900 dark:text-carbon-100 hover:text-brand-500 dark:hover:text-brand-400 transition-colors font-medium text-sm px-1 py-0.5 active:scale-95"
                    onClick={(e) => {
                        e.stopPropagation();
                        onAddNote();
                    }}
                >
                    <Plus className="h-4 w-4" /> <span>Note</span>
                </button>
                <div className="w-px h-4 bg-carbon-200 dark:bg-carbon-700" />
                <button
                    type="button"
                    className="flex items-center gap-1.5 text-carbon-900 dark:text-carbon-100 hover:text-brand-500 dark:hover:text-brand-400 transition-colors font-medium text-sm px-1 py-0.5 active:scale-95"
                    onClick={(e) => {
                        e.stopPropagation();
                        onListenFromHere();
                    }}
                >
                    <Ear className="h-4 w-4" /> <span>Listen</span>
                </button>
            </div>

            {/* Little Arrow pointing down to the selection */}
            <div
                className="w-3 h-3 bg-white/80 dark:bg-carbon-800/90 rotate-45 mx-auto -mt-1.5 shadow-sm border-r border-b border-white/20 dark:border-white/10"
            />
        </div>
    );
}

