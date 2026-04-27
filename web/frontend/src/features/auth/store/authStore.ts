import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { AuthSession, AuthUser } from '../api/authApi';

interface AuthState {
    token: string | null;
    refreshToken: string | null;
    user: AuthUser | null;
    isAuthenticated: boolean;
    requiresRegistration: boolean;
    isInitialized: boolean;
    setToken: (token: string | null) => void;
    setSession: (session: AuthSession) => void;
    setRequiresRegistration: (requires: boolean) => void;
    setInitialized: (initialized: boolean) => void;
    clearSession: () => void;
    logout: () => void;
}

export const useAuthStore = create<AuthState>()(
    persist(
        (set) => ({
            token: null,
            refreshToken: null,
            user: null,
            isAuthenticated: false,
            requiresRegistration: false,
            isInitialized: false,
            setToken: (token) => set({ token, isAuthenticated: !!token }),
            setSession: (session) => set({
                token: session.accessToken,
                refreshToken: session.refreshToken,
                user: session.user,
                isAuthenticated: true,
            }),
            setRequiresRegistration: (requires) => set({ requiresRegistration: requires }),
            setInitialized: (initialized) => set({ isInitialized: initialized }),
            clearSession: () => set({ token: null, refreshToken: null, user: null, isAuthenticated: false }),
            logout: () => {
                set({ token: null, refreshToken: null, user: null, isAuthenticated: false });
                localStorage.removeItem('auth-storage');
            },
        }),
        {
            name: 'auth-storage',
            partialize: (state) => ({
                token: state.token,
                refreshToken: state.refreshToken,
                user: state.user,
            }),
        }
    )
);
