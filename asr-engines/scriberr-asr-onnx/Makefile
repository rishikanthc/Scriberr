.PHONY: sync server test unit integration goldens

ASR_ENGINE_EXTRA ?= cpu

sync:
	uv sync --extra $(ASR_ENGINE_EXTRA)

server:
	uv run asr-engine-server --socket /tmp/asr-engine.sock

unit:
	uv run pytest tests/unit

integration:
	uv run pytest tests/integration

goldens:
	uv run scripts/generate_goldens.py

test:
	uv run pytest
