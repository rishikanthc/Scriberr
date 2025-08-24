import { useState, useCallback } from "react";
import { Button } from "./ui/button";
import { ProfilesTable } from "./ProfilesTable";
import { TranscriptionConfigDialog, type WhisperXParams } from "./TranscriptionConfigDialog";

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

	const handleCreateProfile = useCallback(() => {
		setEditingProfile(null);
		setProfileDialogOpen(true);
	}, []);

	const handleEditProfile = useCallback((profile: TranscriptionProfile) => {
		setEditingProfile(profile);
		setProfileDialogOpen(true);
	}, []);

	const handleProfileSaved = useCallback(async (_params: WhisperXParams & { profileName?: string; profileDescription?: string }) => {
		// Profile saving logic would go here
		// For now, just close the dialog and refresh
		setRefreshTrigger((prev) => prev + 1);
		setProfileDialogOpen(false);
		setEditingProfile(null);
	}, []);

	const handleProfileChange = useCallback(() => {
		setRefreshTrigger((prev) => prev + 1);
	}, []);

	return (
		<div className="space-y-6">
			<div className="bg-gray-50 dark:bg-gray-700/50 rounded-xl p-6">
				<div className="flex items-center justify-between mb-4">
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