import type { ReactNode } from "react";
import { useAuth } from "../contexts/AuthContext";
import { Login } from "../pages/Login";

interface ProtectedRouteProps {
	children: ReactNode;
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
	const { isAuthenticated, login } = useAuth();

	if (!isAuthenticated) {
		return <Login onLogin={login} />;
	}

	return <>{children}</>;
}