import { useState, useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { MainLayout } from "@/components/layout/MainLayout";
import { User, Settings as SettingsIcon, Key, Bot, FileText, Plus, Terminal } from "lucide-react";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Header } from "@/components/Header";
import { ProfileSettings } from "../components/ProfileSettings";
import { AccountSettings } from "../components/AccountSettings";
import { APIKeySettings } from "../components/APIKeySettings";
import { LLMSettings } from "../components/LLMSettings";
import { SummaryTemplateDialog, type SummaryTemplate } from "../components/SummaryTemplateDialog";
import { SummaryTemplatesTable } from "../components/SummaryTemplatesTable";
import { CLISettingsTab } from "../components/CLISettingsTab";
import { useAuth } from "@/features/auth/hooks/useAuth";

export function Settings() {
  const [activeTab, setActiveTab] = useState("transcription");
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();
  const [summaryDialogOpen, setSummaryDialogOpen] = useState(false);
  const [editingSummary, setEditingSummary] = useState<SummaryTemplate | null>(null);
  const [summaryRefresh, setSummaryRefresh] = useState(0);
  const [llmConfigured, setLlmConfigured] = useState(false);

  // Fetch LLM config and models
  useEffect(() => {
    const fetchLLM = async () => {
      try {
        const cfgRes = await fetch('/api/v1/llm/config', { headers: { ...getAuthHeaders() } });
        if (!cfgRes.ok) { setLlmConfigured(false); return; }
        const cfg = await cfgRes.json();
        setLlmConfigured(!!cfg && cfg.is_active);
        // Set configured; models are chosen per-template in dialog now
        if (cfg && cfg.is_active) {
          setLlmConfigured(true);
        }
      } catch {
        setLlmConfigured(false);
      }
    };
    fetchLLM();
  }, [activeTab, getAuthHeaders]);

  return (
    <MainLayout
      header={<Header />}
    >
      {/* Main Content Container with same styling as Homepage */}
      <div className="bg-[var(--bg-card)] border border-[var(--border-subtle)] shadow-[var(--shadow-card)] rounded-[var(--radius-card)] p-2 sm:p-6 mt-8">
        <div className="mb-4 sm:mb-8">
          <h1 className="text-2xl font-display font-bold text-[var(--text-primary)] mb-2">
            Settings
          </h1>
          <p className="text-[var(--text-secondary)]">
            Manage your account settings and transcription profiles.
          </p>
        </div>

        {/* Tabbed Interface */}
        <Tabs
          value={activeTab}
          onValueChange={setActiveTab}
          className="space-y-4 sm:space-y-6"
        >
          <TabsList className="grid w-full grid-cols-6 items-center h-auto bg-[var(--bg-main)]/50 border border-[var(--border-subtle)] p-1 rounded-xl">
            <TabsTrigger
              value="transcription"
              aria-label="Transcription"
              className="flex items-center justify-center gap-2 h-9 py-1.5 data-[state=active]:bg-[var(--bg-card)] data-[state=active]:shadow-sm data-[state=active]:text-[var(--text-primary)] text-[var(--text-tertiary)] hover:text-[var(--text-secondary)] font-medium rounded-lg text-xs sm:text-sm transition-all"
            >
              <SettingsIcon className="h-4 w-4" />
              <span className="hidden sm:inline">Transcription</span>
            </TabsTrigger>
            <TabsTrigger
              value="account"
              aria-label="Account"
              className="flex items-center justify-center gap-2 h-9 py-1.5 data-[state=active]:bg-[var(--bg-card)] data-[state=active]:shadow-sm data-[state=active]:text-[var(--text-primary)] text-[var(--text-tertiary)] hover:text-[var(--text-secondary)] font-medium rounded-lg text-xs sm:text-sm transition-all"
            >
              <User className="h-4 w-4" />
              <span className="hidden sm:inline">Account</span>
            </TabsTrigger>
            <TabsTrigger
              value="apikeys"
              aria-label="API Keys"
              className="flex items-center justify-center gap-2 h-9 py-1.5 data-[state=active]:bg-[var(--bg-card)] data-[state=active]:shadow-sm data-[state=active]:text-[var(--text-primary)] text-[var(--text-tertiary)] hover:text-[var(--text-secondary)] font-medium rounded-lg text-xs sm:text-sm transition-all"
            >
              <Key className="h-4 w-4" />
              <span className="hidden sm:inline">API Keys</span>
            </TabsTrigger>
            <TabsTrigger
              value="llms"
              aria-label="LLMs"
              className="flex items-center justify-center gap-2 h-9 py-1.5 data-[state=active]:bg-[var(--bg-card)] data-[state=active]:shadow-sm data-[state=active]:text-[var(--text-primary)] text-[var(--text-tertiary)] hover:text-[var(--text-secondary)] font-medium rounded-lg text-xs sm:text-sm transition-all"
            >
              <Bot className="h-4 w-4" />
              <span className="hidden sm:inline">LLMs</span>
            </TabsTrigger>
            <TabsTrigger
              value="summary"
              aria-label="Summary"
              className="flex items-center justify-center gap-2 h-9 py-1.5 data-[state=active]:bg-[var(--bg-card)] data-[state=active]:shadow-sm data-[state=active]:text-[var(--text-primary)] text-[var(--text-tertiary)] hover:text-[var(--text-secondary)] font-medium rounded-lg text-xs sm:text-sm transition-all"
            >
              <FileText className="h-4 w-4" />
              <span className="hidden sm:inline">Summary</span>
            </TabsTrigger>
            <TabsTrigger
              value="cli"
              aria-label="CLI Watcher"
              className="flex items-center justify-center gap-2 h-9 py-1.5 data-[state=active]:bg-[var(--bg-card)] data-[state=active]:shadow-sm data-[state=active]:text-[var(--text-primary)] text-[var(--text-tertiary)] hover:text-[var(--text-secondary)] font-medium rounded-lg text-xs sm:text-sm transition-all"
            >
              <Terminal className="h-4 w-4" />
              <span className="hidden sm:inline">CLI Watcher</span>
            </TabsTrigger>
          </TabsList>

          {/* Transcription Tab */}
          <TabsContent value="transcription" className="space-y-6">
            <ProfileSettings />
          </TabsContent>

          {/* Account Tab */}
          <TabsContent value="account" className="space-y-6">
            <AccountSettings />
          </TabsContent>

          {/* API Keys Tab */}
          <TabsContent value="apikeys" className="space-y-6">
            <APIKeySettings />
          </TabsContent>

          {/* LLMs Tab */}
          <TabsContent value="llms" className="space-y-6">
            <LLMSettings />
          </TabsContent>

          {/* Summary Tab */}
          <TabsContent value="summary" className="space-y-6">
            <div className="bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-[var(--radius-card)] p-4 sm:p-6 shadow-sm">
              <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 sm:gap-0 mb-4">
                <div>
                  <h3 className="text-lg font-medium text-[var(--text-primary)]">Summarization Templates</h3>
                  <p className="text-sm text-[var(--text-secondary)] mt-1">Create and manage prompts used to summarize transcripts.</p>
                </div>
                <div className="flex items-center gap-3">
                  <Button
                    variant="outline"
                    onClick={() => { setEditingSummary(null); setSummaryDialogOpen(true); }}
                    disabled={!llmConfigured}
                  >
                    <Plus className="h-4 w-4" /> New Template
                  </Button>
                </div>
              </div>
              {!llmConfigured && (
                <div className="mb-3 text-sm text-[var(--warning-solid)] bg-[var(--warning-translucent)] border border-[var(--warning-solid)]/20 rounded-md px-3 py-2">
                  Configure an LLM provider in the LLMs tab to enable summary templates and model selection.
                </div>
              )}
              <SummaryTemplatesTable onEdit={(tpl) => { setEditingSummary(tpl); setSummaryDialogOpen(true); }} refreshTrigger={summaryRefresh} disabled={!llmConfigured} />
            </div>

            <SummaryTemplateDialog
              open={summaryDialogOpen}
              onOpenChange={(o) => { setSummaryDialogOpen(o); if (!o) setEditingSummary(null); }}
              initial={editingSummary}
              onSave={async (tpl) => {
                const headers: HeadersInit = { 'Content-Type': 'application/json', ...getAuthHeaders() };
                try {
                  if (tpl.id) {
                    await fetch(`/api/v1/summaries/${tpl.id}`, { method: 'PUT', headers, body: JSON.stringify({ name: tpl.name, description: tpl.description, model: tpl.model, prompt: tpl.prompt }) });
                  } else {
                    await fetch('/api/v1/summaries', { method: 'POST', headers, body: JSON.stringify({ name: tpl.name, description: tpl.description, model: tpl.model, prompt: tpl.prompt }) });
                  }
                } finally {
                  // Invalidate cache to propagate changes
                  queryClient.invalidateQueries({ queryKey: ["summaryTemplates"] });

                  // keep user on Summary tab and refresh the list without a full reload
                  setSummaryDialogOpen(false);
                  setEditingSummary(null);
                  setSummaryRefresh((n) => n + 1);
                }
              }}
            />
          </TabsContent>

          {/* CLI Watcher Tab */}
          <TabsContent value="cli" className="space-y-6">
            <CLISettingsTab />
          </TabsContent>
        </Tabs>
      </div>
    </MainLayout>
  );
}
