import { createContext, useContext, useState, useEffect, useCallback } from "react";
import type { ReactNode } from "react";

interface AuthContextType {
	token: string | null;
	isAuthenticated: boolean;
	requiresRegistration: boolean;
	isInitialized: boolean;
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
	const [requiresRegistration, setRequiresRegistration] = useState(false);

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

	// Check registration status and load token on mount
	useEffect(() => {
		const initializeAuth = async () => {
			try {
				// First, check if registration is required
				const response = await fetch("/api/v1/auth/registration-status");
				if (response.ok) {
					const data = await response.json();
					setRequiresRegistration(data.requiresRegistration);
					
					// Only check for existing token if registration is not required
					if (!data.requiresRegistration) {
						const savedToken = localStorage.getItem("scriberr_auth_token");
						if (savedToken) {
							if (isTokenExpired(savedToken)) {
								// Token expired, remove it
								localStorage.removeItem("scriberr_auth_token");
							} else {
								setToken(savedToken);
							}
						}
					}
				}
			} catch (error) {
				console.error("Failed to check registration status:", error);
				// If we can't check status, assume no registration needed and check token
				const savedToken = localStorage.getItem("scriberr_auth_token");
				if (savedToken) {
					if (isTokenExpired(savedToken)) {
						localStorage.removeItem("scriberr_auth_token");
					} else {
						setToken(savedToken);
					}
				}
			} finally {
				setIsInitialized(true);
			}
		};

		initializeAuth();
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
		setRequiresRegistration(false); // Clear registration requirement after successful login/registration
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
		requiresRegistration,
		isInitialized,
		login,
		logout,
		getAuthHeaders,
	};


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