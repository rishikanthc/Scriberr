import { useState, useEffect, useCallback } from "react";
import { MoreVertical, Trash2, Settings, Terminal } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useAuth } from "@/features/auth/hooks/useAuth";
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
} from "@/components/ui/alert-dialog";
import type { WhisperXParams } from "@/components/TranscriptionConfigDialog";

interface TranscriptionProfile {
	id: string;
	name: string;
	description?: string;
	is_default: boolean;
	parameters: WhisperXParams;
	created_at: string;
	updated_at: string;
}

interface ProfilesTableProps {
	refreshTrigger: number;
	onProfileChange: () => void;
	onEditProfile: (profile: TranscriptionProfile) => void;
	onCreateProfile?: () => void;
}

export function ProfilesTable({
	refreshTrigger,
	onProfileChange,
	onEditProfile,
	onCreateProfile,
}: ProfilesTableProps) {
	const { getAuthHeaders } = useAuth();
	const [profiles, setProfiles] = useState<TranscriptionProfile[]>([]);
	const [loading, setLoading] = useState(true);
	const [openPopovers, setOpenPopovers] = useState<Record<string, boolean>>({});
	const [deletingProfiles, setDeletingProfiles] = useState<Set<string>>(
		new Set(),
	);

	const fetchProfiles = useCallback(async () => {
		try {
			setLoading(true);
			const response = await fetch("/api/v1/profiles", {
				headers: {
					...getAuthHeaders(),
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
	}, [getAuthHeaders]);

	useEffect(() => {
		fetchProfiles();
	}, [refreshTrigger, fetchProfiles]);

	const handleDelete = useCallback(
		async (profileId: string) => {
			setOpenPopovers((prev) => ({ ...prev, [profileId]: false }));

			try {
				setDeletingProfiles((prev) => new Set(prev).add(profileId));

				const response = await fetch(`/api/v1/profiles/${profileId}`, {
					method: "DELETE",
					headers: {
						...getAuthHeaders(),
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
		},
		[onProfileChange, getAuthHeaders],
	);

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
						className="bg-carbon-100 dark:bg-carbon-800 rounded-lg p-4 animate-pulse"
					>
						<div className="flex items-center gap-3">
							<div className="h-6 w-6 bg-carbon-200 dark:bg-carbon-600 rounded-md"></div>
							<div className="flex-1 space-y-2">
								<div className="h-4 bg-carbon-200 dark:bg-carbon-600 rounded w-1/3"></div>
								<div className="h-3 bg-carbon-200 dark:bg-carbon-600 rounded w-1/2"></div>
							</div>
							<div className="h-6 w-6 bg-carbon-200 dark:bg-carbon-600 rounded"></div>
						</div>
					</div>
				))}
			</div>
		);
	}

	if (profiles.length === 0) {
		return (
			<div className="text-center py-16">
				<div className="bg-[var(--bg-main)] rounded-full w-16 h-16 mx-auto mb-4 flex items-center justify-center border border-[var(--border-subtle)]">
					<Settings className="h-8 w-8 text-[var(--text-tertiary)]" />
				</div>
				<h3 className="text-lg font-medium text-[var(--text-primary)] mb-2">
					No profiles yet
				</h3>
				<p className="text-[var(--text-secondary)] mb-6 max-w-sm mx-auto">
					Create your first transcription profile to save and reuse your
					preferred settings.
				</p>
				<Button
					onClick={() => onCreateProfile?.()}
					variant="outline"
					className="border-[var(--border-subtle)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-secondary)]"
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
					className="group bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-lg p-4 hover:border-[var(--brand-solid)] transition-all duration-200 cursor-pointer shadow-sm"
					onClick={() => onEditProfile(profile)}
				>
					<div className="flex items-center justify-between">
						<div className="flex items-center gap-3 flex-1 min-w-0">
							<div className="bg-[var(--bg-main)] rounded-md p-1.5 text-[var(--text-tertiary)]">
								<Terminal className="h-3.5 w-3.5" />
							</div>
							<div className="flex-1 min-w-0">
								<div className="flex items-center gap-3">
									<h3 className="text-sm font-medium text-[var(--text-primary)] truncate">
										{profile.name}
									</h3>
									<span className="text-xs text-[var(--text-tertiary)] whitespace-nowrap">
										{formatDate(profile.created_at)}
									</span>
								</div>
								{profile.description && (
									<p className="text-xs text-[var(--text-secondary)] truncate mt-1">
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
										className="h-7 w-7 p-0 hover:bg-carbon-300 dark:hover:bg-carbon-600"
									>
										<MoreVertical className="h-3.5 w-3.5" />
									</Button>
								</PopoverTrigger>
								<PopoverContent className="w-32 bg-[var(--bg-card)] border-[var(--border-subtle)] p-1 text-[var(--text-primary)]">
									<AlertDialog>
										<AlertDialogTrigger asChild>
											<Button
												variant="ghost"
												size="sm"
												className="w-full justify-start h-7 text-xs hover:bg-[var(--error)]/10 text-[var(--error)] hover:text-[var(--error)]"
												disabled={deletingProfiles.has(profile.id)}
											>
												<Trash2 className="mr-2 h-3 w-3" />
												Delete
											</Button>
										</AlertDialogTrigger>
										<AlertDialogContent className="bg-[var(--bg-card)] border-[var(--border-subtle)]">
											<AlertDialogHeader>
												<AlertDialogTitle className="text-[var(--text-primary)]">
													Delete Profile
												</AlertDialogTitle>
												<AlertDialogDescription className="text-[var(--text-secondary)]">
													Are you sure you want to delete "{profile.name}"? This
													action cannot be undone.
												</AlertDialogDescription>
											</AlertDialogHeader>
											<AlertDialogFooter>
												<AlertDialogCancel className="bg-[var(--bg-secondary)] border-[var(--border-subtle)] text-[var(--text-primary)] hover:bg-[var(--bg-main)]">
													Cancel
												</AlertDialogCancel>
												<AlertDialogAction
													className="bg-[var(--error)] text-white hover:bg-[var(--error)]/90"
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
