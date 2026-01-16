import { useAuthStore } from '../features/auth/store/authStore';

declare global {
    interface Window {
        __scriberr_original_fetch?: typeof window.fetch;
    }
}

export function setupAuthInterceptor() {
    if (window.__scriberr_original_fetch) {
        return; // Already setup
    }

    const originalFetch = window.fetch.bind(window);
    window.__scriberr_original_fetch = originalFetch;

    const wrappedFetch: typeof window.fetch = async (input, init) => {
        const url = typeof input === 'string' ? input : (input instanceof URL ? input.href : input.url);
        const isAuthEndpoint = url.includes('/api/v1/auth/');

        // 1. Auto-inject Authorization header if not present and we have a token
        const state = useAuthStore.getState();
        const token = state.token;

        let requestInit = init || {};
        if (token && !isAuthEndpoint) {
            const headers = new Headers(requestInit.headers);
            if (!headers.has('Authorization')) {
                headers.set('Authorization', `Bearer ${token}`);
                requestInit = { ...requestInit, headers };
            }
        }

        // 2. Perform the request
        let res = await originalFetch(input, requestInit);

        // 3. Handle 401 Unauthorized
        if (res.status === 401 && !isAuthEndpoint) {
            // Try to refresh the token
            const newToken = await tryRefresh();
            
            if (newToken) {
                // Retry the original request with the new token
                const retryInit = { ...requestInit };
                const retryHeaders = new Headers(retryInit.headers);
                retryHeaders.set('Authorization', `Bearer ${newToken}`);
                retryInit.headers = retryHeaders;
                
                res = await originalFetch(input, retryInit);
                if (res.status !== 401) return res;
            }

            // If refresh failed or retry still 401, logout
            state.logout();
            if (window.location.pathname !== "/") {
                window.history.pushState({ route: { path: 'home' } }, "", "/");
                window.dispatchEvent(new PopStateEvent('popstate', { state: { route: { path: 'home' } } }));
            }
        }

        return res;
    };

    window.fetch = wrappedFetch;
}

async function tryRefresh(): Promise<string | null> {
    const originalFetch = window.__scriberr_original_fetch || window.fetch;
    const state = useAuthStore.getState();

    try {
        const res = await originalFetch('/api/v1/auth/refresh', { method: 'POST' });
        if (!res.ok) return null;

        const data = await res.json();
        if (data?.token) {
            state.setToken(data.token);
            state.setRequiresRegistration(false);
            return data.token as string;
        }
        return null;
    } catch {
        return null;
    }
}
