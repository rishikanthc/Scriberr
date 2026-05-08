package asrcontract

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const ContractVersionV1 = "asr.provider.v1"

type Capability string

const (
	CapabilityTranscription         Capability = "transcription"
	CapabilityDiarization           Capability = "diarization"
	CapabilitySpeakerIdentification Capability = "speaker_identification"
	CapabilityTranslation           Capability = "translation"
	CapabilityWordTimestamps        Capability = "word_timestamps"
	CapabilitySegmentTimestamps     Capability = "segment_timestamps"
	CapabilityTokenTimestamps       Capability = "token_timestamps"
	CapabilityStreaming             Capability = "streaming"
	CapabilityCustomVocabulary      Capability = "custom_vocabulary"
	CapabilityInitialPrompt         Capability = "initial_prompt"
	CapabilityLanguageDetection     Capability = "language_detection"
	CapabilitySpeakerEmbeddings     Capability = "speaker_embeddings"
)

type Task string

const (
	TaskTranscribe Task = "transcribe"
	TaskTranslate  Task = "translate"
)

type Operation string

const (
	OperationTranscription         Operation = "transcription"
	OperationDiarization           Operation = "diarization"
	OperationSpeakerIdentification Operation = "speaker_identification"
)

type Stage string

const (
	StageAccepted            Stage = "accepted"
	StagePreprocessing       Stage = "preprocessing"
	StagePlanning            Stage = "planning"
	StageChunking            Stage = "chunking"
	StageLoadingDependencies Stage = "loading_dependencies"
	StageLoadingModel        Stage = "loading_model"
	StageRunning             Stage = "running"
	StagePostprocessing      Stage = "postprocessing"
	StageCompleted           Stage = "completed"
	StageFailed              Stage = "failed"
	StageCanceled            Stage = "canceled"
)

type PathMode string

const PathModeMountedFile PathMode = "mounted_file"

type ProviderState string

const (
	ProviderStateStarting  ProviderState = "starting"
	ProviderStateIdle      ProviderState = "idle"
	ProviderStateBusy      ProviderState = "busy"
	ProviderStateDegraded  ProviderState = "degraded"
	ProviderStateUnhealthy ProviderState = "unhealthy"
	ProviderStateStopping  ProviderState = "stopping"
)

type LoadPolicy string

const (
	LoadPolicyAuto    LoadPolicy = "auto"
	LoadPolicyRequire LoadPolicy = "require"
	LoadPolicyReload  LoadPolicy = "reload"
)

type ParameterType string

const (
	ParameterTypeBoolean  ParameterType = "boolean"
	ParameterTypeInteger  ParameterType = "integer"
	ParameterTypeNumber   ParameterType = "number"
	ParameterTypeString   ParameterType = "string"
	ParameterTypeEnum     ParameterType = "enum"
	ParameterTypeDuration ParameterType = "duration"
	ParameterTypePathRef  ParameterType = "path_ref"
)

type ParameterScope string

const (
	ParameterScopeModel       ParameterScope = "model"
	ParameterScopeRuntime     ParameterScope = "runtime"
	ParameterScopeDecoding    ParameterScope = "decoding"
	ParameterScopeChunking    ParameterScope = "chunking"
	ParameterScopeVAD         ParameterScope = "vad"
	ParameterScopeOutput      ParameterScope = "output"
	ParameterScopePostprocess ParameterScope = "postprocess"
)

const (
	CommonParameterRuntimeNumThreads      = "runtime.num_threads"
	CommonParameterDecodingMethod         = "decoding.method"
	CommonParameterChunkingMode           = "chunking.mode"
	CommonParameterChunkingChunkSeconds   = "chunking.chunk_seconds"
	CommonParameterChunkingOverlapSeconds = "chunking.overlap_seconds"
	CommonParameterVADThreshold           = "vad.threshold"
	CommonParameterVADMinSpeechSeconds    = "vad.min_speech_seconds"
	CommonParameterVADMinSilenceSeconds   = "vad.min_silence_seconds"
	CommonParameterVADMaxSpeechSeconds    = "vad.max_speech_seconds"
	CommonParameterVADPaddingSeconds      = "vad.padding_seconds"
	CommonParameterVADMinDurationOn       = "vad.min_duration_on"
	CommonParameterVADMinDurationOff      = "vad.min_duration_off"
	CommonParameterOutputTimestamps       = "output.timestamps"
	CommonParameterOutputWordTimestamps   = "output.word_timestamps"
	CommonParameterOutputTokenTimestamps  = "output.token_timestamps"
	CommonParameterBatchingBatchSize      = "batching.batch_size"
)

type ProviderInfo struct {
	ContractVersion string           `json:"contract_version"`
	Provider        ProviderIdentity `json:"provider"`
	Runtime         RuntimeInfo      `json:"runtime"`
	AudioInput      AudioInputSpec   `json:"audio_input"`
}

type ProviderIdentity struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Vendor  string `json:"vendor,omitempty"`
}

type RuntimeInfo struct {
	DeviceBackends       []string     `json:"device_backends,omitempty"`
	ActiveBackend        string       `json:"active_backend,omitempty"`
	SupportsConcurrent   bool         `json:"supports_concurrent_jobs"`
	MaxConcurrentJobs    int          `json:"max_concurrent_jobs"`
	ProviderCapabilities []Capability `json:"provider_capabilities,omitempty"`
}

type AudioInputSpec struct {
	RequiredSampleRate int      `json:"required_sample_rate"`
	RequiredChannels   int      `json:"required_channels"`
	Formats            []string `json:"formats"`
	PathMode           PathMode `json:"path_mode"`
}

type ModelCard struct {
	ID                   string                  `json:"id"`
	DisplayName          string                  `json:"display_name"`
	Provider             string                  `json:"provider"`
	ModelType            string                  `json:"model_type,omitempty"`
	Version              string                  `json:"version,omitempty"`
	Installed            bool                    `json:"installed"`
	Loaded               bool                    `json:"loaded"`
	Default              bool                    `json:"default"`
	Tasks                []Task                  `json:"tasks,omitempty"`
	Languages            []string                `json:"languages,omitempty"`
	LanguageSupport      *LanguageSupport        `json:"language_support,omitempty"`
	Capabilities         Capabilities            `json:"capabilities"`
	Limits               ModelLimits             `json:"limits,omitempty"`
	ResourceRequirements ResourceRequirements    `json:"resource_requirements,omitempty"`
	Chunking             *ChunkingCapabilities   `json:"chunking,omitempty"`
	Dependencies         []DependencyRequirement `json:"dependencies,omitempty"`
	Artifacts            []ArtifactRequirement   `json:"artifacts,omitempty"`
	ParameterSchema      ParameterSchema         `json:"parameter_schema,omitempty"`
	RecommendedDefaults  map[string]any          `json:"recommended_defaults,omitempty"`
	License              string                  `json:"license,omitempty"`
	SourceURL            string                  `json:"source_url,omitempty"`
	Extensions           map[string]any          `json:"extensions,omitempty"`
}

type ArtifactRequirement struct {
	Key             string `json:"key"`
	Required        bool   `json:"required"`
	ExternalWeights bool   `json:"external_weights,omitempty"`
	Description     string `json:"description,omitempty"`
}

type DependencyRequirement struct {
	ID          string           `json:"id"`
	Required    bool             `json:"required"`
	Description string           `json:"description,omitempty"`
	Activation  []ActivationRule `json:"activation,omitempty"`
}

type ActivationOperator string

const (
	ActivationEquals ActivationOperator = "equals"
)

type ActivationRule struct {
	Parameter string             `json:"parameter"`
	Operator  ActivationOperator `json:"operator"`
	Value     any                `json:"value"`
}

func (m ModelCard) Supports(required ...Capability) bool {
	for _, capability := range required {
		if !m.Capabilities.Supports(capability) {
			return false
		}
	}
	return true
}

type Capabilities struct {
	Transcription         bool            `json:"transcription"`
	Diarization           bool            `json:"diarization"`
	SpeakerIdentification bool            `json:"speaker_identification"`
	Translation           bool            `json:"translation"`
	WordTimestamps        bool            `json:"word_timestamps"`
	SegmentTimestamps     bool            `json:"segment_timestamps"`
	TokenTimestamps       bool            `json:"token_timestamps"`
	Streaming             bool            `json:"streaming"`
	CustomVocabulary      bool            `json:"custom_vocabulary"`
	InitialPrompt         bool            `json:"initial_prompt"`
	LanguageDetection     bool            `json:"language_detection"`
	SpeakerEmbeddings     bool            `json:"speaker_embeddings"`
	Extensions            map[string]bool `json:"extensions,omitempty"`
}

func (c Capabilities) Supports(capability Capability) bool {
	switch capability {
	case CapabilityTranscription:
		return c.Transcription
	case CapabilityDiarization:
		return c.Diarization
	case CapabilitySpeakerIdentification:
		return c.SpeakerIdentification
	case CapabilityTranslation:
		return c.Translation
	case CapabilityWordTimestamps:
		return c.WordTimestamps
	case CapabilitySegmentTimestamps:
		return c.SegmentTimestamps
	case CapabilityTokenTimestamps:
		return c.TokenTimestamps
	case CapabilityStreaming:
		return c.Streaming
	case CapabilityCustomVocabulary:
		return c.CustomVocabulary
	case CapabilityInitialPrompt:
		return c.InitialPrompt
	case CapabilityLanguageDetection:
		return c.LanguageDetection
	case CapabilitySpeakerEmbeddings:
		return c.SpeakerEmbeddings
	default:
		return c.Extensions != nil && c.Extensions[string(capability)]
	}
}

type ModelLimits struct {
	MaxAudioDurationSec *float64 `json:"max_audio_duration_sec,omitempty"`
	RecommendedChunkSec *float64 `json:"recommended_chunk_sec,omitempty"`
}

type ResourceRequirements struct {
	Backends          []string `json:"backends,omitempty"`
	RecommendedVRAMMB *int     `json:"recommended_vram_mb,omitempty"`
	RecommendedRAMMB  *int     `json:"recommended_ram_mb,omitempty"`
}

type LanguageSupport struct {
	Languages []string `json:"languages,omitempty"`
	Mode      string   `json:"mode,omitempty"`
}

type ChunkingCapabilities struct {
	SupportsEngineChunking   bool     `json:"supports_engine_chunking"`
	SupportsProviderChunking bool     `json:"supports_provider_chunking"`
	PreferredMode            string   `json:"preferred_chunking_mode,omitempty"`
	RecommendedChunkSeconds  *float64 `json:"recommended_chunk_seconds,omitempty"`
	MaxChunkSeconds          *float64 `json:"max_chunk_seconds,omitempty"`
	SupportsBatching         bool     `json:"supports_batching"`
	RecommendedBatchSize     *int     `json:"recommended_batch_size,omitempty"`
	MaxBatchSize             *int     `json:"max_batch_size,omitempty"`
}

type ParameterSchema []ParameterDescriptor

type ParameterDescriptor struct {
	Key             string            `json:"key"`
	Label           string            `json:"label,omitempty"`
	Type            ParameterType     `json:"type"`
	Default         any               `json:"default,omitempty"`
	Min             *float64          `json:"min,omitempty"`
	Max             *float64          `json:"max,omitempty"`
	Step            *float64          `json:"step,omitempty"`
	Options         []ParameterOption `json:"options,omitempty"`
	Scope           ParameterScope    `json:"scope"`
	Required        bool              `json:"required,omitempty"`
	Advanced        bool              `json:"advanced,omitempty"`
	ReadOnly        bool              `json:"read_only,omitempty"`
	RequiresReload  bool              `json:"requires_reload,omitempty"`
	ExposeInSummary bool              `json:"expose_in_summary,omitempty"`
	VisibleWhen     []ActivationRule  `json:"visible_when,omitempty"`
}

type ParameterOption struct {
	Value any    `json:"value"`
	Label string `json:"label,omitempty"`
}

type ParameterValueError struct {
	Parameter string
	Reason    string
	Err       error
}

func (e *ParameterValueError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("parameter %q %s: %v", e.Parameter, e.Reason, e.Err)
	}
	return fmt.Sprintf("parameter %q %s", e.Parameter, e.Reason)
}

func (e *ParameterValueError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func ValidateModelCard(card ModelCard) error {
	if strings.TrimSpace(card.ID) == "" {
		return fmt.Errorf("model id is required")
	}
	if strings.TrimSpace(card.Provider) == "" {
		return fmt.Errorf("model provider is required")
	}
	if err := ValidateParameterSchema(card.ParameterSchema); err != nil {
		return err
	}
	if err := validateDependencyActivation(card.ParameterSchema, card.Dependencies); err != nil {
		return err
	}
	return nil
}

func ValidateParameterSchema(schema ParameterSchema) error {
	seen := map[string]struct{}{}
	for _, parameter := range schema {
		key := strings.TrimSpace(parameter.Key)
		if key == "" {
			return fmt.Errorf("parameter key is required")
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("parameter %q is duplicated", key)
		}
		seen[key] = struct{}{}
		if !isCommonParameterKey(key) && !strings.Contains(key, ".") {
			return fmt.Errorf("provider-specific parameter %q must be namespaced", key)
		}
		if !validParameterType(parameter.Type) {
			return fmt.Errorf("parameter %q type is invalid", key)
		}
		if !validParameterScope(parameter.Scope) {
			return fmt.Errorf("parameter %q scope is invalid", key)
		}
		if parameter.Min != nil && parameter.Max != nil && *parameter.Min > *parameter.Max {
			return fmt.Errorf("parameter %q min cannot exceed max", key)
		}
		if parameter.Step != nil && *parameter.Step <= 0 {
			return fmt.Errorf("parameter %q step must be positive", key)
		}
		if parameter.Type == ParameterTypeEnum && len(parameter.Options) == 0 {
			return fmt.Errorf("parameter %q enum options are required", key)
		}
		for _, option := range parameter.Options {
			if option.Value == nil {
				return fmt.Errorf("parameter %q enum option value is required", key)
			}
		}
		if parameter.Default != nil {
			if _, err := validateParameterValue(parameter, parameter.Default); err != nil {
				return fmt.Errorf("parameter %q default is invalid: %w", key, err)
			}
		}
	}
	for _, parameter := range schema {
		key := strings.TrimSpace(parameter.Key)
		for _, rule := range parameter.VisibleWhen {
			if err := validateActivationRule(seen, rule); err != nil {
				return fmt.Errorf("parameter %q visibility rule is invalid: %w", key, err)
			}
		}
	}
	return nil
}

func validateDependencyActivation(schema ParameterSchema, dependencies []DependencyRequirement) error {
	if len(dependencies) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(schema))
	for _, parameter := range schema {
		seen[strings.TrimSpace(parameter.Key)] = struct{}{}
	}
	for _, dependency := range dependencies {
		id := strings.TrimSpace(dependency.ID)
		if id == "" {
			return fmt.Errorf("dependency id is required")
		}
		for _, rule := range dependency.Activation {
			if err := validateActivationRule(seen, rule); err != nil {
				return fmt.Errorf("dependency %q activation rule is invalid: %w", id, err)
			}
		}
	}
	return nil
}

func validateActivationRule(parameters map[string]struct{}, rule ActivationRule) error {
	key := strings.TrimSpace(rule.Parameter)
	if key == "" {
		return fmt.Errorf("parameter is required")
	}
	if _, ok := parameters[key]; !ok {
		return fmt.Errorf("parameter %q is not declared", key)
	}
	if rule.Operator != ActivationEquals {
		return fmt.Errorf("operator %q is not supported", rule.Operator)
	}
	if rule.Value == nil {
		return fmt.Errorf("value is required")
	}
	return nil
}

func ValidateParameterValues(schema ParameterSchema, values map[string]any) (map[string]any, error) {
	if err := ValidateParameterSchema(schema); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}
	byKey := make(map[string]ParameterDescriptor, len(schema))
	for _, parameter := range schema {
		byKey[strings.TrimSpace(parameter.Key)] = parameter
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		parameter, ok := byKey[key]
		if !ok {
			return nil, &ParameterValueError{Parameter: key, Reason: "is not supported"}
		}
		normalized, err := validateParameterValue(parameter, value)
		if err != nil {
			return nil, &ParameterValueError{Parameter: key, Reason: "is invalid", Err: err}
		}
		if parameter.ReadOnly && !readOnlyValueAllowed(parameter, normalized) {
			return nil, &ParameterValueError{Parameter: key, Reason: "is read-only"}
		}
		out[key] = normalized
	}
	return out, nil
}

func readOnlyValueAllowed(parameter ParameterDescriptor, value any) bool {
	if parameter.Default == nil {
		return false
	}
	normalizedDefault, err := validateParameterValue(parameter, parameter.Default)
	if err != nil {
		return false
	}
	return parameterValuesEqual(normalizedDefault, value)
}

func parameterValuesEqual(left any, right any) bool {
	switch typed := left.(type) {
	case string:
		rightString, ok := right.(string)
		return ok && typed == rightString
	case bool:
		rightBool, ok := right.(bool)
		return ok && typed == rightBool
	default:
		leftNumber, leftOK := numericValue(left)
		rightNumber, rightOK := numericValue(right)
		return leftOK && rightOK && leftNumber == rightNumber
	}
}

func validateParameterValue(parameter ParameterDescriptor, value any) (any, error) {
	switch parameter.Type {
	case ParameterTypeBoolean:
		if typed, ok := value.(bool); ok {
			return typed, nil
		}
	case ParameterTypeInteger:
		number, ok := numericValue(value)
		if !ok || math.Trunc(number) != number {
			return nil, fmt.Errorf("must be an integer")
		}
		if err := validateNumberBounds(parameter, number); err != nil {
			return nil, err
		}
		return int64(number), nil
	case ParameterTypeNumber, ParameterTypeDuration:
		number, ok := numericValue(value)
		if !ok {
			return nil, fmt.Errorf("must be a number")
		}
		if err := validateNumberBounds(parameter, number); err != nil {
			return nil, err
		}
		return number, nil
	case ParameterTypeString, ParameterTypePathRef:
		if typed, ok := value.(string); ok {
			return strings.TrimSpace(typed), nil
		}
	case ParameterTypeEnum:
		for _, option := range parameter.Options {
			if optionValueEqual(option.Value, value) {
				return value, nil
			}
		}
		return nil, fmt.Errorf("must be one of the declared enum options")
	}
	return nil, fmt.Errorf("wrong type")
}

func validateNumberBounds(parameter ParameterDescriptor, value float64) error {
	if parameter.Min != nil && value < *parameter.Min {
		return fmt.Errorf("must be at least %v", *parameter.Min)
	}
	if parameter.Max != nil && value > *parameter.Max {
		return fmt.Errorf("must be at most %v", *parameter.Max)
	}
	return nil
}

func numericValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	default:
		return 0, false
	}
}

func optionValueEqual(option any, value any) bool {
	switch typed := option.(type) {
	case string:
		valueString, ok := value.(string)
		return ok && typed == valueString
	case bool:
		valueBool, ok := value.(bool)
		return ok && typed == valueBool
	default:
		optionNumber, optionOK := numericValue(option)
		valueNumber, valueOK := numericValue(value)
		return optionOK && valueOK && optionNumber == valueNumber
	}
}

func validParameterType(value ParameterType) bool {
	switch value {
	case ParameterTypeBoolean, ParameterTypeInteger, ParameterTypeNumber, ParameterTypeString, ParameterTypeEnum, ParameterTypeDuration, ParameterTypePathRef:
		return true
	default:
		return false
	}
}

func validParameterScope(value ParameterScope) bool {
	switch value {
	case ParameterScopeModel, ParameterScopeRuntime, ParameterScopeDecoding, ParameterScopeChunking, ParameterScopeVAD, ParameterScopeOutput, ParameterScopePostprocess:
		return true
	default:
		return false
	}
}

func isCommonParameterKey(key string) bool {
	switch key {
	case CommonParameterRuntimeNumThreads,
		CommonParameterDecodingMethod,
		CommonParameterChunkingMode,
		CommonParameterChunkingChunkSeconds,
		CommonParameterChunkingOverlapSeconds,
		CommonParameterVADThreshold,
		CommonParameterVADMinSpeechSeconds,
		CommonParameterVADMinSilenceSeconds,
		CommonParameterVADMaxSpeechSeconds,
		CommonParameterVADPaddingSeconds,
		CommonParameterVADMinDurationOn,
		CommonParameterVADMinDurationOff,
		CommonParameterOutputTimestamps,
		CommonParameterOutputWordTimestamps,
		CommonParameterOutputTokenTimestamps,
		CommonParameterBatchingBatchSize:
		return true
	default:
		return false
	}
}

type ProviderStatus struct {
	State        ProviderState    `json:"state"`
	ActiveJob    *ActiveJob       `json:"active_job,omitempty"`
	LoadedModels []LoadedModel    `json:"loaded_models,omitempty"`
	Capacity     ProviderCapacity `json:"capacity"`
}

type ActiveJob struct {
	ID        string    `json:"id"`
	Operation Operation `json:"operation"`
	Model     string    `json:"model"`
	Stage     Stage     `json:"stage"`
	Progress  *float64  `json:"progress,omitempty"`
}

type LoadedModel struct {
	ID             string     `json:"id"`
	ResourceKind   string     `json:"resource_kind,omitempty"`
	ResourceRole   string     `json:"resource_role,omitempty"`
	RuntimeBackend string     `json:"runtime_backend,omitempty"`
	Threads        int        `json:"threads,omitempty"`
	ReloadKey      string     `json:"reload_key,omitempty"`
	LoadedAt       *time.Time `json:"loaded_at,omitempty"`
	MemoryMB       *int       `json:"memory_mb,omitempty"`
}

type ProviderCapacity struct {
	MaxConcurrentJobs int `json:"max_concurrent_jobs"`
	AvailableSlots    int `json:"available_slots"`
}

type LoadModelRequest struct {
	Model      string         `json:"model"`
	Operation  Operation      `json:"operation,omitempty"`
	LoadPolicy LoadPolicy     `json:"load_policy,omitempty"`
	Options    map[string]any `json:"options,omitempty"`
}

type UnloadModelRequest struct {
	Model   string         `json:"model"`
	Force   bool           `json:"force,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

type AudioInput struct {
	Path        string   `json:"path"`
	SampleRate  int      `json:"sample_rate"`
	Channels    int      `json:"channels"`
	Format      string   `json:"format"`
	DurationSec *float64 `json:"duration_sec,omitempty"`
}

type TranscriptionRequest struct {
	RequestID  string         `json:"request_id"`
	Audio      AudioInput     `json:"audio"`
	Model      string         `json:"model"`
	LoadPolicy LoadPolicy     `json:"load_policy,omitempty"`
	Task       Task           `json:"task,omitempty"`
	Language   string         `json:"language,omitempty"`
	Features   Capabilities   `json:"features,omitempty"`
	Options    map[string]any `json:"options,omitempty"`
}

type DiarizationRequest struct {
	RequestID  string         `json:"request_id"`
	Audio      AudioInput     `json:"audio"`
	Model      string         `json:"model"`
	LoadPolicy LoadPolicy     `json:"load_policy,omitempty"`
	Inputs     []string       `json:"inputs,omitempty"`
	Options    map[string]any `json:"options,omitempty"`
}

type SpeakerIDRequest struct {
	RequestID  string         `json:"request_id"`
	Audio      AudioInput     `json:"audio"`
	Model      string         `json:"model"`
	LoadPolicy LoadPolicy     `json:"load_policy,omitempty"`
	Inputs     []string       `json:"inputs,omitempty"`
	Options    map[string]any `json:"options,omitempty"`
}

type ProviderProgress struct {
	Stage     Stage     `json:"stage"`
	Progress  *float64  `json:"progress,omitempty"`
	Message   string    `json:"message,omitempty"`
	Operation Operation `json:"operation,omitempty"`
	Model     string    `json:"model,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type TranscriptionResult struct {
	Model    string              `json:"model"`
	Language string              `json:"language,omitempty"`
	Text     string              `json:"text"`
	Segments []TranscriptSegment `json:"segments"`
	Words    []TranscriptWord    `json:"words"`
	Metadata map[string]any      `json:"metadata,omitempty"`
}

type TranscriptSegment struct {
	ID         string   `json:"id,omitempty"`
	Start      float64  `json:"start"`
	End        float64  `json:"end"`
	Speaker    string   `json:"speaker,omitempty"`
	Text       string   `json:"text"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type TranscriptWord struct {
	Start      float64  `json:"start"`
	End        float64  `json:"end"`
	Word       string   `json:"word"`
	Speaker    string   `json:"speaker,omitempty"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type DiarizationResult struct {
	Model    string               `json:"model"`
	Segments []DiarizationSegment `json:"segments"`
	Metadata map[string]any       `json:"metadata,omitempty"`
}

type DiarizationSegment struct {
	Start      float64  `json:"start"`
	End        float64  `json:"end"`
	Speaker    string   `json:"speaker"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type SpeakerIDResult struct {
	Model    string            `json:"model"`
	Speakers []SpeakerIdentity `json:"speakers"`
	Metadata map[string]any    `json:"metadata,omitempty"`
}

type SpeakerIdentity struct {
	Speaker    string         `json:"speaker"`
	Label      string         `json:"label,omitempty"`
	Confidence *float64       `json:"confidence,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type ErrorCode string

const (
	CodeInvalidRequest        ErrorCode = "INVALID_REQUEST"
	CodeUnsupportedOperation  ErrorCode = "UNSUPPORTED_OPERATION"
	CodeUnsupportedFeature    ErrorCode = "UNSUPPORTED_FEATURE"
	CodeUnsupportedModel      ErrorCode = "UNSUPPORTED_MODEL"
	CodeModelNotInstalled     ErrorCode = "MODEL_NOT_INSTALLED"
	CodeAudioNotFound         ErrorCode = "AUDIO_NOT_FOUND"
	CodeAudioInvalid          ErrorCode = "AUDIO_INVALID"
	CodeInsufficientResources ErrorCode = "INSUFFICIENT_RESOURCES"
	CodeProviderBusy          ErrorCode = "PROVIDER_BUSY"
	CodeProviderUnhealthy     ErrorCode = "PROVIDER_UNHEALTHY"
	CodeInferenceFailed       ErrorCode = "INFERENCE_FAILED"
	CodeCanceled              ErrorCode = "CANCELED"
	CodeTimeout               ErrorCode = "TIMEOUT"
)

type ProviderError struct {
	Code      ErrorCode      `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

func NewProviderError(code ErrorCode, message string, retryable bool) *ProviderError {
	return &ProviderError{Code: code, Message: message, Retryable: retryable}
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return e.Message
	}
	if e.Message == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func IsCode(err error, code ErrorCode) bool {
	var providerErr *ProviderError
	return errors.As(err, &providerErr) && providerErr.Code == code
}

func Retryable(err error) bool {
	var providerErr *ProviderError
	return errors.As(err, &providerErr) && providerErr.Retryable
}
