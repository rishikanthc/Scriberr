import { useEffect, useMemo, useState, type ChangeEvent } from "react";
import { Check, X } from "lucide-react";
import { AppButton, IconButton } from "@/shared/ui/Button";
import { defaultProfileParams, normalizeParams, type ModelFamily, type TranscriptionProfile, type WhisperXParams } from "../api/profilesApi";

type ASRProfileDialogProps = {
  open: boolean;
  profile: TranscriptionProfile | null;
  onClose: () => void;
  onSave: (profile: {
    id?: string;
    name: string;
    description: string;
    is_default: boolean;
    options: WhisperXParams;
  }) => Promise<void>;
};

const modelFamilies: Array<{ value: ModelFamily; label: string }> = [
  { value: "whisper", label: "Whisper" },
  { value: "nvidia_parakeet", label: "NVIDIA Parakeet" },
  { value: "nvidia_canary", label: "NVIDIA Canary" },
  { value: "mistral_voxtral", label: "Mistral Voxtral" },
  { value: "openai", label: "OpenAI" },
];

const languages = [
  ["", "Auto-detect"],
  ["en", "English"],
  ["es", "Spanish"],
  ["fr", "French"],
  ["de", "German"],
  ["it", "Italian"],
  ["pt", "Portuguese"],
  ["nl", "Dutch"],
  ["ja", "Japanese"],
  ["ko", "Korean"],
  ["zh", "Chinese"],
  ["hi", "Hindi"],
  ["ar", "Arabic"],
];

const whisperModels = ["tiny", "tiny.en", "base", "base.en", "small", "small.en", "medium", "medium.en", "large-v2", "large-v3"];

export function ASRProfileDialog({ open, profile, onClose, onSave }: ASRProfileDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [isDefault, setIsDefault] = useState(false);
  const [params, setParams] = useState<WhisperXParams>(defaultProfileParams);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) return;
    setName(profile?.name || "");
    setDescription(profile?.description || "");
    setIsDefault(profile?.is_default || false);
    setParams(normalizeParams(profile?.options));
    setError("");
  }, [open, profile]);

  const title = profile ? "Edit profile" : "New profile";
  const modelOptions = useMemo(() => modelOptionsFor(params.model_family), [params.model_family]);

  if (!open) return null;

  const updateParam = <K extends keyof WhisperXParams>(key: K, value: WhisperXParams[K]) => {
    setParams((current) => {
      const next = { ...current, [key]: value };
      if (key === "model_family") {
        next.model = defaultModelFor(value as ModelFamily);
        next.diarize_model = value === "nvidia_parakeet" ? "nvidia_sortformer" : "pyannote";
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
      <section className="scr-profile-modal" role="dialog" aria-modal="true" aria-labelledby="asr-profile-title">
        <header className="scr-modal-header">
          <div>
            <h2 id="asr-profile-title" className="scr-modal-title">{title}</h2>
            <p className="scr-modal-copy">Save a reusable transcription and diarization setup.</p>
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
              <TextField label="Name" value={name} onChange={setName} placeholder="Podcast cleanup" />
              <TextField label="Description" value={description} onChange={setDescription} placeholder="Accurate speaker labels for long conversations" />
            </div>
            <CheckRow label="Set as default profile" checked={isDefault} onChange={setIsDefault} />
          </section>

          <section className="scr-settings-section">
            <h3 className="scr-settings-section-title">Model</h3>
            <div className="scr-form-grid">
              <SelectField label="Provider" value={params.model_family} options={modelFamilies} onChange={(value) => updateParam("model_family", value as ModelFamily)} />
              <SelectField label="Model" value={params.model} options={modelOptions} onChange={(value) => updateParam("model", value)} />
              <SelectField label="Language" value={params.language || ""} options={languages.map(([value, label]) => ({ value, label }))} onChange={(value) => updateParam("language", value || undefined)} />
              <SelectField label="Task" value={params.task} options={[{ value: "transcribe", label: "Transcribe" }, { value: "translate", label: "Translate to English" }]} onChange={(value) => updateParam("task", value)} />
            </div>
          </section>

          <section className="scr-settings-section">
            <h3 className="scr-settings-section-title">Runtime</h3>
            <div className="scr-form-grid">
              <SelectField label="Device" value={params.device} options={[{ value: "auto", label: "Auto" }, { value: "cpu", label: "CPU" }, { value: "cuda", label: "CUDA" }, { value: "mps", label: "Apple GPU" }]} onChange={(value) => updateParam("device", value)} />
              <SelectField label="Compute" value={params.compute_type} options={[{ value: "float32", label: "Float32" }, { value: "float16", label: "Float16" }, { value: "int8", label: "Int8" }]} onChange={(value) => updateParam("compute_type", value)} />
              <NumberField label="Device index" value={params.device_index} min={0} max={8} onChange={(value) => updateParam("device_index", value)} />
              <NumberField label="Threads" value={params.threads} min={0} max={32} onChange={(value) => updateParam("threads", value)} />
            </div>
            <SliderField label="Batch size" value={params.batch_size} min={1} max={64} step={1} onChange={(value) => updateParam("batch_size", value)} />
            <TextField label="Model directory" value={params.model_dir || ""} onChange={(value) => updateParam("model_dir", value || undefined)} placeholder="Optional local model path" />
            <CheckRow label="Use cached models only" checked={params.model_cache_only} onChange={(value) => updateParam("model_cache_only", value)} />
          </section>

          <section className="scr-settings-section">
            <h3 className="scr-settings-section-title">Diarization</h3>
            <CheckRow label="Identify speakers" checked={params.diarize} onChange={(value) => updateParam("diarize", value)} />
            {params.diarize ? (
              <div className="scr-form-grid">
                <SelectField label="Diarization model" value={params.diarize_model} options={[{ value: "pyannote", label: "Pyannote" }, { value: "nvidia_sortformer", label: "NVIDIA Sortformer" }]} onChange={(value) => updateParam("diarize_model", value)} />
                <TextField label="Hugging Face token" type="password" value={params.hf_token || ""} onChange={(value) => updateParam("hf_token", value || undefined)} placeholder="hf_..." />
                <OptionalNumberField label="Min speakers" value={params.min_speakers} min={1} max={20} onChange={(value) => updateParam("min_speakers", value)} />
                <OptionalNumberField label="Max speakers" value={params.max_speakers} min={1} max={20} onChange={(value) => updateParam("max_speakers", value)} />
              </div>
            ) : null}
            <CheckRow label="Store speaker embeddings" checked={params.speaker_embeddings} onChange={(value) => updateParam("speaker_embeddings", value)} />
          </section>

          <section className="scr-settings-section">
            <h3 className="scr-settings-section-title">Detection and Alignment</h3>
            <div className="scr-form-grid">
              <SelectField label="VAD method" value={params.vad_method} options={[{ value: "pyannote", label: "Pyannote" }, { value: "silero", label: "Silero" }]} onChange={(value) => updateParam("vad_method", value)} />
              <NumberField label="Chunk size" value={params.chunk_size} min={5} max={120} onChange={(value) => updateParam("chunk_size", value)} />
              <TextField label="Alignment model" value={params.align_model || ""} onChange={(value) => updateParam("align_model", value || undefined)} placeholder="Optional Hugging Face model" />
              <SelectField label="Interpolation" value={params.interpolate_method} options={[{ value: "nearest", label: "Nearest" }, { value: "linear", label: "Linear" }, { value: "ignore", label: "Ignore" }]} onChange={(value) => updateParam("interpolate_method", value)} />
            </div>
            <SliderField label="VAD onset" value={params.vad_onset} min={0.1} max={0.9} step={0.01} onChange={(value) => updateParam("vad_onset", value)} />
            <SliderField label="VAD offset" value={params.vad_offset} min={0.1} max={0.9} step={0.01} onChange={(value) => updateParam("vad_offset", value)} />
            <CheckRow label="Skip alignment" checked={params.no_align} onChange={(value) => updateParam("no_align", value)} />
            <CheckRow label="Return character alignments" checked={params.return_char_alignments} onChange={(value) => updateParam("return_char_alignments", value)} />
          </section>

          <section className="scr-settings-section">
            <h3 className="scr-settings-section-title">Decoding</h3>
            <div className="scr-form-grid">
              <NumberField label="Best of" value={params.best_of} min={1} max={10} onChange={(value) => updateParam("best_of", value)} />
              <NumberField label="Beam size" value={params.beam_size} min={1} max={10} onChange={(value) => updateParam("beam_size", value)} />
              <DecimalField label="Patience" value={params.patience} min={0.1} max={5} step={0.1} onChange={(value) => updateParam("patience", value)} />
              <DecimalField label="Length penalty" value={params.length_penalty} min={0.1} max={5} step={0.1} onChange={(value) => updateParam("length_penalty", value)} />
            </div>
            <SliderField label="Temperature" value={params.temperature} min={0} max={1} step={0.05} onChange={(value) => updateParam("temperature", value)} />
            <SliderField label="Fallback increment" value={params.temperature_increment_on_fallback} min={0} max={1} step={0.05} onChange={(value) => updateParam("temperature_increment_on_fallback", value)} />
            <div className="scr-form-grid">
              <DecimalField label="Compression threshold" value={params.compression_ratio_threshold} min={0} max={5} step={0.1} onChange={(value) => updateParam("compression_ratio_threshold", value)} />
              <DecimalField label="Logprob threshold" value={params.logprob_threshold} min={-5} max={1} step={0.1} onChange={(value) => updateParam("logprob_threshold", value)} />
              <DecimalField label="No speech threshold" value={params.no_speech_threshold} min={0} max={1} step={0.05} onChange={(value) => updateParam("no_speech_threshold", value)} />
              <TextField label="Suppress tokens" value={params.suppress_tokens || ""} onChange={(value) => updateParam("suppress_tokens", value || undefined)} placeholder="-1 or token IDs" />
            </div>
            <TextAreaField label="Initial prompt" value={params.initial_prompt || ""} onChange={(value) => updateParam("initial_prompt", value || undefined)} />
            <CheckRow label="Condition on previous text" checked={params.condition_on_previous_text} onChange={(value) => updateParam("condition_on_previous_text", value)} />
            <CheckRow label="Suppress numerals" checked={params.suppress_numerals} onChange={(value) => updateParam("suppress_numerals", value)} />
            <CheckRow label="FP16" checked={params.fp16} onChange={(value) => updateParam("fp16", value)} />
          </section>

          <section className="scr-settings-section">
            <h3 className="scr-settings-section-title">Output and Provider</h3>
            <div className="scr-form-grid">
              <SelectField label="Output format" value={params.output_format} options={[{ value: "all", label: "All" }, { value: "json", label: "JSON" }, { value: "srt", label: "SRT" }, { value: "vtt", label: "VTT" }, { value: "txt", label: "Text" }]} onChange={(value) => updateParam("output_format", value)} />
              <SelectField label="Segment resolution" value={params.segment_resolution} options={[{ value: "sentence", label: "Sentence" }, { value: "chunk", label: "Chunk" }]} onChange={(value) => updateParam("segment_resolution", value)} />
              <OptionalNumberField label="Max line width" value={params.max_line_width} min={10} max={120} onChange={(value) => updateParam("max_line_width", value)} />
              <OptionalNumberField label="Max line count" value={params.max_line_count} min={1} max={6} onChange={(value) => updateParam("max_line_count", value)} />
              <OptionalNumberField label="Max new tokens" value={params.max_new_tokens} min={256} max={32768} onChange={(value) => updateParam("max_new_tokens", value)} />
              <TextField label="Callback URL" value={params.callback_url || ""} onChange={(value) => updateParam("callback_url", value || undefined)} placeholder="https://..." />
              <TextField label="Provider API key" type="password" value={params.api_key || ""} onChange={(value) => updateParam("api_key", value || undefined)} placeholder="Optional" />
            </div>
            <SliderField label="Left context" value={params.attention_context_left} min={64} max={512} step={64} onChange={(value) => updateParam("attention_context_left", value)} />
            <SliderField label="Right context" value={params.attention_context_right} min={64} max={512} step={64} onChange={(value) => updateParam("attention_context_right", value)} />
            <CheckRow label="Highlight words" checked={params.highlight_words} onChange={(value) => updateParam("highlight_words", value)} />
            <CheckRow label="Verbose logs" checked={params.verbose} onChange={(value) => updateParam("verbose", value)} />
            <CheckRow label="Print progress" checked={params.print_progress} onChange={(value) => updateParam("print_progress", value)} />
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

function modelOptionsFor(family: ModelFamily) {
  if (family === "whisper") return whisperModels.map((model) => ({ value: model, label: model }));
  if (family === "openai") return ["whisper-1", "gpt-4o-transcribe", "gpt-4o-mini-transcribe"].map((model) => ({ value: model, label: model }));
  if (family === "nvidia_parakeet") return [{ value: "parakeet-tdt-0.6b-v2", label: "Parakeet TDT 0.6B v2" }];
  if (family === "nvidia_canary") return [{ value: "canary-1b", label: "Canary 1B" }];
  return [{ value: "voxtral-mini-3b", label: "Voxtral Mini 3B" }];
}

function defaultModelFor(family: ModelFamily) {
  return modelOptionsFor(family)[0]?.value || "small";
}

function TextField({ label, value, onChange, placeholder, type = "text" }: { label: string; value: string; onChange: (value: string) => void; placeholder?: string; type?: string }) {
  return (
    <label className="scr-control">
      <span>{label}</span>
      <input className="scr-input" type={type} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

function TextAreaField({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="scr-control scr-control-wide">
      <span>{label}</span>
      <textarea className="scr-textarea" value={value} rows={3} placeholder="Context, vocabulary, spelling preferences..." onChange={(event) => onChange(event.target.value)} />
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
      <input className="scr-input" type="number" min={min} max={max} value={value ?? ""} placeholder="Auto" onChange={(event) => onChange(optionalNumber(event))} />
    </label>
  );
}

function SliderField({ label, value, min, max, step, onChange }: { label: string; value: number; min: number; max: number; step: number; onChange: (value: number) => void }) {
  return (
    <label className="scr-slider-row">
      <span>{label}</span>
      <input className="scr-range" type="range" min={min} max={max} step={step} value={value} onChange={(event) => onChange(Number(event.target.value))} />
      <input className="scr-input scr-number-compact" type="number" min={min} max={max} step={step} value={value} onChange={(event) => onChange(Number(event.target.value))} />
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

function optionalNumber(event: ChangeEvent<HTMLInputElement>) {
  return event.target.value === "" ? undefined : Number(event.target.value);
}
