import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AuthState {
    token: string | null;
    isAuthenticated: boolean;
    requiresRegistration: boolean;
    isInitialized: boolean;
    setToken: (token: string | null) => void;
    setRequiresRegistration: (requires: boolean) => void;
    setInitialized: (initialized: boolean) => void;
    logout: () => void;
}

export const useAuthStore = create<AuthState>()(
    persist(
        (set) => ({
            token: null,
            isAuthenticated: false,
            requiresRegistration: false,
            isInitialized: false,
            setToken: (token) => set({ token, isAuthenticated: !!token }),
            setRequiresRegistration: (requires) => set({ requiresRegistration: requires }),
            setInitialized: (initialized) => set({ isInitialized: initialized }),
            logout: () => {
                set({ token: null, isAuthenticated: false });
                localStorage.removeItem('auth-storage');
                // Optional: Call logout endpoint if needed, but side effects strictly in hooks/components usually better
            },
        }),
        {
            name: 'auth-storage',
            partialize: (state) => ({ token: state.token }), // Only persist token
        }
    )
);
