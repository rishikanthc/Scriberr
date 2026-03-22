import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/features/auth/hooks/useAuth";

interface OpenClawProfile {
	id: string;
	name: string;
}

interface SendToOpenClawDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	audioId: string;
	title?: string;
	onSent?: () => void;
}

export function SendToOpenClawDialog({ open, onOpenChange, audioId, title, onSent }: SendToOpenClawDialogProps) {
	const { getAuthHeaders } = useAuth();
	const { toast } = useToast();
	const [profiles, setProfiles] = useState<OpenClawProfile[]>([]);
	const [loadingProfiles, setLoadingProfiles] = useState(false);
	const [sending, setSending] = useState(false);
	const [selectedProfileId, setSelectedProfileId] = useState("");
	const [error, setError] = useState("");

	useEffect(() => {
		if (!open) return;

		let mounted = true;
		const fetchProfiles = async () => {
			try {
				setLoadingProfiles(true);
				setError("");
				const res = await fetch("/api/v1/openclaw/profiles", {
					headers: getAuthHeaders(),
				});
				if (!res.ok) {
					throw new Error("Failed to load OpenClaw profiles");
				}
				const data = await res.json();
				if (!mounted) return;
				const nextProfiles = Array.isArray(data) ? data : [];
				setProfiles(nextProfiles);
				setSelectedProfileId(nextProfiles[0]?.id ?? "");
			} catch (err) {
				if (!mounted) return;
				setError(err instanceof Error ? err.message : "Failed to load OpenClaw profiles");
			} finally {
				if (mounted) {
					setLoadingProfiles(false);
				}
			}
		};

		void fetchProfiles();
		return () => {
			mounted = false;
		};
	}, [open, getAuthHeaders]);

	const disabled = useMemo(() => sending || loadingProfiles || !selectedProfileId, [sending, loadingProfiles, selectedProfileId]);

	const handleSend = async () => {
		if (!selectedProfileId) return;
		setError("");
		try {
			setSending(true);
			const res = await fetch(`/api/v1/transcription/${audioId}/send-openclaw`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					...getAuthHeaders(),
				},
				body: JSON.stringify({ profile_id: selectedProfileId }),
			});
			if (!res.ok) {
				const text = await res.text();
				throw new Error(text || "Failed to send to OpenClaw");
			}
			toast({
				title: "Sent to OpenClaw",
				description: `${title || "This transcript"} was sent successfully.`,
			});
			onSent?.();
			onOpenChange(false);
		} catch (err) {
			setError(err instanceof Error ? err.message : "Failed to send to OpenClaw");
		} finally {
			setSending(false);
		}
	};

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-[560px] bg-[var(--bg-card)] border border-[var(--border-subtle)]">
				<DialogHeader>
					<DialogTitle>Send to OpenClaw</DialogTitle>
					<DialogDescription>
						选择一个预配置 profile，系统会自动通过 SCP 上传 SRT 并触发远端 OpenClaw hook。
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-3">
					<label className="text-sm text-[var(--text-secondary)]">OpenClaw Profile</label>
					<Select value={selectedProfileId} onValueChange={setSelectedProfileId} disabled={loadingProfiles || profiles.length === 0}>
						<SelectTrigger>
							<SelectValue placeholder={loadingProfiles ? "Loading..." : "Select a profile"} />
						</SelectTrigger>
						<SelectContent>
							{profiles.map((profile) => (
								<SelectItem key={profile.id} value={profile.id}>
									{profile.name}
								</SelectItem>
							))}
						</SelectContent>
					</Select>

					{profiles.length === 0 && !loadingProfiles && (
						<div className="text-sm text-[var(--warning-solid)]">
							No profiles found. Please create one in Settings &gt; OpenClaw.
						</div>
					)}

					{error && (
						<div className="text-sm text-[var(--error)] bg-[var(--error)]/10 border border-[var(--error)]/20 rounded-md p-2">
							{error}
						</div>
					)}
				</div>

				<DialogFooter>
					<Button variant="ghost" onClick={() => onOpenChange(false)} disabled={sending}>
						Cancel
					</Button>
					<Button onClick={handleSend} disabled={disabled}>
						{sending ? "Sending..." : "Send"}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
