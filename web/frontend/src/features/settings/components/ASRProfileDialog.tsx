import { useEffect, useMemo, useState } from "react";
import { Check, X } from "lucide-react";
import { AppButton, IconButton } from "@/shared/ui/Button";
import { Select, type SelectOption } from "@/shared/ui/Select";
import {
  type TranscriptionModel,
  type TranscriptionProfile,
  type TranscriptionProfileOptions,
  normalizeProfileOptions,
  type ASRStep,
} from "../api/profilesApi";
import { ASRParameterForm } from "./ASRParameterForm";
import { resolveParameterValues, sanitizeParameterValues } from "./asrParameterValues";

type ASRProfileDialogProps = {
  open: boolean;
  profile: TranscriptionProfile | null;
  models: TranscriptionModel[];
  diarizationModels: TranscriptionModel[];
  onClose: () => void;
  onSave: (profile: {
    id?: string;
    name: string;
    description: string;
    is_default: boolean;
    options: TranscriptionProfileOptions;
  }) => Promise<void>;
};

const fallbackModels: TranscriptionModel[] = [
  { id: "whisper-base", display_name: "Whisper Base", provider: "local", installed: false, default: true, capabilities: { transcription: true, word_timestamps: true } },
  { id: "whisper-small", display_name: "Whisper Small", provider: "local", installed: false, default: false, capabilities: { transcription: true, word_timestamps: true } },
  { id: "parakeet-v2", display_name: "NVIDIA Parakeet TDT v2", provider: "local", installed: false, default: false, capabilities: { transcription: true, word_timestamps: true } },
  { id: "parakeet-v3", display_name: "NVIDIA Parakeet TDT v3", provider: "local", installed: false, default: false, capabilities: { transcription: true, word_timestamps: true } },
];

export function ASRProfileDialog({ open, profile, models, diarizationModels, onClose, onSave }: ASRProfileDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [isDefault, setIsDefault] = useState(false);
  const [options, setOptions] = useState<TranscriptionProfileOptions>({ pipeline: [] });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const availableModels = models.length ? models : fallbackModels;
  const availableDiarizationModels = diarizationModels;

  const modelOptions = useMemo<SelectOption[]>(() => {
    return availableModels.map((model) => ({
      value: model.id,
      label: model.display_name || model.id,
      description: model.installed ? "Installed locally" : "Downloads on use",
    }));
  }, [availableModels]);

  const transcriptionStep = options.pipeline.find((step) => step.kind === "transcription");
  const diarizationStep = options.pipeline.find((step) => step.kind === "diarization");
  const selectedModelID = transcriptionStep?.model || availableModels.find((model) => model.default)?.id || availableModels[0]?.id || "";
  const selectedModel = availableModels.find((model) => model.id === selectedModelID) || null;
  const selectedDiarizationModelID = diarizationStep?.model || availableDiarizationModels.find((model) => model.default)?.id || availableDiarizationModels[0]?.id || "";
  const selectedDiarizationModel = availableDiarizationModels.find((model) => model.id === selectedDiarizationModelID) || null;

  useEffect(() => {
    if (!open) return;
    setName(profile?.name || "");
    setDescription(profile?.description || "");
    setIsDefault(profile?.is_default || false);
    setOptions(ensureTranscriptionStep(normalizeProfileOptions(profile?.options), availableModels));
    setError("");
  }, [availableModels, open, profile]);

  if (!open) return null;

  const updateModel = (modelID: string) => {
    setOptions((current) => withTranscriptionModel(current, availableModels, modelID));
  };

  const updateTranscriptionOptions = (values: Record<string, unknown>) => {
    setOptions((current) => updateStepOptions(current, "transcription", values));
  };

  const updateDiarizationModel = (modelID: string) => {
    setOptions((current) => withDiarizationModel(current, availableDiarizationModels, modelID));
  };

  const updateDiarizationOptions = (values: Record<string, unknown>) => {
    setOptions((current) => updateStepOptions(current, "diarization", values));
  };

  const toggleDiarization = (enabled: boolean) => {
    setOptions((current) => {
      if (!enabled) {
        return { pipeline: current.pipeline.filter((step) => step.kind !== "diarization") };
      }
      return withDiarizationModel(current, availableDiarizationModels, selectedDiarizationModelID);
    });
  };

  const submit = async () => {
    const cleanName = name.trim();
    if (!cleanName) {
      setError("Profile name is required.");
      return;
    }
    setSaving(true);
    setError("");
    try {
      await onSave({
        id: profile?.id,
        name: cleanName,
        description: description.trim(),
        is_default: isDefault,
        options: prepareProfileOptionsForSave(options, availableModels, availableDiarizationModels),
      });
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save profile.");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="scr-modal-backdrop" role="presentation">
      <section className="scr-profile-modal scr-profile-modal-compact" role="dialog" aria-modal="true" aria-labelledby="asr-profile-title">
        <header className="scr-modal-header">
          <div>
            <h2 id="asr-profile-title" className="scr-modal-title">{profile ? "Edit profile" : "New profile"}</h2>
            <p className="scr-modal-copy">Configure the local speech engine request used for transcription.</p>
          </div>
          <IconButton label="Close profile dialog" onClick={onClose}>
            <X size={18} aria-hidden="true" />
          </IconButton>
        </header>

        <div className="scr-modal-body">
          {error ? <div className="scr-alert">{error}</div> : null}

          <section className="scr-settings-section">
            <h3 className="scr-settings-section-title">Profile</h3>
            <div className="scr-form-grid">
              <TextField label="Name" value={name} onChange={setName} placeholder="Accurate local" />
              <TextField label="Description" value={description} onChange={setDescription} placeholder="Local ASR with speaker labels" />
            </div>
            <CheckRow label="Set as default profile" checked={isDefault} onChange={setIsDefault} />
          </section>

          <section className="scr-settings-section">
            <h3 className="scr-settings-section-title">Transcription</h3>
            <div className="scr-form-grid">
              <SelectField label="Model" value={selectedModelID} options={modelOptions} onChange={updateModel} />
            </div>
          </section>

          <ASRParameterForm
            model={selectedModel}
            values={transcriptionStep?.options || {}}
            onChange={updateTranscriptionOptions}
          />

          <section className="scr-settings-section">
            <h3 className="scr-settings-section-title">Diarization</h3>
            <CheckRow label="Identify speakers" checked={Boolean(diarizationStep)} onChange={toggleDiarization} />
            {diarizationStep && availableDiarizationModels.length > 0 ? (
              <div className="scr-form-grid">
                <SelectField label="Model" value={selectedDiarizationModelID} options={diarizationModelOptions(availableDiarizationModels)} onChange={updateDiarizationModel} />
              </div>
            ) : null}
            {diarizationStep && availableDiarizationModels.length === 0 ? <div className="scr-alert">No diarization model card is available.</div> : null}
          </section>

          {diarizationStep ? (
            <ASRParameterForm
              model={selectedDiarizationModel}
              values={diarizationStep.options || {}}
              onChange={updateDiarizationOptions}
            />
          ) : null}
        </div>

        <footer className="scr-modal-footer">
          <AppButton variant="secondary" onClick={onClose}>Cancel</AppButton>
          <AppButton onClick={submit} disabled={saving || !name.trim()}>
            {saving ? "Saving..." : "Save profile"}
          </AppButton>
        </footer>
      </section>
    </div>
  );
}

function TextField({ label, value, onChange, placeholder }: { label: string; value: string; onChange: (value: string) => void; placeholder?: string }) {
  return (
    <label className="scr-control">
      <span>{label}</span>
      <input className="scr-input" value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

function SelectField({ label, value, options, onChange }: { label: string; value: string; options: SelectOption[]; onChange: (value: string) => void }) {
  return <Select label={label} value={value} options={options} onChange={onChange} />;
}

function CheckRow({ label, checked, onChange }: { label: string; checked: boolean; onChange: (checked: boolean) => void }) {
  return (
    <label className="scr-check-row">
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      <span className="scr-check-box" aria-hidden="true">{checked ? <Check size={13} /> : null}</span>
      <span>{label}</span>
    </label>
  );
}

function ensureTranscriptionStep(options: TranscriptionProfileOptions, models: TranscriptionModel[]): TranscriptionProfileOptions {
  if (options.pipeline.some((step) => step.kind === "transcription" && step.model)) {
    return options;
  }
  const model = models.find((item) => item.default) || models[0] || fallbackModels[0];
  return withTranscriptionModel(options, models.length ? models : fallbackModels, model.id);
}

function withTranscriptionModel(options: TranscriptionProfileOptions, models: TranscriptionModel[], modelID: string): TranscriptionProfileOptions {
  const model = models.find((item) => item.id === modelID) || fallbackModels.find((item) => item.id === modelID) || fallbackModels[0];
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

function withDiarizationModel(options: TranscriptionProfileOptions, models: TranscriptionModel[], modelID: string): TranscriptionProfileOptions {
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

function updateStepOptions(options: TranscriptionProfileOptions, kind: ASRStep["kind"], values: Record<string, unknown>): TranscriptionProfileOptions {
  return {
    pipeline: options.pipeline.map((step) => (step.kind === kind ? { ...step, options: values } : step)),
  };
}

function prepareProfileOptionsForSave(options: TranscriptionProfileOptions, models: TranscriptionModel[], diarizationModels: TranscriptionModel[]): TranscriptionProfileOptions {
  const withTranscription = ensureTranscriptionStep(options, models);
  return {
    pipeline: withTranscription.pipeline.map((step) => {
      const model = step.kind === "diarization"
        ? diarizationModels.find((item) => item.id === step.model)
        : models.find((item) => item.id === step.model) || fallbackModels.find((item) => item.id === step.model);
      if (!model) return step;
      return { ...step, options: sanitizeParameterValues(model, resolveParameterValues(model, step.options || {})) };
    }),
  };
}

function diarizationModelOptions(models: TranscriptionModel[]): SelectOption[] {
  return models.map((model) => ({
    value: model.id,
    label: model.display_name || model.id,
    description: model.installed ? "Installed locally" : "Downloads on use",
  }));
}
