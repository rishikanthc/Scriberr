import { useAuthStore } from '../features/auth/store/authStore';
import { refreshSession } from '../features/auth/api/authApi';
import './authTypes';

export async function refreshToken(): Promise<string | null> {
    const state = useAuthStore.getState();
    if (!state.refreshToken) return null;

    try {
        const session = await refreshSession(state.refreshToken);
        state.setSession(session);
        state.setRequiresRegistration(false);
        return session.accessToken;
    } catch {
        state.clearSession();
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
