import { useAuthStore } from '../features/auth/store/authStore';
import { refreshToken, navigateToHome, parseRequestUrl } from './authHelpers';
import './authTypes';

export function setupAuthInterceptor(): void {
    if (window.__scriberr_original_fetch) {
        return;
    }

    const originalFetch = window.fetch.bind(window);
    window.__scriberr_original_fetch = originalFetch;

    const wrappedFetch: typeof window.fetch = async (input, init) => {
        const url = parseRequestUrl(input);
        const isAuthEndpoint = url.includes('/api/v1/auth/');

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

        let response = await originalFetch(input, requestInit);

        if (response.status === 401 && !isAuthEndpoint) {
            const newToken = await refreshToken();

            if (newToken) {
                const retryHeaders = new Headers(requestInit.headers);
                retryHeaders.set('Authorization', `Bearer ${newToken}`);
                const retryInit = { ...requestInit, headers: retryHeaders };

                response = await originalFetch(input, retryInit);
                if (response.status !== 401) return response;
            }

            state.logout();
            navigateToHome();
        }

        return response;
    };

    window.fetch = wrappedFetch;
}
