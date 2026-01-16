import { useAuthStore } from '../features/auth/store/authStore';
import './authTypes';

export async function refreshToken(): Promise<string | null> {
    const originalFetch = window.__scriberr_original_fetch || window.fetch;
    const state = useAuthStore.getState();

    try {
        const response = await originalFetch('/api/v1/auth/refresh', { method: 'POST' });
        if (!response.ok) return null;

        const data = await response.json();
        if (data?.token) {
            state.setToken(data.token);
            state.setRequiresRegistration(false);
            return data.token;
        }
        return null;
    } catch {
        return null;
    }
}

export function navigateToHome(): void {
    if (window.location.pathname !== "/") {
        window.history.pushState({ route: { path: 'home' } }, "", "/");
        window.dispatchEvent(new PopStateEvent('popstate', { state: { route: { path: 'home' } } }));
    }
}

export function parseRequestUrl(input: RequestInfo | URL): string {
    if (typeof input === 'string') return input;
    if (input instanceof URL) return input.href;
    return input.url;
}
