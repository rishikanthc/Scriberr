declare global {
    interface Window {
        __scriberr_original_fetch?: typeof window.fetch;
    }
}

export {};
