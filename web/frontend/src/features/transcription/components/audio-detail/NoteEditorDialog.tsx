import { useState, useEffect } from "react";
import { useIsMobile } from "@/hooks/use-mobile";

interface NoteEditorDialogProps {
    isOpen: boolean;
    quote: string;
    position: { x: number; y: number };
    onSave: (content: string) => void;
    onCancel: () => void;
}

export function NoteEditorDialog({
    isOpen,
    quote,
    position,
    onSave,
    onCancel
}: NoteEditorDialogProps) {
    const isMobile = useIsMobile();
    const [content, setContent] = useState("");

    // Focus textarea on mount
    useEffect(() => {
        if (isOpen) {
            setContent("");
        }
    }, [isOpen]);

    if (!isOpen) return null;

    const style: React.CSSProperties = isMobile ? {
        position: 'fixed',
        left: '50%',
        top: '50%',
        transform: 'translate(-50%, -50%)',
        zIndex: 10001
    } : {
        position: 'fixed',
        left: position.x,
        top: position.y + 18,
        transform: 'translate(-50%, 0)',
        zIndex: 10001
    };

    // Safety check for desktop position to avoid going off-screen
    if (!isMobile) {
        if (style.left && (style.left as number) > window.innerWidth - 200) {
            style.left = window.innerWidth - 200;
        }
    }

    return (
        <div
            style={style}
            className="w-[min(90vw,520px)]"
            onMouseDown={(e) => e.stopPropagation()}
        >
            <div className="bg-white dark:bg-carbon-900 rounded-lg shadow-2xl p-3 pointer-events-auto border border-carbon-200 dark:border-carbon-800">
                <div className="text-xs text-carbon-500 dark:text-carbon-400 border-l-2 border-carbon-300 dark:border-carbon-600 pl-2 italic mb-2 max-h-32 overflow-auto">
                    {quote}
                </div>
                <textarea
                    className="w-full text-sm bg-transparent border rounded-md p-2 border-carbon-300 dark:border-carbon-700 text-carbon-900 dark:text-carbon-100 focus:outline-none focus:ring-2 focus:ring-carbon-500"
                    placeholder="Add a note..."
                    value={content}
                    onChange={e => setContent(e.target.value)}
                    rows={4}
                    autoFocus
                />
                <div className="mt-2 flex items-center justify-end gap-2">
                    <button
                        type="button"
                        className="px-2 py-1 text-sm rounded-md bg-carbon-200 dark:bg-carbon-700 hover:bg-carbon-300 dark:hover:bg-carbon-600 transition-colors"
                        onClick={onCancel}
                    >
                        Cancel
                    </button>
                    <button
                        type="button"
                        className="px-2 py-1 text-sm rounded-md bg-carbon-900 text-white hover:bg-carbon-950 transition-colors"
                        onClick={() => onSave(content)}
                        disabled={!content.trim()}
                    >
                        Save
                    </button>
                </div>
            </div>
        </div>
    );
}
