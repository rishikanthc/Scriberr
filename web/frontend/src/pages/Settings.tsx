import { useState, useEffect } from "react";
import { ArrowLeft, Check, ChevronsUpDown } from "lucide-react";
import { Button } from "../components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "../components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../components/ui/popover";
import { cn } from "../lib/utils";
import { useRouter } from "../contexts/RouterContext";
import { ThemeSwitcher } from "../components/ThemeSwitcher";
import { ScriberrLogo } from "../components/ScriberrLogo";
import { ProfilesTable } from "../components/ProfilesTable";
import { TranscriptionConfigDialog } from "../components/TranscriptionConfigDialog";
import type { WhisperXParams } from "../components/TranscriptionConfigDialog";

interface TranscriptionProfile {
	id: string;
	name: string;
	description?: string;
	is_default: boolean;
	parameters: WhisperXParams;
	created_at: string;
	updated_at: string;
}

export function Settings() {
	const { navigate } = useRouter();
	const [profileDialogOpen, setProfileDialogOpen] = useState(false);
	const [editingProfile, setEditingProfile] = useState<TranscriptionProfile | null>(null);
	const [refreshTrigger, setRefreshTrigger] = useState(0);
	const [profiles, setProfiles] = useState<TranscriptionProfile[]>([]);
	const [defaultProfileId, setDefaultProfileId] = useState<string>("");
	const [comboboxOpen, setComboboxOpen] = useState(false);

	const handleBack = () => {
		navigate({ path: "home" });
	};

	const fetchProfiles = async () => {
		try {
			const response = await fetch("/api/v1/profiles/", {
				headers: {
					"X-API-Key": "dev-api-key-123",
				},
			});

			if (response.ok) {
				const data = await response.json();
				setProfiles(data);
				// Find the default profile
				const defaultProfile = data.find((profile: TranscriptionProfile) => profile.is_default);
				setDefaultProfileId(defaultProfile?.id || "");
			}
		} catch (error) {
			console.error("Error fetching profiles:", error);
		}
	};

	useEffect(() => {
		fetchProfiles();
	}, [refreshTrigger]);

	const handleNewProfile = () => {
		setEditingProfile(null);
		setProfileDialogOpen(true);
	};

	const handleEditProfile = (profile: TranscriptionProfile) => {
		setEditingProfile(profile);
		setProfileDialogOpen(true);
	};

	const handleSaveProfile = async (params: WhisperXParams & { profileName?: string; profileDescription?: string }) => {
		if (!params.profileName) {
			alert("Profile name is required");
			return;
		}

		try {
			const isEditing = editingProfile !== null;
			const url = isEditing 
				? `/api/v1/profiles/${editingProfile.id}`
				: "/api/v1/profiles";
			const method = isEditing ? "PUT" : "POST";

			const response = await fetch(url, {
				method,
				headers: {
					"Content-Type": "application/json",
					"X-API-Key": "dev-api-key-123",
				},
				body: JSON.stringify({
					name: params.profileName,
					description: params.profileDescription || undefined,
					parameters: params,
				}),
			});

			if (response.ok) {
				setProfileDialogOpen(false);
				setEditingProfile(null);
				setRefreshTrigger(prev => prev + 1);
			} else {
				const error = await response.json();
				alert(error.error || `Failed to ${isEditing ? 'update' : 'create'} profile`);
			}
		} catch (error) {
			console.error(`Error ${editingProfile ? 'updating' : 'creating'} profile:`, error);
			alert(`Failed to ${editingProfile ? 'update' : 'create'} profile`);
		}
	};

	const handleSetDefaultProfile = async (profileId: string) => {
		try {
			const response = await fetch(`/api/v1/profiles/${profileId}/set-default`, {
				method: "POST",
				headers: {
					"X-API-Key": "dev-api-key-123",
				},
			});

			if (response.ok) {
				setDefaultProfileId(profileId);
				setRefreshTrigger(prev => prev + 1); // Refresh the profiles list
			} else {
				const error = await response.json();
				alert(error.error || "Failed to set default profile");
			}
		} catch (error) {
			console.error("Error setting default profile:", error);
			alert("Failed to set default profile");
		}
	};

	return (
		<div className="min-h-screen bg-gray-50 dark:bg-gray-900">
			<div className="mx-auto px-8 py-6" style={{ width: "60vw" }}>
				{/* Header */}
				<header className="bg-white dark:bg-gray-800 rounded-xl p-6 mb-6">
					<div className="flex items-center justify-between">
						{/* Left side - Back button and Logo */}
						<div className="flex items-center gap-4">
							<Button 
								onClick={handleBack} 
								variant="ghost" 
								size="sm"
								className="h-10 w-10 p-0 rounded-xl hover:bg-gray-100 dark:hover:bg-gray-700 transition-all duration-200"
							>
								<ArrowLeft className="h-5 w-5 text-gray-600 dark:text-gray-400" />
							</Button>
							<ScriberrLogo />
						</div>
						
						{/* Right side - Theme switcher */}
						<ThemeSwitcher />
					</div>
				</header>

				{/* Settings Content */}
				<div className="bg-white dark:bg-gray-800 rounded-xl p-8">
					{/* Header Section */}
					<div className="flex items-center justify-between mb-8">
						<div>
							<h1 className="text-3xl font-bold text-gray-900 dark:text-gray-50 mb-2">
								Settings
							</h1>
							<p className="text-gray-600 dark:text-gray-400">
								Manage your transcription profiles and application settings
							</p>
						</div>
						<Button
							onClick={handleNewProfile}
							className="bg-blue-500 hover:bg-blue-600 text-white font-medium px-6 py-2.5 rounded-xl transition-all duration-300 hover:scale-[1.02] hover:shadow-lg hover:shadow-blue-500/20"
						>
							New Profile
						</Button>
					</div>

					{/* Profiles Section */}
					<div className="space-y-6">
						<div>
							<h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-1">
								Transcription Profiles
							</h2>
							<p className="text-gray-600 dark:text-gray-400 text-sm mb-6">
								Save and reuse your preferred transcription configurations
							</p>
						</div>
						
						{/* Default Profile Selector */}
						{profiles.length > 0 && (
							<div className="mb-6">
								<label className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 block">
									Default Profile
								</label>
								<Popover open={comboboxOpen} onOpenChange={setComboboxOpen}>
									<PopoverTrigger asChild>
										<Button
											variant="outline"
											role="combobox"
											aria-expanded={comboboxOpen}
											className="w-full justify-between bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
										>
											{defaultProfileId
												? profiles.find((profile) => profile.id === defaultProfileId)?.name || "Select profile..."
												: "Select profile..."
											}
											<ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
										</Button>
									</PopoverTrigger>
									<PopoverContent className="w-full p-0 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
										<Command>
											<CommandInput placeholder="Search profiles..." className="h-9" />
											<CommandList>
												<CommandEmpty>No profile found.</CommandEmpty>
												<CommandGroup>
													{profiles.map((profile) => (
														<CommandItem
															key={profile.id}
															value={profile.name}
															onSelect={() => {
																if (profile.id !== defaultProfileId) {
																	handleSetDefaultProfile(profile.id);
																}
																setComboboxOpen(false);
															}}
															className="cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700"
														>
															<Check
																className={cn(
																	"mr-2 h-4 w-4",
																	defaultProfileId === profile.id ? "opacity-100" : "opacity-0"
																)}
															/>
															<div className="flex flex-col">
																<span className="font-medium">{profile.name}</span>
																{profile.description && (
																	<span className="text-xs text-gray-500 dark:text-gray-400">
																		{profile.description}
																	</span>
																)}
															</div>
														</CommandItem>
													))}
												</CommandGroup>
											</CommandList>
										</Command>
									</PopoverContent>
								</Popover>
								<p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
									This profile will be used as the default configuration for new transcriptions.
								</p>
							</div>
						)}
						
						<ProfilesTable 
							refreshTrigger={refreshTrigger} 
							onProfileChange={() => setRefreshTrigger(prev => prev + 1)}
							onEditProfile={handleEditProfile}
						/>
					</div>
				</div>
			</div>

			{/* Profile Creation/Edit Dialog */}
			<TranscriptionConfigDialog
				open={profileDialogOpen}
				onOpenChange={setProfileDialogOpen}
				onStartTranscription={handleSaveProfile}
				loading={false}
				isProfileMode={true}
				initialParams={editingProfile?.parameters}
				initialName={editingProfile?.name}
				initialDescription={editingProfile?.description}
			/>
		</div>
	);
}