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
import { Slider } from "@/components/ui/slider";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
} from "@/components/ui/accordion";
import { Loader2, Check, XCircle } from "lucide-react";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { FormField, Section, InfoBanner } from "@/components/transcription/FormHelpers";

// ============================================================================
// Types & Constants
// ============================================================================

export interface WhisperXParams {
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
    is_multi_track_enabled: boolean;
    api_key?: string;
    max_new_tokens?: number;
}

interface TranscriptionConfigDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onStartTranscription: (params: WhisperXParams & { profileName?: string; profileDescription?: string }) => void;
    loading?: boolean;
    isProfileMode?: boolean;
    initialParams?: WhisperXParams;
    initialName?: string;
    initialDescription?: string;
    isMultiTrack?: boolean;
    title?: string;
}

const DEFAULT_PARAMS: WhisperXParams = {
    model_family: "whisper",
    model: "small",
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
    diarize: false,
    diarize_model: "pyannote",
    speaker_embeddings: false,
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
    is_multi_track_enabled: false,
    api_key: "",
};

const WHISPER_MODELS = [
    "tiny", "tiny.en", "base", "base.en", "small", "small.en",
    "medium", "medium.en", "large", "large-v1", "large-v2", "large-v3"
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

const VOXTRAL_LANGUAGES = [
    { value: "auto", label: "Auto-detect" },
    { value: "en", label: "English" },
    { value: "zh", label: "Chinese" },
    { value: "de", label: "German" },
    { value: "es", label: "Spanish" },
    { value: "ru", label: "Russian" },
    { value: "ko", label: "Korean" },
    { value: "fr", label: "French" },
    { value: "ja", label: "Japanese" },
    { value: "pt", label: "Portuguese" },
    { value: "tr", label: "Turkish" },
    { value: "pl", label: "Polish" },
    { value: "nl", label: "Dutch" },
    { value: "ar", label: "Arabic" },
    { value: "sv", label: "Swedish" },
    { value: "it", label: "Italian" },
    { value: "id", label: "Indonesian" },
    { value: "hi", label: "Hindi" },
    { value: "fi", label: "Finnish" },
    { value: "vi", label: "Vietnamese" },
    { value: "he", label: "Hebrew" },
    { value: "uk", label: "Ukrainian" },
    { value: "el", label: "Greek" },
];

const PARAM_DESCRIPTIONS = {
    model: "Size of the Whisper model. Larger = more accurate but slower.",
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
    const [params, setParams] = useState<WhisperXParams>(DEFAULT_PARAMS);
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
            const baseParams = initialParams || DEFAULT_PARAMS;
            setParams({
                ...baseParams,
                is_multi_track_enabled: isMultiTrack,
                diarize: isMultiTrack ? false : baseParams.diarize
            });
            setProfileName(initialName);
            setProfileDescription(initialDescription);
        }
    }, [open, initialParams, initialName, initialDescription, isMultiTrack]);

    const updateParam = <K extends keyof WhisperXParams>(key: K, value: WhisperXParams[K]) => {
        setParams(prev => {
            const newParams = { ...prev, [key]: value };
            if (key === 'model_family' && value === 'whisper') {
                newParams.diarize_model = 'pyannote';
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
                                <SelectItem value="mistral_voxtral" className={selectItemClassName}>
                                    Mistral Voxtral
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

                    {params.model_family === "mistral_voxtral" && (
                        <VoxtralConfig
                            params={params}
                            updateParam={updateParam}
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
    params: WhisperXParams;
    updateParam: <K extends keyof WhisperXParams>(key: K, value: WhisperXParams[K]) => void;
    isMultiTrack?: boolean;
}

function WhisperConfig({ params, updateParam, isMultiTrack }: ConfigProps) {
    return (
        <div className="space-y-6">
            {/* Essential Settings */}
            <Section title="Model Settings">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <FormField label="Model Size" description={PARAM_DESCRIPTIONS.model}>
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

                    <FormField label="Task" description={PARAM_DESCRIPTIONS.task}>
                        <Select value={params.task} onValueChange={(v) => updateParam('task', v)}>
                            <SelectTrigger className={selectTriggerClassName}>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className={selectContentClassName}>
                                <SelectItem value="transcribe" className={selectItemClassName}>Transcribe</SelectItem>
                                <SelectItem value="translate" className={selectItemClassName}>Translate to English</SelectItem>
                            </SelectContent>
                        </Select>
                    </FormField>

                    <FormField label="Device" description={PARAM_DESCRIPTIONS.device}>
                        <Select value={params.device} onValueChange={(v) => updateParam('device', v)}>
                            <SelectTrigger className={selectTriggerClassName}>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className={selectContentClassName}>
                                <SelectItem value="cpu" className={selectItemClassName}>CPU</SelectItem>
                                <SelectItem value="cuda" className={selectItemClassName}>GPU (CUDA)</SelectItem>
                            </SelectContent>
                        </Select>
                    </FormField>
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
                            </div>
                        )}
                    </div>
                </Section>
            )}

            {/* Advanced Settings (Accordion) */}
            <Accordion type="single" collapsible className="w-full">
                <AccordionItem value="advanced" className="border border-[var(--border-subtle)] rounded-xl px-4">
                    <AccordionTrigger className="text-sm font-medium text-[var(--text-primary)] hover:no-underline py-4">
                        Advanced Settings
                    </AccordionTrigger>
                    <AccordionContent className="pb-4 space-y-4">
                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                            <FormField label="Compute Type" description={PARAM_DESCRIPTIONS.compute_type}>
                                <Select value={params.compute_type} onValueChange={(v) => updateParam('compute_type', v)}>
                                    <SelectTrigger className={selectTriggerClassName}>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent className={selectContentClassName}>
                                        <SelectItem value="float32" className={selectItemClassName}>Float32 (Accurate)</SelectItem>
                                        <SelectItem value="float16" className={selectItemClassName}>Float16 (Fast)</SelectItem>
                                        <SelectItem value="int8" className={selectItemClassName}>Int8 (Fastest)</SelectItem>
                                    </SelectContent>
                                </Select>
                            </FormField>

                            <FormField label="Batch Size" description={PARAM_DESCRIPTIONS.batch_size}>
                                <Input
                                    type="number"
                                    min={1}
                                    max={64}
                                    value={params.batch_size}
                                    onChange={(e) => updateParam('batch_size', parseInt(e.target.value) || 8)}
                                    className={inputClassName}
                                />
                            </FormField>

                            <FormField label="Beam Size" description={PARAM_DESCRIPTIONS.beam_size}>
                                <Input
                                    type="number"
                                    min={1}
                                    max={10}
                                    value={params.beam_size}
                                    onChange={(e) => updateParam('beam_size', parseInt(e.target.value) || 5)}
                                    className={inputClassName}
                                />
                            </FormField>

                            <FormField label="Temperature" description={PARAM_DESCRIPTIONS.temperature}>
                                <Input
                                    type="number"
                                    min={0}
                                    max={1}
                                    step={0.1}
                                    value={params.temperature}
                                    onChange={(e) => updateParam('temperature', parseFloat(e.target.value) || 0)}
                                    className={inputClassName}
                                />
                            </FormField>
                        </div>

                        <FormField label="Initial Prompt" description={PARAM_DESCRIPTIONS.initial_prompt} optional>
                            <Textarea
                                placeholder="Optional context to guide transcription..."
                                value={params.initial_prompt || ""}
                                onChange={(e) => updateParam('initial_prompt', e.target.value || undefined)}
                                className={`${inputClassName} resize-none min-h-[80px]`}
                                rows={2}
                            />
                        </FormField>

                        <div className="flex items-center gap-3">
                            <Switch
                                id="suppress_numerals"
                                checked={params.suppress_numerals}
                                onCheckedChange={(v) => updateParam('suppress_numerals', v)}
                            />
                            <label htmlFor="suppress_numerals" className="text-sm text-[var(--text-primary)] cursor-pointer">
                                Suppress numerals (write numbers as words)
                            </label>
                        </div>

                        {/* Alignment Settings */}
                        <div className="pt-2 border-t border-[var(--border-subtle)] space-y-4">
                            <div className="flex items-center gap-3">
                                <Switch
                                    id="no_align"
                                    checked={params.no_align}
                                    onCheckedChange={(v) => updateParam('no_align', v)}
                                />
                                <label htmlFor="no_align" className="text-sm text-[var(--text-primary)] cursor-pointer">
                                    Skip word alignment (faster, less precise timestamps)
                                </label>
                            </div>

                            {!params.no_align && (
                                <FormField label="Custom Alignment Model" description="WhisperX-compatible alignment model (e.g., KBLab/wav2vec2-large-voxrex-swedish). Leave empty for default." optional>
                                    <Input
                                        placeholder="model/path or HuggingFace ID"
                                        value={params.align_model || ""}
                                        onChange={(e) => updateParam('align_model', e.target.value || undefined)}
                                        className={inputClassName}
                                    />
                                </FormField>
                            )}
                        </div>
                    </AccordionContent>
                </AccordionItem>
            </Accordion>
        </div>
    );
}

function ParakeetConfig({ params, updateParam, isMultiTrack }: ConfigProps) {
    return (
        <div className="space-y-6">
            {/* Long-form Audio Settings */}
            <Section title="Audio Context" description="Configure how much context the model uses for long audio files">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
                    <div className="space-y-3">
                        <FormField label="Left Context">
                            <Slider
                                value={[params.attention_context_left]}
                                onValueChange={(v) => updateParam('attention_context_left', v[0])}
                                max={512}
                                min={64}
                                step={64}
                                className="w-full"
                            />
                            <div className="flex justify-between text-xs text-[var(--text-tertiary)]">
                                <span>64</span>
                                <span className="font-medium text-[var(--text-primary)]">{params.attention_context_left}</span>
                                <span>512</span>
                            </div>
                        </FormField>
                    </div>

                    <div className="space-y-3">
                        <FormField label="Right Context">
                            <Slider
                                value={[params.attention_context_right]}
                                onValueChange={(v) => updateParam('attention_context_right', v[0])}
                                max={512}
                                min={64}
                                step={64}
                                className="w-full"
                            />
                            <div className="flex justify-between text-xs text-[var(--text-tertiary)]">
                                <span>64</span>
                                <span className="font-medium text-[var(--text-primary)]">{params.attention_context_right}</span>
                                <span>512</span>
                            </div>
                        </FormField>
                    </div>
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
            <Section title="Language Settings">
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

function VoxtralConfig({ params, updateParam }: ConfigProps) {
    return (
        <div className="space-y-6">
            {/* Voxtral Warning Banner */}
            <InfoBanner variant="warning" title="Limited Features">
                Voxtral does not support word-level timestamps. Synchronized playback, audio seeking, and timestamp-based features won't be available.
            </InfoBanner>

            <Section title="Language Settings">
                <FormField label="Language" description="Source language for transcription">
                    <Select value={params.language || "en"} onValueChange={(v) => updateParam('language', v)}>
                        <SelectTrigger className={selectTriggerClassName}>
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent className={selectContentClassName}>
                            {VOXTRAL_LANGUAGES.map((l) => (
                                <SelectItem key={l.value} value={l.value} className={selectItemClassName}>{l.label}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </FormField>
            </Section>

            {/* Advanced Settings */}
            <Accordion type="single" collapsible className="w-full">
                <AccordionItem value="advanced" className="border border-[var(--border-subtle)] rounded-xl px-4">
                    <AccordionTrigger className="text-sm font-medium text-[var(--text-primary)] hover:no-underline py-4">
                        Advanced Settings
                    </AccordionTrigger>
                    <AccordionContent className="pb-4 space-y-4">
                        <FormField label="Max Tokens" description="Maximum number of tokens to generate. Voxtral has a 32k context window and handles up to 30-40 minutes of audio.">
                            <Input
                                type="number"
                                min={1024}
                                max={16384}
                                value={params.max_new_tokens || 8192}
                                onChange={(e) => updateParam('max_new_tokens', parseInt(e.target.value) || 8192)}
                                className={inputClassName}
                            />
                        </FormField>
                    </AccordionContent>
                </AccordionItem>
            </Accordion>
        </div>
    );
}
