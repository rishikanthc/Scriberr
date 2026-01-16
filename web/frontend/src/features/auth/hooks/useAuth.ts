import { useEffect, useRef, useCallback } from 'react';
import { useAuthStore } from '../store/authStore';
import { refreshToken, navigateToHome } from '../../../lib/authHelpers';
import '../../../lib/authTypes';

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

        navigateToHome();
    }, [token, storeLogout]);


    const login = useCallback((newToken: string) => {
        setToken(newToken);
        setRequiresRegistration(false);
    }, [setToken, setRequiresRegistration]);

    useEffect(() => {
        if (tokenCheckIntervalRef.current) clearInterval(tokenCheckIntervalRef.current);

        if (token) {
            const checkTokenExpiry = async () => {
                if (!token) return;
                if (isTokenExpired(token)) {
                    const newToken = await refreshToken();
                    if (!newToken) {
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
    }, [token, isTokenExpired, logout]);

    useEffect(() => {
        const initializeAuth = async () => {
            if (isInitialized) return;

            try {
                const response = await fetch("/api/v1/auth/registration-status");
                if (response.ok) {
                    const data = await response.json();
                    const regEnabled = typeof data.registration_enabled === 'boolean'
                        ? data.registration_enabled
                        : !!data.requiresRegistration;
                    setRequiresRegistration(regEnabled);

                    if (!regEnabled && token && isTokenExpired(token)) {
                        const newToken = await refreshToken();
                        if (!newToken) {
                            logout();
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
    }, [isInitialized, setRequiresRegistration, setInitialized, token, isTokenExpired, logout]);

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
