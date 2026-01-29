#!/usr/bin/env python3
from __future__ import annotations

from pathlib import Path
import shutil
import tempfile

from asr_engine.backend import AsrBackend
from asr_engine.model_manager import ModelManager, ModelSpec
from asr_engine.params import JobParams

MODELS = [
    "nemo-parakeet-tdt-0.6b-v2",
    "nemo-parakeet-tdt-0.6b-v3",
    "nemo-canary-1b-v2",
]

ROOT = Path(__file__).resolve().parents[1]
AUDIO = ROOT / "tests" / "fixtures" / "jfk.wav"
EXPECTED_DIR = ROOT / "tests" / "fixtures" / "expected"


def main() -> int:
    if not AUDIO.exists():
        print(f"Missing audio: {AUDIO}")
        return 2
    EXPECTED_DIR.mkdir(parents=True, exist_ok=True)

    model_manager = ModelManager()
    backend = AsrBackend(model_manager)

    for model_name in MODELS:
        print(f"Loading {model_name}...")
        spec = ModelSpec(model_id=model_name, model_name=model_name)
        model_manager.load(spec)

        with tempfile.TemporaryDirectory() as tmpdir:
            result = backend.transcribe(
                input_path=str(AUDIO),
                output_dir=tmpdir,
                params=JobParams(),
            )
            transcript = Path(result.transcript_path).read_text(encoding="utf-8").strip()
            out_path = EXPECTED_DIR / f"{model_name}.txt"
            out_path.write_text(transcript + "\n", encoding="utf-8")
            print(f"Wrote {out_path}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
