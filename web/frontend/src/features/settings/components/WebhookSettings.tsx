import { useCallback, useEffect, useState } from "react";
import {
  ExternalLink,
  MoreVertical,
  Pencil,
  Plus,
  RotateCcw,
  Save,
  Trash2,
  Webhook as WebhookIcon,
} from "lucide-react";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { useAuth } from "@/features/auth/hooks/useAuth";

type WebhookSubscription = {
  id: string;
  name: string;
  url: string;
  events: string[];
  enabled: boolean;
  has_secret: boolean;
};

type WebhookDelivery = {
  id: string;
  webhook_id: string;
  webhook_name: string;
  event: string;
  job_id: string;
  status: "pending" | "processing" | "succeeded" | "failed";
  attempt_count: number;
  response_status?: number;
  last_error?: string;
  next_attempt_at?: string;
  delivered_at?: string;
  created_at: string;
};

type WebhookForm = {
  name: string;
  url: string;
  secret: string;
  events: string[];
  enabled: boolean;
  clearSecret: boolean;
};

const eventOptions = [
  ["recording.uploaded", "Recording uploaded"],
  ["transcription.completed", "Transcription completed"],
  ["transcription.failed", "Transcription failed"],
  ["summary.completed", "Summary completed"],
  ["summary.failed", "Summary failed"],
] as const;

const emptyForm = (): WebhookForm => ({
  name: "",
  url: "",
  secret: "",
  events: ["recording.uploaded"],
  enabled: true,
  clearSecret: false,
});

export function WebhookSettings() {
  const { getAuthHeaders } = useAuth();
  const [hooks, setHooks] = useState<WebhookSubscription[]>([]);
  const [deliveries, setDeliveries] = useState<WebhookDelivery[]>([]);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingID, setEditingID] = useState<string | null>(null);
  const [form, setForm] = useState<WebhookForm>(emptyForm);
  const [error, setError] = useState("");
  const [listError, setListError] = useState("");
  const [updatingIDs, setUpdatingIDs] = useState<Set<string>>(new Set());

  const load = useCallback(async () => {
    const headers = getAuthHeaders();
    const [hooksResponse, deliveriesResponse] = await Promise.all([
      fetch("/api/v1/webhooks/", { headers }),
      fetch("/api/v1/webhooks/deliveries", { headers }),
    ]);
    if (hooksResponse.ok) {
      setHooks(await hooksResponse.json());
    }
    if (deliveriesResponse.ok) {
      setDeliveries(await deliveriesResponse.json());
    }
  }, [getAuthHeaders]);

  useEffect(() => {
    void load();
  }, [load]);

  const selectedHook = editingID
    ? hooks.find((hook) => hook.id === editingID)
    : undefined;

  const updateForm = <K extends keyof WebhookForm>(
    key: K,
    value: WebhookForm[K],
  ) => setForm((current) => ({ ...current, [key]: value }));

  const toggleEvent = (event: string) => {
    updateForm(
      "events",
      form.events.includes(event)
        ? form.events.filter((configured) => configured !== event)
        : [...form.events, event],
    );
  };

  const closeEditor = () => {
    setEditorOpen(false);
    setEditingID(null);
    setForm(emptyForm());
    setError("");
  };

  const beginCreate = () => {
    setEditingID(null);
    setForm(emptyForm());
    setError("");
    setEditorOpen(true);
  };

  const beginEdit = (hook: WebhookSubscription) => {
    setEditingID(hook.id);
    setForm({
      name: hook.name,
      url: hook.url,
      secret: "",
      events: hook.events,
      enabled: hook.enabled,
      clearSecret: false,
    });
    setError("");
    setEditorOpen(true);
  };

  const save = async () => {
    setError("");
    if (!form.name.trim() || !form.url.trim() || form.events.length === 0) {
      setError("Name, URL, and at least one event are required.");
      return;
    }

    const response = await fetch(
      editingID ? `/api/v1/webhooks/${editingID}` : "/api/v1/webhooks/",
      {
        method: editingID ? "PUT" : "POST",
        headers: { "Content-Type": "application/json", ...getAuthHeaders() },
        body: JSON.stringify({
          name: form.name,
          url: form.url,
          secret: form.secret,
          clear_secret: form.clearSecret,
          events: form.events,
          enabled: form.enabled,
        }),
      },
    );
    if (!response.ok) {
      const body: { error?: string } = await response.json();
      setError(body.error || "Could not save webhook");
      return;
    }

    closeEditor();
    await load();
  };

  const toggleEnabled = async (
    hook: WebhookSubscription,
    enabled: boolean,
  ) => {
    setListError("");
    setUpdatingIDs((current) => new Set(current).add(hook.id));
    setHooks((current) =>
      current.map((item) =>
        item.id === hook.id ? { ...item, enabled } : item,
      ),
    );

    try {
      const response = await fetch(`/api/v1/webhooks/${hook.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", ...getAuthHeaders() },
        body: JSON.stringify({
          name: hook.name,
          url: hook.url,
          events: hook.events,
          enabled,
        }),
      });
      if (!response.ok) {
        throw new Error("Could not update webhook");
      }
    } catch {
      setHooks((current) =>
        current.map((item) =>
          item.id === hook.id ? { ...item, enabled: hook.enabled } : item,
        ),
      );
      setListError(`Could not ${enabled ? "enable" : "disable"} ${hook.name}.`);
    } finally {
      setUpdatingIDs((current) => {
        const next = new Set(current);
        next.delete(hook.id);
        return next;
      });
    }
  };

  const remove = async (hook: WebhookSubscription) => {
    if (!confirm(`Delete ${hook.name}? Delivery history will be retained.`)) {
      return;
    }
    const response = await fetch(`/api/v1/webhooks/${hook.id}`, {
      method: "DELETE",
      headers: getAuthHeaders(),
    });
    if (!response.ok) {
      setError("Could not delete webhook");
      return;
    }
    if (editingID === hook.id) {
      closeEditor();
    }
    await load();
  };

  return (
    <div className="space-y-6">
      <div className="bg-[var(--bg-main)] border border-[var(--border-subtle)] rounded-[var(--radius-card)] p-4 sm:p-6 shadow-sm">
        <div className="flex flex-wrap items-start justify-between gap-3 mb-5">
          <div>
            <div className="flex items-center gap-3 mb-1">
              <WebhookIcon className="h-5 w-5 text-[var(--brand-solid)]" />
              <h3 className="text-lg font-medium text-[var(--text-primary)]">
                Webhooks
              </h3>
            </div>
            <p className="text-sm text-[var(--text-secondary)]">
              Events are queued before delivery and retried after temporary failures.
            </p>
          </div>
          <Button onClick={beginCreate}>
            <Plus className="h-4 w-4" />
            Add Webhook
          </Button>
        </div>

        <div className="space-y-3">
          {hooks.length === 0 && (
            <p className="text-sm text-[var(--text-tertiary)]">
              No webhooks configured yet.
            </p>
          )}
          {listError && (
            <p className="text-sm text-[var(--danger-solid)]">
              {listError}
            </p>
          )}
          {hooks.map((hook) => (
            <div
              key={hook.id}
              className="flex items-center justify-between gap-3 bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-[var(--radius-card)] p-4"
            >
              <div className="min-w-0">
                <p className="font-medium text-[var(--text-primary)]">
                  {hook.name}
                  {!hook.enabled && (
                    <span className="text-xs text-[var(--text-tertiary)] ml-2">
                      disabled
                    </span>
                  )}
                </p>
                <p className="text-sm text-[var(--text-secondary)] break-all">
                  {hook.url}
                </p>
                <p className="text-xs text-[var(--text-tertiary)] mt-1">
                  {hook.events.join(" · ")}
                  {hook.has_secret ? " · signed" : ""}
                </p>
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <Switch
                  checked={hook.enabled}
                  disabled={updatingIDs.has(hook.id)}
                  onCheckedChange={(enabled) =>
                    void toggleEnabled(hook, enabled)
                  }
                  aria-label={`${hook.enabled ? "Disable" : "Enable"} ${hook.name}`}
                />
                <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={`Actions for ${hook.name}`}
                  >
                    <MoreVertical className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  align="end"
                  className="bg-[var(--bg-card)] border-[var(--border-subtle)] text-[var(--text-primary)]"
                >
                  <DropdownMenuItem onSelect={() => beginEdit(hook)}>
                    <Pencil className="h-4 w-4" />
                    Edit
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    variant="destructive"
                    onSelect={() => void remove(hook)}
                  >
                    <Trash2 className="h-4 w-4" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>
          ))}
        </div>
      </div>

      <Accordion type="single" collapsible>
        <AccordionItem
          value="deliveries"
          className="bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-[var(--radius-card)] px-4 sm:px-6 shadow-sm"
        >
          <AccordionTrigger className="hover:no-underline">
            <div className="text-left">
              <h3 className="text-lg font-medium text-[var(--text-primary)]">
                Recent deliveries
              </h3>
              <p className="text-sm font-normal text-[var(--text-secondary)]">
                The latest 50 queued webhook events.
              </p>
            </div>
          </AccordionTrigger>
          <AccordionContent>
            <div className="flex justify-end mb-3">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => void load()}
                aria-label="Refresh deliveries"
              >
                <RotateCcw className="h-4 w-4" />
              </Button>
            </div>
            <div className="space-y-3">
              {deliveries.length === 0 && (
                <p className="text-sm text-[var(--text-tertiary)]">
                  No webhook deliveries yet.
                </p>
              )}
              {deliveries.map((delivery) => (
                <div
                  key={delivery.id}
                  className="bg-[var(--bg-main)] border border-[var(--border-subtle)] rounded-[var(--radius-btn)] p-3"
                >
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <p className="text-sm font-medium text-[var(--text-primary)]">
                      {delivery.webhook_name} · {delivery.event}
                    </p>
                    <span className="text-xs text-[var(--text-secondary)]">
                      {delivery.status} · {delivery.attempt_count} attempt
                      {delivery.attempt_count === 1 ? "" : "s"}
                      {delivery.response_status
                        ? ` · HTTP ${delivery.response_status}`
                        : ""}
                    </span>
                  </div>
                  <p className="text-xs text-[var(--text-tertiary)] mt-1">
                    Job {delivery.job_id} ·{" "}
                    {new Date(delivery.created_at).toLocaleString()}
                  </p>
                  {delivery.last_error && (
                    <p className="text-xs text-[var(--danger-solid)] mt-1 break-words">
                      {delivery.last_error}
                    </p>
                  )}
                </div>
              ))}
            </div>
          </AccordionContent>
        </AccordionItem>
      </Accordion>

      <Dialog
        open={editorOpen}
        onOpenChange={(open) => {
          if (!open) closeEditor();
        }}
      >
        <DialogContent className="max-w-full sm:max-w-2xl w-[calc(100vw-1rem)] max-h-[90vh] overflow-hidden flex flex-col p-0 gap-0 bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-2xl">
          <DialogHeader className="px-6 pt-6 pb-4 border-b border-[var(--border-subtle)]">
            <DialogTitle className="text-xl font-semibold text-[var(--text-primary)]">
              {editingID ? "Edit Webhook" : "Add Webhook"}
            </DialogTitle>
            <DialogDescription className="text-[var(--text-secondary)]">
              Choose where Scriberr sends events and optionally sign each request.
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto px-6 py-6 space-y-5">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-[var(--text-primary)]">
                  Name
                </label>
                <Input
                  value={form.name}
                  onChange={(event) => updateForm("name", event.target.value)}
                  placeholder="Webhook name"
                  aria-label="Webhook name"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-[var(--text-primary)]">
                  URL
                </label>
                <Input
                  value={form.url}
                  onChange={(event) => updateForm("url", event.target.value)}
                  placeholder="https://example.com/webhook"
                  aria-label="Webhook URL"
                  type="url"
                />
              </div>
              <div className="space-y-1.5">
                <div className="flex items-center justify-between gap-2">
                  <label className="text-sm font-medium text-[var(--text-primary)]">
                    Signing secret
                  </label>
                  <a
                    href="https://scriberr.app/docs/webhooks#verifying-signatures"
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1 text-xs text-[var(--brand-solid)] hover:underline"
                  >
                    How signing works
                    <ExternalLink className="h-3 w-3" />
                  </a>
                </div>
                <Input
                  value={form.secret}
                  onChange={(event) => {
                    updateForm("secret", event.target.value);
                    updateForm("clearSecret", false);
                  }}
                  placeholder={
                    selectedHook?.has_secret
                      ? "Leave blank to keep the current secret"
                      : "Optional"
                  }
                  aria-label="Signing secret"
                  type="password"
                />
              </div>
            </div>

            {selectedHook?.has_secret && (
              <div className="flex flex-wrap items-center gap-3">
                <p className="text-xs text-[var(--text-tertiary)]">
                  {form.clearSecret
                    ? "The existing signing secret will be removed."
                    : "A signing secret is currently configured."}
                </p>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() =>
                    updateForm("clearSecret", !form.clearSecret)
                  }
                >
                  {form.clearSecret ? "Keep secret" : "Remove secret"}
                </Button>
              </div>
            )}

            <div>
              <p className="text-sm font-medium text-[var(--text-primary)] mb-2">
                Fire when
              </p>
              <div className="grid gap-2 sm:grid-cols-2">
                {eventOptions.map(([value, label]) => (
                  <label
                    key={value}
                    className="flex items-center gap-2 text-sm text-[var(--text-secondary)]"
                  >
                    <Checkbox
                      checked={form.events.includes(value)}
                      onCheckedChange={() => toggleEvent(value)}
                    />
                    {label}
                  </label>
                ))}
              </div>
            </div>

            {error && (
              <p className="text-sm text-[var(--danger-solid)]">{error}</p>
            )}
          </div>

          <DialogFooter className="px-6 py-4 border-t border-[var(--border-subtle)]">
            <Button variant="ghost" onClick={closeEditor}>
              Cancel
            </Button>
            <Button onClick={() => void save()}>
              {editingID ? (
                <Save className="h-4 w-4" />
              ) : (
                <Plus className="h-4 w-4" />
              )}
              {editingID ? "Save Changes" : "Add Webhook"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
