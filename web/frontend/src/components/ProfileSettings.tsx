import { useState, useCallback, useEffect } from "react";
import { Button } from "./ui/button";
import { ProfilesTable } from "./ProfilesTable";
import { TranscriptionConfigDialog, type WhisperXParams } from "./TranscriptionConfigDialog";
import { useAuth } from "../contexts/AuthContext";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./ui/select";

interface TranscriptionProfile {
	id: string;
	name: string;
	description?: string;
	is_default: boolean;
	parameters: WhisperXParams;
	created_at: string;
	updated_at: string;
}

export function ProfileSettings() {
	const [profileDialogOpen, setProfileDialogOpen] = useState(false);
	const [editingProfile, setEditingProfile] = useState<TranscriptionProfile | null>(null);
	const [refreshTrigger, setRefreshTrigger] = useState(0);
	const [profiles, setProfiles] = useState<TranscriptionProfile[]>([]);
	const [defaultProfile, setDefaultProfile] = useState<TranscriptionProfile | null>(null);
	const [isLoadingProfiles, setIsLoadingProfiles] = useState(true);
    const { getAuthHeaders } = useAuth();

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
			<div className="bg-gray-50 dark:bg-gray-700/50 rounded-xl p-4 sm:p-6">
				<div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 sm:gap-0 mb-4">
					<div>
						<h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
							Transcription Profiles
						</h3>
						<p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
							Manage your saved transcription configurations for quick access.
						</p>
					</div>
					<Button
						onClick={handleCreateProfile}
						className="bg-blue-600 hover:bg-blue-700 text-white"
					>
						Create New Profile
					</Button>
				</div>

				{/* Default Profile Selection */}
				<div className="mb-6 p-4 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
						<div className="flex-1">
							<label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
								Default Profile
							</label>
							<p className="text-xs text-gray-500 dark:text-gray-400">
								The profile to use by default when starting new transcriptions.
							</p>
						</div>
						<div className="w-full sm:w-64">
							<Select
								value={defaultProfile?.id || ""}
								onValueChange={handleDefaultProfileChange}
								disabled={isLoadingProfiles || profiles.length === 0}
							>
								<SelectTrigger>
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
								<SelectContent>
									{profiles.map((profile) => (
										<SelectItem key={profile.id} value={profile.id}>
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
