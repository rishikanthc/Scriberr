import { useState } from "react";
import { Edit3, Plus, Trash2 } from "lucide-react";
import { AppButton, IconButton } from "@/shared/ui/Button";
import { ConfirmDialog } from "@/shared/ui/ConfirmDialog";
import { EmptyState } from "@/shared/ui/EmptyState";
import type { AudioTag, SaveTagPayload } from "@/features/tags/api/tagsApi";
import { useDeleteTag, useSaveTag, useTags } from "@/features/tags/hooks/useTags";
import { TagDialog } from "@/features/tags/components/TagDialog";

export function TagsSettingsPanel() {
  const tagsQuery = useTags();
  const saveTagMutation = useSaveTag();
  const deleteTagMutation = useDeleteTag();
  const tags = tagsQuery.data?.items || [];
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingTag, setEditingTag] = useState<AudioTag | null>(null);
  const [tagToDelete, setTagToDelete] = useState<AudioTag | null>(null);
  const [error, setError] = useState("");

  const openNewTag = () => {
    setEditingTag(null);
    setDialogOpen(true);
  };

  const handleSave = async (payload: SaveTagPayload) => {
    setError("");
    await saveTagMutation.mutateAsync(payload);
  };

  const confirmDelete = async () => {
    if (!tagToDelete) return;
    setError("");
    try {
      await deleteTagMutation.mutateAsync(tagToDelete.id);
      setTagToDelete(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not delete tag.");
    }
  };

  return (
    <section className="scr-settings-panel" aria-label="Tags">
      <div className="scr-settings-panel-head">
        <div>
          <h2 className="scr-settings-heading">Tags</h2>
          <p className="scr-settings-copy">Create labels for grouping recordings by project, topic, or workflow.</p>
        </div>
        <AppButton variant="secondary" className="scr-settings-new-profile" onClick={openNewTag}>
          <Plus size={15} aria-hidden="true" />
          Add tag
        </AppButton>
      </div>

      {error || tagsQuery.error ? (
        <div className="scr-alert">
          {error || (tagsQuery.error instanceof Error ? tagsQuery.error.message : "Could not load tags.")}
        </div>
      ) : null}

      {tagsQuery.isLoading ? (
        <div className="scr-profile-list" aria-label="Loading tags">
          {[0, 1, 2].map((item) => <div className="scr-profile-skeleton" key={item} />)}
        </div>
      ) : tags.length > 0 ? (
        <div className="scr-profile-list">
          {tags.map((tag) => (
            <TagRow
              key={tag.id}
              tag={tag}
              onEdit={() => {
                setEditingTag(tag);
                setDialogOpen(true);
              }}
              onDelete={() => setTagToDelete(tag)}
            />
          ))}
        </div>
      ) : (
        <EmptyState title="No tags yet" description="Add a tag to start organizing audio." />
      )}

      <TagDialog
        open={dialogOpen}
        tag={editingTag}
        tags={tags}
        saving={saveTagMutation.isPending}
        onClose={() => {
          setDialogOpen(false);
          setEditingTag(null);
        }}
        onSave={handleSave}
      />
      <ConfirmDialog
        open={Boolean(tagToDelete)}
        title="Delete tag?"
        description={tagToDelete ? `This will remove "${tagToDelete.name}" from saved tags and assigned audio.` : ""}
        confirmLabel="Delete"
        busy={deleteTagMutation.isPending}
        onCancel={() => {
          if (!deleteTagMutation.isPending) setTagToDelete(null);
        }}
        onConfirm={() => void confirmDelete()}
      />
    </section>
  );
}

function TagRow({ tag, onEdit, onDelete }: { tag: AudioTag; onEdit: () => void; onDelete: () => void }) {
  return (
    <article className="scr-profile-row">
      <button className="scr-profile-main" type="button" onClick={onEdit}>
        <div className="scr-profile-copy">
          <div className="scr-profile-title-row">
            <h3 className="scr-profile-title">{tag.name}</h3>
          </div>
          {tag.description ? <p className="scr-profile-description">{tag.description}</p> : null}
          {tag.when_to_use ? <p className="scr-profile-meta">{tag.when_to_use}</p> : null}
        </div>
      </button>
      <div className="scr-profile-actions">
        <IconButton label={`Edit ${tag.name}`} onClick={onEdit}>
          <Edit3 size={15} aria-hidden="true" />
        </IconButton>
        <IconButton label={`Delete ${tag.name}`} className="scr-icon-danger" onClick={onDelete}>
          <Trash2 size={15} aria-hidden="true" />
        </IconButton>
      </div>
    </article>
  );
}
