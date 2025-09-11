# Extensible Model Interface Architecture

This directory contains the new extensible model interface architecture for Scriberr's transcription and diarization capabilities.

## Overview

The new architecture implements a plugin-style system where:
- **Models self-describe** their capabilities and parameters
- **Auto-registration** happens on package import
- **Parameter validation** is standardized across all models
- **Adding new models** requires minimal code changes

## Architecture Components

### Core Interfaces (`interfaces/`)
- `ModelAdapter`: Base interface for all model adapters
- `TranscriptionAdapter`: Interface for transcription models
- `DiarizationAdapter`: Interface for diarization models
- `ProcessingPipeline`: Interface for complete processing workflows

### Model Registry (`registry/`)
- Auto-discovery and registration of model adapters
- Model selection based on requirements
- Capability matching and scoring
- Parameter validation coordination

### Model Adapters (`adapters/`)
- `WhisperXAdapter`: OpenAI Whisper models with diarization
- `ParakeetAdapter`: NVIDIA Parakeet English transcription
- `CanaryAdapter`: NVIDIA Canary multilingual transcription
- `PyAnnoteAdapter`: PyAnnote speaker diarization
- `SortformerAdapter`: NVIDIA Sortformer diarization

### Processing Pipeline (`pipeline/`)
- Audio format preprocessing
- Model-specific conversions
- Result post-processing

### Unified Service
- `UnifiedTranscriptionService`: Main orchestrator
- `UnifiedJobProcessor`: Legacy queue integration
- Backward compatibility with existing APIs

## Adding a New Model

Adding a new transcription or diarization model is straightforward:

### 1. Create Model Adapter

```go
// adapters/new_model_adapter.go
package adapters

import (
    "context"
    "scriberr/internal/transcription/interfaces"
    "scriberr/internal/transcription/registry"
)

type NewModelAdapter struct {
    *BaseAdapter
}

func NewNewModelAdapter() *NewModelAdapter {
    capabilities := interfaces.ModelCapabilities{
        ModelID:     "new_model",
        ModelFamily: "new_family", 
        DisplayName: "New Model v1.0",
        SupportedLanguages: []string{"en", "es"},
        Features: map[string]bool{
            "timestamps": true,
            "high_quality": true,
        },
    }

    schema := []interfaces.ParameterSchema{
        {
            Name:        "quality",
            Type:        "string",
            Required:    false,
            Default:     "medium",
            Options:     []string{"low", "medium", "high"},
            Description: "Quality setting",
        },
    }

    baseAdapter := NewBaseAdapter("new_model", "/path/to/model", capabilities, schema)
    return &NewModelAdapter{BaseAdapter: baseAdapter}
}

func (n *NewModelAdapter) Transcribe(ctx context.Context, input interfaces.AudioInput, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.TranscriptResult, error) {
    // Implement transcription logic
    return &interfaces.TranscriptResult{
        Text: "Transcribed text",
        Language: "en",
        Segments: []interfaces.TranscriptSegment{},
    }, nil
}

// Auto-register the adapter
func init() {
    registry.RegisterTranscriptionAdapter("new_model", NewNewModelAdapter())
}
```

### 2. Import the Adapter

```go
// Import in main.go or wherever adapters are initialized
import _ "scriberr/internal/transcription/adapters" // Auto-registers all adapters
```

### 3. Use the Model

```go
// The model is now automatically available
service := transcription.NewUnifiedTranscriptionService()
models := service.GetSupportedModels()
// "new_model" will be in the list
```

## Model Selection

The system can automatically select the best model for given requirements:

```go
requirements := interfaces.ModelRequirements{
    Language: "en",
    Features: []string{"timestamps", "high_quality"},
    Quality:  "best",
}

modelID, err := registry.GetRegistry().SelectBestTranscriptionModel(requirements)
```

## Parameter Validation

All models automatically validate their parameters:

```go
params := map[string]interface{}{
    "quality": "high",
    "language": "en",
}

err := adapter.ValidateParameters(params)
```

## Available Models

### Transcription Models

| Model ID | Family | Languages | Features |
|----------|--------|-----------|----------|
| `whisperx` | `whisper` | 90+ languages | Timestamps, Diarization, Translation |
| `parakeet` | `nvidia_parakeet` | English only | Timestamps, Long-form, High Quality |
| `canary` | `nvidia_canary` | 12 languages | Timestamps, Translation, Multilingual |

### Diarization Models

| Model ID | Family | Features |
|----------|--------|----------|
| `pyannote` | `pyannote` | Speaker Detection, Flexible Count |
| `sortformer` | `nvidia_sortformer` | 4-Speaker Optimized, Fast, No Auth |

## Migration from Legacy System

The new system provides backward compatibility:

### Option 1: Drop-in Replacement
```go
// Replace this:
processor := transcription.NewWhisperXService(config)

// With this:
processor := transcription.NewUnifiedJobProcessor()
```

### Option 2: Gradual Migration
```go
// Use feature flags to switch between systems
if useNewArchitecture {
    processor = transcription.NewUnifiedJobProcessor()
} else {
    processor = transcription.NewWhisperXService(config)
}
```

### Option 3: Direct Access
```go
// Access new features while maintaining compatibility
unified := transcription.NewUnifiedJobProcessor()
capabilities := unified.GetSupportedModels()
status := unified.GetModelStatus(ctx)
```

## Environment Setup

Models automatically set up their environments on first use:

```go
// Initialize all models
service := transcription.NewUnifiedTranscriptionService()
err := service.Initialize(ctx) // Downloads and sets up all models
```

## Configuration

Models can be configured via parameters:

```go
// WhisperX with diarization
params := map[string]interface{}{
    "model": "large-v3",
    "device": "cuda", 
    "diarize": true,
    "diarize_model": "pyannote",
    "hf_token": "your_token",
}

// NVIDIA Canary with translation
params := map[string]interface{}{
    "source_lang": "es",
    "target_lang": "en", 
    "task": "translate",
}
```

## Testing

Run tests to verify the architecture:

```bash
go test ./internal/transcription/... -v
```

Benchmark performance:

```bash
go test ./internal/transcription/... -bench=. -benchmem
```

## Future Extensions

The architecture supports:

- **Plugin Loading**: Load models from external packages
- **Model Chaining**: Combine multiple models in sequence
- **Dynamic Configuration**: Hot-reload model configurations
- **Custom Preprocessing**: Add model-specific audio processing
- **Result Fusion**: Combine outputs from multiple models
- **Streaming Support**: Real-time processing capabilities

## Benefits

1. **Extensibility**: Add new models with minimal code
2. **Consistency**: Standardized parameter handling
3. **Discoverability**: Models self-describe capabilities
4. **Validation**: Automatic parameter checking
5. **Selection**: Smart model selection based on requirements
6. **Testing**: Each adapter tested independently
7. **Performance**: Model-specific optimizations
8. **Maintenance**: Clear separation of concerns

This architecture provides a solid foundation for supporting the growing ecosystem of transcription and diarization models while maintaining backward compatibility with existing code.