package adapters

import (
	"context"
	"fmt"

	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/binaries"
	"scriberr/pkg/logger"

	"golang.org/x/sync/singleflight"
)

// Environment readiness cache to avoid repeated expensive UV checks
var (
	envCacheMutex sync.RWMutex
	envCache      = make(map[string]bool)
	requestGroup  singleflight.Group
)

// GetPyTorchCUDAVersion returns the PyTorch CUDA wheel version to use.
// This is configurable via the PYTORCH_CUDA_VERSION environment variable.
// Defaults to "cu126" for CUDA 12.6 (legacy GPUs: GTX 10-series through RTX 40-series).
// Set to "cu128" for CUDA 12.8 (Blackwell GPUs: RTX 50-series).
func GetPyTorchCUDAVersion() string {
	if cudaVersion := os.Getenv("PYTORCH_CUDA_VERSION"); cudaVersion != "" {
		return cudaVersion
	}
	return "cu126" // Default to CUDA 12.6 for legacy compatibility
}

// GetPyTorchWheelURL returns the full PyTorch wheel URL for the configured CUDA version.
func GetPyTorchWheelURL() string {
	return fmt.Sprintf("https://download.pytorch.org/whl/%s", GetPyTorchCUDAVersion())
}

// CheckEnvironmentReady checks if a UV environment is ready with caching and singleflight
func CheckEnvironmentReady(envPath, importStatement string) bool {
	cacheKey := fmt.Sprintf("%s:%s", envPath, importStatement)

	// Check cache first
	envCacheMutex.RLock()
	if ready, exists := envCache[cacheKey]; exists {
		envCacheMutex.RUnlock()
		return ready
	}
	envCacheMutex.RUnlock()

	// Use singleflight to prevent duplicate checks
	result, _, _ := requestGroup.Do(cacheKey, func() (interface{}, error) {
		// Check cache again (double-checked locking)
		envCacheMutex.RLock()
		if ready, exists := envCache[cacheKey]; exists {
			envCacheMutex.RUnlock()
			return ready, nil
		}
		envCacheMutex.RUnlock()

		// Run the actual check
		testCmd := exec.Command(binaries.UV(), "run", "--native-tls", "--project", envPath, "python", "-c", importStatement)
		ready := testCmd.Run() == nil

		// Cache the result
		envCacheMutex.Lock()
		envCache[cacheKey] = ready
		envCacheMutex.Unlock()

		return ready, nil
	})

	return result.(bool)
}

// BaseAdapter provides common functionality for all model adapters
type BaseAdapter struct {
	modelID      string
	modelPath    string
	capabilities interfaces.ModelCapabilities
	schema       []interfaces.ParameterSchema
	initialized  bool
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(modelID, modelPath string, capabilities interfaces.ModelCapabilities, schema []interfaces.ParameterSchema) *BaseAdapter {
	return &BaseAdapter{
		modelID:      modelID,
		modelPath:    modelPath,
		capabilities: capabilities,
		schema:       schema,
		initialized:  false,
	}
}

// GetCapabilities returns the model capabilities
func (b *BaseAdapter) GetCapabilities() interfaces.ModelCapabilities {
	return b.capabilities
}

// GetParameterSchema returns the parameter schema
func (b *BaseAdapter) GetParameterSchema() []interfaces.ParameterSchema {
	return b.schema
}

// GetModelPath returns the model file path
func (b *BaseAdapter) GetModelPath() string {
	return b.modelPath
}

// ValidateParameters validates the provided parameters against the schema
func (b *BaseAdapter) ValidateParameters(params map[string]interface{}) error {
	logger.Info("Validating parameters for model", "model_id", b.modelID, "param_count", len(params))

	// Check for required parameters
	for _, paramSchema := range b.schema {
		value, exists := params[paramSchema.Name]

		if paramSchema.Required && !exists {
			return fmt.Errorf("required parameter missing: %s", paramSchema.Name)
		}

		if !exists {
			continue // Optional parameter not provided
		}

		if err := b.validateParameterValue(paramSchema, value); err != nil {
			return fmt.Errorf("invalid value for parameter %s: %w", paramSchema.Name, err)
		}
	}

	// Check for unknown parameters
	for paramName := range params {
		found := false
		for _, paramSchema := range b.schema {
			if paramSchema.Name == paramName {
				found = true
				break
			}
		}
		if !found {
			logger.Warn("Unknown parameter provided", "model_id", b.modelID, "parameter", paramName)
		}
	}

	return nil
}

// validateParameterValue validates a single parameter value against its schema
//
//nolint:gocyclo // Switch case with type checking is complex
func (b *BaseAdapter) validateParameterValue(schema interfaces.ParameterSchema, value interface{}) error {
	// Type validation
	switch schema.Type {
	case "int":
		intVal, err := b.convertToInt(value)
		if err != nil {
			return fmt.Errorf("expected int, got %T", value)
		}
		if schema.Min != nil && float64(intVal) < *schema.Min {
			return fmt.Errorf("value %d is below minimum %g", intVal, *schema.Min)
		}
		if schema.Max != nil && float64(intVal) > *schema.Max {
			return fmt.Errorf("value %d is above maximum %g", intVal, *schema.Max)
		}

	case "float":
		floatVal, err := b.convertToFloat(value)
		if err != nil {
			return fmt.Errorf("expected float, got %T", value)
		}
		if schema.Min != nil && floatVal < *schema.Min {
			return fmt.Errorf("value %g is below minimum %g", floatVal, *schema.Min)
		}
		if schema.Max != nil && floatVal > *schema.Max {
			return fmt.Errorf("value %g is above maximum %g", floatVal, *schema.Max)
		}

	case "string":
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
		// String validation can be added here if needed in the future
		if len(schema.Options) > 0 && !b.stringInSlice(strVal, schema.Options) {
			return fmt.Errorf("value %s not in allowed options: %v", strVal, schema.Options)
		}

	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}

	case "[]string":
		if reflect.TypeOf(value).Kind() != reflect.Slice {
			return fmt.Errorf("expected slice, got %T", value)
		}
		// Additional slice validation could be added here

	default:
		return fmt.Errorf("unsupported parameter type: %s", schema.Type)
	}

	return nil
}

// convertToInt safely converts various numeric types to int
func (b *BaseAdapter) convertToInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

// convertToFloat safely converts various numeric types to float64
func (b *BaseAdapter) convertToFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// stringInSlice checks if a string is in a slice of strings
func (b *BaseAdapter) stringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// PrepareEnvironment is a base implementation that checks if the model path exists
func (b *BaseAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing environment for model", "model_id", b.modelID, "path", b.modelPath)

	if b.modelPath != "" {
		if err := os.MkdirAll(b.modelPath, 0755); err != nil {
			return fmt.Errorf("failed to create model directory %s: %w", b.modelPath, err)
		}
	}

	b.initialized = true
	logger.Info("Environment prepared for model", "model_id", b.modelID)
	return nil
}

// IsReady checks if the adapter is ready to process jobs
func (b *BaseAdapter) IsReady(ctx context.Context) bool {
	if !b.initialized {
		return false
	}

	// Check if model path exists (if specified)
	if b.modelPath != "" {
		if _, err := os.Stat(b.modelPath); os.IsNotExist(err) {
			return false
		}
	}

	return true
}

// GetEstimatedProcessingTime provides a basic estimation based on audio duration
func (b *BaseAdapter) GetEstimatedProcessingTime(input interfaces.AudioInput) time.Duration {
	// Basic estimation: processing time is typically 10-50% of audio duration
	// This can be overridden by specific adapters for more accurate estimates

	audioDuration := input.Duration
	if audioDuration == 0 {
		// Fallback estimation based on file size (rough approximation)
		// Assume ~1MB per minute for compressed audio
		estimatedMinutes := float64(input.Size) / (1024 * 1024)
		audioDuration = time.Duration(estimatedMinutes * float64(time.Minute))
	}

	// Default estimation: 20% of audio duration
	baseProcessingTime := time.Duration(float64(audioDuration) * 0.2)

	// Adjust based on model requirements
	if b.capabilities.RequiresGPU {
		// GPU models are typically faster
		baseProcessingTime = time.Duration(float64(baseProcessingTime) * 0.5)
	}

	// Add minimum processing time
	minProcessingTime := 5 * time.Second
	if baseProcessingTime < minProcessingTime {
		baseProcessingTime = minProcessingTime
	}

	return baseProcessingTime
}

// GetParameterWithDefault gets a parameter value with fallback to default
func (b *BaseAdapter) GetParameterWithDefault(params map[string]interface{}, paramName string) interface{} {
	if value, exists := params[paramName]; exists {
		return value
	}

	// Find the parameter in schema and return its default
	for _, paramSchema := range b.schema {
		if paramSchema.Name == paramName {
			return paramSchema.Default
		}
	}

	return nil
}

// GetStringParameter safely gets a string parameter
func (b *BaseAdapter) GetStringParameter(params map[string]interface{}, paramName string) string {
	value := b.GetParameterWithDefault(params, paramName)
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

// GetIntParameter safely gets an int parameter
func (b *BaseAdapter) GetIntParameter(params map[string]interface{}, paramName string) int {
	value := b.GetParameterWithDefault(params, paramName)
	if intVal, err := b.convertToInt(value); err == nil {
		return intVal
	}
	return 0
}

// GetFloatParameter safely gets a float parameter
func (b *BaseAdapter) GetFloatParameter(params map[string]interface{}, paramName string) float64 {
	value := b.GetParameterWithDefault(params, paramName)
	if floatVal, err := b.convertToFloat(value); err == nil {
		return floatVal
	}
	return 0.0
}

// GetBoolParameter safely gets a bool parameter
func (b *BaseAdapter) GetBoolParameter(params map[string]interface{}, paramName string) bool {
	value := b.GetParameterWithDefault(params, paramName)
	if boolVal, ok := value.(bool); ok {
		return boolVal
	}
	return false
}

// GetStringSliceParameter safely gets a []string parameter
func (b *BaseAdapter) GetStringSliceParameter(params map[string]interface{}, paramName string) []string {
	value := b.GetParameterWithDefault(params, paramName)
	if slice, ok := value.([]string); ok {
		return slice
	}
	if interfaceSlice, ok := value.([]interface{}); ok {
		var stringSlice []string
		for _, item := range interfaceSlice {
			if str, ok := item.(string); ok {
				stringSlice = append(stringSlice, str)
			}
		}
		return stringSlice
	}
	return []string{}
}

// CreateTempDirectory creates a temporary directory for processing
func (b *BaseAdapter) CreateTempDirectory(procCtx interfaces.ProcessingContext) (string, error) {
	tempDir := filepath.Join(procCtx.TempDirectory, b.modelID, procCtx.JobID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	return tempDir, nil
}

// CleanupTempDirectory removes temporary files
func (b *BaseAdapter) CleanupTempDirectory(tempDir string) {
	if tempDir != "" {
		if err := os.RemoveAll(tempDir); err != nil {
			logger.Warn("Failed to cleanup temp directory", "dir", tempDir, "error", err)
		}
	}
}

// ConvertAudioFormat converts audio to the required format for the model
func (b *BaseAdapter) ConvertAudioFormat(ctx context.Context, input interfaces.AudioInput, targetFormat string, targetSampleRate int) (interfaces.AudioInput, error) {
	// This is a placeholder for audio conversion functionality
	// In a real implementation, this would use FFmpeg or similar to convert audio

	if strings.EqualFold(input.Format, targetFormat) &&
		(targetSampleRate == 0 || input.SampleRate == targetSampleRate) {
		// No conversion needed
		return input, nil
	}

	logger.Info("Audio conversion needed",
		"from_format", input.Format,
		"to_format", targetFormat,
		"from_sample_rate", input.SampleRate,
		"to_sample_rate", targetSampleRate)

	// TODO: Implement actual audio conversion using FFmpeg
	// For now, return the original input and log a warning
	logger.Warn("Audio conversion not yet implemented, using original format")
	return input, nil
}

// ReadLogTail reads the last maxBytes from the log file
func (b *BaseAdapter) ReadLogTail(logPath string, maxBytes int64) (string, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}

	fileSize := stat.Size()
	if fileSize <= maxBytes {
		bytes, err := io.ReadAll(file)
		return string(bytes), err
	}

	start := fileSize - maxBytes
	_, err = file.Seek(start, 0)
	if err != nil {
		return "", err
	}

	bytes := make([]byte, maxBytes)
	_, err = file.Read(bytes)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// ValidateAudioInput checks if the audio input meets model requirements
func (b *BaseAdapter) ValidateAudioInput(input interfaces.AudioInput) error {
	// Check if file exists
	if _, err := os.Stat(input.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("audio file not found: %s", input.FilePath)
	}

	// Check supported formats
	if len(b.capabilities.SupportedFormats) > 0 {
		formatSupported := false
		for _, format := range b.capabilities.SupportedFormats {
			if strings.EqualFold(input.Format, format) {
				formatSupported = true
				break
			}
		}
		if !formatSupported {
			return fmt.Errorf("audio format %s not supported by model %s. Supported formats: %v",
				input.Format, b.modelID, b.capabilities.SupportedFormats)
		}
	}

	// Check file size (basic sanity check)
	if input.Size == 0 {
		return fmt.Errorf("audio file appears to be empty")
	}

	return nil
}

// CreateDefaultMetadata creates default metadata for results
func (b *BaseAdapter) CreateDefaultMetadata(params map[string]interface{}) map[string]string {
	metadata := map[string]string{
		"model_id":      b.modelID,
		"model_family":  b.capabilities.ModelFamily,
		"model_version": b.capabilities.Version,
		"adapter_type":  "base",
	}

	// Add some key parameters to metadata
	if device := b.GetStringParameter(params, "device"); device != "" {
		metadata["device"] = device
	}
	if batchSize := b.GetIntParameter(params, "batch_size"); batchSize > 0 {
		metadata["batch_size"] = strconv.Itoa(batchSize)
	}

	return metadata
}

// LogProcessingStart logs the start of processing
func (b *BaseAdapter) LogProcessingStart(input interfaces.AudioInput, procCtx interfaces.ProcessingContext) {
	logger.Info("Starting model processing",
		"model_id", b.modelID,
		"job_id", procCtx.JobID,
		"audio_file", input.FilePath,
		"audio_format", input.Format,
		"audio_duration", input.Duration,
		"audio_size", input.Size)
}

// LogProcessingEnd logs the end of processing
func (b *BaseAdapter) LogProcessingEnd(procCtx interfaces.ProcessingContext, processingTime time.Duration, err error) {
	if err != nil {
		logger.Error("Model processing failed",
			"model_id", b.modelID,
			"job_id", procCtx.JobID,
			"processing_time", processingTime,
			"error", err)
	} else {
		logger.Info("Model processing completed",
			"model_id", b.modelID,
			"job_id", procCtx.JobID,
			"processing_time", processingTime)
	}
}
