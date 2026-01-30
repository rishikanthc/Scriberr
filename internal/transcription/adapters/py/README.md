# Python Adapters Testing

This directory contains the Python adapter scripts for diarization models used by Scriberr.

## Running Tests

The tests are located in the `tests/` subdirectory of each adapter folder (e.g., `nvidia/tests/`, `pyannote/tests/`). These tests verify that the Python scripts can be executed and produce the expected output.

To run the tests, you need `uv` installed and the diarization environments set up.

### Prerequisites

1.  Ensure you have `uv` installed.
2.  Ensure the `parakeet` (Sortformer) and `pyannote` environments are set up within `data/model-env/`. This is typically handled by the application startup.
3.  Ensure you have the test data available (e.g., `tests/data/AMI-Corpus-IB4002.Mix-Headset-clip.wav`).

### Running Tests with pytest

```bash
# Run NVIDIA Sortformer diarization tests
uv run --with pytest --project data/model-env/parakeet pytest internal/transcription/adapters/py/nvidia/tests

# Run PyAnnote adapter tests
uv run --with pytest --project data/model-env/pyannote pytest internal/transcription/adapters/py/pyannote/tests
```

### Troubleshooting

*   **Audio file not found**: Ensure `tests/data/AMI-Corpus-IB4002.Mix-Headset-clip.wav` exists.
*   **Environment not found**: Ensure `data/model-env/parakeet` and the `pyannote` one exist and is a valid virtual environment. This may not be true if scriberr hasn't run yet.
