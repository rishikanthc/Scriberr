import type { ReactNode } from "react";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { Login } from "@/features/auth/components/Login";
import { Register } from "@/features/auth/components/Register";

interface ProtectedRouteProps {
	children: ReactNode;
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
	const { isAuthenticated, requiresRegistration, isInitialized, login } = useAuth();

	// Show loading while initializing
	if (!isInitialized) {
		return (
			<div className="scr-app flex items-center justify-center">
				<div className="text-center">
					<div className="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-[var(--scr-brand-muted)] border-b-[var(--scr-brand-solid)]"></div>
					<p className="mt-2 text-[var(--scr-text-secondary)]">Loading...</p>
				</div>
			</div>
		);
	}

	// Show registration form if no users exist
	if (requiresRegistration) {
		return <Register onRegister={login} />;
	}

	// Show login form if not authenticated
	if (!isAuthenticated) {
		return <Login onLogin={login} />;
	}

	// Show protected content if authenticated
	return <>{children}</>;
}
