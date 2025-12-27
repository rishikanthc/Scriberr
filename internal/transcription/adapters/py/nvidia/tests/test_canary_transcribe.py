"""Tests for canary_transcribe.py"""
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

def test_canary_transcription_output():
    """Verify Canary transcription output matches expected results."""

    assert AUDIO_FILE.exists(), f"Audio file not found: {AUDIO_FILE}"

    # Locate project root and paths
    project_root = Path(__file__).resolve().parents[6]
    env_path = project_root / "data/whisperx-env/parakeet" # Canary uses the same env as Parakeet in this setup
    script_path = SCRIPT_DIR / "canary_transcribe.py"

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
            "--output", output_file
        ]

        print(f"Running command: {' '.join(cmd)}")

        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            cwd=project_root
        )

        if result.returncode != 0:
            pytest.fail(f"Script failed with error:\n{result.stderr}")

        # Verify output file exists and is valid JSON
        assert os.path.exists(output_file), "Output file was not created"

        with open(output_file, 'r') as f:
            data = json.load(f)

        # Assertions based on the provided sample output
        assert data["source_language"] == "en"
        assert data["target_language"] == "en"
        assert data["task"] == "transcribe"
        assert data["model"] == "canary-1b-v2"

        # Canary output text check
        assert "Most of us" in data["transcription"]
        assert "desktop computers" in data["transcription"]

        assert "word_timestamps" in data
        assert len(data["word_timestamps"]) > 0
        # Check first word
        first_word = data["word_timestamps"][0]
        assert first_word["word"] == "Most"
        # Allow for small floating point differences
        assert abs(first_word["start"] - 0.0) < 0.1

        assert "segment_timestamps" in data
        assert len(data["segment_timestamps"]) > 0

    finally:
        # Cleanup
        if os.path.exists(output_file):
            os.remove(output_file)
