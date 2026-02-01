import { useState, useEffect, memo } from "react";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Loader2, Check, XCircle } from "lucide-react";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { FormField, Section, InfoBanner } from "@/components/transcription/FormHelpers";

// ============================================================================
// Types & Constants
// ============================================================================

export interface TranscriptionParams {
    model_family: string;
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
    target_language?: string;
    pnc?: boolean;
    align_model?: string;
    interpolate_method: string;
    no_align: boolean;
    return_char_alignments: boolean;
    vad_method: string;
    vad_onset: number;
    vad_offset: number;
    chunk_size: number;
    vad_preset: string;
    vad_speech_pad_ms?: number;
    vad_min_silence_ms?: number;
    vad_min_speech_ms?: number;
    vad_max_speech_s?: number;
    diarize: boolean;
    min_speakers?: number;
    max_speakers?: number;
    diarize_model: string;
    speaker_embeddings: boolean;
    diarization_perf_preset: string;
    segmentation_batch_size?: number;
    embedding_batch_size?: number;
    embedding_exclude_overlap?: boolean;
    torch_threads?: number;
    torch_interop_threads?: number;
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
    chunk_len_s: number;
    chunk_batch_size: number;
    segment_gap_s?: number;
    is_multi_track_enabled: boolean;
    api_key?: string;
}

interface TranscriptionConfigDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onStartTranscription: (params: TranscriptionParams & { profileName?: string; profileDescription?: string }) => void;
    loading?: boolean;
    isProfileMode?: boolean;
    initialParams?: TranscriptionParams;
    initialName?: string;
    initialDescription?: string;
    isMultiTrack?: boolean;
    title?: string;
}

const DEFAULT_PARAMS: TranscriptionParams = {
    model_family: "whisper",
    model: "onnx-community/whisper-small",
    model_cache_only: false,
    device: "cpu",
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
    vad_preset: "balanced",
    diarize: false,
    diarize_model: "pyannote",
    speaker_embeddings: false,
    diarization_perf_preset: "auto",
    temperature: 0,
    best_of: 5,
    beam_size: 5,
    patience: 1.0,
    length_penalty: 1.0,
    suppress_numerals: false,
    condition_on_previous_text: false,
    fp16: true,
    temperature_increment_on_fallback: 0.2,
    compression_ratio_threshold: 2.4,
    logprob_threshold: -1.0,
    no_speech_threshold: 0.6,
    highlight_words: false,
    segment_resolution: "sentence",
    print_progress: false,
    attention_context_left: 256,
    attention_context_right: 256,
    chunk_len_s: 300,
    chunk_batch_size: 8,
    segment_gap_s: undefined,
    is_multi_track_enabled: false,
    api_key: "",
    target_language: "en",
    pnc: true,
};

const WHISPER_MODELS = [
    "onnx-community/whisper-tiny",
    "onnx-community/whisper-base",
    "onnx-community/whisper-small",
    "onnx-community/whisper-medium",
    "onnx-community/whisper-large-v2",
    "onnx-community/whisper-large-v3",
    "onnx-community/whisper-large-v3-turbo",
    "whisper-base-ort",
    "whisper-ort",
    "whisper-base"
];

const PARAKEET_MODELS = [
    "nemo-parakeet-tdt-0.6b-v2",
    "nemo-parakeet-tdt-0.6b-v3"
];

const CANARY_MODELS = [
    "nemo-canary-1b-v2"
];

const LANGUAGES = [
    { value: "auto", label: "Auto-detect" },
    { value: "af", label: "Afrikaans" },
    { value: "ar", label: "Arabic" },
    { value: "hy", label: "Armenian" },
    { value: "az", label: "Azerbaijani" },
    { value: "be", label: "Belarusian" },
    { value: "bs", label: "Bosnian" },
    { value: "bg", label: "Bulgarian" },
    { value: "ca", label: "Catalan" },
    { value: "zh", label: "Chinese" },
    { value: "hr", label: "Croatian" },
    { value: "cs", label: "Czech" },
    { value: "da", label: "Danish" },
    { value: "nl", label: "Dutch" },
    { value: "en", label: "English" },
    { value: "et", label: "Estonian" },
    { value: "fi", label: "Finnish" },
    { value: "fr", label: "French" },
    { value: "gl", label: "Galician" },
    { value: "de", label: "German" },
    { value: "el", label: "Greek" },
    { value: "he", label: "Hebrew" },
    { value: "hi", label: "Hindi" },
    { value: "hu", label: "Hungarian" },
    { value: "is", label: "Icelandic" },
    { value: "id", label: "Indonesian" },
    { value: "it", label: "Italian" },
    { value: "ja", label: "Japanese" },
    { value: "kn", label: "Kannada" },
    { value: "kk", label: "Kazakh" },
    { value: "ko", label: "Korean" },
    { value: "lv", label: "Latvian" },
    { value: "lt", label: "Lithuanian" },
    { value: "mk", label: "Macedonian" },
    { value: "ms", label: "Malay" },
    { value: "mr", label: "Marathi" },
    { value: "mi", label: "Maori" },
    { value: "ne", label: "Nepali" },
    { value: "no", label: "Norwegian" },
    { value: "fa", label: "Persian" },
    { value: "pl", label: "Polish" },
    { value: "pt", label: "Portuguese" },
    { value: "ro", label: "Romanian" },
    { value: "ru", label: "Russian" },
    { value: "sr", label: "Serbian" },
    { value: "sk", label: "Slovak" },
    { value: "sl", label: "Slovenian" },
    { value: "es", label: "Spanish" },
    { value: "sw", label: "Swahili" },
    { value: "sv", label: "Swedish" },
    { value: "tl", label: "Tagalog" },
    { value: "ta", label: "Tamil" },
    { value: "th", label: "Thai" },
    { value: "tr", label: "Turkish" },
    { value: "uk", label: "Ukrainian" },
    { value: "ur", label: "Urdu" },
    { value: "vi", label: "Vietnamese" },
    { value: "cy", label: "Welsh" },
];

const CANARY_LANGUAGES = [
    { value: "en", label: "English" },
    { value: "de", label: "German" },
    { value: "es", label: "Spanish" },
    { value: "fr", label: "French" },
];

const PARAM_DESCRIPTIONS = {
    model: "ONNX Whisper model name. Larger = more accurate but slower.",
    language: "Source language. Auto-detect works for most cases.",
    task: "Transcribe in original language or translate to English.",
    device: "CPU (universal), GPU (faster, CUDA required), or AUTO.",
    compute_type: "Float16 (faster), Float32 (accurate), Int8 (fastest).",
    batch_size: "Segments processed at once. Higher = faster but more memory.",
    diarize: "Identify and separate different speakers.",
    diarize_model: "Pyannote (accurate, needs HF token) or NVIDIA Sortformer (up to 4 speakers).",
    temperature: "0 = deterministic, higher = more creative.",
    beam_size: "Search beams. Higher = better quality but slower.",
    vad_method: "Voice detection: Pyannote (accurate) or Silero (fast).",
    initial_prompt: "Context text to guide transcription style.",
    hf_token: "Required for Pyannote diarization models.",
    vad_onset: "Voice detection sensitivity. Lower values (0.3-0.4) catch quieter/distant speakers.",
    vad_offset: "Speech ending sensitivity. Lower values detect speech endings more precisely.",
    vad_preset: "Preset for voice activity detection (segmenting long audio).",
    chunk_len_s: "Chunk length in seconds for long audio (onnx-asr).",
    chunk_batch_size: "Batch size for chunked transcription (onnx-asr).",
    segment_gap_s: "Optional pause threshold (seconds) to split segments (NeMo-style).",
    segmentation_batch_size: "Segmentation batch size (CPU tuning). Higher can be faster but uses more memory.",
    embedding_batch_size: "Embedding batch size (CPU tuning). Higher can be faster but uses more memory.",
    embedding_exclude_overlap: "Skip overlap regions when computing embeddings (faster on CPU).",
    torch_threads: "PyTorch intra-op threads (CPU tuning).",
    torch_interop_threads: "PyTorch inter-op threads (CPU tuning).",
    diarization_perf_preset: "Auto-tune diarization performance based on your CPU/RAM.",
    pnc: "Output punctuation and capitalization (Canary only).",
};

// ============================================================================
// Styled Input/Select Components 
// ============================================================================

const inputClassName = `
  h-11 bg-[var(--bg-main)] border border-[var(--border-subtle)] rounded-xl
  text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)]
  focus:border-[var(--brand-solid)] focus:ring-2 focus:ring-[var(--brand-solid)]/20
  transition-all duration-200
  [color-scheme:light] dark:[color-scheme:dark]
`;

const selectTriggerClassName = `
  h-11 bg-[var(--bg-main)] border border-[var(--border-subtle)] rounded-xl
  text-[var(--text-primary)] shadow-none
  focus:border-[var(--brand-solid)] focus:ring-2 focus:ring-[var(--brand-solid)]/20
`;

const selectContentClassName = `
  bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-xl
`;

const selectItemClassName = `
  text-[var(--text-primary)] rounded-lg mx-1 cursor-pointer
  focus:bg-[var(--brand-light)] focus:text-[var(--brand-solid)]
`;

// ============================================================================
// Main Component
// ============================================================================

export const TranscriptionConfigDialog = memo(function TranscriptionConfigDialog({
    open,
    onOpenChange,
    onStartTranscription,
    loading = false,
    isProfileMode = false,
    initialParams,
    initialName = "",
    initialDescription = "",
    isMultiTrack = false,
    title,
}: TranscriptionConfigDialogProps) {
    const [params, setParams] = useState<TranscriptionParams>(DEFAULT_PARAMS);
    const [profileName, setProfileName] = useState("");
    const [profileDescription, setProfileDescription] = useState("");

    // OpenAI validation state
    const [isValidating, setIsValidating] = useState(false);
    const [validationStatus, setValidationStatus] = useState<'idle' | 'valid' | 'invalid'>('idle');
    const [validationMessage, setValidationMessage] = useState("");
    const { getAuthHeaders } = useAuth();
    const [availableModels, setAvailableModels] = useState<string[]>(["whisper-1"]);

    // Reset when dialog opens
    useEffect(() => {
        if (open) {
            const baseParams = { ...DEFAULT_PARAMS, ...(initialParams || {}) };
            setParams({
                ...baseParams,
                is_multi_track_enabled: isMultiTrack,
                diarize: isMultiTrack ? false : baseParams.diarize
            });
            setProfileName(initialName);
            setProfileDescription(initialDescription);
        }
    }, [open, initialParams, initialName, initialDescription, isMultiTrack]);

    const updateParam = <K extends keyof TranscriptionParams>(key: K, value: TranscriptionParams[K]) => {
        setParams(prev => {
            const newParams = { ...prev, [key]: value };
            if (key === 'model_family') {
                if (value === 'whisper') {
                    newParams.diarize_model = 'pyannote';
                    newParams.model = DEFAULT_PARAMS.model;
                } else if (value === 'nvidia_parakeet') {
                    newParams.model = "nemo-parakeet-tdt-0.6b-v3";
                } else if (value === 'nvidia_canary') {
                    newParams.model = "nemo-canary-1b-v2";
                }
            }
            const perfKeys: Array<keyof TranscriptionParams> = [
                'segmentation_batch_size',
                'embedding_batch_size',
                'embedding_exclude_overlap',
                'torch_threads',
                'torch_interop_threads',
            ];
            if (perfKeys.includes(key)) {
                newParams.diarization_perf_preset = 'custom';
            }
            if (key === 'diarization_perf_preset' && value !== 'custom') {
                newParams.segmentation_batch_size = undefined;
                newParams.embedding_batch_size = undefined;
                newParams.embedding_exclude_overlap = undefined;
                newParams.torch_threads = undefined;
                newParams.torch_interop_threads = undefined;
            }
            return newParams;
        });
    };

    const validateAPIKey = async () => {
        setIsValidating(true);
        setValidationStatus('idle');
        try {
            const response = await fetch('/api/v1/config/openai/validate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
                body: JSON.stringify({ api_key: params.api_key }),
            });
            const data = await response.json();
            if (response.ok && data.valid) {
                setValidationStatus('valid');
                setAvailableModels(data.models || ["whisper-1"]);
                setValidationMessage("API key validated");
            } else {
                setValidationStatus('invalid');
                setValidationMessage(data.error || "Invalid API key");
            }
        } catch {
            setValidationStatus('invalid');
            setValidationMessage("Validation failed");
        } finally {
            setIsValidating(false);
        }
    };

    const handleSubmit = () => {
        if (isProfileMode) {
            onStartTranscription({ ...params, profileName, profileDescription });
        } else {
            onStartTranscription(params);
        }
    };

    const dialogTitle = title || (isProfileMode
        ? (initialName ? `Edit "${initialName}"` : "New Transcription Profile")
        : "Transcription Settings"
    );

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent
                className="max-w-full sm:max-w-2xl w-[calc(100vw-1rem)] max-h-[90vh] overflow-hidden flex flex-col p-0 gap-0 bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-2xl"
                style={{ boxShadow: 'var(--shadow-float)' }}
            >
                {/* Header */}
                <DialogHeader className="px-6 pt-6 pb-4 border-b border-[var(--border-subtle)]">
                    <DialogTitle className="text-xl font-semibold text-[var(--text-primary)]">
                        {dialogTitle}
                    </DialogTitle>
                    <DialogDescription className="text-[var(--text-secondary)] text-sm mt-1">
                        {isProfileMode
                            ? "Configure and save your transcription settings."
                            : "Choose a model and configure transcription parameters."
                        }
                    </DialogDescription>
                </DialogHeader>

                {/* Scrollable Content */}
                <div className="flex-1 overflow-y-auto px-6 py-6 space-y-6">

                    {/* Profile Name/Description (if profile mode) */}
                    {isProfileMode && (
                        <div className="p-4 bg-[var(--bg-main)] rounded-xl border border-[var(--border-subtle)] space-y-4">
                            <FormField label="Profile Name" htmlFor="profileName">
                                <Input
                                    id="profileName"
                                    value={profileName}
                                    onChange={(e) => setProfileName(e.target.value)}
                                    placeholder="My transcription profile"
                                    className={inputClassName}
                                    required
                                />
                            </FormField>
                            <FormField label="Description" htmlFor="profileDesc" optional>
                                <Textarea
                                    id="profileDesc"
                                    value={profileDescription}
                                    onChange={(e) => setProfileDescription(e.target.value)}
                                    placeholder="Describe this profile..."
                                    className={`${inputClassName} resize-none min-h-[80px]`}
                                    rows={2}
                                />
                            </FormField>
                        </div>
                    )}

                    {/* Model Family Selection */}
                    <FormField
                        label="Model Family"
                        description="Choose the AI model for transcription. Each has different capabilities and requirements."
                    >
                        <Select
                            value={params.model_family}
                            onValueChange={(v) => updateParam('model_family', v)}
                        >
                            <SelectTrigger className={selectTriggerClassName}>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className={selectContentClassName}>
                                <SelectItem value="whisper" className={selectItemClassName}>
                                    Whisper
                                </SelectItem>
                                <SelectItem value="nvidia_parakeet" className={selectItemClassName}>
                                    NVIDIA Parakeet
                                </SelectItem>
                                <SelectItem value="nvidia_canary" className={selectItemClassName}>
                                    NVIDIA Canary
                                </SelectItem>
                                <SelectItem value="openai" className={selectItemClassName}>
                                    OpenAI
                                </SelectItem>
                            </SelectContent>
                        </Select>
                    </FormField>

                    {/* Multi-track notice */}
                    {isMultiTrack && (
                        <InfoBanner variant="info" title="Multi-track Audio Detected">
                            Each audio track will be transcribed separately. Speaker diarization is disabled.
                        </InfoBanner>
                    )}

                    {/* Model-Specific Configuration */}
                    {params.model_family === "whisper" && (
                        <WhisperConfig
                            params={params}
                            updateParam={updateParam}
                            isMultiTrack={isMultiTrack}
                        />
                    )}

                    {params.model_family === "nvidia_parakeet" && (
                        <ParakeetConfig
                            params={params}
                            updateParam={updateParam}
                            isMultiTrack={isMultiTrack}
                        />
                    )}

                    {params.model_family === "nvidia_canary" && (
                        <CanaryConfig
                            params={params}
                            updateParam={updateParam}
                            isMultiTrack={isMultiTrack}
                        />
                    )}

                    {params.model_family === "openai" && (
                        <OpenAIConfig
                            params={params}
                            updateParam={updateParam}
                            isValidating={isValidating}
                            validationStatus={validationStatus}
                            validationMessage={validationMessage}
                            availableModels={availableModels}
                            onValidate={validateAPIKey}
                        />
                    )}

                </div>

                {/* Footer */}
                <DialogFooter className="px-6 py-4 border-t border-[var(--border-subtle)] gap-3 sm:gap-2">
                    <Button
                        variant="ghost"
                        onClick={() => onOpenChange(false)}
                        className="rounded-xl text-[var(--text-secondary)] hover:bg-[var(--bg-main)] cursor-pointer"
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleSubmit}
                        disabled={loading || (isProfileMode && !profileName.trim())}
                        className="rounded-xl text-white cursor-pointer bg-gradient-to-r from-[#FFAB40] to-[#FF3D00] hover:opacity-90 active:scale-[0.98] transition-all shadow-lg shadow-orange-500/20"
                    >
                        {loading ? (
                            <>
                                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                Starting...
                            </>
                        ) : (
                            isProfileMode ? "Save Profile" : "Start Transcription"
                        )}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
});

// ============================================================================
// Model-Specific Configuration Components
// ============================================================================

interface ConfigProps {
    params: TranscriptionParams;
    updateParam: <K extends keyof TranscriptionParams>(key: K, value: TranscriptionParams[K]) => void;
    isMultiTrack?: boolean;
}

function WhisperConfig({ params, updateParam, isMultiTrack }: ConfigProps) {
    return (
        <div className="space-y-6">
            {/* Essential Settings */}
            <Section title="Model Settings">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <FormField label="Model" description={PARAM_DESCRIPTIONS.model}>
                        <Select value={params.model} onValueChange={(v) => updateParam('model', v)}>
                            <SelectTrigger className={selectTriggerClassName}>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className={selectContentClassName}>
                                {WHISPER_MODELS.map((m) => (
                                    <SelectItem key={m} value={m} className={selectItemClassName}>{m}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </FormField>

                    <FormField label="Language" description={PARAM_DESCRIPTIONS.language}>
                        <Select value={params.language || "auto"} onValueChange={(v) => updateParam('language', v === "auto" ? undefined : v)}>
                            <SelectTrigger className={selectTriggerClassName}>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className={selectContentClassName}>
                                {LANGUAGES.map((l) => (
                                    <SelectItem key={l.value} value={l.value} className={selectItemClassName}>{l.label}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </FormField>
                </div>
            </Section>

            <Section title="ASR VAD" description="Control how audio is segmented before recognition">
                <div className="space-y-4">
                    <FormField label="VAD Preset" description={PARAM_DESCRIPTIONS.vad_preset}>
                        <Select value={params.vad_preset} onValueChange={(v) => updateParam('vad_preset', v)}>
                            <SelectTrigger className={selectTriggerClassName}>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className={selectContentClassName}>
                                <SelectItem value="conservative" className={selectItemClassName}>Conservative</SelectItem>
                                <SelectItem value="balanced" className={selectItemClassName}>Balanced</SelectItem>
                                <SelectItem value="aggressive" className={selectItemClassName}>Aggressive</SelectItem>
                            </SelectContent>
                        </Select>
                    </FormField>

                    <div className="grid grid-cols-2 gap-4">
                        <FormField label="Speech Pad (ms)" optional>
                            <Input
                                type="number"
                                min={0}
                                placeholder="Default"
                                value={params.vad_speech_pad_ms ?? ""}
                                onChange={(e) => updateParam('vad_speech_pad_ms', e.target.value ? parseInt(e.target.value) : undefined)}
                                className={inputClassName}
                            />
                        </FormField>
                        <FormField label="Min Silence (ms)" optional>
                            <Input
                                type="number"
                                min={0}
                                placeholder="Default"
                                value={params.vad_min_silence_ms ?? ""}
                                onChange={(e) => updateParam('vad_min_silence_ms', e.target.value ? parseInt(e.target.value) : undefined)}
                                className={inputClassName}
                            />
                        </FormField>
                        <FormField label="Min Speech (ms)" optional>
                            <Input
                                type="number"
                                min={0}
                                placeholder="Default"
                                value={params.vad_min_speech_ms ?? ""}
                                onChange={(e) => updateParam('vad_min_speech_ms', e.target.value ? parseInt(e.target.value) : undefined)}
                                className={inputClassName}
                            />
                        </FormField>
                        <FormField label="Max Speech (s)" optional>
                            <Input
                                type="number"
                                min={1}
                                placeholder="Default"
                                value={params.vad_max_speech_s ?? ""}
                                onChange={(e) => updateParam('vad_max_speech_s', e.target.value ? parseInt(e.target.value) : undefined)}
                                className={inputClassName}
                            />
                        </FormField>
                    </div>
                </div>
            </Section>

            {/* Speaker Diarization */}
            {!isMultiTrack && (
                <Section title="Speaker Diarization" description="Identify and separate different speakers in the audio">
                    <div className="space-y-4">
                        <div className="flex items-center gap-3">
                            <Switch
                                id="diarize"
                                checked={params.diarize}
                                onCheckedChange={(v) => updateParam('diarize', v)}
                            />
                            <label htmlFor="diarize" className="text-sm text-[var(--text-primary)] cursor-pointer">
                                Enable speaker identification
                            </label>
                        </div>

                        {params.diarize && (
                            <div className="p-4 bg-[var(--bg-main)] rounded-xl border border-[var(--border-subtle)] space-y-4">
                                <FormField label="Diarization Model">
                                    <Select value={params.diarize_model} onValueChange={(v) => updateParam('diarize_model', v)}>
                                        <SelectTrigger className={selectTriggerClassName}>
                                            <SelectValue />
                                        </SelectTrigger>
                                        <SelectContent className={selectContentClassName}>
                                            <SelectItem value="pyannote" className={selectItemClassName}>Pyannote</SelectItem>
                                            <SelectItem value="nvidia_sortformer" className={selectItemClassName}>NVIDIA Sortformer</SelectItem>
                                        </SelectContent>
                                    </Select>
                                </FormField>

                                <div className="grid grid-cols-2 gap-4">
                                    <FormField label="Min Speakers" optional>
                                        <Input
                                            type="number"
                                            min={1}
                                            max={20}
                                            placeholder="Auto"
                                            value={params.min_speakers || ""}
                                            onChange={(e) => updateParam('min_speakers', e.target.value ? parseInt(e.target.value) : undefined)}
                                            className={inputClassName}
                                        />
                                    </FormField>
                                    <FormField label="Max Speakers" optional>
                                        <Input
                                            type="number"
                                            min={1}
                                            max={20}
                                            placeholder="Auto"
                                            value={params.max_speakers || ""}
                                            onChange={(e) => updateParam('max_speakers', e.target.value ? parseInt(e.target.value) : undefined)}
                                            className={inputClassName}
                                        />
                                    </FormField>
                                </div>

                                {params.diarize_model === "pyannote" && (
                                    <>
                                        <FormField label="Hugging Face Token" description={PARAM_DESCRIPTIONS.hf_token}>
                                            <Input
                                                type="password"
                                                placeholder="hf_..."
                                                value={params.hf_token || ""}
                                                onChange={(e) => updateParam('hf_token', e.target.value || undefined)}
                                                className={inputClassName}
                                            />
                                        </FormField>

                                        <div className="pt-3 border-t border-[var(--border-subtle)]">
                                            <p className="text-xs text-[var(--text-tertiary)] mb-3">Voice Detection Tuning (for noisy/distant audio)</p>
                                            <div className="grid grid-cols-2 gap-4">
                                                <FormField label="VAD Onset" description={PARAM_DESCRIPTIONS.vad_onset}>
                                                    <Input
                                                        type="number"
                                                        min={0.1}
                                                        max={0.9}
                                                        step={0.05}
                                                        value={params.vad_onset}
                                                        onChange={(e) => updateParam('vad_onset', parseFloat(e.target.value) || 0.5)}
                                                        className={inputClassName}
                                                    />
                                                </FormField>
                                                <FormField label="VAD Offset" description={PARAM_DESCRIPTIONS.vad_offset}>
                                                    <Input
                                                        type="number"
                                                        min={0.1}
                                                        max={0.9}
                                                        step={0.05}
                                                        value={params.vad_offset}
                                                        onChange={(e) => updateParam('vad_offset', parseFloat(e.target.value) || 0.363)}
                                                        className={inputClassName}
                                                    />
                                                </FormField>
                                            </div>
                                        </div>

                                        <div className="pt-3 border-t border-[var(--border-subtle)]">
                                            <p className="text-xs text-[var(--text-tertiary)] mb-3">CPU Performance Tuning</p>
                                            <FormField label="Performance Preset" description={PARAM_DESCRIPTIONS.diarization_perf_preset}>
                                                <Select
                                                    value={params.diarization_perf_preset || "auto"}
                                                    onValueChange={(v) => updateParam('diarization_perf_preset', v)}
                                                >
                                                    <SelectTrigger className={selectTriggerClassName}>
                                                        <SelectValue />
                                                    </SelectTrigger>
                                                    <SelectContent className={selectContentClassName}>
                                                        <SelectItem value="auto" className={selectItemClassName}>Auto (recommended)</SelectItem>
                                                        <SelectItem value="low" className={selectItemClassName}>Low</SelectItem>
                                                        <SelectItem value="medium" className={selectItemClassName}>Medium</SelectItem>
                                                        <SelectItem value="high" className={selectItemClassName}>High</SelectItem>
                                                        <SelectItem value="custom" className={selectItemClassName}>Custom</SelectItem>
                                                    </SelectContent>
                                                </Select>
                                            </FormField>

                                            {params.diarization_perf_preset === "custom" && (
                                                <>
                                                    <div className="grid grid-cols-2 gap-4">
                                                        <FormField label="Segmentation Batch" description={PARAM_DESCRIPTIONS.segmentation_batch_size}>
                                                            <Input
                                                                type="number"
                                                                min={1}
                                                                max={32}
                                                                value={params.segmentation_batch_size ?? ""}
                                                                onChange={(e) => updateParam('segmentation_batch_size', e.target.value ? parseInt(e.target.value) : undefined)}
                                                                className={inputClassName}
                                                            />
                                                        </FormField>
                                                        <FormField label="Embedding Batch" description={PARAM_DESCRIPTIONS.embedding_batch_size}>
                                                            <Input
                                                                type="number"
                                                                min={1}
                                                                max={32}
                                                                value={params.embedding_batch_size ?? ""}
                                                                onChange={(e) => updateParam('embedding_batch_size', e.target.value ? parseInt(e.target.value) : undefined)}
                                                                className={inputClassName}
                                                            />
                                                        </FormField>
                                                        <FormField label="Torch Threads" description={PARAM_DESCRIPTIONS.torch_threads}>
                                                            <Input
                                                                type="number"
                                                                min={1}
                                                                max={64}
                                                                value={params.torch_threads ?? ""}
                                                                onChange={(e) => updateParam('torch_threads', e.target.value ? parseInt(e.target.value) : undefined)}
                                                                className={inputClassName}
                                                            />
                                                        </FormField>
                                                        <FormField label="Torch Interop Threads" description={PARAM_DESCRIPTIONS.torch_interop_threads}>
                                                            <Input
                                                                type="number"
                                                                min={1}
                                                                max={8}
                                                                value={params.torch_interop_threads ?? ""}
                                                                onChange={(e) => updateParam('torch_interop_threads', e.target.value ? parseInt(e.target.value) : undefined)}
                                                                className={inputClassName}
                                                            />
                                                        </FormField>
                                                    </div>

                                                    <div className="pt-3">
                                                        <FormField label="Exclude Overlap in Embeddings" description={PARAM_DESCRIPTIONS.embedding_exclude_overlap}>
                                                            <div className="flex items-center gap-3">
                                                                <Switch
                                                                    checked={params.embedding_exclude_overlap ?? true}
                                                                    onCheckedChange={(value) => updateParam('embedding_exclude_overlap', value)}
                                                                />
                                                                <span className="text-sm text-[var(--text-primary)]">Faster on CPU</span>
                                                            </div>
                                                        </FormField>
                                                    </div>
                                                </>
                                            )}
                                        </div>
                                    </>
                                )}
                            </div>
                        )}
                    </div>
                </Section>
            )}
        </div>
    );
}

function ParakeetConfig({ params, updateParam, isMultiTrack }: ConfigProps) {
    return (
        <div className="space-y-6">
            <Section title="Model Selection" description="Choose the Parakeet model variant">
                <FormField label="Model">
                    <Select value={params.model} onValueChange={(v) => updateParam('model', v)}>
                        <SelectTrigger className={selectTriggerClassName}>
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent className={selectContentClassName}>
                            {PARAKEET_MODELS.map((model) => (
                                <SelectItem key={model} value={model} className={selectItemClassName}>{model}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </FormField>
            </Section>

            <Section title="Chunking" description="Control how long audio is split before recognition">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
                    <FormField label="Chunk Length (seconds)" description={PARAM_DESCRIPTIONS.chunk_len_s}>
                        <Input
                            type="number"
                            min={10}
                            step={10}
                            value={params.chunk_len_s}
                            onChange={(e) => updateParam('chunk_len_s', parseFloat(e.target.value) || 300)}
                            className={inputClassName}
                        />
                    </FormField>
                    <FormField label="Chunk Batch Size" description={PARAM_DESCRIPTIONS.chunk_batch_size}>
                        <Input
                            type="number"
                            min={1}
                            step={1}
                            value={params.chunk_batch_size}
                            onChange={(e) => updateParam('chunk_batch_size', parseInt(e.target.value) || 8)}
                            className={inputClassName}
                        />
                    </FormField>
                    <FormField label="Segment Gap (seconds)" optional description={PARAM_DESCRIPTIONS.segment_gap_s}>
                        <Input
                            type="number"
                            min={0}
                            step={0.1}
                            placeholder="None"
                            value={params.segment_gap_s ?? ""}
                            onChange={(e) => updateParam('segment_gap_s', e.target.value ? parseFloat(e.target.value) : undefined)}
                            className={inputClassName}
                        />
                    </FormField>
                </div>
            </Section>

            {/* Diarization for Parakeet */}
            {!isMultiTrack && (
                <Section title="Speaker Diarization">
                    <div className="space-y-4">
                        <div className="flex items-center gap-3">
                            <Switch
                                id="parakeet_diarize"
                                checked={params.diarize}
                                onCheckedChange={(v) => updateParam('diarize', v)}
                            />
                            <label htmlFor="parakeet_diarize" className="text-sm text-[var(--text-primary)] cursor-pointer">
                                Enable speaker identification
                            </label>
                        </div>

                        {params.diarize && (
                            <div className="p-4 bg-[var(--bg-main)] rounded-xl border border-[var(--border-subtle)] space-y-4">
                                <FormField label="Diarization Model">
                                    <Select value={params.diarize_model} onValueChange={(v) => updateParam('diarize_model', v)}>
                                        <SelectTrigger className={selectTriggerClassName}>
                                            <SelectValue />
                                        </SelectTrigger>
                                        <SelectContent className={selectContentClassName}>
                                            <SelectItem value="pyannote" className={selectItemClassName}>Pyannote</SelectItem>
                                            <SelectItem value="nvidia_sortformer" className={selectItemClassName}>NVIDIA Sortformer</SelectItem>
                                        </SelectContent>
                                    </Select>
                                </FormField>

                                <div className="grid grid-cols-2 gap-4">
                                    <FormField label="Min Speakers" optional>
                                        <Input
                                            type="number"
                                            min={1}
                                            max={20}
                                            placeholder="Auto"
                                            value={params.min_speakers || ""}
                                            onChange={(e) => updateParam('min_speakers', e.target.value ? parseInt(e.target.value) : undefined)}
                                            className={inputClassName}
                                        />
                                    </FormField>
                                    <FormField label="Max Speakers" optional>
                                        <Input
                                            type="number"
                                            min={1}
                                            max={20}
                                            placeholder="Auto"
                                            value={params.max_speakers || ""}
                                            onChange={(e) => updateParam('max_speakers', e.target.value ? parseInt(e.target.value) : undefined)}
                                            className={inputClassName}
                                        />
                                    </FormField>
                                </div>

                                {params.diarize_model === "pyannote" && (
                                    <>
                                        <FormField label="Hugging Face Token">
                                            <Input
                                                type="password"
                                                placeholder="hf_..."
                                                value={params.hf_token || ""}
                                                onChange={(e) => updateParam('hf_token', e.target.value || undefined)}
                                                className={inputClassName}
                                            />
                                        </FormField>

                                        <div className="pt-3 border-t border-[var(--border-subtle)]">
                                            <p className="text-xs text-[var(--text-tertiary)] mb-3">Voice Detection Tuning (for noisy/distant audio)</p>
                                            <div className="grid grid-cols-2 gap-4">
                                                <FormField label="VAD Onset" description={PARAM_DESCRIPTIONS.vad_onset}>
                                                    <Input
                                                        type="number"
                                                        min={0.1}
                                                        max={0.9}
                                                        step={0.05}
                                                        value={params.vad_onset}
                                                        onChange={(e) => updateParam('vad_onset', parseFloat(e.target.value) || 0.5)}
                                                        className={inputClassName}
                                                    />
                                                </FormField>
                                                <FormField label="VAD Offset" description={PARAM_DESCRIPTIONS.vad_offset}>
                                                    <Input
                                                        type="number"
                                                        min={0.1}
                                                        max={0.9}
                                                        step={0.05}
                                                        value={params.vad_offset}
                                                        onChange={(e) => updateParam('vad_offset', parseFloat(e.target.value) || 0.363)}
                                                        className={inputClassName}
                                                    />
                                                </FormField>
                                            </div>
                                        </div>

                                        <div className="pt-3 border-t border-[var(--border-subtle)]">
                                            <p className="text-xs text-[var(--text-tertiary)] mb-3">CPU Performance Tuning</p>
                                            <FormField label="Performance Preset" description={PARAM_DESCRIPTIONS.diarization_perf_preset}>
                                                <Select
                                                    value={params.diarization_perf_preset || "auto"}
                                                    onValueChange={(v) => updateParam('diarization_perf_preset', v)}
                                                >
                                                    <SelectTrigger className={selectTriggerClassName}>
                                                        <SelectValue />
                                                    </SelectTrigger>
                                                    <SelectContent className={selectContentClassName}>
                                                        <SelectItem value="auto" className={selectItemClassName}>Auto (recommended)</SelectItem>
                                                        <SelectItem value="low" className={selectItemClassName}>Low</SelectItem>
                                                        <SelectItem value="medium" className={selectItemClassName}>Medium</SelectItem>
                                                        <SelectItem value="high" className={selectItemClassName}>High</SelectItem>
                                                        <SelectItem value="custom" className={selectItemClassName}>Custom</SelectItem>
                                                    </SelectContent>
                                                </Select>
                                            </FormField>

                                            {params.diarization_perf_preset === "custom" && (
                                                <>
                                                    <div className="grid grid-cols-2 gap-4">
                                                        <FormField label="Segmentation Batch" description={PARAM_DESCRIPTIONS.segmentation_batch_size}>
                                                            <Input
                                                                type="number"
                                                                min={1}
                                                                max={32}
                                                                value={params.segmentation_batch_size ?? ""}
                                                                onChange={(e) => updateParam('segmentation_batch_size', e.target.value ? parseInt(e.target.value) : undefined)}
                                                                className={inputClassName}
                                                            />
                                                        </FormField>
                                                        <FormField label="Embedding Batch" description={PARAM_DESCRIPTIONS.embedding_batch_size}>
                                                            <Input
                                                                type="number"
                                                                min={1}
                                                                max={32}
                                                                value={params.embedding_batch_size ?? ""}
                                                                onChange={(e) => updateParam('embedding_batch_size', e.target.value ? parseInt(e.target.value) : undefined)}
                                                                className={inputClassName}
                                                            />
                                                        </FormField>
                                                        <FormField label="Torch Threads" description={PARAM_DESCRIPTIONS.torch_threads}>
                                                            <Input
                                                                type="number"
                                                                min={1}
                                                                max={64}
                                                                value={params.torch_threads ?? ""}
                                                                onChange={(e) => updateParam('torch_threads', e.target.value ? parseInt(e.target.value) : undefined)}
                                                                className={inputClassName}
                                                            />
                                                        </FormField>
                                                        <FormField label="Torch Interop Threads" description={PARAM_DESCRIPTIONS.torch_interop_threads}>
                                                            <Input
                                                                type="number"
                                                                min={1}
                                                                max={8}
                                                                value={params.torch_interop_threads ?? ""}
                                                                onChange={(e) => updateParam('torch_interop_threads', e.target.value ? parseInt(e.target.value) : undefined)}
                                                                className={inputClassName}
                                                            />
                                                        </FormField>
                                                    </div>

                                                    <div className="pt-3">
                                                        <FormField label="Exclude Overlap in Embeddings" description={PARAM_DESCRIPTIONS.embedding_exclude_overlap}>
                                                            <div className="flex items-center gap-3">
                                                                <Switch
                                                                    checked={params.embedding_exclude_overlap ?? true}
                                                                    onCheckedChange={(value) => updateParam('embedding_exclude_overlap', value)}
                                                                />
                                                                <span className="text-sm text-[var(--text-primary)]">Faster on CPU</span>
                                                            </div>
                                                        </FormField>
                                                    </div>
                                                </>
                                            )}
                                        </div>
                                    </>
                                )}
                            </div>
                        )}
                    </div>
                </Section>
            )}
        </div>
    );
}

function CanaryConfig({ params, updateParam, isMultiTrack }: ConfigProps) {
    return (
        <div className="space-y-6">
            <Section title="Model Selection" description="Choose the Canary model variant">
                <FormField label="Model">
                    <Select value={params.model} onValueChange={(v) => updateParam('model', v)}>
                        <SelectTrigger className={selectTriggerClassName}>
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent className={selectContentClassName}>
                            {CANARY_MODELS.map((model) => (
                                <SelectItem key={model} value={model} className={selectItemClassName}>{model}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </FormField>
            </Section>

            <Section title="Language Settings">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <FormField label="Source Language">
                        <Select value={params.language || "en"} onValueChange={(v) => updateParam('language', v)}>
                            <SelectTrigger className={selectTriggerClassName}>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className={selectContentClassName}>
                                {CANARY_LANGUAGES.map((l) => (
                                    <SelectItem key={l.value} value={l.value} className={selectItemClassName}>{l.label}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </FormField>

                    <FormField label="Task">
                        <Select value={params.task} onValueChange={(v) => updateParam('task', v)}>
                            <SelectTrigger className={selectTriggerClassName}>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className={selectContentClassName}>
                                <SelectItem value="transcribe" className={selectItemClassName}>Transcribe</SelectItem>
                                <SelectItem value="translate" className={selectItemClassName}>Translate</SelectItem>
                            </SelectContent>
                        </Select>
                    </FormField>
                </div>

                {params.task === "translate" && (
                    <FormField label="Target Language">
                        <Select value={params.target_language || "en"} onValueChange={(v) => updateParam('target_language', v)}>
                            <SelectTrigger className={selectTriggerClassName}>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className={selectContentClassName}>
                                {CANARY_LANGUAGES.map((l) => (
                                    <SelectItem key={l.value} value={l.value} className={selectItemClassName}>{l.label}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </FormField>
                )}
            </Section>

            <Section title="Output Formatting">
                <div className="flex items-center gap-3">
                    <Switch
                        id="canary_pnc"
                        checked={params.pnc ?? true}
                        onCheckedChange={(v) => updateParam('pnc', v)}
                    />
                    <label htmlFor="canary_pnc" className="text-sm text-[var(--text-primary)] cursor-pointer">
                        Enable punctuation and capitalization
                    </label>
                </div>
            </Section>

            <Section title="ASR VAD" description="Control how audio is segmented before recognition">
                <div className="space-y-4">
                    <FormField label="VAD Preset" description={PARAM_DESCRIPTIONS.vad_preset}>
                        <Select value={params.vad_preset} onValueChange={(v) => updateParam('vad_preset', v)}>
                            <SelectTrigger className={selectTriggerClassName}>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className={selectContentClassName}>
                                <SelectItem value="conservative" className={selectItemClassName}>Conservative</SelectItem>
                                <SelectItem value="balanced" className={selectItemClassName}>Balanced</SelectItem>
                                <SelectItem value="aggressive" className={selectItemClassName}>Aggressive</SelectItem>
                            </SelectContent>
                        </Select>
                    </FormField>

                    <div className="grid grid-cols-2 gap-4">
                        <FormField label="Speech Pad (ms)" optional>
                            <Input
                                type="number"
                                min={0}
                                placeholder="Default"
                                value={params.vad_speech_pad_ms ?? ""}
                                onChange={(e) => updateParam('vad_speech_pad_ms', e.target.value ? parseInt(e.target.value) : undefined)}
                                className={inputClassName}
                            />
                        </FormField>
                        <FormField label="Min Silence (ms)" optional>
                            <Input
                                type="number"
                                min={0}
                                placeholder="Default"
                                value={params.vad_min_silence_ms ?? ""}
                                onChange={(e) => updateParam('vad_min_silence_ms', e.target.value ? parseInt(e.target.value) : undefined)}
                                className={inputClassName}
                            />
                        </FormField>
                        <FormField label="Min Speech (ms)" optional>
                            <Input
                                type="number"
                                min={0}
                                placeholder="Default"
                                value={params.vad_min_speech_ms ?? ""}
                                onChange={(e) => updateParam('vad_min_speech_ms', e.target.value ? parseInt(e.target.value) : undefined)}
                                className={inputClassName}
                            />
                        </FormField>
                        <FormField label="Max Speech (s)" optional>
                            <Input
                                type="number"
                                min={1}
                                placeholder="Default"
                                value={params.vad_max_speech_s ?? ""}
                                onChange={(e) => updateParam('vad_max_speech_s', e.target.value ? parseInt(e.target.value) : undefined)}
                                className={inputClassName}
                            />
                        </FormField>
                    </div>
                </div>
            </Section>

            {/* Diarization for Canary */}
            {!isMultiTrack && (
                <Section title="Speaker Diarization">
                    <div className="space-y-4">
                        <div className="flex items-center gap-3">
                            <Switch
                                id="canary_diarize"
                                checked={params.diarize}
                                onCheckedChange={(v) => updateParam('diarize', v)}
                            />
                            <label htmlFor="canary_diarize" className="text-sm text-[var(--text-primary)] cursor-pointer">
                                Enable speaker identification
                            </label>
                        </div>

                        {params.diarize && (
                            <div className="p-4 bg-[var(--bg-main)] rounded-xl border border-[var(--border-subtle)] space-y-4">
                                <FormField label="Diarization Model">
                                    <Select value={params.diarize_model} onValueChange={(v) => updateParam('diarize_model', v)}>
                                        <SelectTrigger className={selectTriggerClassName}>
                                            <SelectValue />
                                        </SelectTrigger>
                                        <SelectContent className={selectContentClassName}>
                                            <SelectItem value="pyannote" className={selectItemClassName}>Pyannote</SelectItem>
                                            <SelectItem value="nvidia_sortformer" className={selectItemClassName}>NVIDIA Sortformer</SelectItem>
                                        </SelectContent>
                                    </Select>
                                </FormField>

                                <div className="grid grid-cols-2 gap-4">
                                    <FormField label="Min Speakers" optional>
                                        <Input
                                            type="number"
                                            min={1}
                                            max={20}
                                            placeholder="Auto"
                                            value={params.min_speakers || ""}
                                            onChange={(e) => updateParam('min_speakers', e.target.value ? parseInt(e.target.value) : undefined)}
                                            className={inputClassName}
                                        />
                                    </FormField>
                                    <FormField label="Max Speakers" optional>
                                        <Input
                                            type="number"
                                            min={1}
                                            max={20}
                                            placeholder="Auto"
                                            value={params.max_speakers || ""}
                                            onChange={(e) => updateParam('max_speakers', e.target.value ? parseInt(e.target.value) : undefined)}
                                            className={inputClassName}
                                        />
                                    </FormField>
                                </div>

                                {params.diarize_model === "pyannote" && (
                                    <>
                                        <FormField label="Hugging Face Token">
                                            <Input
                                                type="password"
                                                placeholder="hf_..."
                                                value={params.hf_token || ""}
                                                onChange={(e) => updateParam('hf_token', e.target.value || undefined)}
                                                className={inputClassName}
                                            />
                                        </FormField>

                                        <div className="pt-3 border-t border-[var(--border-subtle)]">
                                            <p className="text-xs text-[var(--text-tertiary)] mb-3">Voice Detection Tuning (for noisy/distant audio)</p>
                                            <div className="grid grid-cols-2 gap-4">
                                                <FormField label="VAD Onset" description={PARAM_DESCRIPTIONS.vad_onset}>
                                                    <Input
                                                        type="number"
                                                        min={0.1}
                                                        max={0.9}
                                                        step={0.05}
                                                        value={params.vad_onset}
                                                        onChange={(e) => updateParam('vad_onset', parseFloat(e.target.value) || 0.5)}
                                                        className={inputClassName}
                                                    />
                                                </FormField>
                                                <FormField label="VAD Offset" description={PARAM_DESCRIPTIONS.vad_offset}>
                                                    <Input
                                                        type="number"
                                                        min={0.1}
                                                        max={0.9}
                                                        step={0.05}
                                                        value={params.vad_offset}
                                                        onChange={(e) => updateParam('vad_offset', parseFloat(e.target.value) || 0.363)}
                                                        className={inputClassName}
                                                    />
                                                </FormField>
                                            </div>
                                        </div>

                                        <div className="pt-3 border-t border-[var(--border-subtle)]">
                                            <p className="text-xs text-[var(--text-tertiary)] mb-3">CPU Performance Tuning</p>
                                            <FormField label="Performance Preset" description={PARAM_DESCRIPTIONS.diarization_perf_preset}>
                                                <Select
                                                    value={params.diarization_perf_preset || "auto"}
                                                    onValueChange={(v) => updateParam('diarization_perf_preset', v)}
                                                >
                                                    <SelectTrigger className={selectTriggerClassName}>
                                                        <SelectValue />
                                                    </SelectTrigger>
                                                    <SelectContent className={selectContentClassName}>
                                                        <SelectItem value="auto" className={selectItemClassName}>Auto (recommended)</SelectItem>
                                                        <SelectItem value="low" className={selectItemClassName}>Low</SelectItem>
                                                        <SelectItem value="medium" className={selectItemClassName}>Medium</SelectItem>
                                                        <SelectItem value="high" className={selectItemClassName}>High</SelectItem>
                                                        <SelectItem value="custom" className={selectItemClassName}>Custom</SelectItem>
                                                    </SelectContent>
                                                </Select>
                                            </FormField>

                                            {params.diarization_perf_preset === "custom" && (
                                                <>
                                                    <div className="grid grid-cols-2 gap-4">
                                                        <FormField label="Segmentation Batch" description={PARAM_DESCRIPTIONS.segmentation_batch_size}>
                                                            <Input
                                                                type="number"
                                                                min={1}
                                                                max={32}
                                                                value={params.segmentation_batch_size ?? ""}
                                                                onChange={(e) => updateParam('segmentation_batch_size', e.target.value ? parseInt(e.target.value) : undefined)}
                                                                className={inputClassName}
                                                            />
                                                        </FormField>
                                                        <FormField label="Embedding Batch" description={PARAM_DESCRIPTIONS.embedding_batch_size}>
                                                            <Input
                                                                type="number"
                                                                min={1}
                                                                max={32}
                                                                value={params.embedding_batch_size ?? ""}
                                                                onChange={(e) => updateParam('embedding_batch_size', e.target.value ? parseInt(e.target.value) : undefined)}
                                                                className={inputClassName}
                                                            />
                                                        </FormField>
                                                        <FormField label="Torch Threads" description={PARAM_DESCRIPTIONS.torch_threads}>
                                                            <Input
                                                                type="number"
                                                                min={1}
                                                                max={64}
                                                                value={params.torch_threads ?? ""}
                                                                onChange={(e) => updateParam('torch_threads', e.target.value ? parseInt(e.target.value) : undefined)}
                                                                className={inputClassName}
                                                            />
                                                        </FormField>
                                                        <FormField label="Torch Interop Threads" description={PARAM_DESCRIPTIONS.torch_interop_threads}>
                                                            <Input
                                                                type="number"
                                                                min={1}
                                                                max={8}
                                                                value={params.torch_interop_threads ?? ""}
                                                                onChange={(e) => updateParam('torch_interop_threads', e.target.value ? parseInt(e.target.value) : undefined)}
                                                                className={inputClassName}
                                                            />
                                                        </FormField>
                                                    </div>

                                                    <div className="pt-3">
                                                        <FormField label="Exclude Overlap in Embeddings" description={PARAM_DESCRIPTIONS.embedding_exclude_overlap}>
                                                            <div className="flex items-center gap-3">
                                                                <Switch
                                                                    checked={params.embedding_exclude_overlap ?? true}
                                                                    onCheckedChange={(value) => updateParam('embedding_exclude_overlap', value)}
                                                                />
                                                                <span className="text-sm text-[var(--text-primary)]">Faster on CPU</span>
                                                            </div>
                                                        </FormField>
                                                    </div>
                                                </>
                                            )}
                                        </div>
                                    </>
                                )}
                            </div>
                        )}
                    </div>
                </Section>
            )}
        </div>
    );
}

interface OpenAIConfigProps extends ConfigProps {
    isValidating: boolean;
    validationStatus: 'idle' | 'valid' | 'invalid';
    validationMessage: string;
    availableModels: string[];
    onValidate: () => void;
}

function OpenAIConfig({
    params,
    updateParam,
    isValidating,
    validationStatus,
    validationMessage,
    availableModels,
    onValidate
}: OpenAIConfigProps) {
    return (
        <div className="space-y-6">
            <Section title="API Configuration">
                <div className="space-y-4">
                    <FormField label="OpenAI API Key" description="Your API key. Leave empty to use server default if configured.">
                        <div className="flex gap-2">
                            <Input
                                type="password"
                                placeholder="sk-..."
                                value={params.api_key || ""}
                                onChange={(e) => {
                                    updateParam('api_key', e.target.value);
                                }}
                                className={`${inputClassName} flex-1`}
                            />
                            <Button
                                variant="outline"
                                onClick={onValidate}
                                disabled={isValidating}
                                className="shrink-0 rounded-xl border-[var(--border-subtle)] cursor-pointer"
                            >
                                {isValidating ? <Loader2 className="h-4 w-4 animate-spin" /> : "Validate"}
                            </Button>
                        </div>
                        {validationStatus !== 'idle' && (
                            <div className={`flex items-center gap-2 text-sm mt-2 ${validationStatus === 'valid' ? 'text-[var(--success-solid)]' : 'text-[var(--error)]'
                                }`}>
                                {validationStatus === 'valid' ? <Check className="h-4 w-4" /> : <XCircle className="h-4 w-4" />}
                                <span>{validationMessage}</span>
                            </div>
                        )}
                    </FormField>

                    <FormField label="Model">
                        <Select value={params.model || "whisper-1"} onValueChange={(v) => updateParam('model', v)}>
                            <SelectTrigger className={selectTriggerClassName}>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className={selectContentClassName}>
                                {availableModels.map((m) => (
                                    <SelectItem key={m} value={m} className={selectItemClassName}>{m}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </FormField>

                    <FormField label="Language">
                        <Select value={params.language || "auto"} onValueChange={(v) => updateParam('language', v === "auto" ? undefined : v)}>
                            <SelectTrigger className={selectTriggerClassName}>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className={selectContentClassName}>
                                {LANGUAGES.map((l) => (
                                    <SelectItem key={l.value} value={l.value} className={selectItemClassName}>{l.label}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </FormField>
                </div>
            </Section>

            {params.model && params.model !== "whisper-1" && (
                <InfoBanner variant="warning" title="Limited Features">
                    Word-level timestamps are only supported by whisper-1. Synchronized playback won't be available.
                </InfoBanner>
            )}
        </div>
    );
}
