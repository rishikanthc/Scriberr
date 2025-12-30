import { useEffect, useRef, useState, type ReactNode } from "react";
import { motion, useAnimation, type PanInfo } from "framer-motion";
import { Trash2, Wand2, StopCircle } from "lucide-react";
import { WandAdvancedIcon } from "@/components/icons/WandAdvancedIcon";
import { useIsMobile } from "@/hooks/use-mobile";

interface SwipeableItemProps {
    children: ReactNode;
    onTranscribe: () => void;
    onTranscribeAdvanced: () => void;
    onDelete: () => void;
    onStop?: () => void;
    isProcessing?: boolean;
    isSelectionMode?: boolean;
    shouldShowHint?: boolean;
    onHintComplete?: () => void;
    onSwipeStateChange?: (isSwiping: boolean) => void;
}

// Distance (in px) that cancels a long-press
const LONG_PRESS_CANCEL_DISTANCE = 8;

/**
 * A swipeable card wrapper that reveals action buttons when swiped left.
 * Only active on mobile devices - on desktop, the card is static.
 * 
 * Gesture coordination:
 * - Movement > LONG_PRESS_CANCEL_DISTANCE cancels long-press
 * - Movement > SWIPE_THRESHOLD registers as intentional swipe
 * - After swipe, clicks are suppressed for a short window
 */
export function SwipeableItem({
    children,
    onTranscribe,
    onTranscribeAdvanced,
    onDelete,
    onStop,
    isProcessing = false,
    isSelectionMode = false,
    shouldShowHint = false,
    onHintComplete,
    onSwipeStateChange,
}: SwipeableItemProps) {
    const controls = useAnimation();
    const isMobile = useIsMobile();

    // Gesture state tracking
    const [isOpen, setIsOpen] = useState(false);
    const isDraggingRef = useRef(false);
    const dragStartRef = useRef<{ x: number; y: number } | null>(null);
    const hasMovedRef = useRef(false);
    const suppressClickUntilRef = useRef(0);

    // Width of the action buttons area (3 buttons × 44px + gaps + padding)
    const OPEN_WIDTH = -160;

    const handleDragStart = (_event: MouseEvent | TouchEvent | PointerEvent, info: PanInfo) => {
        isDraggingRef.current = true;
        dragStartRef.current = { x: info.point.x, y: info.point.y };
        hasMovedRef.current = false;
        onSwipeStateChange?.(true);
    };

    const handleDrag = (_event: MouseEvent | TouchEvent | PointerEvent, info: PanInfo) => {
        if (!dragStartRef.current) return;

        const deltaX = Math.abs(info.point.x - dragStartRef.current.x);
        const deltaY = Math.abs(info.point.y - dragStartRef.current.y);

        // Mark as "moved" if beyond threshold (used to cancel long-press and prevent click)
        if (deltaX > LONG_PRESS_CANCEL_DISTANCE || deltaY > LONG_PRESS_CANCEL_DISTANCE) {
            hasMovedRef.current = true;
        }
    };

    const handleDragEnd = async (_event: MouseEvent | TouchEvent | PointerEvent, info: PanInfo) => {
        const offset = info.offset.x;
        const velocity = info.velocity.x;

        isDraggingRef.current = false;
        onSwipeStateChange?.(false);

        // Only suppress click if there was meaningful movement
        if (hasMovedRef.current) {
            suppressClickUntilRef.current = Date.now() + 100;
        }

        // Open if dragged far enough OR flicked fast enough to the left
        if (offset < OPEN_WIDTH / 2 || velocity < -500) {
            setIsOpen(true);
            await controls.start({
                x: OPEN_WIDTH,
                transition: { type: "spring", stiffness: 400, damping: 25 }
            });
        } else {
            // Snap back closed
            setIsOpen(false);
            await controls.start({
                x: 0,
                transition: { type: "spring", stiffness: 400, damping: 25 }
            });
        }

        dragStartRef.current = null;
        hasMovedRef.current = false;
    };

    // Close the drawer programmatically
    const close = async () => {
        setIsOpen(false);
        await controls.start({
            x: 0,
            transition: { type: "spring", stiffness: 400, damping: 25 }
        });
    };

    // Check if a click should be suppressed (because it followed a swipe)
    const shouldSuppressClick = () => {
        return Date.now() < suppressClickUntilRef.current || isDraggingRef.current;
    };

    // Check if the component has moved (for canceling long-press)
    const hasMoved = () => hasMovedRef.current;

    // Expose gesture utilities through data attributes for parent coordination
    useEffect(() => {
        // We use a custom event to communicate gesture state to parent
        // This is a more React-friendly way than imperative refs
    }, []);

    // Discovery nudge animation for first-time visitors
    useEffect(() => {
        if (shouldShowHint && isMobile) {
            const runNudge = async () => {
                // Wait for page to settle
                await new Promise(resolve => setTimeout(resolve, 800));
                // Nudge left 40px
                await controls.start({ x: -40, transition: { duration: 0.3 } });
                // Snap back with bounce
                await controls.start({ x: 0, transition: { type: "spring", bounce: 0.4 } });
                // Mark hint as shown
                onHintComplete?.();
            };
            runNudge();
        }
    }, [shouldShowHint, isMobile, controls, onHintComplete]);

    // Handle action button clicks - close drawer after action
    const handleAction = (action: () => void) => {
        close();
        action();
    };

    // Close drawer when clicking outside (on the card content) while drawer is open
    const handleContentClick = (e: React.MouseEvent) => {
        if (isOpen) {
            e.stopPropagation();
            close();
            return;
        }
    };

    return (
        <div
            className="relative group"
            data-swipeable="true"
            data-suppress-click={shouldSuppressClick()}
            data-has-moved={hasMoved()}
        >
            {/* --- LAYER 0: The Action Buttons (Hidden underneath, mobile only) --- */}
            <div className="absolute inset-y-0 right-0 w-[170px] flex items-center justify-end pr-2 gap-2 rounded-2xl z-0 md:hidden">
                {/* Transcribe (Primary) */}
                <button
                    onClick={() => handleAction(onTranscribe)}
                    className="w-11 h-11 flex items-center justify-center rounded-full bg-[var(--brand-light)] text-[var(--brand-solid)] shadow-sm active:scale-95 transition-transform cursor-pointer"
                    aria-label="Transcribe"
                >
                    <Wand2 size={18} />
                </button>

                {/* Transcribe Advanced (Secondary) */}
                <button
                    onClick={() => handleAction(onTranscribeAdvanced)}
                    className="w-11 h-11 flex items-center justify-center rounded-full bg-gray-100 text-gray-600 shadow-sm active:scale-95 transition-transform cursor-pointer"
                    aria-label="Transcribe Advanced"
                >
                    <WandAdvancedIcon className="h-[18px] w-[18px]" />
                </button>

                {/* Delete or Stop (Destructive - furthest right) */}
                {isProcessing && onStop ? (
                    <button
                        onClick={() => handleAction(onStop)}
                        className="w-11 h-11 flex items-center justify-center rounded-full bg-amber-50 text-amber-600 shadow-sm active:scale-95 transition-transform cursor-pointer"
                        aria-label="Stop Transcription"
                    >
                        <StopCircle size={18} />
                    </button>
                ) : (
                    <button
                        onClick={() => handleAction(onDelete)}
                        className="w-11 h-11 flex items-center justify-center rounded-full bg-red-50 text-[var(--error)] shadow-sm active:scale-95 transition-transform cursor-pointer"
                        aria-label="Delete"
                    >
                        <Trash2 size={18} />
                    </button>
                )}
            </div>

            {/* --- LAYER 1: The Content Card (Draggable on mobile) --- */}
            <motion.div
                // Disable drag on desktop or when in selection mode
                drag={isMobile && !isSelectionMode ? "x" : false}
                dragConstraints={{ left: OPEN_WIDTH, right: 0 }}
                dragElastic={0.1} // Rubber band resistance
                dragDirectionLock={true}
                // Add drag threshold to distinguish from taps
                dragSnapToOrigin={false}
                onDragStart={handleDragStart}
                onDrag={handleDrag}
                onDragEnd={handleDragEnd}
                animate={controls}
                whileTap={isMobile && !isSelectionMode ? { scale: 0.98 } : undefined}
                onClick={handleContentClick}
                style={{
                    touchAction: isMobile ? "pan-y" : "auto",
                    x: 0
                }}
                className="relative z-10"
            >
                {children}
            </motion.div>
        </div>
    );
}
