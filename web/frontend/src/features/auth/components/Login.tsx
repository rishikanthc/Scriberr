import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ScriberrLogo } from "@/components/ScriberrLogo";
import { useNavigate } from "react-router-dom";
import { ThemeSwitcher } from "@/components/ThemeSwitcher";

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

	return (
		<div className="min-h-screen bg-carbon-50 dark:bg-carbon-900 flex items-center justify-center">
			<div className="absolute top-8 right-8">
				<ThemeSwitcher />
			</div>

			<div className="w-full max-w-md space-y-8">
				<div className="text-center">
					<div className="flex justify-center mb-6">
						<ScriberrLogo onClick={() => navigate('/')} />
					</div>
					<h2 className="text-3xl font-bold text-carbon-900 dark:text-carbon-100">
						Sign in to Scriberr
					</h2>
					<p className="mt-2 text-carbon-600 dark:text-carbon-400">
						Access your audio transcription workspace
					</p>
				</div>

				<Card className="bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700">
					<CardHeader>
						<CardTitle className="text-carbon-900 dark:text-carbon-100">Login</CardTitle>
						<CardDescription className="text-carbon-600 dark:text-carbon-400">
							Enter your credentials to continue
						</CardDescription>
					</CardHeader>
					<CardContent>
						<form onSubmit={handleSubmit} className="space-y-4">
							{error && (
								<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3">
									<p className="text-red-700 dark:text-red-300 text-sm">{error}</p>
								</div>
							)}

							<div className="space-y-2">
								<Label htmlFor="username" className="text-carbon-700 dark:text-carbon-300">
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
									className="bg-white dark:bg-carbon-800 border-carbon-300 dark:border-carbon-600 text-carbon-900 dark:text-carbon-100"
								/>
							</div>

							<div className="space-y-2">
								<Label htmlFor="password" className="text-carbon-700 dark:text-carbon-300">
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
									className="bg-white dark:bg-carbon-800 border-carbon-300 dark:border-carbon-600 text-carbon-900 dark:text-carbon-100"
								/>
							</div>

							<Button
								type="submit"
								className="w-full bg-blue-600 hover:bg-blue-700 text-white"
								disabled={loading || !username.trim() || !password.trim()}
							>
								{loading ? "Signing in..." : "Sign In"}
							</Button>
						</form>
					</CardContent>
				</Card>

				<div className="text-center">
					<p className="text-sm text-carbon-600 dark:text-carbon-400">
						Secure authentication required for API key management
					</p>
				</div>
			</div>
		</div>
	);
}
