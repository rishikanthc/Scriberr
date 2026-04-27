import { useEffect, useMemo, useState } from "react";
import { Field } from "@/shared/ui/Field";
import { Select, type SelectOption } from "@/shared/ui/Select";
import { useLLMProviderSettings, useSaveLLMProviderSettings } from "@/features/settings/hooks/useLLMProvider";

export function LLMProviderPanel() {
  const providerQuery = useLLMProviderSettings();
  const saveProvider = useSaveLLMProviderSettings();
  const settings = providerQuery.data;
  const { error: saveError, isPending: isSaving, mutate: saveSettings } = saveProvider;
  const [baseURL, setBaseURL] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [largeModel, setLargeModel] = useState("");
  const [smallModel, setSmallModel] = useState("");
  const [dirty, setDirty] = useState(false);
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (dirty) return;
    setBaseURL(settings?.base_url || "");
    setApiKey("");
    setLargeModel(settings?.large_model || "");
    setSmallModel(settings?.small_model || "");
  }, [dirty, settings?.base_url, settings?.large_model, settings?.small_model]);

  const connectionStatus = useMemo(() => {
    if (!settings?.configured) return "Not connected";
    const provider = settings.provider === "ollama" ? "Ollama" : "OpenAI compatible";
    const modelCopy = `${settings.model_count} ${settings.model_count === 1 ? "model" : "models"}`;
    return `${provider} connected · ${modelCopy}`;
  }, [settings?.configured, settings?.model_count, settings?.provider]);

  const modelOptions = useMemo<SelectOption[]>(() => {
    const models = settings?.models || [];
    if (models.length === 0) {
      return [{ value: "", label: "No models available" }];
    }
    return [
      { value: "", label: "Not selected" },
      ...models.map((model) => ({ value: model, label: model })),
    ];
  }, [settings?.models]);

  const modelSelectDisabled = !settings?.configured || (settings.models || []).length === 0;
  const canAutoSave = useMemo(() => isValidProviderURL(baseURL), [baseURL]);

  useEffect(() => {
    if (!dirty || providerQuery.isLoading || !canAutoSave) return;

    const timeout = window.setTimeout(() => {
      setMessage("");
      saveSettings(
        {
          base_url: baseURL.trim(),
          api_key: apiKey.trim() || undefined,
          large_model: largeModel || undefined,
          small_model: smallModel || undefined,
        },
        {
          onSuccess: (saved) => {
            setDirty(false);
            setApiKey("");
            setMessage(`Saved · ${saved.model_count} ${saved.model_count === 1 ? "model" : "models"}.`);
          },
          onError: () => {
            setMessage("");
          },
        }
      );
    }, 800);

    return () => window.clearTimeout(timeout);
  }, [apiKey, baseURL, canAutoSave, dirty, largeModel, providerQuery.isLoading, saveSettings, smallModel]);

  useEffect(() => {
    if (!dirty) return;
    if (message) {
      setMessage("");
    }
  }, [dirty, message]);

  return (
    <section className="scr-settings-panel" aria-label="LLM settings">
      {providerQuery.error ? (
        <div className="scr-alert">
          {providerQuery.error instanceof Error ? providerQuery.error.message : "Could not load LLM provider settings."}
        </div>
      ) : null}

      {providerQuery.isLoading ? (
        <div className="scr-provider-skeleton" aria-label="Loading LLM provider settings" />
      ) : (
        <div className="scr-provider-form">
          <section className="scr-provider-section" aria-label="LLM providers">
            <div className="scr-provider-section-header">
              <h2 className="scr-settings-heading">LLM providers</h2>
              <p className="scr-settings-copy">
                Connect a local or hosted OpenAI-compatible endpoint for chat, summaries, and model-assisted workflows.
              </p>
            </div>

            <div className="scr-provider-section-controls">
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
            </div>
          </section>

          <section className="scr-provider-section" aria-label="Model configuration">
            <div className="scr-provider-section-header">
              <h2 className="scr-settings-heading">Model configuration</h2>
              <p className="scr-settings-copy">
                Choose the default models Scriberr should use for larger and smaller LLM tasks.
              </p>
            </div>

            <div className="scr-provider-section-controls">
              <div>
                <div className="scr-provider-model-grid">
                  <Select
                    label="Large LLM"
                    value={largeModel}
                    options={modelOptions}
                    disabled={modelSelectDisabled}
                    onChange={(value) => {
                      setLargeModel(value);
                      setDirty(true);
                      setMessage("");
                    }}
                  />
                  <Select
                    label="Small LLM"
                    value={smallModel}
                    options={modelOptions}
                    disabled={modelSelectDisabled}
                    onChange={(value) => {
                      setSmallModel(value);
                      setDirty(true);
                      setMessage("");
                    }}
                  />
                </div>
              </div>
            </div>
          </section>

          <div className="scr-provider-section-controls">
            {saveError ? (
              <div className="scr-alert">
                {saveError instanceof Error ? saveError.message : "Could not connect to this provider."}
              </div>
            ) : null}
            {isSaving ? <div className="scr-provider-autosave">Saving changes...</div> : null}
            {message ? <div className="scr-success">{message}</div> : null}
          </div>
        </div>
      )}
    </section>
  );
}

function isValidProviderURL(value: string) {
  try {
    const url = new URL(value.trim());
    return url.protocol === "http:" || url.protocol === "https:";
  } catch {
    return false;
  }
}
