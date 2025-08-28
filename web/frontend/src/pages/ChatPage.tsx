import { useEffect, useState } from "react";
import { useRouter } from "../contexts/RouterContext";
import { ChatInterface } from "../components/ChatInterface";
import { Button } from "../components/ui/button";
import { ArrowLeft, Sidebar } from "lucide-react";
import { ThemeSwitcher } from "../components/ThemeSwitcher";
import { useAuth } from "../contexts/AuthContext";
import { ChatSessionsSidebar } from "../components/ChatSessionsSidebar";

export function ChatPage() {
	const { currentRoute, navigate } = useRouter();
	const audioId = currentRoute.params?.audioId;
	const sessionId = currentRoute.params?.sessionId;
	const { getAuthHeaders } = useAuth();
	const [audioTitle, setAudioTitle] = useState<string | null>(null);
	const [showSidebar, setShowSidebar] = useState(true);

	useEffect(() => {
		// If we somehow landed on chat without required params, bounce home
		if (!audioId) {
			navigate({ path: "home" });
		}
	}, [audioId, navigate]);

	useEffect(() => {
		if (!audioId) return;
		const fetchTitle = async () => {
			try {
				const res = await fetch(`/api/v1/transcription/${audioId}`, {
					headers: getAuthHeaders(),
				});
				if (res.ok) {
					const data = await res.json();
					setAudioTitle(data?.title || null);
				} else {
					setAudioTitle(null);
				}
			} catch {
				setAudioTitle(null);
			}
		};
		fetchTitle();
	}, [audioId, getAuthHeaders]);

	if (!audioId) return null;

	return (
		<div className="text-gray-700 dark:text-gray-100 bg-white dark:bg-gray-900 h-screen max-h-[100dvh] overflow-auto flex flex-row justify-end">
			{/* Sidebar */}
			{showSidebar && (
				<div className="fixed inset-y-0 left-0 z-40 w-80 bg-white dark:bg-gray-850 md:relative md:translate-x-0">
					<ChatSessionsSidebar
						transcriptionId={audioId}
						activeSessionId={sessionId}
						onSessionChange={(newSessionId) => {
							if (!newSessionId) {
								navigate({ path: "audio-detail", params: { id: audioId } });
							} else {
								navigate({ path: "chat", params: { audioId, sessionId: newSessionId } });
							}
						}}
					/>
				</div>
			)}

			{/* Overlay for mobile sidebar */}
			{showSidebar && (
				<div 
					className="fixed inset-0 bg-black bg-opacity-50 z-30 md:hidden"
					onClick={() => setShowSidebar(false)}
				/>
			)}

			{/* Main Content Area */}
			<div className={`transition-width duration-200 ease-in-out ${
				showSidebar ? 'md:max-w-[calc(100%-320px)]' : ''
			} w-full max-w-full flex flex-col`}>
				{/* Top Navigation Bar */}
				<div className="h-14 bg-white dark:bg-gray-900 flex items-center px-4 md:px-6 z-10">
					<div className="flex items-center gap-3 flex-1">
						{/* Sidebar Toggle Button */}
						<Button
							variant="ghost"
							size="sm"
							onClick={() => setShowSidebar(!showSidebar)}
							className="p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100"
							title="Toggle Sidebar"
						>
							<Sidebar className="h-4 w-4" />
						</Button>

						{/* Back Button */}
						<Button
							variant="ghost"
							size="sm"
							onClick={() => navigate({ path: "audio-detail", params: { id: audioId } })}
							className="gap-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100"
						>
							<ArrowLeft className="h-4 w-4" />
							Back to Transcript
						</Button>

						{/* Title */}
						<div className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
							{audioTitle || "Chat Session"}
						</div>
					</div>

					{/* Right side controls */}
					<div className="flex items-center gap-2">
						<ThemeSwitcher />
					</div>
				</div>

				{/* Chat Interface */}
				<div className="flex-1 h-0 bg-white dark:bg-gray-900">
					<ChatInterface
						transcriptionId={audioId}
						activeSessionId={sessionId}
						hideSidebar
						onSessionChange={(newSessionId) => {
							if (newSessionId) {
								navigate({ path: "chat", params: { audioId, sessionId: newSessionId } });
							} else {
								// Stay on chat page but remove sessionId from URL
								navigate({ path: "chat", params: { audioId } });
							}
						}}
					/>
				</div>
			</div>
		</div>
	);
}

