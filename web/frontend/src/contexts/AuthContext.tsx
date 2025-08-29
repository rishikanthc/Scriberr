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
    // Force navigate to login (home) for any unauthorized state
    if (window.location.pathname !== "/") {
      window.history.pushState({ route: { path: 'home' } }, "", "/");
      window.dispatchEvent(new PopStateEvent('popstate', { state: { route: { path: 'home' } } as any }));
    }
  }, [token]);

	// Check registration status and load token on mount
	useEffect(() => {
    const initializeAuth = async () => {
			try {
				// First, check if registration is required
				const response = await fetch("/api/v1/auth/registration-status");
				if (response.ok) {
                const data = await response.json();
                // Support both legacy and current API response shapes
                const regEnabled =
                  typeof data.registration_enabled === 'boolean'
                    ? data.registration_enabled
                    : !!data.requiresRegistration;
                setRequiresRegistration(regEnabled);
					
					// Only check for existing token if registration is not required
                    if (!regEnabled) {
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

  // Helper: attempt to refresh JWT via cookie refresh token
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
  }, [])

  // Globally intercept 401s: try refresh once, then retry original, else logout
  useEffect(() => {
    const originalFetch = window.fetch.bind(window);

    const wrappedFetch: typeof window.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
      try {
        let res = await originalFetch(input, init);
        if (res.status === 401) {
          // Try silent refresh once
          const newToken = await tryRefresh()
          if (newToken) {
            // Retry original request with updated Authorization header if provided
            const newInit: RequestInit | undefined = init ? { ...init } : undefined
            if (newInit?.headers && typeof newInit.headers === 'object') {
              (newInit.headers as any)['Authorization'] = `Bearer ${newToken}`
            }
            res = await originalFetch(input, newInit)
            if (res.status !== 401) return res
          }
          // Still unauthorized: force logout
          logout()
        }
        return res;
      } catch (err) {
        // Network or other errors just propagate
        throw err;
      }
    };

    window.fetch = wrappedFetch as any;
    return () => {
      window.fetch = originalFetch;
    };
  }, [logout, tryRefresh]);

  // Proactive refresh shortly before expiry
  useEffect(() => {
    if (!token) return
    const id = setInterval(async () => {
      if (token && isTokenExpired(token)) {
        const ok = await tryRefresh()
        if (!ok) {
          logout()
        }
      }
    }, 60_000)
    return () => clearInterval(id)
  }, [token, isTokenExpired, tryRefresh, logout])

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
