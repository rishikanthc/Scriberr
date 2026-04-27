export type SummaryWidgetContextSource = "summary" | "transcript";

export type SummaryWidget = {
  id: string;
  name: string;
  description: string;
  always_enabled: boolean;
  when_to_use: string;
  context_source: SummaryWidgetContextSource;
  prompt: string;
  render_markdown: boolean;
  display_title: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
};

export type SummaryWidgetPayload = {
  id?: string;
  name: string;
  description: string;
  always_enabled: boolean;
  when_to_use: string;
  context_source: SummaryWidgetContextSource;
  prompt: string;
  render_markdown: boolean;
  display_title: string;
  enabled: boolean;
};

type SummaryWidgetListResponse = {
  items?: SummaryWidget[];
};

export async function listSummaryWidgets(headers?: Record<string, string>) {
  const response = await fetch("/api/v1/settings/summary-widgets", { headers });
  if (!response.ok) throw new Error(await readError(response));
  const data = (await response.json()) as SummaryWidgetListResponse;
  return (data.items || []).map(normalizeSummaryWidget);
}

export async function saveSummaryWidget(widget: SummaryWidgetPayload, headers?: Record<string, string>) {
  const response = await fetch(widget.id ? `/api/v1/settings/summary-widgets/${widget.id}` : "/api/v1/settings/summary-widgets", {
    method: widget.id ? "PATCH" : "POST",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify({
      name: widget.name,
      description: widget.description,
      always_enabled: widget.always_enabled,
      when_to_use: widget.when_to_use,
      context_source: widget.context_source,
      prompt: widget.prompt,
      render_markdown: widget.render_markdown,
      display_title: widget.display_title,
      enabled: widget.enabled,
    }),
  });
  if (!response.ok) throw new Error(await readError(response));
  return normalizeSummaryWidget((await response.json()) as SummaryWidget);
}

export async function deleteSummaryWidget(widgetId: string, headers?: Record<string, string>) {
  const response = await fetch(`/api/v1/settings/summary-widgets/${widgetId}`, { method: "DELETE", headers });
  if (!response.ok) throw new Error(await readError(response));
}

function normalizeSummaryWidget(widget: SummaryWidget): SummaryWidget {
  return {
    ...widget,
    description: widget.description || "",
    when_to_use: widget.when_to_use || "",
    context_source: widget.context_source === "transcript" ? "transcript" : "summary",
    enabled: widget.enabled ?? true,
  };
}

async function readError(response: Response) {
  try {
    const data = await response.json();
    return data?.error?.message || data?.message || response.statusText;
  } catch {
    return response.statusText;
  }
}
