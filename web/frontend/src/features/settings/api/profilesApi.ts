export type ASRCapability =
  | "transcription"
  | "diarization"
  | "speaker_identification"
  | "translation"
  | "word_timestamps"
  | "segment_timestamps"
  | "token_timestamps"
  | "streaming"
  | "custom_vocabulary"
  | "initial_prompt"
  | "language_detection"
  | "speaker_embeddings";

export type ASRCapabilities = {
  transcription?: boolean;
  diarization?: boolean;
  speaker_identification?: boolean;
  translation?: boolean;
  word_timestamps?: boolean;
  segment_timestamps?: boolean;
  token_timestamps?: boolean;
  streaming?: boolean;
  custom_vocabulary?: boolean;
  initial_prompt?: boolean;
  language_detection?: boolean;
  speaker_embeddings?: boolean;
  extensions?: Record<string, boolean>;
};

export type ParameterOption = {
  value: unknown;
  label?: string;
};

export type ActivationRule = {
  parameter: string;
  operator: "equals";
  value: unknown;
};

export type ParameterDescriptor = {
  key: string;
  label?: string;
  type: "boolean" | "integer" | "number" | "string" | "enum" | "duration" | "path_ref";
  default?: unknown;
  min?: number;
  max?: number;
  step?: number;
  options?: ParameterOption[];
  scope: "model" | "runtime" | "decoding" | "chunking" | "vad" | "output" | "postprocess";
  required?: boolean;
  advanced?: boolean;
  read_only?: boolean;
  requires_reload?: boolean;
  expose_in_summary?: boolean;
  visible_when?: ActivationRule[];
};

export type ASRModelCard = {
  id: string;
  display_name: string;
  provider: string;
  model_type?: string;
  version?: string;
  installed: boolean;
  loaded?: boolean;
  default: boolean;
  tasks?: string[];
  languages?: string[];
  capabilities: ASRCapabilities;
  chunking?: Record<string, unknown>;
  dependencies?: unknown[];
  artifacts?: unknown[];
  parameter_schema?: ParameterDescriptor[];
  recommended_defaults?: Record<string, unknown>;
  license?: string;
};

export type ASRStepKind = "transcription" | "diarization" | "speaker_identification";

export type ASRStep = {
  kind: ASRStepKind;
  provider?: string;
  model?: string;
  model_family?: string;
  options?: Record<string, unknown>;
};

export type TranscriptionProfileOptions = {
  pipeline: ASRStep[];
};

export type TranscriptionProfile = {
  id: string;
  name: string;
  description: string;
  is_default: boolean;
  options: TranscriptionProfileOptions;
  parameters?: TranscriptionProfileOptions;
  created_at: string;
  updated_at: string;
};

export type TranscriptionModel = {
  id: string;
  name: string;
  provider: string;
  installed: boolean;
  default: boolean;
  capabilities: string[];
};

type ProfileListResponse = {
  items?: TranscriptionProfile[];
};

type ModelListResponse = {
  items?: ASRModelCard[];
};

export async function listProfiles(headers?: Record<string, string>) {
  const response = await fetch("/api/v1/profiles", { headers });
  if (!response.ok) throw new Error(await readError(response));
  const data = (await response.json()) as ProfileListResponse | TranscriptionProfile[];
  const items = Array.isArray(data) ? data : data.items || [];
  return items.map(normalizeProfile);
}

export async function listTranscriptionModels(headers?: Record<string, string>) {
  const models = await listASRModels(["transcription"], headers);
  return models.map(modelCardToTranscriptionModel);
}

export async function listASRModels(capabilities?: ASRCapability[], headers?: Record<string, string>) {
  const query = new URLSearchParams();
  if (capabilities && capabilities.length > 0) {
    query.set("capability", capabilities.join(","));
  }
  const response = await fetch(`/api/v1/models${query.toString() ? `?${query.toString()}` : ""}`, { headers });
  if (!response.ok) throw new Error(await readError(response));
  const data = (await response.json()) as ModelListResponse;
  return (data.items || []).filter((model) => !capabilities?.length || capabilities.some((capability) => modelSupportsCapability(model, capability)));
}

export async function saveProfile(profile: {
  id?: string;
  name: string;
  description: string;
  is_default: boolean;
  options: TranscriptionProfileOptions;
}, headers?: Record<string, string>) {
  const payload = {
    name: profile.name,
    description: profile.description,
    is_default: profile.is_default,
    options: profile.options,
  };
  const response = await fetch(profile.id ? `/api/v1/profiles/${profile.id}` : "/api/v1/profiles", {
    method: profile.id ? "PATCH" : "POST",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await readError(response));
  return normalizeProfile((await response.json()) as TranscriptionProfile);
}

export async function deleteProfile(profileId: string, headers?: Record<string, string>) {
  const response = await fetch(`/api/v1/profiles/${profileId}`, { method: "DELETE", headers });
  if (!response.ok) throw new Error(await readError(response));
}

function normalizeProfile(profile: TranscriptionProfile): TranscriptionProfile {
  return {
    ...profile,
    description: profile.description || "",
    options: normalizeProfileOptions(profile.options || profile.parameters),
  };
}

export function normalizeProfileOptions(options?: TranscriptionProfileOptions): TranscriptionProfileOptions {
  return { pipeline: Array.isArray(options?.pipeline) ? options.pipeline : [] };
}

function modelCardToTranscriptionModel(model: ASRModelCard): TranscriptionModel {
  return {
    id: model.id,
    name: model.display_name || model.id,
    provider: model.provider,
    installed: model.installed,
    default: model.default,
    capabilities: capabilitiesToList(model.capabilities),
  };
}

function capabilitiesToList(capabilities: ASRCapabilities) {
  const out: string[] = [];
  for (const [key, value] of Object.entries(capabilities)) {
    if (key === "extensions") continue;
    if (value === true) out.push(key);
  }
  if (capabilities.extensions) {
    for (const [key, value] of Object.entries(capabilities.extensions)) {
      if (value === true) out.push(key);
    }
  }
  return out;
}

function modelSupportsCapability(model: ASRModelCard, capability: ASRCapability) {
  const direct = model.capabilities[capability as keyof ASRCapabilities];
  return direct === true || model.capabilities.extensions?.[capability] === true;
}

async function readError(response: Response) {
  try {
    const data = await response.json();
    return data?.error?.message || data?.message || response.statusText;
  } catch {
    return response.statusText;
  }
}
