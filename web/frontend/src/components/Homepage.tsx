import { useState } from "react";
import { Header } from "./Header";
import { AudioFilesTable } from "./AudioFilesTable";
import { useAuth } from "../contexts/AuthContext";

export function Homepage() {
	const { getAuthHeaders } = useAuth();
	const [refreshTrigger, setRefreshTrigger] = useState(0);

	const handleFileSelect = async (file: File) => {
		const formData = new FormData();
		formData.append("audio", file);
		formData.append("title", file.name.replace(/\.[^/.]+$/, ""));

		try {
			const response = await fetch("/api/v1/transcription/upload", {
				method: "POST",
				headers: {
					...getAuthHeaders(),
				},
				body: formData,
			});

			if (response.ok) {
				// Refresh the table to show the new file
				setRefreshTrigger((prev) => prev + 1);
			} else {
				alert("Failed to upload file");
			}
		} catch {
			alert("Error uploading file");
		}
	};

	const handleTranscribe = () => {
		// Refresh table when transcription starts
		setRefreshTrigger((prev) => prev + 1);
	};

	return (
		<div className="min-h-screen bg-gray-50 dark:bg-gray-900">
			<div className="mx-auto px-8 py-6" style={{ width: "60vw" }}>
				<Header onFileSelect={handleFileSelect} />
				<AudioFilesTable
					refreshTrigger={refreshTrigger}
					onTranscribe={handleTranscribe}
				/>
			</div>
		</div>
	);
}
