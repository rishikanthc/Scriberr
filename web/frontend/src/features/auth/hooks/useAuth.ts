import { useEffect, useRef, useCallback } from 'react';
import { useAuthStore } from '../store/authStore';
import { refreshToken, navigateToHome } from '../../../lib/authHelpers';
import '../../../lib/authTypes';
import { getCurrentUser, getRegistrationStatus, logoutSession, type AuthSession } from '../api/authApi';

export function useAuth() {
    const {
        token,
        refreshToken: storedRefreshToken,
        user,
        requiresRegistration,
        isInitialized,
        setToken,
        setSession,
        setRequiresRegistration,
        setInitialized,
        clearSession,
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
        const refreshTokenToRevoke = useAuthStore.getState().refreshToken;
        storeLogout();
        logoutSession(refreshTokenToRevoke).catch(() => { });

        navigateToHome();
    }, [storeLogout]);


    const login = useCallback((sessionOrToken: AuthSession | string) => {
        if (typeof sessionOrToken === "string") {
            setToken(sessionOrToken);
        } else {
            setSession(sessionOrToken);
        }
        setRequiresRegistration(false);
    }, [setToken, setSession, setRequiresRegistration]);

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
                const registrationEnabled = await getRegistrationStatus();
                setRequiresRegistration(registrationEnabled);

                if (registrationEnabled) {
                    clearSession();
                    return;
                }

                if (!token) return;

                if (isTokenExpired(token)) {
                    const newToken = await refreshToken();
                    if (!newToken) clearSession();
                    return;
                }

                const currentUser = await getCurrentUser(token);
                const currentState = useAuthStore.getState();
                if (currentState.token && currentState.refreshToken) {
                    setSession({
                        accessToken: currentState.token,
                        refreshToken: currentState.refreshToken,
                        user: currentUser,
                    });
                }
            } catch (error) {
                console.error("Failed check reg status", error);
                if (token) clearSession();
            } finally {
                setInitialized(true);
            }
        };
        initializeAuth();
    }, [isInitialized, setRequiresRegistration, setInitialized, token, isTokenExpired, clearSession, setSession]);

    return {
        token,
        refreshToken: storedRefreshToken,
        user,
        isAuthenticated,
        requiresRegistration,
        isInitialized,
        login,
        logout,
        getAuthHeaders
    };
}
