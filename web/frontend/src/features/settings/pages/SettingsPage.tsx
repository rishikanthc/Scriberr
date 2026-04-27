import { useCallback, useEffect, useMemo, useState } from "react";
import { Edit3, Plus, Trash2 } from "lucide-react";
import { Sidebar } from "@/features/home/components/HomePage";
import { AppButton, IconButton } from "@/shared/ui/Button";
import { ConfirmDialog } from "@/shared/ui/ConfirmDialog";
import { EmptyState } from "@/shared/ui/EmptyState";
import { ASRProfileDialog } from "../components/ASRProfileDialog";
import {
  deleteProfile,
  listProfiles,
  listTranscriptionModels,
  saveProfile,
  type TranscriptionModel,
  type TranscriptionProfile,
  type TranscriptionProfileOptions,
} from "../api/profilesApi";

type SettingsTab = "general" | "asr";

export function Settings() {
  const [activeTab, setActiveTab] = useState<SettingsTab>("asr");
  const [profiles, setProfiles] = useState<TranscriptionProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [editingProfile, setEditingProfile] = useState<TranscriptionProfile | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [profileToDelete, setProfileToDelete] = useState<TranscriptionProfile | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [models, setModels] = useState<TranscriptionModel[]>([]);

  const loadProfiles = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setProfiles(await listProfiles());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load profiles.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadProfiles();
  }, [loadProfiles]);

  useEffect(() => {
    listTranscriptionModels()
      .then(setModels)
      .catch(() => setModels([]));
  }, []);

  const defaultProfile = useMemo(() => profiles.find((profile) => profile.is_default), [profiles]);

  const openNewProfile = () => {
    setEditingProfile(null);
    setDialogOpen(true);
  };

  const handleSave = async (profile: { id?: string; name: string; description: string; is_default: boolean; options: TranscriptionProfileOptions }) => {
    await saveProfile(profile);
    await loadProfiles();
  };

  const confirmDelete = async () => {
    if (!profileToDelete) return;
    setDeleting(true);
    setError("");
    try {
      await deleteProfile(profileToDelete.id);
      setProfileToDelete(null);
      await loadProfiles();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not delete profile.");
    } finally {
      setDeleting(false);
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
          </div>

          <div className="scr-settings-content">
            {activeTab === "general" ? (
              <EmptyState title="General settings" description="General preferences will appear here." />
            ) : (
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

                {error ? <div className="scr-alert">{error}</div> : null}

                {loading ? (
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
            )}
          </div>
        </main>
      </div>

      <ASRProfileDialog
        open={dialogOpen}
        profile={editingProfile}
        models={models}
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
        busy={deleting}
        onCancel={() => {
          if (!deleting) setProfileToDelete(null);
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
