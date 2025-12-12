import { useCallback, useRef, useState } from 'react';

interface Options {
    shouldPreventDefault?: boolean;
    delay?: number;
}

export const useLongPress = (
    onLongPress: (e: React.MouseEvent | React.TouchEvent) => void,
    onClick: (e: React.MouseEvent | React.TouchEvent) => void,
    { shouldPreventDefault = true, delay = 500 }: Options = {}
) => {
    const [longPressTriggered, setLongPressTriggered] = useState(false);
    const timeout = useRef<NodeJS.Timeout | undefined>(undefined);
    const target = useRef<EventTarget | null>(null);

    const start = useCallback(
        (e: React.MouseEvent | React.TouchEvent) => {
            if (shouldPreventDefault && e.target) {
                target.current = e.target;
            }
            timeout.current = setTimeout(() => {
                onLongPress(e);
                setLongPressTriggered(true);
            }, delay);
        },
        [onLongPress, delay, shouldPreventDefault]
    );

    const clear = useCallback(
        (e: React.MouseEvent | React.TouchEvent, shouldTriggerClick = true) => {
            if (timeout.current) {
                clearTimeout(timeout.current);
            }
            if (shouldTriggerClick && !longPressTriggered) {
                onClick(e);
            }
            setLongPressTriggered(false);
            if (shouldPreventDefault && target.current) {
                target.current = null;
            }
        },
        [shouldPreventDefault, onClick, longPressTriggered]
    );

    return {
        onMouseDown: (e: React.MouseEvent) => start(e),
        onTouchStart: (e: React.TouchEvent) => start(e),
        onMouseUp: (e: React.MouseEvent) => clear(e),
        onMouseLeave: (e: React.MouseEvent) => clear(e, false),
        onTouchEnd: (e: React.TouchEvent) => clear(e),
    };
};
