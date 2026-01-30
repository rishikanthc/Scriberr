# ASR Engine (onnx-asr + gRPC)

A structured ASR engine daemon that exposes a gRPC API to transcribe audio using onnx-asr (Whisper/Parakeet/Canary).

## Layout

- `src/asr_engine/` core engine
- `proto/` gRPC proto definition
- `tests/` unit + integration tests
- `tests/fixtures/jfk.wav` sample audio for integration tests

## Quick start (uv)

```bash
uv sync
uv run asr-engine-server --socket /tmp/asr-engine.sock
```

## Tests

```bash
uv run pytest
```

Notes:
- Integration tests will download model weights via `onnx_asr` if not cached.
- Golden transcripts live under `tests/fixtures/expected/` and must be generated from `tests/fixtures/jfk.wav`.

## Makefile (uv-only)

```bash
make sync
make goldens
make test
```
