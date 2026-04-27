import { type FormEvent, type ReactNode, useEffect, useMemo, useState } from "react";
import { CheckCircle2, KeyRound, Loader2, Server } from "lucide-react";
import { AppButton } from "@/shared/ui/Button";
import { Field } from "@/shared/ui/Field";
import { EmptyState } from "@/shared/ui/EmptyState";
import { useLLMProviderSettings, useSaveLLMProviderSettings } from "@/features/settings/hooks/useLLMProvider";

export function LLMProviderPanel() {
  const providerQuery = useLLMProviderSettings();
  const saveProvider = useSaveLLMProviderSettings();
  const settings = providerQuery.data;
  const [baseURL, setBaseURL] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [dirty, setDirty] = useState(false);
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (dirty) return;
    setBaseURL(settings?.base_url || "");
    setApiKey("");
  }, [dirty, settings?.base_url]);

  const providerLabel = useMemo(() => {
    if (!settings?.provider) return "Not configured";
    if (settings.provider === "openai_compatible") return "OpenAI compatible";
    if (settings.provider === "ollama") return "Ollama";
    return settings.provider;
  }, [settings?.provider]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setMessage("");
    try {
      const saved = await saveProvider.mutateAsync({
        base_url: baseURL.trim(),
        api_key: apiKey.trim() || undefined,
      });
      setDirty(false);
      setApiKey("");
      setMessage(`Connected to ${saved.model_count} ${saved.model_count === 1 ? "model" : "models"}.`);
    } catch {
      setMessage("");
    }
  };

  return (
    <section className="scr-settings-panel" aria-label="LLM provider settings">
      <div className="scr-settings-panel-head">
        <div>
          <h2 className="scr-settings-heading">LLM providers</h2>
          <p className="scr-settings-copy">
            Connect a local or hosted OpenAI-compatible endpoint for chat, summaries, and model-assisted workflows.
          </p>
        </div>
      </div>

      {providerQuery.error ? (
        <div className="scr-alert">
          {providerQuery.error instanceof Error ? providerQuery.error.message : "Could not load LLM provider settings."}
        </div>
      ) : null}

      {providerQuery.isLoading ? (
        <div className="scr-provider-skeleton" aria-label="Loading LLM provider settings" />
      ) : (
        <div className="scr-provider-layout">
          <form className="scr-provider-form" onSubmit={(event) => void handleSubmit(event)}>
            <div className="scr-provider-status-row">
              <ProviderStatus icon={<Server size={16} aria-hidden="true" />} label="Provider" value={providerLabel} />
              <ProviderStatus
                icon={<KeyRound size={16} aria-hidden="true" />}
                label="API key"
                value={settings?.has_api_key ? settings.key_preview || "Configured" : "Not set"}
              />
              <ProviderStatus
                icon={<CheckCircle2 size={16} aria-hidden="true" />}
                label="Models"
                value={settings?.configured ? String(settings.model_count) : "Not tested"}
              />
            </div>

            <div className="scr-provider-fields">
              <Field
                id="llm-provider-base-url"
                label="Base endpoint"
                type="url"
                inputMode="url"
                autoComplete="url"
                placeholder="https://api.openai.com/v1"
                value={baseURL}
                onChange={(event) => {
                  setBaseURL(event.target.value);
                  setDirty(true);
                  setMessage("");
                }}
                required
              />
              <Field
                id="llm-provider-api-key"
                label={settings?.has_api_key ? "API key (leave blank to keep current)" : "API key (optional)"}
                type="password"
                autoComplete="off"
                placeholder={settings?.has_api_key ? "Current key is saved" : "sk-..."}
                value={apiKey}
                onChange={(event) => {
                  setApiKey(event.target.value);
                  setDirty(true);
                  setMessage("");
                }}
              />
            </div>

            {saveProvider.error ? (
              <div className="scr-alert">
                {saveProvider.error instanceof Error ? saveProvider.error.message : "Could not connect to this provider."}
              </div>
            ) : null}
            {message ? <div className="scr-success">{message}</div> : null}

            <div className="scr-provider-actions">
              <AppButton className="scr-provider-save" type="submit" disabled={saveProvider.isPending || baseURL.trim() === ""}>
                {saveProvider.isPending ? <Loader2 className="scr-spin" size={15} aria-hidden="true" /> : <CheckCircle2 size={15} aria-hidden="true" />}
                {saveProvider.isPending ? "Testing" : "Save provider"}
              </AppButton>
            </div>
          </form>

          {settings?.configured && settings.models.length > 0 ? (
            <div className="scr-provider-models" aria-label="Available LLM models">
              <h3 className="scr-provider-models-title">Available models</h3>
              <div className="scr-provider-model-list">
                {settings.models.slice(0, 8).map((model) => (
                  <span className="scr-provider-model" key={model}>
                    {model}
                  </span>
                ))}
              </div>
            </div>
          ) : (
            <EmptyState title="No provider saved" description="Save a provider to make its models available in Scriberr." />
          )}
        </div>
      )}
    </section>
  );
}

function ProviderStatus({ icon, label, value }: { icon: ReactNode; label: string; value: string }) {
  return (
    <div className="scr-provider-status">
      <span className="scr-provider-status-icon">{icon}</span>
      <span className="scr-provider-status-copy">
        <span className="scr-provider-status-label">{label}</span>
        <span className="scr-provider-status-value">{value}</span>
      </span>
    </div>
  );
}
