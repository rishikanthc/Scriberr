export type ModelFamily = "whisper" | "nemo_transducer";

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

export type TranscriptionProfileOptions = {
  model_family: ModelFamily;
  model: string;
  language?: string;
  task: "transcribe" | "translate";
  threads: number;
  tail_paddings?: number;
  decoding_method: "greedy_search" | "modified_beam_search";
  chunking_strategy: "fixed" | "vad";
  diarize: boolean;
  diarize_model: "diarization-default";
  num_speakers: number;
  diarization_threshold: number;
  min_duration_on: number;
  min_duration_off: number;
};

export type TranscriptionProfile = {
  id: string;
  name: string;
  description: string;
  is_default: boolean;
  options: TranscriptionProfileOptions;
  parameters?: Partial<TranscriptionProfileOptions>;
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

export const defaultProfileParams: TranscriptionProfileOptions = {
  model_family: "whisper",
  model: "whisper-base",
  task: "transcribe",
  threads: 0,
  decoding_method: "greedy_search",
  diarize: false,
  chunking_strategy: "fixed",
  diarize_model: "diarization-default",
  num_speakers: 0,
  diarization_threshold: 0.5,
  min_duration_on: 0.2,
  min_duration_off: 0.3,
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
    options: {
      model: profile.options.model,
      language: profile.options.language || undefined,
      task: profile.options.task,
      threads: profile.options.threads,
      tail_paddings: profile.options.tail_paddings,
      decoding_method: profile.options.decoding_method,
      chunking_strategy: profile.options.chunking_strategy,
      diarize: profile.options.diarize,
      diarization: profile.options.diarize,
      diarize_model: "diarization-default",
      num_speakers: profile.options.num_speakers,
      diarization_threshold: profile.options.diarization_threshold,
      min_duration_on: profile.options.min_duration_on,
      min_duration_off: profile.options.min_duration_off,
    },
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

export function normalizeParams(params?: Partial<TranscriptionProfileOptions> & { diarization?: boolean }): TranscriptionProfileOptions {
  const model = params?.model || defaultProfileParams.model;
  return {
    ...defaultProfileParams,
    ...params,
    model,
    model_family: params?.model_family || familyForModel(model),
    language: params?.language || undefined,
    task: params?.task === "translate" ? "translate" : "transcribe",
    decoding_method: params?.decoding_method === "modified_beam_search" ? "modified_beam_search" : "greedy_search",
    chunking_strategy: params?.chunking_strategy === "vad" ? "vad" : "fixed",
    diarize: params?.diarize ?? Boolean(params?.diarization),
    diarize_model: "diarization-default",
    num_speakers: params?.num_speakers || 0,
    diarization_threshold: params?.diarization_threshold || 0.5,
    min_duration_on: params?.min_duration_on || 0.2,
    min_duration_off: params?.min_duration_off || 0.3,
  };
}

export function familyForModel(model: string): ModelFamily {
  if (model.startsWith("parakeet")) return "nemo_transducer";
  return "whisper";
}

function normalizeProfile(profile: TranscriptionProfile): TranscriptionProfile {
  return {
    ...profile,
    description: profile.description || "",
    options: normalizeParams(profile.options || profile.parameters),
  };
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
