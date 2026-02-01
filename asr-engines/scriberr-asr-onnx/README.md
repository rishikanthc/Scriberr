# ASR Engine (onnx-asr + gRPC)

A structured ASR engine daemon that exposes a gRPC API to transcribe audio using onnx-asr (Whisper/Parakeet/Canary).

## Layout

- `src/asr_engine/` core engine
- `proto/` gRPC proto definition
- `tests/` unit + integration tests
- `tests/fixtures/jfk.wav` sample audio for integration tests

## Quick start (uv)

```bash
uv sync --extra cpu
uv run asr-engine-server --socket /tmp/asr-engine.sock
```

### GPU setup

Install the GPU dependency profile when CUDA is available:

```bash
uv sync --extra gpu
```

Notes:
- The engine selects CUDA automatically if the CUDA execution provider is available.
- In the repo root, `make dev` and `make asr-engine-dev` will auto-select `cpu` or `gpu`.
  Override with `ASR_ENGINE_EXTRA=cpu|gpu` or `ASR_ENGINE_DEVICE=cpu|gpu`.

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
