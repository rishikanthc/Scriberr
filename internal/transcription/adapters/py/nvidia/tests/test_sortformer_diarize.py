"""Tests for sortformer_diarize.py"""
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

def test_sortformer_diarization_output():
    """Verify Sortformer diarization output matches expected results."""

    assert AUDIO_FILE.exists(), f"Audio file not found: {AUDIO_FILE}"

    # Locate project root and paths
    project_root = Path(__file__).resolve().parents[6]
    env_path = project_root / "data/whisperx-env/parakeet"
    script_path = SCRIPT_DIR / "sortformer_diarize.py"

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
            output_file
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

        # Assertions based on the provided sample output
        assert data["model"] == "nvidia/diar_streaming_sortformer_4spk-v2"
        assert "segments" in data
        assert len(data["segments"]) > 0

        # Check speakers
        assert "speakers" in data
        assert len(data["speakers"]) == 4 # Based on sample output which found 4 speakers
        assert "speaker_0" in data["speakers"]
        assert "speaker_1" in data["speakers"]

        # Check first segment
        first_segment = data["segments"][0]
        assert "start" in first_segment
        assert "end" in first_segment
        assert "speaker" in first_segment
        assert first_segment["start"] == 0.0
        assert first_segment["speaker"] == "speaker_0"

        assert data["speaker_count"] == 4

        assert "total_segments" in data
        assert data["total_segments"] > 15

        assert "total_duration" in data
        assert data["total_duration"] > 17

    finally:
        # Cleanup
        if os.path.exists(output_file):
            os.remove(output_file)
