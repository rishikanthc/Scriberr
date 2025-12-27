"""Tests for pyannote_diarize.py"""
import pytest
import subprocess
from pathlib import Path

# Paths
SCRIPT_DIR = Path(__file__).parent.parent
SCRIPT_PATH = SCRIPT_DIR / "pyannote_diarize.py"

# TODO: Add proper diarization testing once a dummy HF token or mock pipeline is available.
# uv run --project data/whisperx-env/pyannote/ python internal/transcription/adapters/py/pyannote/pyannote_diarize.py --output=/tmp/pyan.json --hf-token $HF_TOKEN tests/data/AMI-Corpus-IB4002.Mix-Headset-clip.wav
def test_pyannote_diarize_exists():
    """Verify pyannote_diarize.py exists."""
    assert SCRIPT_PATH.exists(), "pyannote_diarize.py should exist"


def test_pyannote_diarize_help():
    """Verify pyannote_diarize.py --help works."""

    # Locate project root (Scriberr directory)
    # This file is in internal/transcription/adapters/py/pyannote/tests/
    project_root = Path(__file__).resolve().parents[6]
    env_path = project_root / "data/whisperx-env/pyannote"

    assert env_path.exists(), f"Environment not found at: {env_path}"



    cmd = [
        "uv", "run",
        "--project", str(env_path),
        "python", str(SCRIPT_PATH),
        "--help"
    ]

    print(f"Running command: {' '.join(cmd)}")
    result = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        cwd=project_root
    )

    assert result.returncode == 0
    assert "usage: pyannote_diarize.py" in result.stdout
    assert "--hf-token" in result.stdout
    assert "--model" in result.stdout
