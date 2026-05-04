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
