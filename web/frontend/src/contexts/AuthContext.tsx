import { createContext, useContext, useState, useEffect, useCallback } from "react";
import type { ReactNode } from "react";

interface AuthContextType {
	token: string | null;
	isAuthenticated: boolean;
	login: (token: string) => void;
	logout: () => void;
	getAuthHeaders: () => { Authorization?: string };
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
	children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
	const [token, setToken] = useState<string | null>(null);
	const [isInitialized, setIsInitialized] = useState(false);

	// Check if token is expired
	const isTokenExpired = useCallback((token: string): boolean => {
		try {
			const payload = JSON.parse(atob(token.split(".")[1]));
			const currentTime = Date.now() / 1000;
			// Check if token will expire in the next 5 minutes
			return payload.exp && payload.exp <= (currentTime + 300);
		} catch (error) {
			console.error("Invalid token format:", error);
			return true;
		}
	}, []);

	// Logout function
	const logout = useCallback(() => {
		setToken(null);
		localStorage.removeItem("scriberr_auth_token");
		// Call logout endpoint to invalidate token server-side (optional)
		fetch("/api/v1/auth/logout", {
			method: "POST",
			headers: {
				"Authorization": token ? `Bearer ${token}` : "",
			},
		}).catch(() => {
			// Ignore errors in logout call
		});
	}, [token]);

	// Load token from localStorage on mount and setup auto-logout
	useEffect(() => {
		const savedToken = localStorage.getItem("scriberr_auth_token");
		if (savedToken) {
			if (isTokenExpired(savedToken)) {
				// Token expired, remove it
				localStorage.removeItem("scriberr_auth_token");
			} else {
				setToken(savedToken);
			}
		}
		setIsInitialized(true);
	}, [isTokenExpired]);

	// Setup auto-logout when token expires
	useEffect(() => {
		if (!token) return;

		const checkTokenExpiry = () => {
			if (isTokenExpired(token)) {
				console.log("Token expired, logging out...");
				logout();
			}
		};

		// Check token expiry every minute
		const interval = setInterval(checkTokenExpiry, 60000);
		
		// Cleanup interval on unmount or token change
		return () => clearInterval(interval);
	}, [token, isTokenExpired, logout]);

	const login = (newToken: string) => {
		setToken(newToken);
		localStorage.setItem("scriberr_auth_token", newToken);
	};

	const getAuthHeaders = () => {
		if (token) {
			return { Authorization: `Bearer ${token}` };
		}
		return {};
	};

	const value = {
		token,
		isAuthenticated: !!token && isInitialized,
		login,
		logout,
		getAuthHeaders,
	};

	// Show loading state until initialization is complete
	if (!isInitialized) {
		return (
			<div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
				<div className="text-center">
					<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
					<p className="mt-2 text-gray-600 dark:text-gray-400">Loading...</p>
				</div>
			</div>
		);
	}

	return (
		<AuthContext.Provider value={value}>
			{children}
		</AuthContext.Provider>
	);
}

export function useAuth() {
	const context = useContext(AuthContext);
	if (context === undefined) {
		throw new Error("useAuth must be used within an AuthProvider");
	}
	return context;
}