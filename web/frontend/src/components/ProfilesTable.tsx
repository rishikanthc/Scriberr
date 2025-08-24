import { useState, useEffect, useCallback } from "react";
import { MoreVertical, Trash2, Settings, Terminal } from "lucide-react";
import { Button } from "./ui/button";
import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "./ui/popover";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
	AlertDialogTrigger,
} from "./ui/alert-dialog";
import type { WhisperXParams } from "./TranscriptionConfigDialog";

interface TranscriptionProfile {
	id: string;
	name: string;
	description?: string;
	parameters: WhisperXParams;
	created_at: string;
	updated_at: string;
}

interface ProfilesTableProps {
	refreshTrigger: number;
	onProfileChange: () => void;
	onEditProfile: (profile: TranscriptionProfile) => void;
}

export function ProfilesTable({ refreshTrigger, onProfileChange, onEditProfile }: ProfilesTableProps) {
	const [profiles, setProfiles] = useState<TranscriptionProfile[]>([]);
	const [loading, setLoading] = useState(true);
	const [openPopovers, setOpenPopovers] = useState<Record<string, boolean>>({});
	const [deletingProfiles, setDeletingProfiles] = useState<Set<string>>(new Set());

	const fetchProfiles = useCallback(async () => {
		try {
			setLoading(true);
			const response = await fetch("/api/v1/profiles", {
				headers: {
					"X-API-Key": "dev-api-key-123",
				},
			});

			if (response.ok) {
				const data = await response.json();
				setProfiles(data);
			} else {
				console.error("Failed to fetch profiles");
			}
		} catch (error) {
			console.error("Error fetching profiles:", error);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		fetchProfiles();
	}, [refreshTrigger, fetchProfiles]);

	const handleDelete = useCallback(async (profileId: string) => {
		setOpenPopovers((prev) => ({ ...prev, [profileId]: false }));

		try {
			setDeletingProfiles((prev) => new Set(prev).add(profileId));
			
			const response = await fetch(`/api/v1/profiles/${profileId}`, {
				method: "DELETE",
				headers: {
					"X-API-Key": "dev-api-key-123",
				},
			});

			if (response.ok) {
				onProfileChange();
			} else {
				alert("Failed to delete profile");
			}
		} catch {
			alert("Error deleting profile");
		} finally {
			setDeletingProfiles((prev) => {
				const newSet = new Set(prev);
				newSet.delete(profileId);
				return newSet;
			});
		}
	}, [onProfileChange]);

	const formatDate = useCallback((dateString: string) => {
		return new Date(dateString).toLocaleDateString("en-US", {
			year: "numeric",
			month: "short",
			day: "numeric",
			hour: "2-digit",
			minute: "2-digit",
		});
	}, []);


	if (loading) {
		return (
			<div className="space-y-2">
				{[...Array(3)].map((_, i) => (
					<div
						key={i}
						className="bg-gray-100 dark:bg-gray-800 rounded-lg p-4 animate-pulse"
					>
						<div className="flex items-center gap-3">
							<div className="h-6 w-6 bg-gray-200 dark:bg-gray-600 rounded-md"></div>
							<div className="flex-1 space-y-2">
								<div className="h-4 bg-gray-200 dark:bg-gray-600 rounded w-1/3"></div>
								<div className="h-3 bg-gray-200 dark:bg-gray-600 rounded w-1/2"></div>
							</div>
							<div className="h-6 w-6 bg-gray-200 dark:bg-gray-600 rounded"></div>
						</div>
					</div>
				))}
			</div>
		);
	}

	if (profiles.length === 0) {
		return (
			<div className="text-center py-16">
				<div className="bg-gray-100 dark:bg-gray-700 rounded-full w-16 h-16 mx-auto mb-4 flex items-center justify-center">
					<Settings className="h-8 w-8 text-gray-400 dark:text-gray-500" />
				</div>
				<h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
					No profiles yet
				</h3>
				<p className="text-gray-600 dark:text-gray-400 mb-6 max-w-sm mx-auto">
					Create your first transcription profile to save and reuse your preferred settings.
				</p>
				<Button
					onClick={() => {}}
					variant="outline"
					className="border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-400"
				>
					Create Profile
				</Button>
			</div>
		);
	}

	return (
		<div className="space-y-2">
			{profiles.map((profile) => (
				<div
					key={profile.id}
					className="group bg-gray-100 dark:bg-gray-800 rounded-lg p-4 hover:bg-gray-200 dark:hover:bg-gray-700 transition-all duration-200 cursor-pointer"
					onClick={() => onEditProfile(profile)}
				>
					<div className="flex items-center justify-between">
						<div className="flex items-center gap-3 flex-1 min-w-0">
							<div className="bg-gray-200 dark:bg-gray-700 rounded-md p-1.5">
								<Terminal className="h-3.5 w-3.5 text-gray-500 dark:text-gray-400" />
							</div>
							<div className="flex-1 min-w-0">
								<div className="flex items-center gap-3">
									<h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
										{profile.name}
									</h3>
									<span className="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">
										{formatDate(profile.created_at)}
									</span>
								</div>
								{profile.description && (
									<p className="text-xs text-gray-500 dark:text-gray-400 truncate mt-1">
										{profile.description}
									</p>
								)}
							</div>
						</div>
						
						<div 
							className="opacity-0 group-hover:opacity-100 transition-opacity duration-200"
							onClick={(e) => e.stopPropagation()}
						>
							<Popover
								open={openPopovers[profile.id] || false}
								onOpenChange={(open) =>
									setOpenPopovers((prev) => ({
										...prev,
										[profile.id]: open,
									}))
								}
							>
								<PopoverTrigger asChild>
									<Button
										variant="ghost"
										size="sm"
										className="h-7 w-7 p-0 hover:bg-gray-300 dark:hover:bg-gray-600"
									>
										<MoreVertical className="h-3.5 w-3.5" />
									</Button>
								</PopoverTrigger>
								<PopoverContent className="w-32 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-600 p-1">
									<AlertDialog>
										<AlertDialogTrigger asChild>
											<Button
												variant="ghost"
												size="sm"
												className="w-full justify-start h-7 text-xs hover:bg-gray-100 dark:hover:bg-gray-700 text-red-500 dark:text-red-400 hover:text-red-600 dark:hover:text-red-300"
												disabled={deletingProfiles.has(profile.id)}
											>
												<Trash2 className="mr-2 h-3 w-3" />
												Delete
											</Button>
										</AlertDialogTrigger>
										<AlertDialogContent className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
											<AlertDialogHeader>
												<AlertDialogTitle className="text-gray-900 dark:text-gray-100">
													Delete Profile
												</AlertDialogTitle>
												<AlertDialogDescription className="text-gray-600 dark:text-gray-400">
													Are you sure you want to delete "{profile.name}"? This action cannot be undone.
												</AlertDialogDescription>
											</AlertDialogHeader>
											<AlertDialogFooter>
												<AlertDialogCancel className="bg-gray-100 dark:bg-gray-700 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-600">
													Cancel
												</AlertDialogCancel>
												<AlertDialogAction
													className="bg-red-600 text-white hover:bg-red-700"
													onClick={() => handleDelete(profile.id)}
												>
													Delete
												</AlertDialogAction>
											</AlertDialogFooter>
										</AlertDialogContent>
									</AlertDialog>
								</PopoverContent>
							</Popover>
						</div>
					</div>
				</div>
			))}
		</div>
	);
}