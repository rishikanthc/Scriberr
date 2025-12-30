import { useEffect, useState } from 'react';

const STORAGE_KEY = 'scriberr_swipe_hint_shown';

/**
 * Hook to manage the first-visit swipe discovery hint.
 * Returns whether to show the "nudge" animation and a function to mark it as shown.
 */
export function useSwipeHint() {
    const [shouldShowHint, setShouldShowHint] = useState(false);

    useEffect(() => {
        // Check if hint has been shown before
        const hasShown = localStorage.getItem(STORAGE_KEY);
        if (!hasShown) {
            setShouldShowHint(true);
        }
    }, []);

    const markHintShown = () => {
        localStorage.setItem(STORAGE_KEY, 'true');
        setShouldShowHint(false);
    };

    return { shouldShowHint, markHintShown };
}
