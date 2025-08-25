import { useEffect, useState } from "react";
import { useRouter } from "../contexts/RouterContext";
import { ChatInterface } from "../components/ChatInterface";
import { Button } from "../components/ui/button";
import { ArrowLeft } from "lucide-react";
import { ThemeSwitcher } from "../components/ThemeSwitcher";
import { useAuth } from "../contexts/AuthContext";
import { SidebarProvider, Sidebar, SidebarInset, SidebarTrigger } from "../components/ui/sidebar";
import { ChatSessionsSidebar } from "../components/ChatSessionsSidebar";

export function ChatPage() {
	const { currentRoute, navigate } = useRouter();
	const audioId = currentRoute.params?.audioId;
	const sessionId = currentRoute.params?.sessionId;
	const { getAuthHeaders } = useAuth();
	const [audioTitle, setAudioTitle] = useState<string | null>(null);

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
		<SidebarProvider defaultOpen={true} width={320}>
			<div className="relative h-[100dvh] overflow-hidden bg-gray-50 dark:bg-gray-900">
				{/* Sidebar fixed on left, below header */}
				<Sidebar topOffset={56}>
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
				</Sidebar>

				{/* Main inset shifts when sidebar opens */}
				<SidebarInset className="h-full flex flex-col">
					{/* Header (fixed height) */}
					<div className="h-14 bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60 z-10 shadow-sm flex items-center">
						<div className="w-full max-w-[1400px] mx-auto px-4 flex items-center justify-between gap-3">
							<div className="flex items-center gap-2">
								<SidebarTrigger className="sm:hidden">Menu</SidebarTrigger>
								<Button
									variant="ghost"
									size="sm"
									onClick={() => navigate({ path: "audio-detail", params: { id: audioId } })}
									className="gap-2"
								>
									<ArrowLeft className="h-4 w-4" />
									Back to Transcript
								</Button>
							</div>
							<div className="flex items-center gap-3">
								<div className="text-sm text-muted-foreground truncate max-w-[50vw]">
									{audioTitle || audioId}
								</div>
								<ThemeSwitcher />
							</div>
						</div>
					</div>

					{/* Chat content (no page scroll) */}
					<div className="flex-1 min-h-0 overflow-hidden">
						<ChatInterface
							transcriptionId={audioId}
							activeSessionId={sessionId}
							hideSidebar
							onSessionChange={(newSessionId) => {
								if (!newSessionId) {
									navigate({ path: "audio-detail", params: { id: audioId } });
								} else {
									navigate({ path: "chat", params: { audioId, sessionId: newSessionId } });
								}
							}}
						/>
					</div>
				</SidebarInset>
			</div>
		</SidebarProvider>
	);
}
