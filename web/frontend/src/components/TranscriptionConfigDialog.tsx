import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Slider } from "@/components/ui/slider";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Separator } from "@/components/ui/separator";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Info } from "lucide-react";

export interface WhisperXParams {
  // Model parameters
  model: string;
  model_cache_only: boolean;
  model_dir?: string;
  
  // Device and computation
  device: string;
  device_index: number;
  batch_size: number;
  compute_type: string;
  threads: number;
  
  // Output settings
  output_format: string;
  verbose: boolean;
  
  // Task and language
  task: string;
  language?: string;
  
  // Alignment settings
  align_model?: string;
  interpolate_method: string;
  no_align: boolean;
  return_char_alignments: boolean;
  
  // VAD settings
  vad_method: string;
  vad_onset: number;
  vad_offset: number;
  chunk_size: number;
  
  // Diarization settings
  diarize: boolean;
  min_speakers?: number;
  max_speakers?: number;
  diarize_model: string;
  speaker_embeddings: boolean;
  
  // Transcription quality settings
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
  
  // Output formatting
  max_line_width?: number;
  max_line_count?: number;
  highlight_words: boolean;
  segment_resolution: string;
  
  // Token and progress
  hf_token?: string;
  print_progress: boolean;
}

// Parameter descriptions for hover cards
const PARAM_DESCRIPTIONS = {
  model: "Size of the Whisper model to use. Larger models are more accurate but slower and require more memory.",
  language: "Source language of the audio. Leave as auto-detect for automatic language detection.",
  task: "Whether to transcribe the audio or translate it to English.",
  device: "Processing device: CPU (slower, universal), GPU (faster, requires CUDA), or Auto (automatic selection).",
  compute_type: "Precision type: Float16 (faster, less memory), Float32 (more accurate), Int8 (fastest, least accurate).",
  batch_size: "Number of audio segments processed simultaneously. Higher values are faster but use more memory.",
  diarize: "Enable speaker diarization to identify and separate different speakers in the audio.",
  min_speakers: "Minimum number of speakers expected in the audio (leave empty for automatic detection).",
  max_speakers: "Maximum number of speakers expected in the audio (leave empty for automatic detection).",
  diarize_model: "Diarization model to use. Version 3.1 is newer and more accurate, while 3.0 is more stable and faster.",
  temperature: "Controls randomness in output. 0 = deterministic, higher values = more creative but less accurate.",
  beam_size: "Number of beams for beam search decoding. Higher values improve quality but are slower.",
  best_of: "Number of candidate sequences when sampling. Higher values improve quality but are slower.",
  patience: "Patience factor for beam search. Higher values wait longer for better sequences.",
  length_penalty: "Penalty applied to longer sequences. >1 favors longer, <1 favors shorter sequences.",
  initial_prompt: "Optional text to provide context for the first transcription window.",
  suppress_numerals: "Suppress numeric symbols and currency symbols during transcription sampling.",
  condition_on_previous_text: "Use previous transcription output as context for next segment (may cause repetition loops).",
  vad_method: "Voice Activity Detection method: Pyannote (more accurate) or Silero (faster).",
  vad_onset: "Sensitivity threshold for detecting speech start. Lower = more sensitive to quiet speech.",
  vad_offset: "Sensitivity threshold for detecting speech end. Lower = continues longer into silence.",
  chunk_size: "Duration in seconds for merging adjacent speech segments detected by VAD.",
  compression_ratio_threshold: "Fail transcription if text compression ratio exceeds this value (indicates repetitive output).",
  logprob_threshold: "Fail transcription if average log probability is below this value (indicates low confidence).",
  no_speech_threshold: "Consider segment as silence if no-speech probability exceeds this value.",
  suppress_tokens: "Comma-separated token IDs to suppress during generation (e.g., -1 for default special tokens).",
  no_align: "Skip phoneme-level alignment for faster processing but less precise word timestamps.",
  return_char_alignments: "Include character-level timing alignments in the output (increases processing time).",
  fp16: "Use 16-bit floating point precision for faster inference with slightly reduced accuracy.",
  output_format: "File format(s) to generate: SRT (subtitles), VTT (web), TXT (plain text), JSON (structured), TSV (tabular), or All.",
  segment_resolution: "How to break up transcription: Sentence (natural breaks) or Chunk (fixed VAD segments).",
  max_line_width: "Maximum characters per line in subtitle formats before text wrapping.",
  max_line_count: "Maximum number of lines per subtitle segment.",
  highlight_words: "Add word-level timing highlights in SRT/VTT formats (underlines words as spoken).",
  verbose: "Show detailed progress and debug messages during transcription.",
  print_progress: "Display processing progress information in the console output.",
  hf_token: "Hugging Face API token required for accessing private or gated models."
};

interface TranscriptionConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onStartTranscription: (params: WhisperXParams & { profileName?: string; profileDescription?: string }) => void;
  loading?: boolean;
  isProfileMode?: boolean;
  initialParams?: WhisperXParams;
  initialName?: string;
  initialDescription?: string;
}

const DEFAULT_PARAMS: WhisperXParams = {
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
  diarize_model: "pyannote/speaker-diarization-3.1",
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
};

const WHISPER_MODELS = [
  "tiny", "tiny.en",
  "base", "base.en",
  "small", "small.en", 
  "medium", "medium.en",
  "large", "large-v1", "large-v2", "large-v3"
];

const LANGUAGES = [
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
  { value: "ca", label: "Catalan" },
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
  { value: "ms", label: "Malay" },
  { value: "cs", label: "Czech" },
  { value: "ro", label: "Romanian" },
  { value: "da", label: "Danish" },
  { value: "hu", label: "Hungarian" },
  { value: "ta", label: "Tamil" },
  { value: "no", label: "Norwegian" },
  { value: "th", label: "Thai" },
  { value: "ur", label: "Urdu" },
  { value: "hr", label: "Croatian" },
  { value: "bg", label: "Bulgarian" },
  { value: "lt", label: "Lithuanian" },
  { value: "la", label: "Latin" },
  { value: "mi", label: "Maori" },
  { value: "ml", label: "Malayalam" },
  { value: "cy", label: "Welsh" },
  { value: "sk", label: "Slovak" },
  { value: "te", label: "Telugu" },
  { value: "fa", label: "Persian" },
  { value: "lv", label: "Latvian" },
  { value: "bn", label: "Bengali" },
  { value: "sr", label: "Serbian" },
  { value: "az", label: "Azerbaijani" },
  { value: "sl", label: "Slovenian" },
  { value: "kn", label: "Kannada" },
  { value: "et", label: "Estonian" },
  { value: "mk", label: "Macedonian" },
  { value: "br", label: "Breton" },
  { value: "eu", label: "Basque" },
  { value: "is", label: "Icelandic" },
  { value: "hy", label: "Armenian" },
  { value: "ne", label: "Nepali" },
  { value: "mn", label: "Mongolian" },
  { value: "bs", label: "Bosnian" },
  { value: "kk", label: "Kazakh" },
  { value: "sq", label: "Albanian" },
  { value: "sw", label: "Swahili" },
  { value: "gl", label: "Galician" },
  { value: "mr", label: "Marathi" },
  { value: "pa", label: "Punjabi" },
  { value: "si", label: "Sinhala" },
  { value: "km", label: "Khmer" },
  { value: "sn", label: "Shona" },
  { value: "yo", label: "Yoruba" },
  { value: "so", label: "Somali" },
  { value: "af", label: "Afrikaans" },
  { value: "oc", label: "Occitan" },
  { value: "ka", label: "Georgian" },
  { value: "be", label: "Belarusian" },
  { value: "tg", label: "Tajik" },
  { value: "sd", label: "Sindhi" },
  { value: "gu", label: "Gujarati" },
  { value: "am", label: "Amharic" },
  { value: "yi", label: "Yiddish" },
  { value: "lo", label: "Lao" },
  { value: "uz", label: "Uzbek" },
  { value: "fo", label: "Faroese" },
  { value: "ht", label: "Haitian Creole" },
  { value: "ps", label: "Pashto" },
  { value: "tk", label: "Turkmen" },
  { value: "nn", label: "Nynorsk" },
  { value: "mt", label: "Maltese" },
  { value: "sa", label: "Sanskrit" },
  { value: "lb", label: "Luxembourgish" },
  { value: "my", label: "Myanmar" },
  { value: "bo", label: "Tibetan" },
  { value: "tl", label: "Tagalog" },
  { value: "mg", label: "Malagasy" },
  { value: "as", label: "Assamese" },
  { value: "tt", label: "Tatar" },
  { value: "haw", label: "Hawaiian" },
  { value: "ln", label: "Lingala" },
  { value: "ha", label: "Hausa" },
  { value: "ba", label: "Bashkir" },
  { value: "jw", label: "Javanese" },
  { value: "su", label: "Sundanese" },
];

export function TranscriptionConfigDialog({
  open,
  onOpenChange,
  onStartTranscription,
  loading = false,
  isProfileMode = false,
  initialParams,
  initialName = "",
  initialDescription = "",
}: TranscriptionConfigDialogProps) {
  const [params, setParams] = useState<WhisperXParams>(DEFAULT_PARAMS);
  const [profileName, setProfileName] = useState("");
  const [profileDescription, setProfileDescription] = useState("");

  // Reset to defaults or initial values when dialog opens
  useEffect(() => {
    if (open) {
      setParams(initialParams || DEFAULT_PARAMS);
      setProfileName(initialName);
      setProfileDescription(initialDescription);
    }
  }, [open, initialParams, initialName, initialDescription]);

  const updateParam = <K extends keyof WhisperXParams>(
    key: K,
    value: WhisperXParams[K]
  ) => {
    setParams(prev => ({ ...prev, [key]: value }));
  };

  const handleStartTranscription = () => {
    if (isProfileMode) {
      onStartTranscription({ ...params, profileName, profileDescription });
    } else {
      onStartTranscription(params);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700 p-8">
        <DialogHeader className="mb-6">
          <DialogTitle className="text-gray-900 dark:text-gray-100">
            {isProfileMode 
              ? (initialName ? `Edit "${initialName}"` : "New Transcription Profile")
              : "Transcription Configuration"
            }
          </DialogTitle>
          <DialogDescription className="text-gray-600 dark:text-gray-400">
            {isProfileMode 
              ? (initialName ? "Update your transcription profile settings." : "Create a new profile to save and reuse your transcription settings.")
              : "Configure WhisperX parameters for your transcription. Advanced settings allow fine-tuning quality and performance."
            }
          </DialogDescription>
        </DialogHeader>

        {isProfileMode && (
          <div className="mb-6 space-y-4">
            <div className="space-y-2">
              <Label htmlFor="profileName" className="text-gray-700 dark:text-gray-300 font-medium">
                Profile Name
              </Label>
              <Input
                id="profileName"
                value={profileName}
                onChange={(e) => setProfileName(e.target.value)}
                placeholder="Enter a name for this profile..."
                className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="profileDescription" className="text-gray-700 dark:text-gray-300 font-medium">
                Description <span className="text-gray-500 dark:text-gray-400 font-normal">(optional)</span>
              </Label>
              <Textarea
                id="profileDescription"
                value={profileDescription}
                onChange={(e) => setProfileDescription(e.target.value)}
                placeholder="Describe this profile's purpose..."
                className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 resize-none"
                rows={2}
              />
            </div>
          </div>
        )}

        <Tabs defaultValue="basic" className="w-full">
          <TabsList className="grid w-full grid-cols-4 bg-gray-100 dark:bg-gray-800">
            <TabsTrigger value="basic" className="data-[state=active]:bg-white data-[state=active]:dark:bg-gray-700 text-gray-700 dark:text-gray-300">Basic</TabsTrigger>
            <TabsTrigger value="quality" className="data-[state=active]:bg-white data-[state=active]:dark:bg-gray-700 text-gray-700 dark:text-gray-300">Quality</TabsTrigger>
            <TabsTrigger value="advanced" className="data-[state=active]:bg-white data-[state=active]:dark:bg-gray-700 text-gray-700 dark:text-gray-300">Advanced</TabsTrigger>
            <TabsTrigger value="diarization" className="data-[state=active]:bg-white data-[state=active]:dark:bg-gray-700 text-gray-700 dark:text-gray-300">Diarization</TabsTrigger>
          </TabsList>

          <TabsContent value="basic" className="space-y-8 mt-6">
            <div className="grid grid-cols-2 gap-8">
              <div className="space-y-6">
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Label htmlFor="model" className="text-gray-700 dark:text-gray-300">Model Size</Label>
                    <HoverCard>
                      <HoverCardTrigger asChild>
                        <Info className="h-4 w-4 text-gray-400 cursor-help" />
                      </HoverCardTrigger>
                      <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                        <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.model}</p>
                      </HoverCardContent>
                    </HoverCard>
                  </div>
                  <Select
                    value={params.model}
                    onValueChange={(value) => updateParam('model', value)}
                  >
                    <SelectTrigger className="mt-3 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                      {WHISPER_MODELS.map((model) => (
                        <SelectItem key={model} value={model} className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">
                          {model}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Label htmlFor="language" className="text-gray-700 dark:text-gray-300">Language</Label>
                    <HoverCard>
                      <HoverCardTrigger asChild>
                        <Info className="h-4 w-4 text-gray-400 cursor-help" />
                      </HoverCardTrigger>
                      <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                        <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.language}</p>
                      </HoverCardContent>
                    </HoverCard>
                  </div>
                  <Select
                    value={params.language || "auto"}
                    onValueChange={(value) => updateParam('language', value === "auto" ? undefined : value)}
                  >
                    <SelectTrigger className="mt-3 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 max-h-60">
                      {LANGUAGES.map((lang) => (
                        <SelectItem key={lang.value} value={lang.value} className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">
                          {lang.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Label htmlFor="task" className="text-gray-700 dark:text-gray-300">Task</Label>
                    <HoverCard>
                      <HoverCardTrigger asChild>
                        <Info className="h-4 w-4 text-gray-400 cursor-help" />
                      </HoverCardTrigger>
                      <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                        <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.task}</p>
                      </HoverCardContent>
                    </HoverCard>
                  </div>
                  <Select
                    value={params.task}
                    onValueChange={(value) => updateParam('task', value)}
                  >
                    <SelectTrigger className="mt-3 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                      <SelectItem value="transcribe" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">Transcribe</SelectItem>
                      <SelectItem value="translate" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">Translate to English</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="space-y-6">
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Label htmlFor="device" className="text-gray-700 dark:text-gray-300">Device</Label>
                    <HoverCard>
                      <HoverCardTrigger asChild>
                        <Info className="h-4 w-4 text-gray-400 cursor-help" />
                      </HoverCardTrigger>
                      <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                        <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.device}</p>
                      </HoverCardContent>
                    </HoverCard>
                  </div>
                  <Select
                    value={params.device}
                    onValueChange={(value) => updateParam('device', value)}
                  >
                    <SelectTrigger className="mt-3 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                      <SelectItem value="cpu" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">CPU</SelectItem>
                      <SelectItem value="cuda" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">GPU (CUDA)</SelectItem>
                      <SelectItem value="auto" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">Auto</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Label htmlFor="compute_type" className="text-gray-700 dark:text-gray-300">Compute Type</Label>
                    <HoverCard>
                      <HoverCardTrigger asChild>
                        <Info className="h-4 w-4 text-gray-400 cursor-help" />
                      </HoverCardTrigger>
                      <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                        <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.compute_type}</p>
                      </HoverCardContent>
                    </HoverCard>
                  </div>
                  <Select
                    value={params.compute_type}
                    onValueChange={(value) => updateParam('compute_type', value)}
                  >
                    <SelectTrigger className="mt-3 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                      <SelectItem value="float16" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">Float16 (Faster)</SelectItem>
                      <SelectItem value="float32" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">Float32 (More Accurate)</SelectItem>
                      <SelectItem value="int8" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">Int8 (Fastest)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="batch_size" className="text-gray-700 dark:text-gray-300">Batch Size: {params.batch_size}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.batch_size}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.batch_size]}
                    onValueChange={([value]) => updateParam('batch_size', value)}
                    min={1}
                    max={32}
                    step={1}
                    className="mt-3"
                  />
                </div>

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="threads" className="text-gray-700 dark:text-gray-300">Threads: {params.threads}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">Number of threads used by torch for CPU inference. 0 means auto-detect.</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.threads]}
                    onValueChange={([value]) => updateParam('threads', value)}
                    min={0}
                    max={16}
                    step={1}
                    className="mt-3"
                  />
                </div>
              </div>
            </div>

          </TabsContent>

          <TabsContent value="quality" className="space-y-8 mt-6">
            <div className="grid grid-cols-2 gap-8">
              <div className="space-y-6">
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="temperature" className="text-gray-700 dark:text-gray-300">Temperature: {params.temperature}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.temperature}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.temperature]}
                    onValueChange={([value]) => updateParam('temperature', value)}
                    min={0}
                    max={1}
                    step={0.1}
                    className="mt-3"
                  />
                </div>

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="beam_size" className="text-gray-700 dark:text-gray-300">Beam Size: {params.beam_size}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.beam_size}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.beam_size]}
                    onValueChange={([value]) => updateParam('beam_size', value)}
                    min={1}
                    max={20}
                    step={1}
                    className="mt-3"
                  />
                </div>

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="best_of" className="text-gray-700 dark:text-gray-300">Best Of: {params.best_of}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.best_of}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.best_of]}
                    onValueChange={([value]) => updateParam('best_of', value)}
                    min={1}
                    max={20}
                    step={1}
                    className="mt-3"
                  />
                </div>
              </div>

              <div className="space-y-6">
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="patience" className="text-gray-700 dark:text-gray-300">Patience: {params.patience}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.patience}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.patience]}
                    onValueChange={([value]) => updateParam('patience', value)}
                    min={0.1}
                    max={3.0}
                    step={0.1}
                    className="mt-3"
                  />
                </div>

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="length_penalty" className="text-gray-700 dark:text-gray-300">Length Penalty: {params.length_penalty}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.length_penalty}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.length_penalty]}
                    onValueChange={([value]) => updateParam('length_penalty', value)}
                    min={0.1}
                    max={3.0}
                    step={0.1}
                    className="mt-3"
                  />
                </div>

                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Label htmlFor="initial_prompt" className="text-gray-700 dark:text-gray-300">Initial Prompt</Label>
                    <HoverCard>
                      <HoverCardTrigger asChild>
                        <Info className="h-4 w-4 text-gray-400 cursor-help" />
                      </HoverCardTrigger>
                      <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                        <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.initial_prompt}</p>
                      </HoverCardContent>
                    </HoverCard>
                  </div>
                  <Textarea
                    placeholder="Optional text to provide as context for the first window"
                    value={params.initial_prompt || ""}
                    onChange={(e) => updateParam('initial_prompt', e.target.value || undefined)}
                    rows={3}
                    className="mt-3 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 placeholder:text-gray-500 dark:placeholder:text-gray-400"
                  />
                </div>
              </div>
            </div>

            <Separator className="bg-gray-200 dark:bg-gray-700 my-8" />

            <div className="space-y-6">
              <div className="flex items-center space-x-2">
                <Switch
                  id="suppress_numerals"
                  checked={params.suppress_numerals}
                  onCheckedChange={(checked) => updateParam('suppress_numerals', checked)}
                />
                <Label htmlFor="suppress_numerals" className="text-gray-700 dark:text-gray-300">Suppress Numerals</Label>
                <HoverCard>
                  <HoverCardTrigger asChild>
                    <Info className="h-4 w-4 text-gray-400 cursor-help" />
                  </HoverCardTrigger>
                  <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                    <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.suppress_numerals}</p>
                  </HoverCardContent>
                </HoverCard>
              </div>

              <div className="flex items-center space-x-2">
                <Switch
                  id="condition_on_previous_text"
                  checked={params.condition_on_previous_text}
                  onCheckedChange={(checked) => updateParam('condition_on_previous_text', checked)}
                />
                <Label htmlFor="condition_on_previous_text" className="text-gray-700 dark:text-gray-300">Condition on Previous Text</Label>
                <HoverCard>
                  <HoverCardTrigger asChild>
                    <Info className="h-4 w-4 text-gray-400 cursor-help" />
                  </HoverCardTrigger>
                  <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                    <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.condition_on_previous_text}</p>
                  </HoverCardContent>
                </HoverCard>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="advanced" className="space-y-8 mt-6">
            <div className="grid grid-cols-2 gap-8">
              <div className="space-y-6">
                <h4 className="font-medium text-gray-900 dark:text-gray-100">VAD Settings</h4>
                
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Label htmlFor="vad_method" className="text-gray-700 dark:text-gray-300">VAD Method</Label>
                    <HoverCard>
                      <HoverCardTrigger asChild>
                        <Info className="h-4 w-4 text-gray-400 cursor-help" />
                      </HoverCardTrigger>
                      <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                        <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.vad_method}</p>
                      </HoverCardContent>
                    </HoverCard>
                  </div>
                  <Select
                    value={params.vad_method}
                    onValueChange={(value) => updateParam('vad_method', value)}
                  >
                    <SelectTrigger className="mt-3 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                      <SelectItem value="pyannote" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">Pyannote</SelectItem>
                      <SelectItem value="silero" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">Silero</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="vad_onset" className="text-gray-700 dark:text-gray-300">VAD Onset: {params.vad_onset}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.vad_onset}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.vad_onset]}
                    onValueChange={([value]) => updateParam('vad_onset', value)}
                    min={0.1}
                    max={1.0}
                    step={0.01}
                    className="mt-3"
                  />
                </div>

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="vad_offset" className="text-gray-700 dark:text-gray-300">VAD Offset: {params.vad_offset}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.vad_offset}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.vad_offset]}
                    onValueChange={([value]) => updateParam('vad_offset', value)}
                    min={0.1}
                    max={1.0}
                    step={0.01}
                    className="mt-3"
                  />
                </div>

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="chunk_size" className="text-gray-700 dark:text-gray-300">Chunk Size: {params.chunk_size}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.chunk_size}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.chunk_size]}
                    onValueChange={([value]) => updateParam('chunk_size', value)}
                    min={5}
                    max={60}
                    step={1}
                    className="mt-3"
                  />
                </div>
              </div>

              <div className="space-y-6">
                <h4 className="font-medium">Detection Thresholds</h4>
                
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="compression_ratio_threshold" className="text-gray-700 dark:text-gray-300">Compression Ratio: {params.compression_ratio_threshold}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.compression_ratio_threshold}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.compression_ratio_threshold]}
                    onValueChange={([value]) => updateParam('compression_ratio_threshold', value)}
                    min={1.0}
                    max={5.0}
                    step={0.1}
                    className="mt-3"
                  />
                </div>

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="logprob_threshold" className="text-gray-700 dark:text-gray-300">Log Probability: {params.logprob_threshold}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.logprob_threshold}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.logprob_threshold]}
                    onValueChange={([value]) => updateParam('logprob_threshold', value)}
                    min={-3.0}
                    max={0.0}
                    step={0.1}
                    className="mt-3"
                  />
                </div>

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="no_speech_threshold" className="text-gray-700 dark:text-gray-300">No Speech: {params.no_speech_threshold}</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.no_speech_threshold}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                  </div>
                  <Slider
                    value={[params.no_speech_threshold]}
                    onValueChange={([value]) => updateParam('no_speech_threshold', value)}
                    min={0.1}
                    max={1.0}
                    step={0.01}
                    className="mt-3"
                  />
                </div>

                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Label htmlFor="suppress_tokens" className="text-gray-700 dark:text-gray-300">Suppress Tokens</Label>
                    <HoverCard>
                      <HoverCardTrigger asChild>
                        <Info className="h-4 w-4 text-gray-400 cursor-help" />
                      </HoverCardTrigger>
                      <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                        <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.suppress_tokens}</p>
                      </HoverCardContent>
                    </HoverCard>
                  </div>
                  <Input
                    placeholder="-1 (default: suppress special characters)"
                    value={params.suppress_tokens || ""}
                    onChange={(e) => updateParam('suppress_tokens', e.target.value || undefined)}
                    className="mt-3"
                  />
                </div>
              </div>
            </div>

            <Separator className="my-8" />

            <div className="space-y-6">
              <div className="flex items-center space-x-2">
                <Switch
                  id="no_align"
                  checked={params.no_align}
                  onCheckedChange={(checked) => updateParam('no_align', checked)}
                />
                <Label htmlFor="no_align" className="text-gray-700 dark:text-gray-300">Disable Alignment</Label>
                <HoverCard>
                  <HoverCardTrigger asChild>
                    <Info className="h-4 w-4 text-gray-400 cursor-help" />
                  </HoverCardTrigger>
                  <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                    <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.no_align}</p>
                  </HoverCardContent>
                </HoverCard>
              </div>

              <div className="flex items-center space-x-2">
                <Switch
                  id="return_char_alignments"
                  checked={params.return_char_alignments}
                  onCheckedChange={(checked) => updateParam('return_char_alignments', checked)}
                />
                <Label htmlFor="return_char_alignments" className="text-gray-700 dark:text-gray-300">Character-level Alignments</Label>
                <HoverCard>
                  <HoverCardTrigger asChild>
                    <Info className="h-4 w-4 text-gray-400 cursor-help" />
                  </HoverCardTrigger>
                  <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                    <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.return_char_alignments}</p>
                  </HoverCardContent>
                </HoverCard>
              </div>

              <div className="flex items-center space-x-2">
                <Switch
                  id="fp16"
                  checked={params.fp16}
                  onCheckedChange={(checked) => updateParam('fp16', checked)}
                />
                <Label htmlFor="fp16" className="text-gray-700 dark:text-gray-300">FP16 Inference</Label>
                <HoverCard>
                  <HoverCardTrigger asChild>
                    <Info className="h-4 w-4 text-gray-400 cursor-help" />
                  </HoverCardTrigger>
                  <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                    <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.fp16}</p>
                  </HoverCardContent>
                </HoverCard>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="diarization" className="space-y-8 mt-6">
            <div>
              <div className="flex items-center space-x-2 mb-6">
                <Switch
                  id="diarize"
                  checked={params.diarize}
                  onCheckedChange={(checked) => updateParam('diarize', checked)}
                />
                <Label htmlFor="diarize" className="text-gray-700 dark:text-gray-300">Enable Speaker Diarization</Label>
                <HoverCard>
                  <HoverCardTrigger asChild>
                    <Info className="h-4 w-4 text-gray-400 cursor-help" />
                  </HoverCardTrigger>
                  <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                    <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.diarize}</p>
                  </HoverCardContent>
                </HoverCard>
              </div>

              {params.diarize && (
                <div className="p-6 border border-gray-200 dark:border-gray-700 rounded-lg bg-gray-50 dark:bg-gray-800 space-y-6">
                  <div>
                    <div className="flex items-center gap-2 mb-2">
                      <Label htmlFor="diarize_model" className="text-gray-700 dark:text-gray-300">Diarization Model</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.diarize_model}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                    <Select
                      value={params.diarize_model}
                      onValueChange={(value) => updateParam('diarize_model', value)}
                    >
                      <SelectTrigger className="mt-3 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                        <SelectItem value="pyannote/speaker-diarization-3.1" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">
                          pyannote/speaker-diarization-3.1 (Recommended)
                        </SelectItem>
                        <SelectItem value="pyannote/speaker-diarization-3.0" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">
                          pyannote/speaker-diarization-3.0 (Stable)
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  
                  <div className="grid grid-cols-2 gap-6">
                    <div>
                      <div className="flex items-center gap-2 mb-2">
                        <Label htmlFor="min_speakers" className="text-gray-700 dark:text-gray-300">Min Speakers</Label>
                        <HoverCard>
                          <HoverCardTrigger asChild>
                            <Info className="h-4 w-4 text-gray-400 cursor-help" />
                          </HoverCardTrigger>
                          <HoverCardContent className="w-64 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                            <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.min_speakers}</p>
                          </HoverCardContent>
                        </HoverCard>
                      </div>
                      <Input
                        type="number"
                        min="1"
                        max="20"
                        placeholder="Auto-detect"
                        value={params.min_speakers || ""}
                        onChange={(e) => updateParam('min_speakers', e.target.value ? parseInt(e.target.value) : undefined)}
                        className="mt-3 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
                      />
                    </div>
                    <div>
                      <div className="flex items-center gap-2 mb-2">
                        <Label htmlFor="max_speakers" className="text-gray-700 dark:text-gray-300">Max Speakers</Label>
                        <HoverCard>
                          <HoverCardTrigger asChild>
                            <Info className="h-4 w-4 text-gray-400 cursor-help" />
                          </HoverCardTrigger>
                          <HoverCardContent className="w-64 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                            <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.max_speakers}</p>
                          </HoverCardContent>
                        </HoverCard>
                      </div>
                      <Input
                        type="number"
                        min="1"
                        max="20"
                        placeholder="Auto-detect"
                        value={params.max_speakers || ""}
                        onChange={(e) => updateParam('max_speakers', e.target.value ? parseInt(e.target.value) : undefined)}
                        className="mt-3 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
                      />
                    </div>
                  </div>

                  <Separator className="my-6" />

                  <div>
                    <div className="flex items-center gap-2 mb-2">
                      <Label htmlFor="hf_token" className="text-gray-700 dark:text-gray-300">Hugging Face Token</Label>
                      <HoverCard>
                        <HoverCardTrigger asChild>
                          <Info className="h-4 w-4 text-gray-400 cursor-help" />
                        </HoverCardTrigger>
                        <HoverCardContent className="w-80 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                          <p className="text-sm text-gray-700 dark:text-gray-300">{PARAM_DESCRIPTIONS.hf_token}</p>
                        </HoverCardContent>
                      </HoverCard>
                    </div>
                    <Input
                      type="password"
                      placeholder="Required for diarization models"
                      value={params.hf_token || ""}
                      onChange={(e) => updateParam('hf_token', e.target.value || undefined)}
                      className="mt-3 bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
                    />
                  </div>
                </div>
              )}

              {!params.diarize && (
                <div className="p-8 text-center border border-gray-200 dark:border-gray-700 rounded-lg bg-gray-50 dark:bg-gray-800">
                  <div className="text-4xl mb-3 opacity-50">🎤</div>
                  <h3 className="text-lg font-medium text-gray-600 dark:text-gray-300 mb-2">
                    Speaker Diarization Disabled
                  </h3>
                  <p className="text-gray-500 dark:text-gray-400 text-sm">
                    Enable speaker diarization to identify and separate different speakers in your audio.
                  </p>
                </div>
              )}
            </div>
          </TabsContent>
        </Tabs>

        <DialogFooter className="gap-2 border-t border-gray-200 dark:border-gray-700 pt-6 mt-8">
          <Button variant="outline" onClick={() => onOpenChange(false)} className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer">
            Cancel
          </Button>
          <Button 
            onClick={handleStartTranscription} 
            disabled={loading || (isProfileMode && !profileName.trim())} 
            className="bg-blue-600 dark:bg-blue-700 hover:bg-blue-700 dark:hover:bg-blue-800 text-white cursor-pointer disabled:cursor-not-allowed"
          >
            {loading 
              ? (isProfileMode ? "Saving..." : "Starting...") 
              : (isProfileMode ? "Save Profile" : "Start Transcription")
            }
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}