import { useState, useEffect } from "react";
import { User, Settings as SettingsIcon, Key, Bot, FileText, Plus } from "lucide-react";
import {
	Tabs,
	TabsContent,
	TabsList,
	TabsTrigger,
} from "../components/ui/tabs";
import { Header } from "../components/Header";
import { ProfileSettings } from "../components/ProfileSettings";
import { AccountSettings } from "../components/AccountSettings";
import { APIKeySettings } from "../components/APIKeySettings";
import { LLMSettings } from "../components/LLMSettings";
import { SummaryTemplateDialog, type SummaryTemplate } from "../components/SummaryTemplateDialog";
import { SummaryTemplatesTable } from "../components/SummaryTemplatesTable";
import { useAuth } from "../contexts/AuthContext";

export function Settings() {
  const [activeTab, setActiveTab] = useState("profiles");
  const { getAuthHeaders } = useAuth();
  const [summaryDialogOpen, setSummaryDialogOpen] = useState(false);
  const [editingSummary, setEditingSummary] = useState<SummaryTemplate | null>(null);
  const [summaryRefresh, setSummaryRefresh] = useState(0);
  const [llmConfigured, setLlmConfigured] = useState(false);

  // Fetch LLM config and models
  useEffect(() => {
    const fetchLLM = async () => {
      try {
        const cfgRes = await fetch('/api/v1/llm/config', { headers: { ...getAuthHeaders() }});
        if (!cfgRes.ok) { setLlmConfigured(false); return; }
        const cfg = await cfgRes.json();
        setLlmConfigured(!!cfg && cfg.is_active);
        // Set configured; models are chosen per-template in dialog now
        if (cfg && cfg.is_active) {
          setLlmConfigured(true);
        }
      } catch (e) {
        setLlmConfigured(false);
      }
    };
    fetchLLM();
  }, []);


	// Dummy function for file select (Settings page doesn't upload files)
	const handleFileSelect = () => {
		// No file upload in settings
	};

	return (
		<div className="min-h-screen bg-gray-50 dark:bg-gray-900">
			<div className="mx-auto w-full max-w-6xl px-2 sm:px-6 md:px-8 py-3 sm:py-6">
				{/* Use the same Header component as Homepage */}
				<Header onFileSelect={handleFileSelect} />

				{/* Main Content Container with same styling as Homepage */}
				<div className="bg-white dark:bg-gray-800 rounded-xl p-2 sm:p-6 mt-4 sm:mt-6">
					<div className="mb-4 sm:mb-8">
						<h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-2">
							Settings
						</h1>
						<p className="text-gray-600 dark:text-gray-400">
							Manage your account settings and transcription profiles.
						</p>
					</div>

					{/* Tabbed Interface */}
						<Tabs
							value={activeTab}
							onValueChange={setActiveTab}
							className="space-y-4 sm:space-y-6"
						>
            <TabsList className="grid w-full grid-cols-5 items-center h-auto bg-gray-100 dark:bg-gray-800 p-1 rounded-xl">
                            <TabsTrigger
                                value="profiles"
                                aria-label="Profiles"
                                className="flex items-center justify-center gap-2 h-9 py-1.5 data-[state=active]:bg-white dark:data-[state=active]:bg-gray-700 data-[state=active]:text-gray-900 dark:data-[state=active]:text-gray-100 text-gray-600 dark:text-gray-400 font-medium rounded-lg text-xs sm:text-sm"
                            >
									<SettingsIcon className="h-4 w-4" />
									<span className="hidden sm:inline">Profiles</span>
								</TabsTrigger>
                            <TabsTrigger
                                value="account"
                                aria-label="Account"
                                className="flex items-center justify-center gap-2 h-9 py-1.5 data-[state=active]:bg-white dark:data-[state=active]:bg-gray-700 data-[state=active]:text-gray-900 dark:data-[state=active]:text-gray-100 text-gray-600 dark:text-gray-400 font-medium rounded-lg text-xs sm:text-sm"
                            >
									<User className="h-4 w-4" />
									<span className="hidden sm:inline">Account</span>
								</TabsTrigger>
                            <TabsTrigger
                                value="apikeys"
                                aria-label="API Keys"
                                className="flex items-center justify-center gap-2 h-9 py-1.5 data-[state=active]:bg-white dark:data-[state=active]:bg-gray-700 data-[state=active]:text-gray-900 dark:data-[state=active]:text-gray-100 text-gray-600 dark:text-gray-400 font-medium rounded-lg text-xs sm:text-sm"
                            >
									<Key className="h-4 w-4" />
									<span className="hidden sm:inline">API Keys</span>
								</TabsTrigger>
            <TabsTrigger
              value="llms"
              aria-label="LLMs"
              className="flex items-center justify-center gap-2 h-9 py-1.5 data-[state=active]:bg-white dark:data-[state=active]:bg-gray-700 data-[state=active]:text-gray-900 dark:data-[state=active]:text-gray-100 text-gray-600 dark:text-gray-400 font-medium rounded-lg text-xs sm:text-sm"
            >
              <Bot className="h-4 w-4" />
              <span className="hidden sm:inline">LLMs</span>
            </TabsTrigger>
            <TabsTrigger
              value="summary"
              aria-label="Summary"
              className="flex items-center justify-center gap-2 h-9 py-1.5 data-[state=active]:bg-white dark:data-[state=active]:bg-gray-700 data-[state=active]:text-gray-900 dark:data-[state=active]:text-gray-100 text-gray-600 dark:text-gray-400 font-medium rounded-lg text-xs sm:text-sm"
            >
              <FileText className="h-4 w-4" />
              <span className="hidden sm:inline">Summary</span>
            </TabsTrigger>
							</TabsList>

						{/* Profiles Tab */}
						<TabsContent value="profiles" className="space-y-6">
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
            <div className="bg-gray-50 dark:bg-gray-700/50 rounded-xl p-4 sm:p-6">
              <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 sm:gap-0 mb-4">
                <div>
                  <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">Summarization Templates</h3>
                  <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">Create and manage prompts used to summarize transcripts.</p>
                </div>
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => { setEditingSummary(null); setSummaryDialogOpen(true); }}
                    disabled={!llmConfigured}
                    className={`inline-flex items-center gap-2 px-3 py-2 rounded-md cursor-pointer ${llmConfigured ? 'bg-blue-600 hover:bg-blue-700 text-white' : 'bg-gray-300 text-gray-500 cursor-not-allowed'}`}
                  >
                    <Plus className="h-4 w-4" /> New Template
                  </button>
                </div>
              </div>
              {!llmConfigured && (
                <div className="mb-3 text-sm text-gray-700 dark:text-gray-300 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-md px-3 py-2">
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
                    await fetch(`/api/v1/summaries/${tpl.id}`, { method: 'PUT', headers, body: JSON.stringify({ name: tpl.name, description: tpl.description, prompt: tpl.prompt }) });
                  } else {
                    await fetch('/api/v1/summaries', { method: 'POST', headers, body: JSON.stringify({ name: tpl.name, description: tpl.description, prompt: tpl.prompt }) });
                  }
                } finally {
                  // keep user on Summary tab and refresh the list without a full reload
                  setSummaryDialogOpen(false);
                  setEditingSummary(null);
                  setSummaryRefresh((n) => n + 1);
                }
              }}
            />
          </TabsContent>
					</Tabs>
				</div>
			</div>
		</div>
	);
}
