import type { ASRStep, TranscriptionModel, TranscriptionProfileOptions } from "../api/profilesApi";
import { resolveParameterValues, sanitizeParameterValues } from "./asrParameterValues";

export const fallbackTranscriptionModels: TranscriptionModel[] = [
  { id: "whisper-base", display_name: "Whisper Base", provider: "local", installed: false, default: true, capabilities: { transcription: true, word_timestamps: true } },
  { id: "whisper-small", display_name: "Whisper Small", provider: "local", installed: false, default: false, capabilities: { transcription: true, word_timestamps: true } },
  { id: "parakeet-v2", display_name: "NVIDIA Parakeet TDT v2", provider: "local", installed: false, default: false, capabilities: { transcription: true, word_timestamps: true } },
  { id: "parakeet-v3", display_name: "NVIDIA Parakeet TDT v3", provider: "local", installed: false, default: false, capabilities: { transcription: true, word_timestamps: true } },
];

export function ensureTranscriptionStep(options: TranscriptionProfileOptions, models: TranscriptionModel[]): TranscriptionProfileOptions {
  if (options.pipeline.some((step) => step.kind === "transcription" && step.model)) {
    return options;
  }
  const source = models.length ? models : fallbackTranscriptionModels;
  const model = source.find((item) => item.default) || source[0];
  return withTranscriptionModel(options, source, model.id);
}

export function withTranscriptionModel(options: TranscriptionProfileOptions, models: TranscriptionModel[], modelID: string): TranscriptionProfileOptions {
  const model = models.find((item) => item.id === modelID) || fallbackTranscriptionModels.find((item) => item.id === modelID) || fallbackTranscriptionModels[0];
  const existingStep = options.pipeline.find((step) => step.kind === "transcription");
  const nextStep = {
    ...(existingStep || {}),
    kind: "transcription" as const,
    provider: model.provider,
    model: model.id,
    options: sanitizeParameterValues(model, resolveParameterValues(model, existingStep?.options || {})),
  };
  const otherSteps = options.pipeline.filter((step) => step.kind !== "transcription");
  return { pipeline: [nextStep, ...otherSteps] };
}

export function withDiarizationModel(options: TranscriptionProfileOptions, models: TranscriptionModel[], modelID: string): TranscriptionProfileOptions {
  const model = models.find((item) => item.id === modelID) || models[0];
  if (!model) return options;
  const existingStep = options.pipeline.find((step) => step.kind === "diarization");
  const nextStep = {
    ...(existingStep || {}),
    kind: "diarization" as const,
    provider: model.provider,
    model: model.id,
    options: sanitizeParameterValues(model, resolveParameterValues(model, existingStep?.options || {})),
  };
  const otherSteps = options.pipeline.filter((step) => step.kind !== "diarization");
  return { pipeline: [...otherSteps, nextStep] };
}

export function updateStepOptions(options: TranscriptionProfileOptions, kind: ASRStep["kind"], values: Record<string, unknown>): TranscriptionProfileOptions {
  return {
    pipeline: options.pipeline.map((step) => (step.kind === kind ? { ...step, options: values } : step)),
  };
}

export function prepareProfileOptionsForSave(options: TranscriptionProfileOptions, models: TranscriptionModel[], diarizationModels: TranscriptionModel[]): TranscriptionProfileOptions {
  const withTranscription = ensureTranscriptionStep(options, models);
  return {
    pipeline: withTranscription.pipeline.map((step) => {
      const model = step.kind === "diarization"
        ? diarizationModels.find((item) => item.id === step.model)
        : models.find((item) => item.id === step.model) || fallbackTranscriptionModels.find((item) => item.id === step.model);
      if (!model) return step;
      return { ...step, options: sanitizeParameterValues(model, resolveParameterValues(model, step.options || {})) };
    }),
  };
}
