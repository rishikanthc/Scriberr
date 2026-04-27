import { useMemo, useState } from "react";
import { Edit3, FileText, Plus, Trash2 } from "lucide-react";
import { AppButton, IconButton } from "@/shared/ui/Button";
import { ConfirmDialog } from "@/shared/ui/ConfirmDialog";
import { EmptyState } from "@/shared/ui/EmptyState";
import type { SummaryWidget, SummaryWidgetPayload } from "@/features/settings/api/summaryWidgetsApi";
import { useDeleteSummaryWidget, useSaveSummaryWidget, useSummaryWidgets } from "@/features/settings/hooks/useSummaryWidgets";
import { SummaryWidgetDialog } from "@/features/settings/components/SummaryWidgetDialog";

export function SummaryWidgetsPanel() {
  const widgetsQuery = useSummaryWidgets();
  const saveWidgetMutation = useSaveSummaryWidget();
  const deleteWidgetMutation = useDeleteSummaryWidget();
  const [editingWidget, setEditingWidget] = useState<SummaryWidget | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [widgetToDelete, setWidgetToDelete] = useState<SummaryWidget | null>(null);
  const [error, setError] = useState("");

  const widgets = useMemo(() => widgetsQuery.data || [], [widgetsQuery.data]);

  const openNewWidget = () => {
    setEditingWidget(null);
    setDialogOpen(true);
  };

  const saveWidget = async (widget: SummaryWidgetPayload) => {
    setError("");
    await saveWidgetMutation.mutateAsync(widget);
  };

  const confirmDelete = async () => {
    if (!widgetToDelete) return;
    setError("");
    try {
      await deleteWidgetMutation.mutateAsync(widgetToDelete.id);
      setWidgetToDelete(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not delete widget.");
    }
  };

  return (
    <>
      <section className="scr-settings-panel" aria-label="Summarization widgets">
        <div className="scr-settings-panel-head">
          <div>
            <h2 className="scr-settings-heading">Widgets</h2>
            <p className="scr-settings-copy">
              Extract repeatable sections from generated summaries and transcripts after automatic summarization completes.
            </p>
          </div>
          <AppButton variant="secondary" className="scr-settings-new-profile" onClick={openNewWidget}>
            <Plus size={15} aria-hidden="true" />
            New widget
          </AppButton>
        </div>

        {error || widgetsQuery.error ? (
          <div className="scr-alert">
            {error || (widgetsQuery.error instanceof Error ? widgetsQuery.error.message : "Could not load widgets.")}
          </div>
        ) : null}

        {widgetsQuery.isLoading ? (
          <div className="scr-profile-list" aria-label="Loading widgets">
            {[0, 1, 2].map((item) => <div className="scr-profile-skeleton" key={item} />)}
          </div>
        ) : widgets.length > 0 ? (
          <div className="scr-profile-list">
            {widgets.map((widget) => (
              <SummaryWidgetRow
                key={widget.id}
                widget={widget}
                onEdit={() => {
                  setEditingWidget(widget);
                  setDialogOpen(true);
                }}
                onDelete={() => setWidgetToDelete(widget)}
              />
            ))}
          </div>
        ) : (
          <EmptyState title="No widgets yet" description="Create a widget to extract custom sections from future summaries." />
        )}
      </section>

      <SummaryWidgetDialog
        open={dialogOpen}
        widget={editingWidget}
        onClose={() => {
          setDialogOpen(false);
          setEditingWidget(null);
        }}
        onSave={saveWidget}
      />
      <ConfirmDialog
        open={Boolean(widgetToDelete)}
        title="Delete widget?"
        description={widgetToDelete ? `This will remove "${widgetToDelete.name}" from future summary runs.` : ""}
        confirmLabel="Delete"
        busy={deleteWidgetMutation.isPending}
        onCancel={() => {
          if (!deleteWidgetMutation.isPending) setWidgetToDelete(null);
        }}
        onConfirm={() => void confirmDelete()}
      />
    </>
  );
}

function SummaryWidgetRow({ widget, onEdit, onDelete }: { widget: SummaryWidget; onEdit: () => void; onDelete: () => void }) {
  const mode = widget.always_enabled ? "always enabled" : "conditional";
  const context = widget.context_source === "transcript" ? "transcript context" : "summary context";
  const markdown = widget.render_markdown ? "markdown" : "plain text";
  const enabled = widget.enabled ? "enabled" : "disabled";
  const meta = [enabled, mode, context, markdown, widget.display_title].join(" · ");

  return (
    <article className="scr-profile-row">
      <button className="scr-profile-main" type="button" onClick={onEdit}>
        <div className="scr-profile-copy">
          <div className="scr-profile-title-row">
            <FileText size={15} aria-hidden="true" />
            <h3 className="scr-profile-title">{widget.name}</h3>
            {!widget.enabled ? <span className="scr-profile-badge">Disabled</span> : null}
          </div>
          {widget.description ? <p className="scr-profile-description">{widget.description}</p> : null}
          <p className="scr-profile-meta">{meta}</p>
        </div>
      </button>
      <div className="scr-profile-actions">
        <IconButton label={`Edit ${widget.name}`} onClick={onEdit}>
          <Edit3 size={15} aria-hidden="true" />
        </IconButton>
        <IconButton label={`Delete ${widget.name}`} className="scr-icon-danger" onClick={onDelete}>
          <Trash2 size={15} aria-hidden="true" />
        </IconButton>
      </div>
    </article>
  );
}
