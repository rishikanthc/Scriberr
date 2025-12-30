import { useState, useEffect } from "react";
import { CheckCircle, AlertCircle, Loader2, Clock } from "lucide-react";
import { useAuth } from "@/features/auth/hooks/useAuth";

interface MergeStatusBadgeProps {
	jobId: string;
	mergeStatus?: string;
	mergeError?: string;
	className?: string;
}

export function MergeStatusBadge({ jobId, mergeStatus: initialStatus, mergeError: initialError, className }: MergeStatusBadgeProps) {
	const { getAuthHeaders } = useAuth();
	const [status, setStatus] = useState(initialStatus || "none");
	const [error, setError] = useState(initialError);

	// Poll for status updates when processing
	useEffect(() => {
		if (status === "processing" || status === "pending") {
			const interval = setInterval(async () => {
				try {
					const response = await fetch(`/api/v1/transcription/${jobId}/merge-status`, {
						headers: getAuthHeaders(),
					});
					
					if (response.ok) {
						const data = await response.json();
						setStatus(data.merge_status);
						setError(data.merge_error);
						
						// Stop polling when processing is complete
						if (data.merge_status === "completed" || data.merge_status === "failed") {
							clearInterval(interval);
						}
					}
				} catch (error) {
					console.error("Failed to fetch merge status:", error);
				}
			}, 2000); // Poll every 2 seconds

			return () => clearInterval(interval);
		}
	}, [status, jobId, getAuthHeaders]);

	if (status === "none") {
		return null; // Don't show badge for single-track files
	}

	const getStatusConfig = () => {
		switch (status) {
			case "pending":
				return {
					icon: Clock,
					text: "Merging pending",
					className: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300",
				};
			case "processing":
				return {
					icon: Loader2,
					text: "Merging...",
					className: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
					animate: true,
				};
			case "completed":
				return {
					icon: CheckCircle,
					text: "Merged",
					className: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
				};
			case "failed":
				return {
					icon: AlertCircle,
					text: "Merge failed",
					className: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300",
				};
			default:
				return null;
		}
	};

	const config = getStatusConfig();
	if (!config) return null;

	const Icon = config.icon;

	return (
		<span 
			className={`inline-flex items-center gap-1 px-2 py-1 text-xs font-medium rounded-md ${config.className} ${className || ""}`}
			title={error || undefined}
		>
			<Icon 
				className={`h-3 w-3 ${config.animate ? 'animate-spin' : ''}`} 
			/>
			{config.text}
		</span>
	);
}