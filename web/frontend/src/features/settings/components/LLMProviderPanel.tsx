import { type FormEvent, useEffect, useMemo, useState } from "react";
import { CheckCircle2, Loader2 } from "lucide-react";
import { AppButton } from "@/shared/ui/Button";
import { Field } from "@/shared/ui/Field";
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

  const connectionStatus = useMemo(() => {
    if (!settings?.configured) return "Not connected";
    const provider = settings.provider === "ollama" ? "Ollama" : "OpenAI compatible";
    const modelCopy = `${settings.model_count} ${settings.model_count === 1 ? "model" : "models"}`;
    return `${provider} connected · ${modelCopy}`;
  }, [settings?.configured, settings?.model_count, settings?.provider]);

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
            <div className="scr-provider-connection" data-connected={settings?.configured === true}>
              <span className="scr-provider-connection-dot" aria-hidden="true" />
              <span>{connectionStatus}</span>
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
        </div>
      )}
    </section>
  );
}
