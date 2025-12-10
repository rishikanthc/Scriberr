import { useEffect, useRef, useCallback } from 'react';
import { useAuthStore } from '../store/authStore';

export function useAuth() {
    const {
        token,
        isAuthenticated,
        requiresRegistration,
        isInitialized,
        setToken,
        setRequiresRegistration,
        setInitialized,
        logout: storeLogout
    } = useAuthStore();

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
            window.dispatchEvent(new PopStateEvent('popstate', { state: { route: { path: 'home' } } as any }));
        }
    }, [token, storeLogout]);


    const login = useCallback((newToken: string) => {
        setToken(newToken);
        setRequiresRegistration(false);
    }, [setToken, setRequiresRegistration]);


    const tryRefresh = useCallback(async (): Promise<string | null> => {
        try {
            const res = await fetch('/api/v1/auth/refresh', { method: 'POST' })
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
            const originalFetch = window.fetch.bind(window);
            const wrappedFetch: typeof window.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
                try {
                    let res = await originalFetch(input, init);
                    if (res.status === 401) {
                        const newToken = await tryRefresh()
                        if (newToken) {
                            const newInit: RequestInit | undefined = init ? { ...init } : undefined
                            if (newInit?.headers && typeof newInit.headers === 'object') {
                                (newInit.headers as any)['Authorization'] = `Bearer ${newToken}`
                            }
                            res = await originalFetch(input, newInit)
                            if (res.status !== 401) return res
                        }
                        logout()
                    }
                    return res;
                } catch (err) {
                    throw err;
                }
            };
            window.fetch = wrappedFetch as any;
            fetchWrapperSetupRef.current = true;
            return () => { window.fetch = originalFetch; };
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
