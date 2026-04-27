export type ModelFamily = "whisper" | "nemo_transducer" | "canary";

export type TranscriptionProfileOptions = {
  model_family: ModelFamily;
  model: string;
  language?: string;
  task: "transcribe" | "translate";
  threads: number;
  tail_paddings?: number;
  enable_token_timestamps?: boolean;
  enable_segment_timestamps?: boolean;
  canary_source_language: string;
  canary_target_language: string;
  canary_use_punctuation?: boolean;
  decoding_method: "greedy_search" | "modified_beam_search";
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
  canary_source_language: "en",
  canary_target_language: "en",
  canary_use_punctuation: true,
  diarize: false,
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
  items?: TranscriptionModel[];
};

export async function listProfiles() {
  const response = await fetch("/api/v1/profiles");
  if (!response.ok) throw new Error(await readError(response));
  const data = (await response.json()) as ProfileListResponse | TranscriptionProfile[];
  const items = Array.isArray(data) ? data : data.items || [];
  return items.map(normalizeProfile);
}

export async function listTranscriptionModels() {
  const response = await fetch("/api/v1/models/transcription");
  if (!response.ok) throw new Error(await readError(response));
  const data = (await response.json()) as ModelListResponse;
  return (data.items || []).filter((model) => model.capabilities.includes("transcription"));
}

export async function saveProfile(profile: {
  id?: string;
  name: string;
  description: string;
  is_default: boolean;
  options: TranscriptionProfileOptions;
}) {
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
      enable_token_timestamps: profile.options.enable_token_timestamps,
      enable_segment_timestamps: profile.options.enable_segment_timestamps,
      canary_source_language: profile.options.canary_source_language,
      canary_target_language: profile.options.canary_target_language,
      canary_use_punctuation: profile.options.canary_use_punctuation,
      decoding_method: profile.options.decoding_method,
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
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await readError(response));
  return normalizeProfile((await response.json()) as TranscriptionProfile);
}

export async function deleteProfile(profileId: string) {
  const response = await fetch(`/api/v1/profiles/${profileId}`, { method: "DELETE" });
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
    canary_source_language: params?.canary_source_language || "en",
    canary_target_language: params?.canary_target_language || "en",
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
  if (model.startsWith("canary")) return "canary";
  return "whisper";
}

function normalizeProfile(profile: TranscriptionProfile): TranscriptionProfile {
  return {
    ...profile,
    description: profile.description || "",
    options: normalizeParams(profile.options || profile.parameters),
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
