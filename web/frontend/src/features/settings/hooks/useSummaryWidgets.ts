import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import {
  deleteSummaryWidget,
  listSummaryWidgets,
  saveSummaryWidget,
  type SummaryWidgetPayload,
} from "@/features/settings/api/summaryWidgetsApi";

export const summaryWidgetsQueryKey = ["summary-widgets"] as const;

export function useSummaryWidgets() {
  const { getAuthHeaders, isAuthenticated } = useAuth();

  return useQuery({
    queryKey: summaryWidgetsQueryKey,
    queryFn: () => listSummaryWidgets(getAuthHeaders()),
    enabled: isAuthenticated,
  });
}

export function useSaveSummaryWidget() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (widget: SummaryWidgetPayload) => saveSummaryWidget(widget, getAuthHeaders()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: summaryWidgetsQueryKey });
    },
  });
}

export function useDeleteSummaryWidget() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (widgetId: string) => deleteSummaryWidget(widgetId, getAuthHeaders()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: summaryWidgetsQueryKey });
    },
  });
}
