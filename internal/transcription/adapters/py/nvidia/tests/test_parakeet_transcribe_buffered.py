"""Tests for parakeet_transcribe_buffered.py"""
import pytest
import subprocess
import json
import os
import tempfile
from pathlib import Path

# Paths
SCRIPT_DIR = Path(__file__).parent.parent
TEST_DATA_DIR = Path(__file__).parent.parent.parent.parent.parent.parent.parent / "tests/data"
AUDIO_FILE = TEST_DATA_DIR / "AMI-Corpus-IB4002.Mix-Headset-clip.wav"

def test_parakeet_buffered_transcription_output():
    """Verify Parakeet buffered transcription output matches expected results."""

    assert AUDIO_FILE.exists(), f"Audio file not found: {AUDIO_FILE}"

    # Locate project root and paths
    project_root = Path(__file__).resolve().parents[6]
    env_path = project_root / "data/whisperx-env/parakeet"
    script_path = SCRIPT_DIR / "parakeet_transcribe_buffered.py"

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
            "--output", output_file,
            # Use a small chunk length to force buffering behavior on our test file which is 19 sec long
            "--chunk-len", "10"
        ]

        print(f"Running command: {' '.join(cmd)}")

        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            cwd=project_root
        )

        if result.returncode != 0:
            pytest.fail(f"Script failed with error:\n{result.stderr}\nStdout:\n{result.stdout}")

        # Verify output file exists and is valid JSON
        assert os.path.exists(output_file), "Output file was not created"

        with open(output_file, 'r') as f:
            data = json.load(f)

        # Assertions
        assert data["language"] == "en"
        assert data["model"] == "parakeet-tdt-0.6b-v3"
        assert data.get("buffered") is True
        assert "transcription" in data
        assert len(data["transcription"]) > 0

        # Check that we have timestamps
        assert "word_timestamps" in data
        assert len(data["word_timestamps"]) > 0

        assert "segment_timestamps" in data
        assert len(data["segment_timestamps"]) > 0

    finally:
        # Cleanup
        if os.path.exists(output_file):
            os.remove(output_file)
