from __future__ import annotations

from pathlib import Path

import numpy as np
import soundfile as sf


class AudioLoadError(RuntimeError):
    pass


def load_audio(path: str | Path, sample_rate: int = 16000) -> tuple[np.ndarray, int]:
    audio_path = Path(path)
    if not audio_path.exists():
        raise AudioLoadError(f"Audio file not found: {audio_path}")

    audio, sr = sf.read(str(audio_path), dtype="float32", always_2d=False)
    if audio.ndim == 2:
        audio = np.mean(audio, axis=1)

    if sr != sample_rate:
        try:
            import librosa

            audio = librosa.resample(audio, orig_sr=sr, target_sr=sample_rate)
            sr = sample_rate
        except Exception as exc:
            raise AudioLoadError(
                f"Audio sample rate is {sr} Hz; expected {sample_rate} Hz. "
                "Resampling requires librosa or an external tool."
            ) from exc

    return audio.astype(np.float32), sr
