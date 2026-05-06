# ASR Provider Author Guide

Scriberr external ASR providers run as independent services. Scriberr owns queueing, audio preprocessing, persistence, transcript merging, and user-facing APIs. Providers own model execution only.

## Transport

The current external provider transport is REST over HTTP. Providers should expose JSON endpoints under `/v1`.

Required control endpoints:

- `GET /v1/health`
- `GET /v1/provider`
- `GET /v1/models`
- `GET /v1/status`
- `GET /v1/models/loaded`
- `POST /v1/models/{model}:load`
- `POST /v1/models/{model}:unload`

Required job endpoints:

- `POST /v1/jobs`
- `GET /v1/jobs/{job_id}`
- `GET /v1/jobs/{job_id}/events`
- `DELETE /v1/jobs/{job_id}`

`GET /v1/jobs/{job_id}/events` can return sporadic progress events. It does not need to stream.

## Execution Rules

- A provider processes one job at a time unless its model card/runtime explicitly advertises higher capacity.
- Scriberr may split pipeline steps across providers.
- Providers must treat mounted audio paths as read-only.
- Providers should return quickly from `POST /v1/jobs` with a provider-local `job_id`, then expose progress/result through polling.
- `DELETE /v1/jobs/{job_id}` should cancel best-effort and be idempotent.

## Audio Input

Scriberr preprocesses audio before provider execution.

Baseline expectation:

- WAV
- 16 kHz
- mono
- mounted file path accessible inside the provider container

Providers declare exact expectations in `ProviderInfo.audio_input`.

## Model Cards

Every model must publish a model card from `GET /v1/models`.

Required fields:

- `id`
- `display_name`
- `provider`
- `family`
- `installed`
- `loaded`
- `default`
- `capabilities`

Capabilities should be precise. Do not advertise diarization, speaker identification, token timestamps, custom vocabulary, or streaming unless the model endpoint actually supports them.

Optional fields:

- `tasks`
- `languages`
- `limits`
- `resource_requirements`
- `parameter_schema`
- `license`
- `version`

`parameter_schema` is reserved for provider-specific options. Avoid secrets, host paths, URLs, or credentials in schema examples and runtime options.

### Model Registry V2

Providers should treat `GET /v1/models` as a model registry, not a static name list. Scriberr uses this registry for backend validation, profile defaults, execution planning, and frontend profile controls.

Each model entry should describe:

- stable `id`, display name, provider, family, version, license, and install/load state
- supported tasks such as `transcription`, `diarization`, `speaker_identification`, and `audio_tagging`
- supported languages, including whether language selection is automatic, fixed, or user-configurable
- output capabilities such as word timestamps, segment timestamps, token timestamps, speaker labels, language detection, translation, custom vocabulary, and streaming
- runtime capabilities such as CPU, CUDA, CoreML, thread support, batching support, and expected memory class
- chunking capabilities and preferences
- typed parameter schema with defaults and validation limits

Do not expose implementation paths, cache directories, signed URLs, credentials, or machine-local details in model cards.

### Parameter Schema

`parameter_schema` is the source of truth for dynamic ASR profile forms. Scriberr validates submitted profile parameters against this schema before queue execution, even when the frontend generated the form from the same schema.

Each parameter should include:

- `key`: stable machine-readable ID
- `label`: short user-facing label
- `type`: `boolean`, `integer`, `number`, `string`, `enum`, `duration`, or `path_ref`
- `default`
- optional `min`, `max`, `step`, and `options`
- `scope`: `model`, `runtime`, `decoding`, `chunking`, `vad`, `output`, or `postprocess`
- `advanced`: whether the frontend should hide it behind advanced controls by default
- `requires_reload`: whether changing the value requires model/recognizer recreation

Prefer provider-neutral parameter keys where the behavior is shared. For example, use `runtime.num_threads`, `decoding.method`, `chunking.mode`, and `output.timestamps` instead of inventing provider-specific names for common concepts.

Provider-specific parameters are allowed, but they must be namespaced, bounded, and documented. Example: `sherpa.whisper.tail_paddings`.

### Chunking And Batching Metadata

Providers must declare how chunking should be handled by the execution runtime that owns model decode:

- `supports_engine_chunking`: the execution runtime may split audio before decode.
- `supports_provider_chunking`: provider can accept full audio and handle chunking internally.
- `preferred_chunking_mode`: `fixed`, `vad`, `provider`, or `none`.
- `recommended_chunk_seconds` and `max_chunk_seconds` when engine chunking is supported.
- `supports_batching`, `recommended_batch_size`, and `max_batch_size` when batch decode is supported.

The Scriberr backend does not pre-chunk audio for providers. It passes normalized audio plus validated pipeline step options. The bundled local provider delegates fixed-window, VAD, batching, timestamp offsetting, and stitching to `scriberr-engine` through direct Go APIs. External REST providers should perform any required long-form planning internally.

Batching metadata must reflect real runtime behavior. If an API supports batch decode but CPU throughput is worse in practice, advertise batching as supported but set `recommended_batch_size` to `1`.

### Sherpa-ONNX Notes

Sherpa-ONNX has a mostly common offline decode lifecycle after recognizer construction, but model configuration differs by family.

Common offline settings include:

- `feat.sample_rate`
- `feat.feature_dim`
- `model.provider`
- `model.num_threads`
- `model.debug`
- `model.model_type`
- `model.tokens`
- `decoding_method`
- `max_active_paths`
- `hotwords_file`
- `hotwords_score`
- `blank_penalty`
- `rule_fsts`
- `rule_fars`
- `lm.model`
- `lm.scale`

Whisper-specific settings include:

- `whisper.encoder`
- `whisper.decoder`
- `whisper.language`
- `whisper.task`
- `whisper.tail_paddings`
- `whisper.enable_token_timestamps`
- `whisper.enable_segment_timestamps`

Parakeet/NeMo transducer settings include:

- `transducer.encoder`
- `transducer.decoder`
- `transducer.joiner`
- `tokens`
- `model_type: nemo_transducer`

Whisper language/task can be user-configurable. Parakeet language should usually be model metadata unless the selected model explicitly supports runtime language selection.

## Progress

Use these stages where applicable:

- `accepted`
- `preprocessing`
- `loading_model`
- `transcribing`
- `diarizing`
- `identifying_speakers`
- `postprocessing`
- `completed`
- `failed`
- `canceled`

Progress values are optional and should be `0.0` to `1.0` when present.

## Errors

Return provider errors as:

```json
{
  "error": {
    "code": "PROVIDER_BUSY",
    "message": "provider is busy",
    "retryable": true
  }
}
```

Use the shared error codes in `internal/transcription/asrcontract`. Do not include local paths, URLs with credentials, API keys, tokens, or raw stack traces in messages or details.

## Contract Tests

Use `internal/transcription/engineprovider/contracttest.RunProviderContract` for Go providers/adapters. The remote example server test in `internal/transcription/engineprovider/remote/example_provider_test.go` shows the minimum REST behavior expected by Scriberr.

## Internal Backend Guardrails

Backend provider adapters should keep these rules:

- The bundled sherpa path must remain a direct Go adapter over `scriberr-engine`; do not add REST between Scriberr and the local engine.
- Internal provider capabilities are derived from `Models()`; providers should not expose a separate capability list.
- Execution methods are task-specific. Implement only the task interfaces a provider can actually run.
- Profile options must be descriptor-keyed values under `pipeline[].options`; do not add new flat `ASRParams` fields for model/runtime/chunking controls.
