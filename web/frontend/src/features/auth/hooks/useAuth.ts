import { useEffect, useRef, useCallback } from 'react';
import { useAuthStore } from '../store/authStore';

declare global {
    interface Window {
        __scriberr_original_fetch?: typeof window.fetch;
    }
}

export function useAuth() {
    const {
        token,
        requiresRegistration,
        isInitialized,
        setToken,
        setRequiresRegistration,
        setInitialized,
        logout: storeLogout
    } = useAuthStore();

    const isAuthenticated = !!token;

    const tokenCheckIntervalRef = useRef<NodeJS.Timeout | null>(null);
    const fetchWrapperSetupRef = useRef(false);

    const getAuthHeaders = useCallback((): Record<string, string> => {
        if (token) {
            return { Authorization: `Bearer ${token}` };
        }
        return {};
    }, [token]);

    // Memoize expensive token expiry check
    const isTokenExpired = useCallback((tokenToCheck: string): boolean => {
        try {
            const payload = JSON.parse(atob(tokenToCheck.split(".")[1]));
            const currentTime = Date.now() / 1000;
            // Check if token will expire in the next 5 minutes
            return payload.exp && payload.exp <= (currentTime + 300);
        } catch (error) {
            console.error("Invalid token format:", error);
            return true;
        }
    }, []);

    const logout = useCallback(() => {
        storeLogout();
        fetch("/api/v1/auth/logout", {
            method: "POST",
            headers: {
                "Authorization": token ? `Bearer ${token}` : "",
            },
        }).catch(() => { });

        if (window.location.pathname !== "/") {
            // Force navigation handled by RouterContext or window.location if critical
            window.history.pushState({ route: { path: 'home' } }, "", "/");
            window.dispatchEvent(new PopStateEvent('popstate', { state: { route: { path: 'home' } } }));
        }
    }, [token, storeLogout]);


    const login = useCallback((newToken: string) => {
        setToken(newToken);
        setRequiresRegistration(false);
    }, [setToken, setRequiresRegistration]);


    const tryRefresh = useCallback(async (): Promise<string | null> => {
        try {
            const fetchToUse = window.__scriberr_original_fetch || window.fetch;
            const res = await fetchToUse('/api/v1/auth/refresh', { method: 'POST' })
            if (!res.ok) return null
            const data = await res.json()
            if (data?.token) {
                login(data.token)
                return data.token as string
            }
            return null
        } catch {
            return null
        }
    }, [login])


    // Consolidated token management
    useEffect(() => {
        if (!fetchWrapperSetupRef.current) {
            if (!window.__scriberr_original_fetch) {
                window.__scriberr_original_fetch = window.fetch.bind(window);
            }

            const originalFetch = window.__scriberr_original_fetch!;
            const wrappedFetch: typeof window.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
                const url = typeof input === 'string' ? input : (input instanceof URL ? input.href : input.url);
                const isAuthEndpoint = url.includes('/api/v1/auth/');

                let res = await originalFetch(input, init);
                if (res.status === 401 && !isAuthEndpoint) {
                    const newToken = await tryRefresh()
                    if (newToken) {
                        const newInit: RequestInit = init ? { ...init } : {};
                        const headers = new Headers(newInit.headers);
                        headers.set('Authorization', `Bearer ${newToken}`);
                        newInit.headers = headers;

                        res = await originalFetch(input, newInit)
                        if (res.status !== 401) return res
                    }
                    logout()
                }
                return res;
            };
            window.fetch = wrappedFetch;
            fetchWrapperSetupRef.current = true;
            // Note: We don't restore originalFetch on unmount because other components
            // also use useAuth and expect the wrapped version. This is a bit hacky
            // but safer than multiple re-wrapping/unwrapping.
        }

        if (tokenCheckIntervalRef.current) clearInterval(tokenCheckIntervalRef.current);

        if (token) {
            const checkTokenExpiry = async () => {
                if (!token) return;
                if (isTokenExpired(token)) {
                    const newToken = await tryRefresh();
                    if (!newToken) logout();
                }
            };
            tokenCheckIntervalRef.current = setInterval(checkTokenExpiry, 60000);
            checkTokenExpiry();
        }

        return () => {
            if (tokenCheckIntervalRef.current) clearInterval(tokenCheckIntervalRef.current);
        };
    }, [token, isTokenExpired, logout, tryRefresh]);

    // Initial check (equivalent to old AuthProvider mount effect)
    useEffect(() => {
        const initializeAuth = async () => {
            if (isInitialized) return; // Don't run if already initialized

            try {
                const response = await fetch("/api/v1/auth/registration-status");
                if (response.ok) {
                    const data = await response.json();
                    const regEnabled = typeof data.registration_enabled === 'boolean' ? data.registration_enabled : !!data.requiresRegistration;
                    setRequiresRegistration(regEnabled);

                    if (!regEnabled) {
                        // Check token validity if present
                        if (token && isTokenExpired(token)) {
                            // Try refresh or logout
                            const Refreshed = await tryRefresh();
                            if (!Refreshed) logout();
                        }
                    }
                }
            } catch (error) {
                console.error("Failed check reg status", error);
            } finally {
                setInitialized(true);
            }
        };
        initializeAuth();
    }, [isInitialized, setRequiresRegistration, setInitialized, token, isTokenExpired, tryRefresh, logout]);

    return {
        token,
        isAuthenticated,
        requiresRegistration,
        isInitialized,
        login,
        logout,
        getAuthHeaders
    };
}
