import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScriberrLogo } from "@/components/ScriberrLogo";
import { useNavigate } from "react-router-dom";
import { ThemeSwitcher } from "@/components/ThemeSwitcher";
import { Loader2, AlertCircle } from "lucide-react";

interface LoginProps {
	onLogin: (token: string) => void;
}

export function Login({ onLogin }: LoginProps) {
	const navigate = useNavigate();
	const [username, setUsername] = useState("");
	const [password, setPassword] = useState("");
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState("");

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		setError("");
		setLoading(true);

		try {
			const response = await fetch("/api/v1/auth/login", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
				},
				body: JSON.stringify({
					username,
					password,
				}),
			});

			if (response.ok) {
				const data = await response.json();
				onLogin(data.token);
			} else {
				const error = await response.json();
				setError(error.error || "Login failed");
			}
		} catch (error) {
			console.error("Login error:", error);
			setError("Network error. Please try again.");
		} finally {
			setLoading(false);
		}
	};

	const isFormValid = username.trim() && password.trim();

	return (
		<div className="min-h-screen bg-[var(--bg-main)] flex flex-col">
			{/* Theme Switcher - Top Right */}
			<div className="absolute top-6 right-6 z-10">
				<ThemeSwitcher />
			</div>

			{/* Main Content - Centered */}
			<div className="flex-1 flex items-center justify-center px-6 py-12">
				<div className="w-full max-w-sm space-y-8">
					{/* Logo & Header */}
					<div className="text-center space-y-6">
						<div
							className="flex justify-center cursor-pointer transition-transform hover:scale-105 active:scale-95"
							onClick={() => navigate('/')}
						>
							<ScriberrLogo />
						</div>

						<div className="space-y-2">
							<h1 className="text-2xl font-semibold text-[var(--text-primary)] tracking-tight">
								Welcome back
							</h1>
							<p className="text-[var(--text-secondary)] text-sm">
								Sign in to continue to your workspace
							</p>
						</div>
					</div>

					{/* Login Form Card */}
					<div
						className="bg-[var(--bg-card)] rounded-2xl border border-[var(--border-subtle)] p-6 space-y-6"
						style={{ boxShadow: 'var(--shadow-card)' }}
					>
						<form onSubmit={handleSubmit} className="space-y-5">
							{/* Error Message */}
							{error && (
								<div className="flex items-start gap-3 bg-red-50 dark:bg-red-950/50 border border-red-200 dark:border-red-900 rounded-xl p-3.5">
									<AlertCircle className="h-5 w-5 text-[var(--error)] shrink-0 mt-0.5" />
									<p className="text-[var(--error)] text-sm">{error}</p>
								</div>
							)}

							{/* Username Field */}
							<div className="space-y-2">
								<Label
									htmlFor="username"
									className="text-sm font-medium text-[var(--text-primary)]"
								>
									Username
								</Label>
								<Input
									id="username"
									type="text"
									placeholder="Enter your username"
									value={username}
									onChange={(e) => setUsername(e.target.value)}
									disabled={loading}
									required
									autoComplete="username"
									className="h-11 bg-[var(--bg-main)] border-[var(--border-subtle)] rounded-xl
										text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)]
										focus:border-[var(--brand-solid)] focus:ring-2 focus:ring-[var(--brand-solid)]/20
										transition-all duration-200"
								/>
							</div>

							{/* Password Field */}
							<div className="space-y-2">
								<Label
									htmlFor="password"
									className="text-sm font-medium text-[var(--text-primary)]"
								>
									Password
								</Label>
								<Input
									id="password"
									type="password"
									placeholder="Enter your password"
									value={password}
									onChange={(e) => setPassword(e.target.value)}
									disabled={loading}
									required
									autoComplete="current-password"
									className="h-11 bg-[var(--bg-main)] border-[var(--border-subtle)] rounded-xl
										text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)]
										focus:border-[var(--brand-solid)] focus:ring-2 focus:ring-[var(--brand-solid)]/20
										transition-all duration-200"
								/>
							</div>

							{/* Submit Button */}
							<Button
								type="submit"
								disabled={loading || !isFormValid}
								className="w-full h-11 rounded-xl font-medium text-white
									bg-gradient-to-r from-[#FFAB40] to-[#FF3D00]
									hover:opacity-90 active:scale-[0.98]
									disabled:opacity-50 disabled:cursor-not-allowed
									transition-all duration-200 cursor-pointer"
							>
								{loading ? (
									<>
										<Loader2 className="mr-2 h-4 w-4 animate-spin" />
										Signing in...
									</>
								) : (
									"Sign In"
								)}
							</Button>
						</form>
					</div>

					{/* Footer Note */}
					<p className="text-center text-xs text-[var(--text-tertiary)]">
						Secure authentication for your transcription workspace
					</p>
				</div>
			</div>
		</div>
	);
}
