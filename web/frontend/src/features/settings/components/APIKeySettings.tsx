import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { APIKeyTable } from "./APIKeyTable";
import { APIKeyCreateDialog } from "./APIKeyCreateDialog";
import { APIKeyDisplayDialog } from "./APIKeyDisplayDialog";

interface CreatedAPIKey {
	id: string;
	name: string;
	description?: string;
	key: string;
	created_at: string;
}

export function APIKeySettings() {
	const [createDialogOpen, setCreateDialogOpen] = useState(false);
	const [displayDialogOpen, setDisplayDialogOpen] = useState(false);
	const [createdKey, setCreatedKey] = useState<CreatedAPIKey | null>(null);
	const [refreshTrigger, setRefreshTrigger] = useState(0);

	const handleCreateAPIKey = useCallback(() => {
		setCreateDialogOpen(true);
	}, []);

	const handleKeyCreated = useCallback(async (keyData: CreatedAPIKey) => {
		setCreatedKey(keyData);
		setCreateDialogOpen(false);
		setDisplayDialogOpen(true);
		setRefreshTrigger((prev) => prev + 1);
	}, []);

	const handleKeyChange = useCallback(() => {
		setRefreshTrigger((prev) => prev + 1);
	}, []);

	const handleDisplayDialogClose = useCallback(() => {
		setDisplayDialogOpen(false);
		setCreatedKey(null);
	}, []);

	return (
		<div className="space-y-6">
			<div className="bg-[var(--bg-main)]/50 border border-[var(--border-subtle)] rounded-[var(--radius-card)] p-4 sm:p-6 shadow-sm">
				<div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 sm:gap-0 mb-4">
					<div>
						<h3 className="text-lg font-medium text-[var(--text-primary)]">
							API Keys
						</h3>
						<p className="text-sm text-[var(--text-secondary)] mt-1">
							Manage your API keys for external access to Scriberr.
						</p>
					</div>
					<Button
						onClick={handleCreateAPIKey}
						className="!bg-[var(--brand-gradient)] hover:!opacity-90 !text-black dark:!text-white shadow-lg shadow-orange-500/20 border-none"
					>
						Create New API Key
					</Button>
				</div>

				<APIKeyTable
					refreshTrigger={refreshTrigger}
					onKeyChange={handleKeyChange}
				/>
			</div>

			<APIKeyCreateDialog
				open={createDialogOpen}
				onOpenChange={setCreateDialogOpen}
				onKeyCreated={handleKeyCreated}
			/>

			<APIKeyDisplayDialog
				open={displayDialogOpen}
				onOpenChange={setDisplayDialogOpen}
				apiKey={createdKey}
				onClose={handleDisplayDialogClose}
			/>
		</div>
	);
}
