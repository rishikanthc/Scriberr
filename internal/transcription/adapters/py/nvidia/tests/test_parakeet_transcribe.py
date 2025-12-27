"""Tests for parakeet_transcribe.py"""
import pytest
import subprocess
import json
import os
from pathlib import Path

# Paths
SCRIPT_DIR = Path(__file__).parent.parent
TEST_DATA_DIR = Path(__file__).parent.parent.parent.parent.parent.parent.parent / "tests/data"
AUDIO_FILE = TEST_DATA_DIR / "AMI-Corpus-IB4002.Mix-Headset-clip.wav"

import tempfile

def test_parakeet_transcription_output():
    """Verify Parakeet transcription output matches expected results."""

    assert AUDIO_FILE.exists(), f"Audio file not found: {AUDIO_FILE}"

    # Construct command
    # uv run --project data/whisperx-env/parakeet python internal/transcription/adapters/py/nvidia/parakeet_transcribe.py ...

    # Locate project root (Scriberr directory)
    # This file is in internal/transcription/adapters/py/nvidia/tests/
    project_root = Path(__file__).resolve().parents[6]
    env_path = project_root / "data/whisperx-env/parakeet"
    script_path = SCRIPT_DIR / "parakeet_transcribe.py"

    assert env_path.exists(), f"Environment not found at: {env_path}"

    # Create a temporary file for output
    with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as tmp_file:
        output_file = tmp_file.name

    try:
        cmd = [
            "uv", "run",
            "--project", str(env_path),
            "python", str(script_path),
            str(AUDIO_FILE),
            "--timestamps",
            "--context-left", "256",
            "--context-right", "256",
            "--output", output_file
        ]

        print(f"Running command: {' '.join(cmd)}")

        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            cwd=project_root # Run from project root to ensure paths are correct if relative
        )

        if result.returncode != 0:
            pytest.fail(f"Script failed with error:\n{result.stderr}")

        # Verify output file exists and is valid JSON
        assert os.path.exists(output_file), "Output file was not created"

        with open(output_file, 'r') as f:
            data = json.load(f)

        # Assertions based on the provided sample output
        assert data["language"] == "en"
        assert data["model"] == "parakeet-tdt-0.6b-v3"
        assert "transcription" in data
        assert "First of all, have desktop computers" in data["transcription"]
        assert "reading room" in data["transcription"]

        assert "word_timestamps" in data
        assert len(data["word_timestamps"]) > 0
        # Check first word
        first_word = data["word_timestamps"][0]
        assert first_word["word"] == "First"
        # Allow for small floating point differences
        assert abs(first_word["start"] - 0.4) < 0.1

        assert "segment_timestamps" in data
        assert len(data["segment_timestamps"]) > 0

        assert data["context"]["left"] == 256
        assert data["context"]["right"] == 256

    finally:
        # Cleanup
        if os.path.exists(output_file):
            os.remove(output_file)
