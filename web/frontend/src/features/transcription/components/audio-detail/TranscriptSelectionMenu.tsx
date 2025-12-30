import { Plus, Ear } from "lucide-react";
import type { SelectionMenuState } from "@/features/transcription/hooks/useSelectionMenu";
import { useIsDesktop } from "@/hooks/useIsDesktop";

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
    const isDesktop = useIsDesktop();

    if (!menuState || !menuState.visible) return null;

    // MOBILE: Fixed bottom-center bar (completely out of the way of selection)
    if (!isDesktop) {
        return (
            <div
                className="fixed bottom-0 left-0 right-0 z-[10000] flex justify-center pb-6 pt-2"
                style={{
                    // Ensure this container doesn't block touch events on the page
                    pointerEvents: 'none',
                    // Safe area for devices with home indicators
                    paddingBottom: 'max(1.5rem, env(safe-area-inset-bottom))'
                }}
            >
                {/* The Pill - only this is interactive */}
                <div
                    className="glass shadow-2xl rounded-full px-5 py-3 flex items-center gap-3 border border-white/20 dark:border-white/10 animate-in fade-in slide-in-from-bottom-4 duration-300"
                    style={{ pointerEvents: 'auto' }}
                    onTouchStart={(e) => e.stopPropagation()}
                >
                    <button
                        type="button"
                        className="flex items-center gap-2 text-carbon-900 dark:text-carbon-100 hover:text-brand-500 dark:hover:text-brand-400 transition-colors font-medium text-sm px-2 py-1 active:scale-95"
                        onClick={(e) => {
                            e.stopPropagation();
                            onAddNote();
                        }}
                    >
                        <Plus className="h-5 w-5" /> <span>Add Note</span>
                    </button>
                    <div className="w-px h-5 bg-carbon-200 dark:bg-carbon-700" />
                    <button
                        type="button"
                        className="flex items-center gap-2 text-carbon-900 dark:text-carbon-100 hover:text-brand-500 dark:hover:text-brand-400 transition-colors font-medium text-sm px-2 py-1 active:scale-95"
                        onClick={(e) => {
                            e.stopPropagation();
                            onListenFromHere();
                        }}
                    >
                        <Ear className="h-5 w-5" /> <span>Listen</span>
                    </button>
                </div>
            </div>
        );
    }

    // DESKTOP: Position above the selection
    return (
        <div
            style={{
                position: 'fixed',
                left: menuState.x,
                top: menuState.y,
                transform: 'translate(-50%, -100%)',
                zIndex: 10000,
                pointerEvents: 'auto'
            }}
            onMouseDown={(e) => e.stopPropagation()}
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

            {/* Arrow pointing DOWN to the selection */}
            <div
                className="w-3 h-3 bg-white/80 dark:bg-carbon-800/90 rotate-45 mx-auto -mt-1.5 shadow-sm border-r border-b border-white/20 dark:border-white/10"
            />
        </div>
    );
}


