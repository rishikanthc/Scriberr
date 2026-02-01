.PHONY: sync server test unit

DIAR_ENGINE_EXTRA ?= cpu

sync:
	uv sync --extra $(DIAR_ENGINE_EXTRA)

server:
	uv run diar-engine-server --socket /tmp/diar-engine.sock

unit:
	uv run pytest tests/unit

test:
	uv run pytest
