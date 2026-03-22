import { useCallback, useEffect, useRef, useState } from "react";
import type { ChangeEvent } from "react";
import { Pencil, Plus, Send, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useAuth } from "@/features/auth/hooks/useAuth";

interface OpenClawProfile {
	id: string;
	name: string;
	ip: string;
	hook_name: string;
	message: string;
	has_ssh_key: boolean;
	has_hook_key: boolean;
}

interface ProfileFormState {
	name: string;
	ip: string;
	ssh_key: string;
	hook_key: string;
	hook_name: string;
	message: string;
}

const emptyForm: ProfileFormState = {
	name: "",
	ip: "",
	ssh_key: "",
	hook_key: "",
	hook_name: "Dashboard",
	message: "请根据上传的 SRT 输出会议总结，提炼关键决策、行动项和风险。",
};

export function OpenClawProfileSettings() {
	const { getAuthHeaders } = useAuth();
	const [profiles, setProfiles] = useState<OpenClawProfile[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState("");
	const [dialogOpen, setDialogOpen] = useState(false);
	const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
	const [editing, setEditing] = useState<OpenClawProfile | null>(null);
	const [deleting, setDeleting] = useState<OpenClawProfile | null>(null);
	const [saving, setSaving] = useState(false);
	const [form, setForm] = useState<ProfileFormState>(emptyForm);
	const [sshKeyFileName, setSshKeyFileName] = useState("");
	const sshKeyInputRef = useRef<HTMLInputElement | null>(null);

	const fetchProfiles = useCallback(async () => {
		try {
			setLoading(true);
			setError("");
			const res = await fetch("/api/v1/openclaw/profiles", {
				headers: getAuthHeaders(),
			});
			if (!res.ok) {
				throw new Error("Failed to fetch OpenClaw profiles");
			}
			const data = await res.json();
			setProfiles(Array.isArray(data) ? data : []);
		} catch (err) {
			setError(err instanceof Error ? err.message : "Failed to fetch OpenClaw profiles");
		} finally {
			setLoading(false);
		}
	}, [getAuthHeaders]);

	useEffect(() => {
		void fetchProfiles();
	}, [fetchProfiles]);

	const openCreateDialog = () => {
		setEditing(null);
		setForm(emptyForm);
		setSshKeyFileName("");
		setDialogOpen(true);
	};

	const openEditDialog = (profile: OpenClawProfile) => {
		setEditing(profile);
		setForm({
			name: profile.name,
			ip: profile.ip,
			ssh_key: "",
			hook_key: "",
			hook_name: profile.hook_name,
			message: profile.message,
		});
		setSshKeyFileName(profile.has_ssh_key ? "已保存 SSH Key，可重新选择文件覆盖" : "");
		setDialogOpen(true);
	};

	const triggerSshKeyFilePicker = () => {
		sshKeyInputRef.current?.click();
	};

	const onSshKeyFileSelected = async (event: ChangeEvent<HTMLInputElement>) => {
		const file = event.target.files?.[0];
		if (!file) return;
		try {
			const keyContent = await file.text();
			setForm((state) => ({ ...state, ssh_key: keyContent }));
			setSshKeyFileName(file.name);
			setError("");
		} catch {
			setError("Failed to read SSH key file");
		} finally {
			event.target.value = "";
		}
	};

	const onSave = async () => {
		setError("");
		if (!form.name.trim() || !form.ip.trim() || !form.hook_name.trim() || !form.message.trim()) {
			setError("Name, host, hook name, and message are required");
			return;
		}
		if (!editing && (!form.ssh_key.trim() || !form.hook_key.trim())) {
			setError("SSH key and hook key are required");
			return;
		}

		try {
			setSaving(true);
			const isEditing = !!editing;
			const url = isEditing ? `/api/v1/openclaw/profiles/${editing?.id}` : "/api/v1/openclaw/profiles";
			const method = isEditing ? "PUT" : "POST";
			const res = await fetch(url, {
				method,
				headers: {
					"Content-Type": "application/json",
					...getAuthHeaders(),
				},
				body: JSON.stringify(form),
			});
			if (!res.ok) {
				const text = await res.text();
				throw new Error(text || "Failed to save profile");
			}

			setDialogOpen(false);
			setEditing(null);
			setForm(emptyForm);
			setSshKeyFileName("");
			await fetchProfiles();
		} catch (err) {
			setError(err instanceof Error ? err.message : "Failed to save profile");
		} finally {
			setSaving(false);
		}
	};

	const requestDelete = (profile: OpenClawProfile) => {
		setDeleting(profile);
		setDeleteDialogOpen(true);
	};

	const confirmDelete = async () => {
		if (!deleting) return;
		try {
			const res = await fetch(`/api/v1/openclaw/profiles/${deleting.id}`, {
				method: "DELETE",
				headers: getAuthHeaders(),
			});
			if (!res.ok) {
				const text = await res.text();
				throw new Error(text || "Failed to delete profile");
			}
			setDeleteDialogOpen(false);
			setDeleting(null);
			await fetchProfiles();
		} catch (err) {
			setError(err instanceof Error ? err.message : "Failed to delete profile");
		}
	};

	return (
		<div className="space-y-6">
			{error && (
				<div className="bg-[var(--error)]/10 border border-[var(--error)]/20 rounded-lg p-3">
					<p className="text-[var(--error)] text-sm">{error}</p>
				</div>
			)}

			<div className="bg-[var(--bg-main)]/50 border border-[var(--border-subtle)] rounded-[var(--radius-card)] p-4 sm:p-6 shadow-sm">
				<div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 mb-4">
					<div>
						<h3 className="text-lg font-medium text-[var(--text-primary)] flex items-center gap-2">
							<Send className="h-5 w-5 text-[var(--brand-solid)]" />
							OpenClaw Profiles
						</h3>
						<p className="text-sm text-[var(--text-secondary)] mt-1">
							配置远端连接和 OpenClaw hook，用于发送 SRT 并触发总结。
						</p>
					</div>
					<Button onClick={openCreateDialog} className="!bg-[var(--brand-gradient)] hover:!opacity-90 !text-black dark:!text-white border-none">
						<Plus className="h-4 w-4 mr-2" />
						New Profile
					</Button>
				</div>

				{loading ? (
					<div className="py-6 text-sm text-[var(--text-secondary)]">Loading profiles...</div>
				) : profiles.length === 0 ? (
					<div className="py-10 text-center text-sm text-[var(--text-secondary)] border border-dashed border-[var(--border-subtle)] rounded-lg">
						No OpenClaw profiles yet.
					</div>
				) : (
					<div className="space-y-3">
						{profiles.map((profile) => (
							<div key={profile.id} className="flex items-center justify-between gap-3 bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-lg p-4">
								<div className="min-w-0">
									<div className="text-sm font-medium text-[var(--text-primary)] truncate">{profile.name}</div>
									<div className="text-xs text-[var(--text-secondary)] truncate mt-1">{profile.ip}</div>
									<div className="text-xs text-[var(--text-tertiary)] truncate mt-1">Hook Name: {profile.hook_name}</div>
								</div>
								<div className="flex items-center gap-2">
									<Button variant="outline" size="sm" onClick={() => openEditDialog(profile)}>
										<Pencil className="h-3.5 w-3.5 mr-1" />
										Edit
									</Button>
									<Button variant="outline" size="sm" onClick={() => requestDelete(profile)} className="text-[var(--error)] border-[var(--error)]/20 hover:bg-[var(--error)]/10">
										<Trash2 className="h-3.5 w-3.5 mr-1" />
										Delete
									</Button>
								</div>
							</div>
						))}
					</div>
				)}
			</div>

			<Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
				<DialogContent className="max-w-[760px] bg-[var(--bg-card)] border border-[var(--border-subtle)]">
					<DialogHeader>
						<DialogTitle>{editing ? "Edit OpenClaw Profile" : "New OpenClaw Profile"}</DialogTitle>
						<DialogDescription>
							保存后即可在录音详情页中直接选择该 profile 发送转录结果。
						</DialogDescription>
					</DialogHeader>

					<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
						<div className="space-y-2">
							<label className="text-sm text-[var(--text-secondary)]">Profile Name</label>
							<Input value={form.name} onChange={(e) => setForm((state) => ({ ...state, name: e.target.value }))} placeholder="e.g. OpenClaw Prod" />
						</div>
						<div className="space-y-2">
							<label className="text-sm text-[var(--text-secondary)]">Host (supports user@host)</label>
							<Input value={form.ip} onChange={(e) => setForm((state) => ({ ...state, ip: e.target.value }))} placeholder="user@example-host" />
						</div>
						<div className="space-y-2 md:col-span-2">
							<label className="text-sm text-[var(--text-secondary)]">SSH Private Key</label>
							<div className="space-y-2">
								<input
									ref={sshKeyInputRef}
									type="file"
									className="hidden"
									accept=".pem,.key,.txt,application/octet-stream"
									onChange={onSshKeyFileSelected}
								/>
								<Button type="button" variant="outline" onClick={triggerSshKeyFilePicker}>
									Choose SSH Key File
								</Button>
								<p className="text-xs text-[var(--text-secondary)]">
									{sshKeyFileName || (editing ? "未修改，将继续使用已保存 SSH Key" : "未选择文件")}
								</p>
							</div>
						</div>
						<div className="space-y-2">
							<label className="text-sm text-[var(--text-secondary)]">Hook Key</label>
							<Input
								type="password"
								value={form.hook_key}
								onChange={(e) => setForm((state) => ({ ...state, hook_key: e.target.value }))}
								placeholder={editing ? "留空则保留当前值" : "Bearer token value"}
							/>
						</div>
						<div className="space-y-2">
							<label className="text-sm text-[var(--text-secondary)]">Hook Name</label>
							<Input value={form.hook_name} onChange={(e) => setForm((state) => ({ ...state, hook_name: e.target.value }))} placeholder="Dashboard" />
						</div>
						<div className="space-y-2 md:col-span-2">
							<label className="text-sm text-[var(--text-secondary)]">Message</label>
							<Textarea value={form.message} onChange={(e) => setForm((state) => ({ ...state, message: e.target.value }))} rows={4} placeholder="Prompt to OpenClaw agent" />
						</div>
					</div>

					<DialogFooter>
						<Button variant="ghost" onClick={() => setDialogOpen(false)}>Cancel</Button>
						<Button onClick={onSave} disabled={saving}>
							{saving ? "Saving..." : editing ? "Update" : "Create"}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			<AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
				<AlertDialogContent className="bg-[var(--bg-card)] border border-[var(--border-subtle)]">
					<AlertDialogHeader>
						<AlertDialogTitle>Delete OpenClaw Profile</AlertDialogTitle>
						<AlertDialogDescription>
							{deleting ? `Are you sure you want to delete "${deleting.name}"?` : "Are you sure?"}
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel onClick={() => setDeleting(null)}>Cancel</AlertDialogCancel>
						<AlertDialogAction onClick={confirmDelete}>Delete</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}
