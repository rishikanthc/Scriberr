import { createContext, useContext, useState, useEffect } from "react";
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

	// Load token from localStorage on mount
	useEffect(() => {
		const savedToken = localStorage.getItem("scriberr_auth_token");
		if (savedToken) {
			// Validate token by checking if it's expired
			try {
				const payload = JSON.parse(atob(savedToken.split(".")[1]));
				const currentTime = Date.now() / 1000;
				
				if (payload.exp && payload.exp > currentTime) {
					setToken(savedToken);
				} else {
					// Token expired, remove it
					localStorage.removeItem("scriberr_auth_token");
				}
			} catch (error) {
				// Invalid token format, remove it
				console.error("Invalid token format:", error);
				localStorage.removeItem("scriberr_auth_token");
			}
		}
	}, []);

	const login = (newToken: string) => {
		setToken(newToken);
		localStorage.setItem("scriberr_auth_token", newToken);
	};

	const logout = () => {
		setToken(null);
		localStorage.removeItem("scriberr_auth_token");
	};

	const getAuthHeaders = () => {
		if (token) {
			return { Authorization: `Bearer ${token}` };
		}
		return {};
	};

	const value = {
		token,
		isAuthenticated: !!token,
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