import { useState, useEffect } from "react";
import { CheckCircle, Clock, XCircle, Loader2 } from "lucide-react";

interface AudioFile {
	id: string;
	title?: string;
	status: "uploaded" | "pending" | "processing" | "completed" | "failed";
	created_at: string;
	audio_path: string;
}

interface AudioFilesTableProps {
	refreshTrigger: number;
}

export function AudioFilesTable({ refreshTrigger }: AudioFilesTableProps) {
	const [audioFiles, setAudioFiles] = useState<AudioFile[]>([]);
	const [loading, setLoading] = useState(true);

	const fetchAudioFiles = async () => {
		try {
			const response = await fetch("/api/v1/transcription/list", {
				headers: {
					"X-API-Key": "dev-api-key-123",
				},
			});

			if (response.ok) {
				const data = await response.json();
				setAudioFiles(data.jobs || []);
			}
		} catch (error) {
			console.error("Failed to fetch audio files:", error);
		} finally {
			setLoading(false);
		}
	};

	useEffect(() => {
		fetchAudioFiles();
	}, [refreshTrigger]);

	const getStatusIcon = (status: string) => {
		const iconSize = 16;

		switch (status) {
			case "completed":
				return <CheckCircle size={iconSize} className="text-magnum-400" />;
			case "processing":
				return (
					<Loader2 size={iconSize} className="text-blue-400 animate-spin" />
				);
			case "failed":
				return <XCircle size={iconSize} className="text-magenta-400" />;
			case "pending":
				return <Clock size={iconSize} className="text-purple-400" />;
			case "uploaded":
			default:
				return <Clock size={iconSize} className="text-neon-100" />;
		}
	};

	const formatDate = (dateString: string) => {
		return new Date(dateString).toLocaleDateString("en-US", {
			year: "numeric",
			month: "short",
			day: "numeric",
			hour: "2-digit",
			minute: "2-digit",
		});
	};

	const getFileName = (audioPath: string) => {
		const parts = audioPath.split("/");
		return parts[parts.length - 1];
	};

	if (loading) {
		return (
			<div className="bg-gray-800 rounded-xl p-6">
				<div className="animate-pulse">
					<div className="h-4 bg-gray-700 rounded w-1/4 mb-6"></div>
					<div className="space-y-3">
						{[...Array(5)].map((_, i) => (
							<div key={i} className="h-12 bg-gray-700/50 rounded-lg"></div>
						))}
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="bg-gray-800 rounded-xl overflow-hidden">
			<div className="p-6">
				<h2 className="text-xl font-semibold text-gray-50 mb-2">Audio Files</h2>
				<p className="text-gray-400 text-sm">
					{audioFiles.length} file{audioFiles.length !== 1 ? "s" : ""} total
				</p>
			</div>

			{audioFiles.length === 0 ? (
				<div className="p-12 text-center">
					<div className="text-5xl mb-4 opacity-50">🎵</div>
					<h3 className="text-lg font-medium text-gray-300 mb-2">
						No audio files yet
					</h3>
					<p className="text-gray-500">
						Upload your first audio file to get started
					</p>
				</div>
			) : (
				<div className="overflow-x-auto">
					<table className="w-full">
						<thead className="bg-gray-700">
							<tr>
								<th className="text-left px-6 py-3 text-sm font-medium text-gray-300">
									Title
								</th>
								<th className="text-left px-6 py-3 text-sm font-medium text-gray-300">
									Date Added
								</th>
								<th className="text-left px-6 py-3 text-sm font-medium text-gray-300">
									Status
								</th>
							</tr>
						</thead>
						<tbody>
							{audioFiles.map((file) => (
								<tr
									key={file.id}
									className="hover:bg-gray-700 transition-colors duration-200"
								>
									<td className="px-6 py-3">
										<span className="text-gray-50 font-medium">
											{file.title || getFileName(file.audio_path)}
										</span>
									</td>
									<td className="px-6 py-3 text-gray-300 text-sm">
										{formatDate(file.created_at)}
									</td>
									<td className="px-6 py-3">{getStatusIcon(file.status)}</td>
								</tr>
							))}
						</tbody>
					</table>
				</div>
			)}
		</div>
	);
}
