from pathlib import Path
import numpy as np
import soundfile as sf

from asr_engine.audio_io import load_audio


def test_load_audio_mono(tmp_path: Path):
    sr = 16000
    t = np.linspace(0, 1, sr, endpoint=False)
    left = 0.5 * np.sin(2 * np.pi * 220 * t)
    right = 0.5 * np.sin(2 * np.pi * 440 * t)
    stereo = np.stack([left, right], axis=1)

    wav_path = tmp_path / "test.wav"
    sf.write(wav_path, stereo, sr)

    audio, out_sr = load_audio(wav_path, sample_rate=sr)
    assert out_sr == sr
    assert audio.ndim == 1
    assert len(audio) == sr
