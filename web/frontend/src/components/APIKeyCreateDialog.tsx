import { useState } from "react";
import { Button } from "./ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "./ui/dialog";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Textarea } from "./ui/textarea";
import { useAuth } from "../contexts/AuthContext";

interface CreatedAPIKey {
	id: string;
	name: string;
	description?: string;
	key: string;
	created_at: string;
}

interface APIKeyCreateDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onKeyCreated: (keyData: CreatedAPIKey) => void;
}

export function APIKeyCreateDialog({
	open,
	onOpenChange,
	onKeyCreated,
}: APIKeyCreateDialogProps) {
	const [name, setName] = useState("");
	const [description, setDescription] = useState("");
	const [isCreating, setIsCreating] = useState(false);
	const [error, setError] = useState("");
	const { getAuthHeaders } = useAuth();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		
		if (!name.trim()) {
			setError("Name is required");
			return;
		}

		setIsCreating(true);
		setError("");

		try {
			const response = await fetch("/api/v1/api-keys/", {
				method: "POST",
				headers: {
					...getAuthHeaders(),
					"Content-Type": "application/json",
				},
				body: JSON.stringify({
					name: name.trim(),
					description: description.trim() || undefined,
				}),
			});

			if (response.ok) {
				const data = await response.json();
				// Backend returns the key data directly, not under api_key
				const keyData = {
					id: data.id.toString(),
					name: data.name,
					description: data.description,
					key: data.key,
					created_at: new Date().toISOString()
				};
				onKeyCreated(keyData);
				setName("");
				setDescription("");
				setError("");
			} else {
				const errorData = await response.json();
				setError(errorData.error || "Failed to create API key");
			}
		} catch (error) {
			console.error("Error creating API key:", error);
			setError("Failed to create API key. Please try again.");
		} finally {
			setIsCreating(false);
		}
	};

	const handleOpenChange = (newOpen: boolean) => {
		if (!isCreating) {
			onOpenChange(newOpen);
			if (!newOpen) {
				setName("");
				setDescription("");
				setError("");
			}
		}
	};

	return (
		<Dialog open={open} onOpenChange={handleOpenChange}>
			<DialogContent className="sm:max-w-md bg-white dark:bg-carbon-800 border border-carbon-200 dark:border-carbon-700">
				<DialogHeader>
					<DialogTitle>Create New API Key</DialogTitle>
					<DialogDescription>
						Create a new API key for external access to Scriberr. Give it a
						descriptive name to help you identify it later.
					</DialogDescription>
				</DialogHeader>

				<form onSubmit={handleSubmit} className="space-y-4">
					<div className="space-y-2">
						<Label htmlFor="name">Name *</Label>
						<Input
							id="name"
							placeholder="e.g., My App Integration"
							value={name}
							onChange={(e) => setName(e.target.value)}
							maxLength={100}
							disabled={isCreating}
						/>
						<div className="text-xs text-carbon-500 dark:text-carbon-400">
							A friendly name to identify this API key
						</div>
					</div>

					<div className="space-y-2">
						<Label htmlFor="description">Description</Label>
						<Textarea
							id="description"
							placeholder="Optional description of what this key will be used for"
							value={description}
							onChange={(e) => setDescription(e.target.value)}
							rows={3}
							disabled={isCreating}
						/>
					</div>

					{error && (
						<div className="text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
							{error}
						</div>
					)}

					<DialogFooter>
						<Button
							type="button"
							variant="outline"
							onClick={() => handleOpenChange(false)}
							disabled={isCreating}
						>
							Cancel
						</Button>
						<Button
							type="submit"
							disabled={isCreating || !name.trim()}
							className="bg-blue-600 hover:bg-blue-700 text-white"
						>
							{isCreating ? "Creating..." : "Create API Key"}
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}