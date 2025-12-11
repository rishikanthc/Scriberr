import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Copy, Check, AlertTriangle } from "lucide-react";

interface CreatedAPIKey {
	id: string;
	name: string;
	description?: string;
	key: string;
	created_at: string;
}

interface APIKeyDisplayDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	apiKey: CreatedAPIKey | null;
	onClose: () => void;
}

export function APIKeyDisplayDialog({
	open,
	onOpenChange,
	apiKey,
	onClose,
}: APIKeyDisplayDialogProps) {
	const [copied, setCopied] = useState(false);

	const handleCopy = async () => {
		if (!apiKey?.key) return;

		try {
			await navigator.clipboard.writeText(apiKey.key);
			setCopied(true);
			setTimeout(() => setCopied(false), 2000);
		} catch (error) {
			console.error("Failed to copy to clipboard:", error);
			// Fallback for older browsers
			const textArea = document.createElement("textarea");
			textArea.value = apiKey.key;
			document.body.appendChild(textArea);
			textArea.select();
			document.execCommand("copy");
			document.body.removeChild(textArea);
			setCopied(true);
			setTimeout(() => setCopied(false), 2000);
		}
	};

	const handleClose = () => {
		setCopied(false);
		onClose();
	};

	const handleOpenChange = (newOpen: boolean) => {
		if (!newOpen) {
			handleClose();
		}
		onOpenChange(newOpen);
	};

	if (!apiKey) return null;

	return (
		<Dialog open={open} onOpenChange={handleOpenChange}>
			<DialogContent className="sm:max-w-lg bg-white dark:bg-carbon-800 border border-carbon-200 dark:border-carbon-700">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Check className="h-5 w-5 text-green-600" />
						API Key Created Successfully
					</DialogTitle>
					<DialogDescription>
						Your API key has been created. Copy it now and store it securely -
						you won't be able to see the full key again.
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-4">
					<div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
						<div className="flex items-start gap-3">
							<AlertTriangle className="h-5 w-5 text-yellow-600 dark:text-yellow-400 mt-0.5 flex-shrink-0" />
							<div className="text-sm text-yellow-800 dark:text-yellow-200">
								<div className="font-medium mb-1">Important Security Notice</div>
								<div>
									This is the only time you'll see the full API key. Make sure to copy
									and store it in a secure location before closing this dialog.
								</div>
							</div>
						</div>
					</div>

					<div className="space-y-3">
						<div>
							<Label className="text-sm font-medium">Name</Label>
							<div className="text-carbon-900 dark:text-carbon-100 mt-1">
								{apiKey.name}
							</div>
						</div>

						{apiKey.description && (
							<div>
								<Label className="text-sm font-medium">Description</Label>
								<div className="text-carbon-600 dark:text-carbon-400 mt-1">
									{apiKey.description}
								</div>
							</div>
						)}

						<div>
							<Label htmlFor="api-key" className="text-sm font-medium">
								API Key
							</Label>
							<div className="flex items-center gap-2 mt-1">
								<Input
									id="api-key"
									value={apiKey.key}
									readOnly
									className="font-mono text-sm bg-carbon-50 dark:bg-carbon-800"
									onClick={(e) => (e.target as HTMLInputElement).select()}
								/>
								<Button
									onClick={handleCopy}
									size="sm"
									variant="outline"
									className="flex-shrink-0"
									disabled={copied}
								>
									{copied ? (
										<Check className="h-4 w-4 text-green-600" />
									) : (
										<Copy className="h-4 w-4" />
									)}
								</Button>
							</div>
							{copied && (
								<div className="text-sm text-green-600 dark:text-green-400 mt-1">
									Copied to clipboard!
								</div>
							)}
						</div>

						<div className="bg-carbon-50 dark:bg-carbon-800 rounded-lg p-3">
							<div className="text-sm text-carbon-600 dark:text-carbon-400">
								<div className="font-medium mb-1">Usage Example:</div>
								<div className="font-mono text-xs bg-white dark:bg-carbon-900 p-2 rounded border">
									curl -H "X-API-Key: {apiKey.key}" http://localhost:8080/api/v1/...
								</div>
							</div>
						</div>
					</div>
				</div>

				<DialogFooter>
					<Button onClick={handleClose} className="w-full">
						I've Saved My API Key
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}