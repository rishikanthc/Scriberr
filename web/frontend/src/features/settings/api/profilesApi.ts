export type ModelFamily = "whisper" | "nvidia_parakeet" | "nvidia_canary" | "mistral_voxtral" | "openai";

export type WhisperXParams = {
  model_family: ModelFamily;
  model: string;
  model_cache_only: boolean;
  model_dir?: string;
  device: string;
  device_index: number;
  batch_size: number;
  compute_type: string;
  threads: number;
  output_format: string;
  verbose: boolean;
  task: string;
  language?: string;
  align_model?: string;
  interpolate_method: string;
  no_align: boolean;
  return_char_alignments: boolean;
  vad_method: string;
  vad_onset: number;
  vad_offset: number;
  chunk_size: number;
  diarize: boolean;
  min_speakers?: number;
  max_speakers?: number;
  diarize_model: string;
  speaker_embeddings: boolean;
  temperature: number;
  best_of: number;
  beam_size: number;
  patience: number;
  length_penalty: number;
  suppress_tokens?: string;
  suppress_numerals: boolean;
  initial_prompt?: string;
  condition_on_previous_text: boolean;
  fp16: boolean;
  temperature_increment_on_fallback: number;
  compression_ratio_threshold: number;
  logprob_threshold: number;
  no_speech_threshold: number;
  max_line_width?: number;
  max_line_count?: number;
  highlight_words: boolean;
  segment_resolution: string;
  hf_token?: string;
  print_progress: boolean;
  attention_context_left: number;
  attention_context_right: number;
  callback_url?: string;
  api_key?: string;
  max_new_tokens?: number;
};

export type TranscriptionProfile = {
  id: string;
  name: string;
  description: string;
  is_default: boolean;
  options: WhisperXParams;
  parameters?: WhisperXParams;
  created_at: string;
  updated_at: string;
};

export const defaultProfileParams: WhisperXParams = {
  model_family: "whisper",
  model: "small",
  model_cache_only: false,
  device: "auto",
  device_index: 0,
  batch_size: 8,
  compute_type: "float32",
  threads: 0,
  output_format: "all",
  verbose: true,
  task: "transcribe",
  interpolate_method: "nearest",
  no_align: false,
  return_char_alignments: false,
  vad_method: "pyannote",
  vad_onset: 0.5,
  vad_offset: 0.363,
  chunk_size: 30,
  diarize: false,
  diarize_model: "pyannote",
  speaker_embeddings: false,
  temperature: 0,
  best_of: 5,
  beam_size: 5,
  patience: 1,
  length_penalty: 1,
  suppress_numerals: false,
  condition_on_previous_text: false,
  fp16: true,
  temperature_increment_on_fallback: 0.2,
  compression_ratio_threshold: 2.4,
  logprob_threshold: -1,
  no_speech_threshold: 0.6,
  highlight_words: false,
  segment_resolution: "sentence",
  print_progress: false,
  attention_context_left: 256,
  attention_context_right: 256,
  api_key: "",
};

type ProfileListResponse = {
  items?: TranscriptionProfile[];
};

export async function listProfiles() {
  const response = await fetch("/api/v1/profiles");
  if (!response.ok) throw new Error(await readError(response));
  const data = (await response.json()) as ProfileListResponse | TranscriptionProfile[];
  const items = Array.isArray(data) ? data : data.items || [];
  return items.map(normalizeProfile);
}

export async function saveProfile(profile: {
  id?: string;
  name: string;
  description: string;
  is_default: boolean;
  options: WhisperXParams;
}) {
  const payload = {
    name: profile.name,
    description: profile.description,
    is_default: profile.is_default,
    options: {
      ...profile.options,
      language: profile.options.language || undefined,
      diarization: profile.options.diarize,
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

export function normalizeParams(params?: Partial<WhisperXParams>): WhisperXParams {
  return {
    ...defaultProfileParams,
    ...params,
    language: params?.language || undefined,
    diarize: params?.diarize ?? Boolean((params as Partial<WhisperXParams> & { diarization?: boolean })?.diarization),
  };
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
