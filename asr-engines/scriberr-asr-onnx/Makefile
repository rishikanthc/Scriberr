.PHONY: sync server test unit integration goldens

sync:
	uv sync

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
