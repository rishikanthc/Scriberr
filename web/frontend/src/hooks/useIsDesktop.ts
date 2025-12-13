import { useState, useEffect } from 'react';

/**
 * Detects if the user is on a device with a fine pointer (mouse/trackpad).
 * Uses the CSS Media Interaction API instead of User Agent strings.
 * 
 * @returns true if the user has a precise pointer (mouse), false for touch devices
 */
export function useIsDesktop(): boolean {
    // Default to true for SSR, check on mount
    const [isDesktop, setIsDesktop] = useState(true);

    useEffect(() => {
        // 'pointer: fine' typically means a mouse or trackpad
        const media = window.matchMedia('(pointer: fine)');

        const update = (e: MediaQueryListEvent) => setIsDesktop(e.matches);

        // Set initial value
        setIsDesktop(media.matches);

        // Listen for changes (e.g., detaching a tablet keyboard/mouse)
        media.addEventListener('change', update);
        return () => media.removeEventListener('change', update);
    }, []);

    return isDesktop;
}
