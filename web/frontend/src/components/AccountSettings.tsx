import { useState } from "react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Separator } from "./ui/separator";
import { Eye, EyeOff, User, Lock, Check, X } from "lucide-react";
import { useAuth } from "../contexts/AuthContext";

interface PasswordStrength {
	hasMinLength: boolean;
	hasUppercase: boolean;
	hasLowercase: boolean;
	hasNumber: boolean;
	hasSpecialChar: boolean;
}

export function AccountSettings() {
	const { getAuthHeaders, logout } = useAuth();
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState("");
	const [success, setSuccess] = useState("");

	// Username change state
	const [newUsername, setNewUsername] = useState("");
	const [usernamePassword, setUsernamePassword] = useState("");
	const [showUsernamePassword, setShowUsernamePassword] = useState(false);

	// Password change state
	const [currentPassword, setCurrentPassword] = useState("");
	const [newPassword, setNewPassword] = useState("");
	const [confirmPassword, setConfirmPassword] = useState("");
	const [showCurrentPassword, setShowCurrentPassword] = useState(false);
	const [showNewPassword, setShowNewPassword] = useState(false);
	const [showConfirmPassword, setShowConfirmPassword] = useState(false);

	// Password strength validation
	const checkPasswordStrength = (pwd: string): PasswordStrength => ({
		hasMinLength: pwd.length >= 8,
		hasUppercase: /[A-Z]/.test(pwd),
		hasLowercase: /[a-z]/.test(pwd),
		hasNumber: /\d/.test(pwd),
		hasSpecialChar: /[!@#$%^&*(),.?":{}|<>]/.test(pwd),
	});

	const passwordStrength = checkPasswordStrength(newPassword);
	const isPasswordValid = Object.values(passwordStrength).every(Boolean);
	const passwordsMatch = newPassword === confirmPassword && confirmPassword.length > 0;

	const handleUsernameChange = async (e: React.FormEvent) => {
		e.preventDefault();
		setError("");
		setSuccess("");
		setLoading(true);

		try {
			const response = await fetch("/api/v1/auth/change-username", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					...getAuthHeaders(),
				},
				body: JSON.stringify({
					newUsername,
					password: usernamePassword,
				}),
			});

			if (response.ok) {
				setSuccess("Username changed successfully!");
				setNewUsername("");
				setUsernamePassword("");
			} else {
				const errorData = await response.json();
				setError(errorData.error || "Failed to change username");
			}
		} catch (error) {
			console.error("Username change error:", error);
			setError("Network error. Please try again.");
		} finally {
			setLoading(false);
		}
	};

	const handlePasswordChange = async (e: React.FormEvent) => {
		e.preventDefault();
		setError("");
		setSuccess("");

		if (!isPasswordValid) {
			setError("Please ensure your new password meets all requirements");
			return;
		}

		if (!passwordsMatch) {
			setError("New passwords do not match");
			return;
		}

		setLoading(true);

		try {
			const response = await fetch("/api/v1/auth/change-password", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					...getAuthHeaders(),
				},
				body: JSON.stringify({
					currentPassword,
					newPassword,
					confirmPassword,
				}),
			});

			if (response.ok) {
				setSuccess("Password changed successfully! You will be logged out shortly...");
				setCurrentPassword("");
				setNewPassword("");
				setConfirmPassword("");
				
				// Auto-logout after 2 seconds
				setTimeout(() => {
					logout();
				}, 2000);
			} else {
				const errorData = await response.json();
				setError(errorData.error || "Failed to change password");
			}
		} catch (error) {
			console.error("Password change error:", error);
			setError("Network error. Please try again.");
		} finally {
			setLoading(false);
		}
	};

	const PasswordStrengthIndicator = ({ label, met }: { label: string; met: boolean }) => (
		<div className={`flex items-center gap-2 text-sm ${met ? 'text-green-600 dark:text-green-400' : 'text-gray-500 dark:text-gray-400'}`}>
			{met ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}
			<span>{label}</span>
		</div>
	);

	return (
		<div className="space-y-6">
			{/* Error/Success Messages */}
			{error && (
				<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3">
					<p className="text-red-700 dark:text-red-300 text-sm">{error}</p>
				</div>
			)}
			
			{success && (
				<div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-3">
					<p className="text-green-700 dark:text-green-300 text-sm">{success}</p>
				</div>
			)}

			{/* Username Change Section */}
			<div className="bg-gray-50 dark:bg-gray-700/50 rounded-xl p-6">
				<div className="mb-4">
					<div className="flex items-center space-x-2 mb-2">
						<User className="h-5 w-5 text-blue-600 dark:text-blue-400" />
						<h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">Change Username</h3>
					</div>
					<p className="text-sm text-gray-600 dark:text-gray-400">
						Update your account username. You'll need to verify your current password.
					</p>
				</div>
				<div>
					<form onSubmit={handleUsernameChange} className="space-y-4">
						<div className="space-y-2">
							<Label htmlFor="newUsername" className="text-gray-700 dark:text-gray-300">
								New Username
							</Label>
							<Input
								id="newUsername"
								type="text"
								placeholder="Enter new username (3-50 characters)"
								value={newUsername}
								onChange={(e) => setNewUsername(e.target.value)}
								disabled={loading}
								required
								minLength={3}
								maxLength={50}
								className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
							/>
						</div>

						<div className="space-y-2">
							<Label htmlFor="usernamePassword" className="text-gray-700 dark:text-gray-300">
								Current Password
							</Label>
							<div className="relative">
								<Input
									id="usernamePassword"
									type={showUsernamePassword ? "text" : "password"}
									placeholder="Enter your current password"
									value={usernamePassword}
									onChange={(e) => setUsernamePassword(e.target.value)}
									disabled={loading}
									required
									className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 pr-10"
								/>
								<button
									type="button"
									onClick={() => setShowUsernamePassword(!showUsernamePassword)}
									className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
								>
									{showUsernamePassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
								</button>
							</div>
						</div>

						<Button
							type="submit"
							className="bg-blue-600 hover:bg-blue-700 text-white"
							disabled={loading || !newUsername.trim() || !usernamePassword.trim()}
						>
							{loading ? "Changing Username..." : "Change Username"}
						</Button>
					</form>
				</div>
			</div>

			<Separator className="bg-gray-200 dark:bg-gray-700" />

			{/* Password Change Section */}
			<div className="bg-gray-50 dark:bg-gray-700/50 rounded-xl p-6">
				<div className="mb-4">
					<div className="flex items-center space-x-2 mb-2">
						<Lock className="h-5 w-5 text-red-600 dark:text-red-400" />
						<h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">Change Password</h3>
					</div>
					<p className="text-sm text-gray-600 dark:text-gray-400">
						Update your account password. You'll be automatically logged out after changing your password.
					</p>
				</div>
				<div>
					<form onSubmit={handlePasswordChange} className="space-y-4">
						<div className="space-y-2">
							<Label htmlFor="currentPassword" className="text-gray-700 dark:text-gray-300">
								Current Password
							</Label>
							<div className="relative">
								<Input
									id="currentPassword"
									type={showCurrentPassword ? "text" : "password"}
									placeholder="Enter your current password"
									value={currentPassword}
									onChange={(e) => setCurrentPassword(e.target.value)}
									disabled={loading}
									required
									className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 pr-10"
								/>
								<button
									type="button"
									onClick={() => setShowCurrentPassword(!showCurrentPassword)}
									className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
								>
									{showCurrentPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
								</button>
							</div>
						</div>

						<div className="space-y-2">
							<Label htmlFor="newPassword" className="text-gray-700 dark:text-gray-300">
								New Password
							</Label>
							<div className="relative">
								<Input
									id="newPassword"
									type={showNewPassword ? "text" : "password"}
									placeholder="Create a new secure password"
									value={newPassword}
									onChange={(e) => setNewPassword(e.target.value)}
									disabled={loading}
									required
									className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 pr-10"
								/>
								<button
									type="button"
									onClick={() => setShowNewPassword(!showNewPassword)}
									className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
								>
									{showNewPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
								</button>
							</div>

							{newPassword && (
								<div className="mt-3 space-y-2 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
									<p className="text-sm font-medium text-gray-700 dark:text-gray-300">Password Requirements:</p>
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
							<Label htmlFor="confirmPassword" className="text-gray-700 dark:text-gray-300">
								Confirm New Password
							</Label>
							<div className="relative">
								<Input
									id="confirmPassword"
									type={showConfirmPassword ? "text" : "password"}
									placeholder="Confirm your new password"
									value={confirmPassword}
									onChange={(e) => setConfirmPassword(e.target.value)}
									disabled={loading}
									required
									className={`bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 pr-10 ${
										confirmPassword && !passwordsMatch ? 'border-red-300 dark:border-red-600' : ''
									}`}
								/>
								<button
									type="button"
									onClick={() => setShowConfirmPassword(!showConfirmPassword)}
									className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
								>
									{showConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
								</button>
							</div>

							{confirmPassword && (
								<div className={`flex items-center gap-2 text-sm ${
									passwordsMatch ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'
								}`}>
									{passwordsMatch ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}
									<span>{passwordsMatch ? "Passwords match" : "Passwords do not match"}</span>
								</div>
							)}
						</div>

						<div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3">
							<p className="text-yellow-800 dark:text-yellow-300 text-sm font-medium">⚠️ Warning</p>
							<p className="text-yellow-700 dark:text-yellow-300 text-sm mt-1">
								You will be automatically logged out after changing your password and will need to log in again with your new credentials.
							</p>
						</div>

						<Button
							type="submit"
							className="bg-red-600 hover:bg-red-700 text-white"
							disabled={loading || !currentPassword.trim() || !isPasswordValid || !passwordsMatch}
						>
							{loading ? "Changing Password..." : "Change Password"}
						</Button>
					</form>
				</div>
			</div>
		</div>
	);
}