export type LLMProviderSettings = {
  configured: boolean;
  provider: "openai_compatible" | "ollama" | string;
  base_url: string;
  has_api_key: boolean;
  key_preview?: string | null;
  model_count: number;
  models: string[];
  large_model?: string | null;
  small_model?: string | null;
  updated_at?: string;
};

export type SaveLLMProviderPayload = {
  base_url: string;
  api_key?: string;
  large_model?: string;
  small_model?: string;
};

export async function getLLMProviderSettings(headers?: Record<string, string>) {
  const response = await fetch("/api/v1/settings/llm-provider", { headers });
  if (!response.ok) throw new Error(await readError(response));
  const data = (await response.json()) as LLMProviderSettings;
  return {
    ...data,
    models: data.models || [],
  };
}

export async function saveLLMProviderSettings(payload: SaveLLMProviderPayload, headers?: Record<string, string>) {
  const response = await fetch("/api/v1/settings/llm-provider", {
    method: "PUT",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify({
      base_url: payload.base_url,
      api_key: payload.api_key || undefined,
      large_model: payload.large_model || undefined,
      small_model: payload.small_model || undefined,
    }),
  });
  if (!response.ok) throw new Error(await readError(response));
  const data = (await response.json()) as LLMProviderSettings;
  return {
    ...data,
    models: data.models || [],
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
