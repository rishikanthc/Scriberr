import { useMemo, useState } from "react";
import { Edit3, Plus, Trash2 } from "lucide-react";
import { Sidebar } from "@/features/home/components/HomePage";
import { AppButton, IconButton } from "@/shared/ui/Button";
import { ConfirmDialog } from "@/shared/ui/ConfirmDialog";
import { EmptyState } from "@/shared/ui/EmptyState";
import { useDeleteProfile, useProfiles, useSaveProfile, useTranscriptionModels } from "@/features/settings/hooks/useProfiles";
import { ASRProfileDialog } from "../components/ASRProfileDialog";
import { LLMProviderPanel } from "../components/LLMProviderPanel";
import { SummaryWidgetsPanel } from "../components/SummaryWidgetsPanel";
import { TagsSettingsPanel } from "@/features/tags/components/TagsSettingsPanel";
import type { TranscriptionProfile, TranscriptionProfileOptions } from "../api/profilesApi";

type SettingsTab = "general" | "asr" | "llm" | "summarization" | "tags";

export function Settings() {
  const [activeTab, setActiveTab] = useState<SettingsTab>("asr");
  const [error, setError] = useState("");
  const [editingProfile, setEditingProfile] = useState<TranscriptionProfile | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [profileToDelete, setProfileToDelete] = useState<TranscriptionProfile | null>(null);
  const profilesQuery = useProfiles();
  const modelsQuery = useTranscriptionModels();
  const saveProfileMutation = useSaveProfile();
  const deleteProfileMutation = useDeleteProfile();
  const profiles = profilesQuery.data || [];

  const defaultProfile = useMemo(() => profiles.find((profile) => profile.is_default), [profiles]);

  const openNewProfile = () => {
    setEditingProfile(null);
    setDialogOpen(true);
  };

  const handleSave = async (profile: { id?: string; name: string; description: string; is_default: boolean; options: TranscriptionProfileOptions }) => {
    setError("");
    try {
      await saveProfileMutation.mutateAsync(profile);
    } catch (err) {
      throw new Error(err instanceof Error ? err.message : "Could not save profile.");
    }
  };

  const confirmDelete = async () => {
    if (!profileToDelete) return;
    setError("");
    try {
      await deleteProfileMutation.mutateAsync(profileToDelete.id);
      setProfileToDelete(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not delete profile.");
    }
  };

  return (
    <div className="scr-app">
      <div className="scr-shell">
        <Sidebar activeItem="settings" />
        <main className="scr-main">
          <div className="scr-settings-topline">
            <h1 className="scr-settings-title">Settings</h1>
          </div>

          <div className="scr-settings-tabs" role="tablist" aria-label="Settings sections">
            <button className="scr-settings-tab" data-active={activeTab === "general"} type="button" role="tab" aria-selected={activeTab === "general"} onClick={() => setActiveTab("general")}>
              General
            </button>
            <button className="scr-settings-tab" data-active={activeTab === "asr"} type="button" role="tab" aria-selected={activeTab === "asr"} onClick={() => setActiveTab("asr")}>
              ASR
            </button>
            <button className="scr-settings-tab" data-active={activeTab === "llm"} type="button" role="tab" aria-selected={activeTab === "llm"} onClick={() => setActiveTab("llm")}>
              LLM Providers
            </button>
            <button className="scr-settings-tab" data-active={activeTab === "summarization"} type="button" role="tab" aria-selected={activeTab === "summarization"} onClick={() => setActiveTab("summarization")}>
              Summarization
            </button>
            <button className="scr-settings-tab" data-active={activeTab === "tags"} type="button" role="tab" aria-selected={activeTab === "tags"} onClick={() => setActiveTab("tags")}>
              Tags
            </button>
          </div>

          <div className="scr-settings-content">
            {activeTab === "general" ? (
              <EmptyState title="General settings" description="General preferences will appear here." />
            ) : activeTab === "asr" ? (
              <section className="scr-settings-panel" aria-label="ASR profiles">
                <div className="scr-settings-panel-head">
                  <div>
                    <h2 className="scr-settings-heading">Transcription profiles</h2>
                    <p className="scr-settings-copy">
                      Save engine, model, decoding, diarization, and output presets for repeatable transcription runs.
                    </p>
                  </div>
                  <AppButton variant="secondary" className="scr-settings-new-profile" onClick={openNewProfile}>
                    <Plus size={15} aria-hidden="true" />
                    New profile
                  </AppButton>
                </div>

                {error || profilesQuery.error ? (
                  <div className="scr-alert">
                    {error || (profilesQuery.error instanceof Error ? profilesQuery.error.message : "Could not load profiles.")}
                  </div>
                ) : null}

                {profilesQuery.isLoading ? (
                  <div className="scr-profile-list" aria-label="Loading profiles">
                    {[0, 1, 2].map((item) => <div className="scr-profile-skeleton" key={item} />)}
                  </div>
                ) : profiles.length > 0 ? (
                  <div className="scr-profile-list">
                    {profiles.map((profile) => (
                      <ProfileRow
                        key={profile.id}
                        profile={profile}
                        isDefault={defaultProfile?.id === profile.id}
                        onEdit={() => {
                          setEditingProfile(profile);
                          setDialogOpen(true);
                        }}
                        onDelete={() => setProfileToDelete(profile)}
                      />
                    ))}
                  </div>
                ) : (
                  <EmptyState title="No profiles yet" description="Create a profile to save your preferred ASR settings." />
                )}
              </section>
            ) : activeTab === "llm" ? (
              <LLMProviderPanel />
            ) : activeTab === "tags" ? (
              <TagsSettingsPanel />
            ) : (
              <SummaryWidgetsPanel />
            )}
          </div>
        </main>
      </div>

      <ASRProfileDialog
        open={dialogOpen}
        profile={editingProfile}
        models={modelsQuery.data || []}
        onClose={() => {
          setDialogOpen(false);
          setEditingProfile(null);
        }}
        onSave={handleSave}
      />
      <ConfirmDialog
        open={Boolean(profileToDelete)}
        title="Delete profile?"
        description={profileToDelete ? `This will remove "${profileToDelete.name}" from your saved ASR profiles.` : ""}
        confirmLabel="Delete"
        busy={deleteProfileMutation.isPending}
        onCancel={() => {
          if (!deleteProfileMutation.isPending) setProfileToDelete(null);
        }}
        onConfirm={() => void confirmDelete()}
      />
    </div>
  );
}

function ProfileRow({ profile, isDefault, onEdit, onDelete }: { profile: TranscriptionProfile; isDefault: boolean; onEdit: () => void; onDelete: () => void }) {
  const optionSummary = [
    profile.options.model,
    profile.options.language || "auto language",
    profile.options.task,
    profile.options.chunking_strategy === "vad" ? "VAD chunks" : "fixed chunks",
    profile.options.diarize ? "diarization on" : "diarization off",
  ].join(" · ");

  return (
    <article className="scr-profile-row">
      <button className="scr-profile-main" type="button" onClick={onEdit}>
        <div className="scr-profile-copy">
          <div className="scr-profile-title-row">
            <h3 className="scr-profile-title">{profile.name}</h3>
            {isDefault ? (
              <span className="scr-profile-badge">
                Default
              </span>
            ) : null}
          </div>
          {profile.description ? <p className="scr-profile-description">{profile.description}</p> : null}
          <p className="scr-profile-meta">{optionSummary}</p>
        </div>
      </button>
      <div className="scr-profile-actions">
        <IconButton label={`Edit ${profile.name}`} onClick={onEdit}>
          <Edit3 size={15} aria-hidden="true" />
        </IconButton>
        <IconButton label={`Delete ${profile.name}`} className="scr-icon-danger" onClick={onDelete}>
          <Trash2 size={15} aria-hidden="true" />
        </IconButton>
      </div>
    </article>
  );
}
