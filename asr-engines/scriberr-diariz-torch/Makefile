.PHONY: sync server test unit

sync:
	uv sync

server:
	uv run diar-engine-server --socket /tmp/diar-engine.sock

unit:
	uv run pytest tests/unit

test:
	uv run pytest
