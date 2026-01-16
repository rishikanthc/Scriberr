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

    // Consolidated token management
    useEffect(() => {
        if (tokenCheckIntervalRef.current) clearInterval(tokenCheckIntervalRef.current);

        if (token) {
            const checkTokenExpiry = async () => {
                if (!token) return;
                if (isTokenExpired(token)) {
                    // Refresh is handled by the fetch interceptor
                    // We just need to trigger a request to an auth-protected endpoint if we want to force it
                    // or we can call the refresh endpoint directly here if we prefer to keep the timer.
                    try {
                        const originalFetch = window.__scriberr_original_fetch || window.fetch;
                        const res = await originalFetch('/api/v1/auth/refresh', { method: 'POST' });
                        if (res.ok) {
                            const data = await res.json();
                            if (data?.token) {
                                login(data.token);
                            }
                        } else {
                            logout();
                        }
                    } catch {
                        logout();
                    }
                }
            };
            tokenCheckIntervalRef.current = setInterval(checkTokenExpiry, 60000);
            checkTokenExpiry();
        }

        return () => {
            if (tokenCheckIntervalRef.current) clearInterval(tokenCheckIntervalRef.current);
        };
    }, [token, isTokenExpired, logout, login]);

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
                            // Fetch interceptor will handle the 401 if it happens, 
                            // but we can be proactive here.
                            const originalFetch = window.__scriberr_original_fetch || window.fetch;
                            const res = await originalFetch('/api/v1/auth/refresh', { method: 'POST' });
                            if (res.ok) {
                                const data = await res.json();
                                if (data?.token) login(data.token);
                            } else {
                                logout();
                            }
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
    }, [isInitialized, setRequiresRegistration, setInitialized, token, isTokenExpired, logout, login]);

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
