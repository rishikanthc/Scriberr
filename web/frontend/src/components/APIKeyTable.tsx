import { useState, useEffect } from "react";
import { Button } from "./ui/button";
import { Trash2, Calendar, Clock } from "lucide-react";
import { useAuth } from "../contexts/AuthContext";

interface APIKey {
	id: string;
	name: string;
	description?: string;
	key_preview: string;
	created_at: string;
	last_used?: string;
}

interface APIKeyTableProps {
	refreshTrigger: number;
	onKeyChange: () => void;
}

export function APIKeyTable({ refreshTrigger, onKeyChange }: APIKeyTableProps) {
	const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
	const [loading, setLoading] = useState(true);
	const [deletingId, setDeletingId] = useState<string | null>(null);
	const { getAuthHeaders } = useAuth();

	const fetchAPIKeys = async () => {
		try {
			const response = await fetch("/api/v1/api-keys/", {
				headers: getAuthHeaders(),
			});

			if (response.ok) {
				const data = await response.json();
				setApiKeys(data.api_keys || []);
			} else {
				console.error("Failed to fetch API keys");
				setApiKeys([]);
			}
		} catch (error) {
			console.error("Error fetching API keys:", error);
			setApiKeys([]);
		} finally {
			setLoading(false);
		}
	};

	useEffect(() => {
		fetchAPIKeys();
	}, [refreshTrigger]);

	const handleDelete = async (id: string) => {
		if (!confirm("Are you sure you want to delete this API key? This action cannot be undone.")) {
			return;
		}

		setDeletingId(id);
		try {
			const response = await fetch(`/api/v1/api-keys/${id}`, {
				method: "DELETE",
				headers: getAuthHeaders(),
			});

			if (response.ok) {
				onKeyChange();
			} else {
				console.error("Failed to delete API key");
			}
		} catch (error) {
			console.error("Error deleting API key:", error);
		} finally {
			setDeletingId(null);
		}
	};

	const formatDate = (dateString: string) => {
		return new Date(dateString).toLocaleDateString("en-US", {
			year: "numeric",
			month: "short",
			day: "numeric",
		});
	};

	const formatDateTime = (dateString: string) => {
		return new Date(dateString).toLocaleString("en-US", {
			year: "numeric",
			month: "short",
			day: "numeric",
			hour: "2-digit",
			minute: "2-digit",
		});
	};

	if (loading) {
		return (
			<div className="flex items-center justify-center h-32">
				<div className="text-gray-500 dark:text-gray-400">Loading API keys...</div>
			</div>
		);
	}

	if (apiKeys.length === 0) {
		return (
			<div className="text-center py-8">
				<div className="text-gray-500 dark:text-gray-400 mb-2">
					No API keys found
				</div>
				<div className="text-sm text-gray-400 dark:text-gray-500">
					Create your first API key to get started with external access
				</div>
			</div>
		);
	}

	return (
		<div className="overflow-x-auto">
			<table className="w-full">
				<thead>
					<tr className="border-b border-gray-200 dark:border-gray-600">
						<th className="text-left py-3 px-4 font-medium text-gray-700 dark:text-gray-300">
							Name
						</th>
						<th className="text-left py-3 px-4 font-medium text-gray-700 dark:text-gray-300">
							Description
						</th>
						<th className="text-left py-3 px-4 font-medium text-gray-700 dark:text-gray-300">
							Key Preview
						</th>
						<th className="text-left py-3 px-4 font-medium text-gray-700 dark:text-gray-300">
							Created
						</th>
						<th className="text-left py-3 px-4 font-medium text-gray-700 dark:text-gray-300">
							Last Used
						</th>
						<th className="text-right py-3 px-4 font-medium text-gray-700 dark:text-gray-300">
							Actions
						</th>
					</tr>
				</thead>
				<tbody>
					{apiKeys.map((apiKey) => (
						<tr
							key={apiKey.id}
							className="border-b border-gray-100 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800/50"
						>
							<td className="py-3 px-4">
								<div className="font-medium text-gray-900 dark:text-gray-100">
									{apiKey.name}
								</div>
							</td>
							<td className="py-3 px-4">
								<div className="text-sm text-gray-600 dark:text-gray-400">
									{apiKey.description || "—"}
								</div>
							</td>
							<td className="py-3 px-4">
								<div className="font-mono text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
									{apiKey.key_preview}
								</div>
							</td>
							<td className="py-3 px-4">
								<div className="flex items-center text-sm text-gray-600 dark:text-gray-400">
									<Calendar className="h-4 w-4 mr-1" />
									{formatDate(apiKey.created_at)}
								</div>
							</td>
							<td className="py-3 px-4">
								<div className="flex items-center text-sm text-gray-600 dark:text-gray-400">
									{apiKey.last_used ? (
										<>
											<Clock className="h-4 w-4 mr-1" />
											{formatDateTime(apiKey.last_used)}
										</>
									) : (
										"Never"
									)}
								</div>
							</td>
							<td className="py-3 px-4 text-right">
								<Button
									variant="ghost"
									size="sm"
									onClick={() => handleDelete(apiKey.id)}
									disabled={deletingId === apiKey.id}
									className="text-red-600 hover:text-red-700 hover:bg-red-50 dark:text-red-400 dark:hover:text-red-300 dark:hover:bg-red-900/20"
								>
									<Trash2 className="h-4 w-4" />
								</Button>
							</td>
						</tr>
					))}
				</tbody>
			</table>
		</div>
	);
}