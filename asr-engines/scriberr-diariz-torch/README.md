# Diarization Engine (PyTorch + gRPC)

A structured diarization engine daemon that exposes the same gRPC API as the ASR engine,
backed by PyAnnote and NVIDIA Sortformer.

## Layout

- `src/diar_engine/` core engine
- `proto/` gRPC proto definition (same interface as ASR engine)
- `tests/` unit tests

## Supported models

- PyAnnote (default): `pyannote/speaker-diarization-community-1`
- PyAnnote (alt): `pyannote/speaker-diarization-3.1`
- NVIDIA Sortformer: `nvidia/diar_streaming_sortformer_4spk-v2`

Notes:
- PyAnnote models require accepting the model license and providing a Hugging Face token.
  Set `HF_TOKEN` in the environment or pass `hf_token` in job params.
- Sortformer uses a local `.nemo` file if `model_path` is provided, otherwise it downloads
  the model file from Hugging Face.

## Quick start (uv)

```bash
uv sync
uv run diar-engine-server --socket /tmp/diar-engine.sock
```

## Tests

```bash
uv run pytest
```

## Makefile (uv-only)

```bash
make sync
make test
```
