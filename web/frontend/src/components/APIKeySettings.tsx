import { useState, useCallback } from "react";
import { Button } from "./ui/button";
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
			<div className="bg-gray-50 dark:bg-gray-700/50 rounded-xl p-4 sm:p-6">
				<div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 sm:gap-0 mb-4">
					<div>
						<h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
							API Keys
						</h3>
						<p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
							Manage your API keys for external access to Scriberr.
						</p>
					</div>
					<Button
						onClick={handleCreateAPIKey}
						className="bg-blue-600 hover:bg-blue-700 text-white"
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
