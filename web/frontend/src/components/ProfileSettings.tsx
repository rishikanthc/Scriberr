import { useState, useCallback } from "react";
import { Button } from "./ui/button";
import { ProfilesTable } from "./ProfilesTable";
import { TranscriptionConfigDialog, type WhisperXParams } from "./TranscriptionConfigDialog";
import { useAuth } from "../contexts/AuthContext";

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
    const { getAuthHeaders } = useAuth();

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
