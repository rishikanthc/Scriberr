import { useState, useCallback, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { ProfilesTable } from "./ProfilesTable";
import { TranscriptionConfigDialog, type WhisperXParams } from "@/components/TranscriptionConfigDialog";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Settings } from "lucide-react";

interface TranscriptionProfile {
	id: string;
	name: string;
	description?: string;
	is_default: boolean;
	parameters: WhisperXParams;
	created_at: string;
	updated_at: string;
}

interface UserSettings {
	auto_transcription_enabled: boolean;
	default_profile_id?: string;
}

export function ProfileSettings() {
	const [profileDialogOpen, setProfileDialogOpen] = useState(false);
	const [editingProfile, setEditingProfile] = useState<TranscriptionProfile | null>(null);
	const [refreshTrigger, setRefreshTrigger] = useState(0);
	const [profiles, setProfiles] = useState<TranscriptionProfile[]>([]);
	const [defaultProfile, setDefaultProfile] = useState<TranscriptionProfile | null>(null);
	const [isLoadingProfiles, setIsLoadingProfiles] = useState(true);
	const { getAuthHeaders } = useAuth();

	// User settings state
	const [userSettings, setUserSettings] = useState<UserSettings | null>(null);
	const [settingsLoading, setSettingsLoading] = useState(true);
	const [error, setError] = useState("");
	const [success, setSuccess] = useState("");

	// Load profiles and default profile
	const loadProfiles = useCallback(async () => {
		try {
			setIsLoadingProfiles(true);

			// Load all profiles
			const profilesRes = await fetch('/api/v1/profiles', {
				headers: getAuthHeaders()
			});
			if (profilesRes.ok) {
				const profilesData = await profilesRes.json();
				setProfiles(profilesData);
			}

			// Load user's default profile
			const defaultRes = await fetch('/api/v1/user/default-profile', {
				headers: getAuthHeaders()
			});
			if (defaultRes.ok) {
				const defaultData = await defaultRes.json();
				setDefaultProfile(defaultData);
			} else if (defaultRes.status === 404) {
				// No default profile set, that's okay
				setDefaultProfile(null);
			}
		} catch (error) {
			console.error('Failed to load profiles:', error);
		} finally {
			setIsLoadingProfiles(false);
		}
	}, [getAuthHeaders]);

	// Handle default profile change
	const handleDefaultProfileChange = useCallback(async (profileId: string) => {
		try {
			const res = await fetch('/api/v1/user/default-profile', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					...getAuthHeaders()
				},
				body: JSON.stringify({ profile_id: profileId })
			});

			if (res.ok) {
				// Update local state
				const selectedProfile = profiles.find(p => p.id === profileId);
				setDefaultProfile(selectedProfile || null);
			} else {
				const error = await res.text();
				console.error('Failed to set default profile:', error);
				alert('Failed to set default profile');
			}
		} catch (error) {
			console.error('Failed to set default profile:', error);
			alert('Failed to set default profile');
		}
	}, [profiles, getAuthHeaders]);

	// Load profiles on component mount and when refresh trigger changes
	useEffect(() => {
		loadProfiles();
	}, [loadProfiles, refreshTrigger]);

	// Load user settings on component mount
	useEffect(() => {
		const loadUserSettings = async () => {
			try {
				const response = await fetch("/api/v1/user/settings", {
					headers: getAuthHeaders(),
				});

				if (response.ok) {
					const settings = await response.json();
					setUserSettings(settings);
				} else {
					console.error("Failed to load user settings");
				}
			} catch (error) {
				console.error("Error loading user settings:", error);
			} finally {
				setSettingsLoading(false);
			}
		};

		loadUserSettings();
	}, [getAuthHeaders]);

	// Handle auto-transcription toggle
	const handleAutoTranscriptionToggle = async (enabled: boolean) => {
		setError("");
		setSuccess("");

		try {
			const response = await fetch("/api/v1/user/settings", {
				method: "PUT",
				headers: {
					"Content-Type": "application/json",
					...getAuthHeaders(),
				},
				body: JSON.stringify({
					auto_transcription_enabled: enabled,
				}),
			});

			if (response.ok) {
				const updatedSettings = await response.json();
				setUserSettings(updatedSettings);
				setSuccess(`Auto-transcription ${enabled ? "enabled" : "disabled"} successfully!`);
			} else {
				const errorData = await response.json();
				setError(errorData.error || "Failed to update setting");
			}
		} catch (error) {
			console.error("Error updating auto-transcription setting:", error);
			setError("Network error. Please try again.");
		}
	};

	const handleCreateProfile = useCallback(() => {
		setEditingProfile(null);
		setProfileDialogOpen(true);
	}, []);

	const handleEditProfile = useCallback((profile: TranscriptionProfile) => {
		setEditingProfile(profile);
		setProfileDialogOpen(true);
	}, []);

	const handleProfileSaved = useCallback(async (payload: WhisperXParams & { profileName?: string; profileDescription?: string }) => {
		try {
			const name = (payload.profileName || "").trim();
			const description = (payload.profileDescription || "").trim();
			if (!name) {
				alert("Profile name is required");
				return;
			}

			// eslint-disable-next-line @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any
			const { profileName: _pn, profileDescription: _pd, ...paramRest } = payload as any;
			const body = {
				name,
				description: description || undefined,
				parameters: paramRest as WhisperXParams,
			};

			let res: Response;
			if (editingProfile) {
				// Preserve current default flag unless changed elsewhere
				res = await fetch(`/api/v1/profiles/${editingProfile.id}`, {
					method: "PUT",
					headers: { "Content-Type": "application/json", ...getAuthHeaders() },
					body: JSON.stringify({
						...body,
						id: editingProfile.id,
						is_default: editingProfile.is_default,
					}),
				});
			} else {
				res = await fetch(`/api/v1/profiles`, {
					method: "POST",
					headers: { "Content-Type": "application/json", ...getAuthHeaders() },
					body: JSON.stringify(body),
				});
			}

			if (!res.ok) {
				const text = await res.text();
				alert(`Failed to save profile: ${res.status} ${text}`);
				return;
			}

			setRefreshTrigger((prev) => prev + 1);
			setProfileDialogOpen(false);
			setEditingProfile(null);
		} catch (e) {
			console.error("Failed to save profile", e);
			alert("Failed to save profile");
		}
	}, [editingProfile, getAuthHeaders]);

	const handleProfileChange = useCallback(() => {
		setRefreshTrigger((prev) => prev + 1);
	}, []);

	return (
		<div className="space-y-6">
			{/* Error/Success Messages */}
			{error && (
				<div className="bg-[var(--error)]/10 border border-[var(--error)]/20 rounded-lg p-3">
					<p className="text-[var(--error)] text-sm">{error}</p>
				</div>
			)}

			{success && (
				<div className="bg-[var(--success-translucent)] border border-[var(--success-solid)]/20 rounded-lg p-3">
					<p className="text-[var(--success-solid)] text-sm">{success}</p>
				</div>
			)}

			{/* Auto-Transcription Settings */}
			<div className="bg-[var(--bg-main)]/50 border border-[var(--border-subtle)] rounded-[var(--radius-card)] p-4 sm:p-6 shadow-sm">
				<div className="mb-4">
					<div className="flex items-center space-x-2 mb-2">
						<Settings className="h-5 w-5 text-[var(--brand-solid)]" />
						<h3 className="text-lg font-medium text-[var(--text-primary)]">Auto-Transcription</h3>
					</div>
					<p className="text-sm text-[var(--text-secondary)]">
						Configure automatic transcription behavior for uploaded files.
					</p>
				</div>

				{settingsLoading ? (
					<div className="flex items-center space-x-2 py-4">
						<div className="animate-spin rounded-full h-4 w-4 border-b-2 border-carbon-600 dark:border-carbon-400"></div>
						<span className="text-sm text-carbon-600 dark:text-carbon-400">Loading settings...</span>
					</div>
				) : (
					<div className="flex items-center justify-between py-2">
						<div>
							<Label htmlFor="auto-transcription" className="text-[var(--text-primary)] font-medium">
								Automatic Transcription on Upload
							</Label>
							<p className="text-sm text-[var(--text-secondary)] mt-1">
								When enabled, uploaded audio files will automatically be queued for transcription using your default profile.
							</p>
						</div>
						<Switch
							id="auto-transcription"
							checked={userSettings?.auto_transcription_enabled || false}
							onCheckedChange={handleAutoTranscriptionToggle}
							disabled={settingsLoading}
						/>
					</div>
				)}
			</div>

			{/* Transcription Profiles */}
			<div className="bg-[var(--bg-main)]/50 border border-[var(--border-subtle)] rounded-[var(--radius-card)] p-4 sm:p-6 shadow-sm">
				<div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 sm:gap-0 mb-4">
					<div>
						<h3 className="text-lg font-medium text-[var(--text-primary)]">
							Transcription Profiles
						</h3>
						<p className="text-sm text-[var(--text-secondary)] mt-1">
							Manage your saved transcription configurations for quick access.
						</p>
					</div>
					<Button
						onClick={handleCreateProfile}
						className="!bg-[var(--brand-gradient)] hover:!opacity-90 !text-black dark:!text-white shadow-lg shadow-orange-500/20 border-none"
					>
						Create New Profile
					</Button>
				</div>

				{/* Default Profile Selection */}
				<div className="mb-6 p-4 bg-[var(--bg-card)] rounded-lg border border-[var(--border-subtle)]">
					<div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
						<div className="flex-1">
							<label className="block text-sm font-medium text-[var(--text-primary)] mb-1">
								Default Profile
							</label>
							<p className="text-xs text-[var(--text-secondary)]">
								The profile to use by default when starting new transcriptions.
							</p>
						</div>
						<div className="w-full sm:w-64">
							<Select
								value={defaultProfile?.id || ""}
								onValueChange={handleDefaultProfileChange}
								disabled={isLoadingProfiles || profiles.length === 0}
							>
								<SelectTrigger className="bg-[var(--bg-main)] border-[var(--border-subtle)] text-[var(--text-primary)]">
									<SelectValue
										placeholder={
											isLoadingProfiles
												? "Loading..."
												: profiles.length === 0
													? "No profiles available"
													: "Select default profile"
										}
									/>
								</SelectTrigger>
								<SelectContent className="bg-[var(--bg-card)] border-[var(--border-subtle)] text-[var(--text-primary)]">
									{profiles.map((profile) => (
										<SelectItem key={profile.id} value={profile.id} className="focus:bg-[var(--bg-secondary)] focus:text-[var(--text-primary)]">
											{profile.name}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
					</div>
				</div>

				<ProfilesTable
					refreshTrigger={refreshTrigger}
					onProfileChange={handleProfileChange}
					onEditProfile={handleEditProfile}
					onCreateProfile={handleCreateProfile}
				/>
			</div>

			<TranscriptionConfigDialog
				open={profileDialogOpen}
				onOpenChange={(open) => {
					setProfileDialogOpen(open);
					if (!open) {
						setEditingProfile(null);
					}
				}}
				onStartTranscription={handleProfileSaved}
				isProfileMode={true}
				initialParams={editingProfile?.parameters}
				initialName={editingProfile?.name}
				initialDescription={editingProfile?.description}
			/>
		</div>
	);
}
