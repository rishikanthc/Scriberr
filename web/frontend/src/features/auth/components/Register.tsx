import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ScriberrLogo } from "@/components/ScriberrLogo";
import { useNavigate } from "react-router-dom";
import { ThemeSwitcher } from "@/components/ThemeSwitcher";
import { Eye, EyeOff, Check, X } from "lucide-react";

interface RegisterProps {
	onRegister: (token: string) => void;
}

interface PasswordStrength {
	hasMinLength: boolean;
	hasUppercase: boolean;
	hasLowercase: boolean;
	hasNumber: boolean;
	hasSpecialChar: boolean;
}

export function Register({ onRegister }: RegisterProps) {
	const navigate = useNavigate();
	const [username, setUsername] = useState("");
	const [password, setPassword] = useState("");
	const [confirmPassword, setConfirmPassword] = useState("");
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState("");
	const [showPassword, setShowPassword] = useState(false);
	const [showConfirmPassword, setShowConfirmPassword] = useState(false);

	// Password strength validation
	const checkPasswordStrength = (pwd: string): PasswordStrength => ({
		hasMinLength: pwd.length >= 8,
		hasUppercase: /[A-Z]/.test(pwd),
		hasLowercase: /[a-z]/.test(pwd),
		hasNumber: /\d/.test(pwd),
		hasSpecialChar: /[!@#$%^&*(),.?":{}|<>]/.test(pwd),
	});

	const passwordStrength = checkPasswordStrength(password);
	const isPasswordValid = Object.values(passwordStrength).every(Boolean);
	const passwordsMatch = password === confirmPassword && confirmPassword.length > 0;

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		setError("");

		if (!isPasswordValid) {
			setError("Please ensure your password meets all requirements");
			return;
		}

		if (!passwordsMatch) {
			setError("Passwords do not match");
			return;
		}

		setLoading(true);

		try {
			const response = await fetch("/api/v1/auth/register", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
				},
				body: JSON.stringify({
					username,
					password,
					confirmPassword,
				}),
			});

			if (response.ok) {
				const data = await response.json();
				onRegister(data.token);
			} else {
				const error = await response.json();
				setError(error.error || "Registration failed");
			}
		} catch (error) {
			console.error("Registration error:", error);
			setError("Network error. Please try again.");
		} finally {
			setLoading(false);
		}
	};

	const PasswordStrengthIndicator = ({ label, met }: { label: string; met: boolean }) => (
		<div className={`flex items-center gap-2 text-sm ${met ? 'text-green-600 dark:text-green-400' : 'text-carbon-500 dark:text-carbon-400'}`}>
			{met ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}
			<span>{label}</span>
		</div>
	);

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
						Welcome to Scriberr
					</h2>
					<p className="mt-2 text-carbon-600 dark:text-carbon-400">
						Create your admin account to get started
					</p>
				</div>

				<Card className="bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700">
					<CardHeader>
						<CardTitle className="text-carbon-900 dark:text-carbon-100">Setup Admin Account</CardTitle>
						<CardDescription className="text-carbon-600 dark:text-carbon-400">
							This will be the only account that can access this Scriberr instance
						</CardDescription>
					</CardHeader>
					<CardContent>
						<form onSubmit={handleSubmit} className="space-y-6">
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
									placeholder="Choose a username (3-50 characters)"
									value={username}
									onChange={(e) => setUsername(e.target.value)}
									disabled={loading}
									required
									minLength={3}
									maxLength={50}
									className="bg-white dark:bg-carbon-800 border-carbon-300 dark:border-carbon-600 text-carbon-900 dark:text-carbon-100"
								/>
							</div>

							<div className="space-y-2">
								<Label htmlFor="password" className="text-carbon-700 dark:text-carbon-300">
									Password
								</Label>
								<div className="relative">
									<Input
										id="password"
										type={showPassword ? "text" : "password"}
										placeholder="Create a secure password"
										value={password}
										onChange={(e) => setPassword(e.target.value)}
										disabled={loading}
										required
										className="bg-white dark:bg-carbon-800 border-carbon-300 dark:border-carbon-600 text-carbon-900 dark:text-carbon-100 pr-10"
									/>
									<Button
										type="button"
										variant="ghost"
										size="icon"
										onClick={() => setShowPassword(!showPassword)}
										className="absolute right-1 top-1/2 -translate-y-1/2 h-7 w-7"
									>
										{showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
									</Button>
								</div>

								{password && (
									<div className="mt-3 space-y-2 p-3 bg-carbon-50 dark:bg-carbon-800 rounded-lg">
										<p className="text-sm font-medium text-carbon-700 dark:text-carbon-300">Password Requirements:</p>
										<div className="grid grid-cols-1 gap-1">
											<PasswordStrengthIndicator label="At least 8 characters" met={passwordStrength.hasMinLength} />
											<PasswordStrengthIndicator label="One uppercase letter" met={passwordStrength.hasUppercase} />
											<PasswordStrengthIndicator label="One lowercase letter" met={passwordStrength.hasLowercase} />
											<PasswordStrengthIndicator label="One number" met={passwordStrength.hasNumber} />
											<PasswordStrengthIndicator label="One special character" met={passwordStrength.hasSpecialChar} />
										</div>
									</div>
								)}
							</div>

							<div className="space-y-2">
								<Label htmlFor="confirmPassword" className="text-carbon-700 dark:text-carbon-300">
									Confirm Password
								</Label>
								<div className="relative">
									<Input
										id="confirmPassword"
										type={showConfirmPassword ? "text" : "password"}
										placeholder="Confirm your password"
										value={confirmPassword}
										onChange={(e) => setConfirmPassword(e.target.value)}
										disabled={loading}
										required
										className={`bg-white dark:bg-carbon-800 border-carbon-300 dark:border-carbon-600 text-carbon-900 dark:text-carbon-100 pr-10 ${confirmPassword && !passwordsMatch ? 'border-red-300 dark:border-red-600' : ''
											}`}
									/>
									<Button
										type="button"
										variant="ghost"
										size="icon"
										onClick={() => setShowConfirmPassword(!showConfirmPassword)}
										className="absolute right-1 top-1/2 -translate-y-1/2 h-7 w-7"
									>
										{showConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
									</Button>
								</div>

								{confirmPassword && (
									<div className={`flex items-center gap-2 text-sm ${passwordsMatch ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'
										}`}>
										{passwordsMatch ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}
										<span>{passwordsMatch ? "Passwords match" : "Passwords do not match"}</span>
									</div>
								)}
							</div>

							<Button
								type="submit"
								variant="brand"
								className="w-full"
								disabled={loading || !username.trim() || !isPasswordValid || !passwordsMatch}
							>
								{loading ? "Creating Account..." : "Create Admin Account"}
							</Button>
						</form>
					</CardContent>
				</Card>

				<div className="text-center">
					<p className="text-sm text-carbon-600 dark:text-carbon-400">
						This account will have full administrative access to your Scriberr instance
					</p>
				</div>
			</div>
		</div>
	);
}
