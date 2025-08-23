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

interface TranscriptionConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onStartTranscription: (params: WhisperXParams) => void;
  loading?: boolean;
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
}: TranscriptionConfigDialogProps) {
  const [params, setParams] = useState<WhisperXParams>(DEFAULT_PARAMS);

  // Reset to defaults when dialog opens
  useEffect(() => {
    if (open) {
      setParams(DEFAULT_PARAMS);
    }
  }, [open]);

  const updateParam = <K extends keyof WhisperXParams>(
    key: K,
    value: WhisperXParams[K]
  ) => {
    setParams(prev => ({ ...prev, [key]: value }));
  };

  const handleStartTranscription = () => {
    onStartTranscription(params);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700">
        <DialogHeader>
          <DialogTitle className="text-gray-900 dark:text-gray-100">Transcription Configuration</DialogTitle>
          <DialogDescription className="text-gray-600 dark:text-gray-400">
            Configure WhisperX parameters for your transcription. Advanced settings allow fine-tuning quality and performance.
          </DialogDescription>
        </DialogHeader>

        <Tabs defaultValue="basic" className="w-full">
          <TabsList className="grid w-full grid-cols-4 bg-gray-100 dark:bg-gray-800">
            <TabsTrigger value="basic" className="data-[state=active]:bg-white data-[state=active]:dark:bg-gray-700 text-gray-700 dark:text-gray-300">Basic</TabsTrigger>
            <TabsTrigger value="quality" className="data-[state=active]:bg-white data-[state=active]:dark:bg-gray-700 text-gray-700 dark:text-gray-300">Quality</TabsTrigger>
            <TabsTrigger value="advanced" className="data-[state=active]:bg-white data-[state=active]:dark:bg-gray-700 text-gray-700 dark:text-gray-300">Advanced</TabsTrigger>
            <TabsTrigger value="output" className="data-[state=active]:bg-white data-[state=active]:dark:bg-gray-700 text-gray-700 dark:text-gray-300">Output</TabsTrigger>
          </TabsList>

          <TabsContent value="basic" className="space-y-6">
            <div className="grid grid-cols-2 gap-6">
              <div className="space-y-4">
                <div>
                  <Label htmlFor="model" className="text-gray-700 dark:text-gray-300">Model Size</Label>
                  <Select
                    value={params.model}
                    onValueChange={(value) => updateParam('model', value)}
                  >
                    <SelectTrigger className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
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
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Larger models are more accurate but slower
                  </p>
                </div>

                <div>
                  <Label htmlFor="language" className="text-gray-700 dark:text-gray-300">Language</Label>
                  <Select
                    value={params.language || "auto"}
                    onValueChange={(value) => updateParam('language', value === "auto" ? undefined : value)}
                  >
                    <SelectTrigger className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
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
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Leave as auto-detect for automatic language detection
                  </p>
                </div>

                <div>
                  <Label htmlFor="task" className="text-gray-700 dark:text-gray-300">Task</Label>
                  <Select
                    value={params.task}
                    onValueChange={(value) => updateParam('task', value)}
                  >
                    <SelectTrigger className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                      <SelectItem value="transcribe" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">Transcribe</SelectItem>
                      <SelectItem value="translate" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">Translate to English</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <Label htmlFor="device" className="text-gray-700 dark:text-gray-300">Device</Label>
                  <Select
                    value={params.device}
                    onValueChange={(value) => updateParam('device', value)}
                  >
                    <SelectTrigger className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
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
                  <Label htmlFor="compute_type" className="text-gray-700 dark:text-gray-300">Compute Type</Label>
                  <Select
                    value={params.compute_type}
                    onValueChange={(value) => updateParam('compute_type', value)}
                  >
                    <SelectTrigger className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
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
                  <div className="flex items-center justify-between">
                    <Label htmlFor="batch_size" className="text-gray-700 dark:text-gray-300">Batch Size: {params.batch_size}</Label>
                  </div>
                  <Slider
                    value={[params.batch_size]}
                    onValueChange={([value]) => updateParam('batch_size', value)}
                    min={1}
                    max={32}
                    step={1}
                    className="mt-2"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Higher values are faster but use more memory
                  </p>
                </div>
              </div>
            </div>

            <Separator className="bg-gray-200 dark:bg-gray-700" />

            <div>
              <div className="flex items-center space-x-2">
                <Switch
                  id="diarize"
                  checked={params.diarize}
                  onCheckedChange={(checked) => updateParam('diarize', checked)}
                />
                <Label htmlFor="diarize" className="text-gray-700 dark:text-gray-300">Speaker Diarization</Label>
              </div>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                Identify different speakers in the audio
              </p>

              {params.diarize && (
                <div className="grid grid-cols-2 gap-4 mt-4 p-4 border border-gray-200 dark:border-gray-700 rounded-lg bg-gray-50 dark:bg-gray-800">
                  <div>
                    <Label htmlFor="min_speakers" className="text-gray-700 dark:text-gray-300">Min Speakers</Label>
                    <Input
                      type="number"
                      min="1"
                      max="20"
                      value={params.min_speakers || ""}
                      onChange={(e) => updateParam('min_speakers', e.target.value ? parseInt(e.target.value) : undefined)}
                      className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
                    />
                  </div>
                  <div>
                    <Label htmlFor="max_speakers" className="text-gray-700 dark:text-gray-300">Max Speakers</Label>
                    <Input
                      type="number"
                      min="1"
                      max="20"
                      value={params.max_speakers || ""}
                      onChange={(e) => updateParam('max_speakers', e.target.value ? parseInt(e.target.value) : undefined)}
                      className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100"
                    />
                  </div>
                </div>
              )}
            </div>
          </TabsContent>

          <TabsContent value="quality" className="space-y-6">
            <div className="grid grid-cols-2 gap-6">
              <div className="space-y-4">
                <div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="temperature" className="text-gray-700 dark:text-gray-300">Temperature: {params.temperature}</Label>
                  </div>
                  <Slider
                    value={[params.temperature]}
                    onValueChange={([value]) => updateParam('temperature', value)}
                    min={0}
                    max={1}
                    step={0.1}
                    className="mt-2"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Higher values increase randomness in output
                  </p>
                </div>

                <div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="beam_size" className="text-gray-700 dark:text-gray-300">Beam Size: {params.beam_size}</Label>
                  </div>
                  <Slider
                    value={[params.beam_size]}
                    onValueChange={([value]) => updateParam('beam_size', value)}
                    min={1}
                    max={20}
                    step={1}
                    className="mt-2"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Number of beams for beam search (higher = better quality)
                  </p>
                </div>

                <div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="best_of" className="text-gray-700 dark:text-gray-300">Best Of: {params.best_of}</Label>
                  </div>
                  <Slider
                    value={[params.best_of]}
                    onValueChange={([value]) => updateParam('best_of', value)}
                    min={1}
                    max={20}
                    step={1}
                    className="mt-2"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Number of candidates when sampling
                  </p>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="patience" className="text-gray-700 dark:text-gray-300">Patience: {params.patience}</Label>
                  </div>
                  <Slider
                    value={[params.patience]}
                    onValueChange={([value]) => updateParam('patience', value)}
                    min={0.1}
                    max={3.0}
                    step={0.1}
                    className="mt-2"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Patience value for beam decoding
                  </p>
                </div>

                <div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="length_penalty" className="text-gray-700 dark:text-gray-300">Length Penalty: {params.length_penalty}</Label>
                  </div>
                  <Slider
                    value={[params.length_penalty]}
                    onValueChange={([value]) => updateParam('length_penalty', value)}
                    min={0.1}
                    max={3.0}
                    step={0.1}
                    className="mt-2"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Token length penalty coefficient
                  </p>
                </div>

                <div>
                  <Label htmlFor="initial_prompt" className="text-gray-700 dark:text-gray-300">Initial Prompt</Label>
                  <Textarea
                    placeholder="Optional text to provide as context for the first window"
                    value={params.initial_prompt || ""}
                    onChange={(e) => updateParam('initial_prompt', e.target.value || undefined)}
                    rows={3}
                    className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 placeholder:text-gray-500 dark:placeholder:text-gray-400"
                  />
                </div>
              </div>
            </div>

            <Separator className="bg-gray-200 dark:bg-gray-700" />

            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <Switch
                  id="suppress_numerals"
                  checked={params.suppress_numerals}
                  onCheckedChange={(checked) => updateParam('suppress_numerals', checked)}
                />
                <Label htmlFor="suppress_numerals" className="text-gray-700 dark:text-gray-300">Suppress Numerals</Label>
              </div>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                Suppress numeric symbols and currency symbols during sampling
              </p>

              <div className="flex items-center space-x-2">
                <Switch
                  id="condition_on_previous_text"
                  checked={params.condition_on_previous_text}
                  onCheckedChange={(checked) => updateParam('condition_on_previous_text', checked)}
                />
                <Label htmlFor="condition_on_previous_text" className="text-gray-700 dark:text-gray-300">Condition on Previous Text</Label>
              </div>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                Use previous output as prompt for next window (may cause loops)
              </p>
            </div>
          </TabsContent>

          <TabsContent value="advanced" className="space-y-6">
            <div className="grid grid-cols-2 gap-6">
              <div className="space-y-4">
                <h4 className="font-medium text-gray-900 dark:text-gray-100">VAD Settings</h4>
                
                <div>
                  <Label htmlFor="vad_method" className="text-gray-700 dark:text-gray-300">VAD Method</Label>
                  <Select
                    value={params.vad_method}
                    onValueChange={(value) => updateParam('vad_method', value)}
                  >
                    <SelectTrigger className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                      <SelectItem value="pyannote" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">Pyannote</SelectItem>
                      <SelectItem value="silero" className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700">Silero</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="vad_onset">VAD Onset: {params.vad_onset}</Label>
                  </div>
                  <Slider
                    value={[params.vad_onset]}
                    onValueChange={([value]) => updateParam('vad_onset', value)}
                    min={0.1}
                    max={1.0}
                    step={0.01}
                    className="mt-2"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Onset threshold for VAD (reduce if speech not detected)
                  </p>
                </div>

                <div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="vad_offset">VAD Offset: {params.vad_offset}</Label>
                  </div>
                  <Slider
                    value={[params.vad_offset]}
                    onValueChange={([value]) => updateParam('vad_offset', value)}
                    min={0.1}
                    max={1.0}
                    step={0.01}
                    className="mt-2"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Offset threshold for VAD
                  </p>
                </div>

                <div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="chunk_size">Chunk Size: {params.chunk_size}</Label>
                  </div>
                  <Slider
                    value={[params.chunk_size]}
                    onValueChange={([value]) => updateParam('chunk_size', value)}
                    min={5}
                    max={60}
                    step={1}
                    className="mt-2"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Chunk size for merging VAD segments (seconds)
                  </p>
                </div>
              </div>

              <div className="space-y-4">
                <h4 className="font-medium">Detection Thresholds</h4>
                
                <div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="compression_ratio_threshold">Compression Ratio: {params.compression_ratio_threshold}</Label>
                  </div>
                  <Slider
                    value={[params.compression_ratio_threshold]}
                    onValueChange={([value]) => updateParam('compression_ratio_threshold', value)}
                    min={1.0}
                    max={5.0}
                    step={0.1}
                    className="mt-2"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Fail if compression ratio is higher than this
                  </p>
                </div>

                <div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="logprob_threshold">Log Probability: {params.logprob_threshold}</Label>
                  </div>
                  <Slider
                    value={[params.logprob_threshold]}
                    onValueChange={([value]) => updateParam('logprob_threshold', value)}
                    min={-3.0}
                    max={0.0}
                    step={0.1}
                    className="mt-2"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Fail if average log probability is lower than this
                  </p>
                </div>

                <div>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="no_speech_threshold">No Speech: {params.no_speech_threshold}</Label>
                  </div>
                  <Slider
                    value={[params.no_speech_threshold]}
                    onValueChange={([value]) => updateParam('no_speech_threshold', value)}
                    min={0.1}
                    max={1.0}
                    step={0.01}
                    className="mt-2"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Consider segment as silence if probability is higher
                  </p>
                </div>

                <div>
                  <Label htmlFor="suppress_tokens">Suppress Tokens</Label>
                  <Input
                    placeholder="-1 (default: suppress special characters)"
                    value={params.suppress_tokens || ""}
                    onChange={(e) => updateParam('suppress_tokens', e.target.value || undefined)}
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Comma-separated token IDs to suppress
                  </p>
                </div>
              </div>
            </div>

            <Separator />

            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <Switch
                  id="no_align"
                  checked={params.no_align}
                  onCheckedChange={(checked) => updateParam('no_align', checked)}
                />
                <Label htmlFor="no_align">Disable Alignment</Label>
              </div>
              <p className="text-xs text-muted-foreground">
                Skip phoneme-level alignment (faster but less precise timestamps)
              </p>

              <div className="flex items-center space-x-2">
                <Switch
                  id="return_char_alignments"
                  checked={params.return_char_alignments}
                  onCheckedChange={(checked) => updateParam('return_char_alignments', checked)}
                />
                <Label htmlFor="return_char_alignments">Character-level Alignments</Label>
              </div>
              <p className="text-xs text-muted-foreground">
                Return character-level alignments in output
              </p>

              <div className="flex items-center space-x-2">
                <Switch
                  id="fp16"
                  checked={params.fp16}
                  onCheckedChange={(checked) => updateParam('fp16', checked)}
                />
                <Label htmlFor="fp16">FP16 Inference</Label>
              </div>
              <p className="text-xs text-muted-foreground">
                Use 16-bit floating point for inference (faster, less accurate)
              </p>
            </div>
          </TabsContent>

          <TabsContent value="output" className="space-y-6">
            <div className="grid grid-cols-2 gap-6">
              <div className="space-y-4">
                <div>
                  <Label htmlFor="output_format">Output Format</Label>
                  <Select
                    value={params.output_format}
                    onValueChange={(value) => updateParam('output_format', value)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All formats</SelectItem>
                      <SelectItem value="srt">SRT</SelectItem>
                      <SelectItem value="vtt">VTT</SelectItem>
                      <SelectItem value="txt">TXT</SelectItem>
                      <SelectItem value="json">JSON</SelectItem>
                      <SelectItem value="tsv">TSV</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <Label htmlFor="segment_resolution">Segment Resolution</Label>
                  <Select
                    value={params.segment_resolution}
                    onValueChange={(value) => updateParam('segment_resolution', value)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="sentence">Sentence</SelectItem>
                      <SelectItem value="chunk">Chunk</SelectItem>
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground mt-1">
                    How to break up segments in the output
                  </p>
                </div>

                <div>
                  <Label htmlFor="max_line_width">Max Line Width</Label>
                  <Input
                    type="number"
                    min="10"
                    max="200"
                    placeholder="No limit"
                    value={params.max_line_width || ""}
                    onChange={(e) => updateParam('max_line_width', e.target.value ? parseInt(e.target.value) : undefined)}
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Maximum characters per line before breaking
                  </p>
                </div>

                <div>
                  <Label htmlFor="max_line_count">Max Line Count</Label>
                  <Input
                    type="number"
                    min="1"
                    max="20"
                    placeholder="No limit"
                    value={params.max_line_count || ""}
                    onChange={(e) => updateParam('max_line_count', e.target.value ? parseInt(e.target.value) : undefined)}
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Maximum lines per segment
                  </p>
                </div>
              </div>

              <div className="space-y-4">
                <div className="flex items-center space-x-2">
                  <Switch
                    id="highlight_words"
                    checked={params.highlight_words}
                    onCheckedChange={(checked) => updateParam('highlight_words', checked)}
                  />
                  <Label htmlFor="highlight_words">Highlight Words</Label>
                </div>
                <p className="text-xs text-muted-foreground">
                  Underline each word as it's spoken in SRT/VTT
                </p>

                <div className="flex items-center space-x-2">
                  <Switch
                    id="verbose"
                    checked={params.verbose}
                    onCheckedChange={(checked) => updateParam('verbose', checked)}
                  />
                  <Label htmlFor="verbose">Verbose Output</Label>
                </div>
                <p className="text-xs text-muted-foreground">
                  Show progress and debug messages
                </p>

                <div className="flex items-center space-x-2">
                  <Switch
                    id="print_progress"
                    checked={params.print_progress}
                    onCheckedChange={(checked) => updateParam('print_progress', checked)}
                  />
                  <Label htmlFor="print_progress">Print Progress</Label>
                </div>
                <p className="text-xs text-muted-foreground">
                  Print progress information during processing
                </p>

                <Separator />

                <div>
                  <Label htmlFor="hf_token">Hugging Face Token</Label>
                  <Input
                    type="password"
                    placeholder="Optional HF token for private models"
                    value={params.hf_token || ""}
                    onChange={(e) => updateParam('hf_token', e.target.value || undefined)}
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Required for some private models
                  </p>
                </div>
              </div>
            </div>
          </TabsContent>
        </Tabs>

        <DialogFooter className="gap-2 border-t border-gray-200 dark:border-gray-700 pt-4">
          <Button variant="outline" onClick={() => onOpenChange(false)} className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700">
            Cancel
          </Button>
          <Button onClick={handleStartTranscription} disabled={loading} className="bg-blue-600 dark:bg-blue-700 hover:bg-blue-700 dark:hover:bg-blue-800 text-white">
            {loading ? "Starting..." : "Start Transcription"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}