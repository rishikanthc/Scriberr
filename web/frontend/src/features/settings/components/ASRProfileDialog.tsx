import { useEffect, useMemo, useState } from "react";
import { Check, X } from "lucide-react";
import { AppButton, IconButton } from "@/shared/ui/Button";
import {
  defaultProfileParams,
  familyForModel,
  normalizeParams,
  type TranscriptionModel,
  type TranscriptionProfile,
  type TranscriptionProfileOptions,
} from "../api/profilesApi";

type ASRProfileDialogProps = {
  open: boolean;
  profile: TranscriptionProfile | null;
  models: TranscriptionModel[];
  onClose: () => void;
  onSave: (profile: {
    id?: string;
    name: string;
    description: string;
    is_default: boolean;
    options: TranscriptionProfileOptions;
  }) => Promise<void>;
};

const languageOptions = [
  { value: "", label: "Model default / auto" },
  { value: "en", label: "English" },
  { value: "es", label: "Spanish" },
  { value: "fr", label: "French" },
  { value: "de", label: "German" },
  { value: "it", label: "Italian" },
  { value: "pt", label: "Portuguese" },
  { value: "nl", label: "Dutch" },
  { value: "ja", label: "Japanese" },
  { value: "ko", label: "Korean" },
  { value: "zh", label: "Chinese" },
];

const canaryLanguageOptions = [
  { value: "en", label: "English" },
  { value: "es", label: "Spanish" },
  { value: "de", label: "German" },
  { value: "fr", label: "French" },
];

const fallbackModels: TranscriptionModel[] = [
  { id: "whisper-base", name: "Whisper Base", provider: "local", installed: false, default: true, capabilities: ["transcription", "word_timestamps"] },
  { id: "whisper-small", name: "Whisper Small", provider: "local", installed: false, default: false, capabilities: ["transcription", "word_timestamps"] },
  { id: "parakeet-v2", name: "NVIDIA Parakeet TDT v2", provider: "local", installed: false, default: false, capabilities: ["transcription", "word_timestamps"] },
  { id: "canary-180m", name: "NVIDIA NeMo Canary 180m", provider: "local", installed: false, default: false, capabilities: ["transcription", "word_timestamps"] },
];

export function ASRProfileDialog({ open, profile, models, onClose, onSave }: ASRProfileDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [isDefault, setIsDefault] = useState(false);
  const [params, setParams] = useState<TranscriptionProfileOptions>(defaultProfileParams);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const modelOptions = useMemo(() => {
    const source = models.length ? models : fallbackModels;
    return source.map((model) => ({
      value: model.id,
      label: `${model.name}${model.installed ? "" : " (downloads on use)"}`,
    }));
  }, [models]);

  useEffect(() => {
    if (!open) return;
    const initial = normalizeParams(profile?.options);
    setName(profile?.name || "");
    setDescription(profile?.description || "");
    setIsDefault(profile?.is_default || false);
    setParams(initial);
    setError("");
  }, [open, profile]);

  if (!open) return null;

  const updateParam = <K extends keyof TranscriptionProfileOptions>(key: K, value: TranscriptionProfileOptions[K]) => {
    setParams((current) => {
      const next = { ...current, [key]: value };
      if (key === "model") {
        next.model_family = familyForModel(String(value));
      }
      return next;
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
        options: normalizeParams(params),
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
              <SelectField label="Model" value={params.model} options={modelOptions} onChange={(value) => updateParam("model", value)} />
              <SelectField label="Language" value={params.language || ""} options={languageOptions} onChange={(value) => updateParam("language", value || undefined)} />
              <SelectField label="Task" value={params.task} options={[{ value: "transcribe", label: "Transcribe" }, { value: "translate", label: "Translate to English" }]} onChange={(value) => updateParam("task", value as TranscriptionProfileOptions["task"])} />
              <SelectField label="Decoding" value={params.decoding_method} options={[{ value: "greedy_search", label: "Greedy search" }, { value: "modified_beam_search", label: "Modified beam search" }]} onChange={(value) => updateParam("decoding_method", value as TranscriptionProfileOptions["decoding_method"])} />
              <NumberField label="Threads" value={params.threads} min={0} max={32} onChange={(value) => updateParam("threads", value)} />
              <OptionalNumberField label="Tail paddings" value={params.tail_paddings} min={-1} max={16} onChange={(value) => updateParam("tail_paddings", value)} />
            </div>
          </section>

          {params.model_family === "canary" ? (
            <section className="scr-settings-section">
              <h3 className="scr-settings-section-title">Canary</h3>
              <div className="scr-form-grid">
                <SelectField label="Source language" value={params.canary_source_language} options={canaryLanguageOptions} onChange={(value) => updateParam("canary_source_language", value)} />
                <SelectField label="Target language" value={params.canary_target_language} options={canaryLanguageOptions} onChange={(value) => updateParam("canary_target_language", value)} />
              </div>
              <CheckRow label="Use punctuation and capitalization" checked={params.canary_use_punctuation ?? true} onChange={(value) => updateParam("canary_use_punctuation", value)} />
            </section>
          ) : null}

          <section className="scr-settings-section">
            <h3 className="scr-settings-section-title">Diarization</h3>
            <CheckRow label="Identify speakers" checked={params.diarize} onChange={(value) => updateParam("diarize", value)} />
            {params.diarize ? (
              <>
                <div className="scr-fixed-option">
                  <span>Model</span>
                  <span>Pyannote + 3D-Speaker</span>
                </div>
                <div className="scr-form-grid">
                  <NumberField label="Known speakers" value={params.num_speakers} min={0} max={20} onChange={(value) => updateParam("num_speakers", value)} />
                  <DecimalField label="Clustering threshold" value={params.diarization_threshold} min={0.05} max={1} step={0.05} onChange={(value) => updateParam("diarization_threshold", value)} />
                  <DecimalField label="Min speech duration" value={params.min_duration_on} min={0.05} max={2} step={0.05} onChange={(value) => updateParam("min_duration_on", value)} />
                  <DecimalField label="Min silence duration" value={params.min_duration_off} min={0.05} max={2} step={0.05} onChange={(value) => updateParam("min_duration_off", value)} />
                </div>
              </>
            ) : null}
          </section>
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

function SelectField({ label, value, options, onChange }: { label: string; value: string; options: Array<{ value: string; label: string }>; onChange: (value: string) => void }) {
  return (
    <label className="scr-control">
      <span>{label}</span>
      <select className="scr-select" value={value} onChange={(event) => onChange(event.target.value)}>
        {options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
      </select>
    </label>
  );
}

function NumberField({ label, value, min, max, onChange }: { label: string; value: number; min: number; max: number; onChange: (value: number) => void }) {
  return <DecimalField label={label} value={value} min={min} max={max} step={1} onChange={onChange} />;
}

function DecimalField({ label, value, min, max, step, onChange }: { label: string; value: number; min: number; max: number; step: number; onChange: (value: number) => void }) {
  return (
    <label className="scr-control">
      <span>{label}</span>
      <input className="scr-input" type="number" min={min} max={max} step={step} value={value} onChange={(event) => onChange(Number(event.target.value))} />
    </label>
  );
}

function OptionalNumberField({ label, value, min, max, onChange }: { label: string; value?: number; min: number; max: number; onChange: (value: number | undefined) => void }) {
  return (
    <label className="scr-control">
      <span>{label}</span>
      <input className="scr-input" type="number" min={min} max={max} value={value ?? ""} placeholder="Default" onChange={(event) => onChange(event.target.value === "" ? undefined : Number(event.target.value))} />
    </label>
  );
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
