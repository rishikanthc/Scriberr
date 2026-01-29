#!/usr/bin/env python3
import sys
import json
import logging
from pathlib import Path
import numpy as np
import onnxruntime as rt
import soundfile as sf
import onnx_asr

# Silence the noise
logging.basicConfig(level=logging.ERROR)


def get_vad_params(preset: str = "balanced"):
    """
    Return a dict of vad parameters for three presets:
      - conservative: fewer cuts, more padding (best for WER, slower)
      - balanced: reasonable middle-ground
      - aggressive: more splits, lower latency (may hurt WER)
    """
    presets = {
        "conservative": {
            "speech_pad_ms": 400,
            "min_silence_duration_ms": 800,
            "min_speech_duration_ms": 300,
            "max_speech_duration_s": 30,
            # optional thresholds if the VAD backend supports them:
            # "threshold": 0.45,
            # "neg_threshold": 0.15,
        },
        "balanced": {
            "speech_pad_ms": 300,
            "min_silence_duration_ms": 600,
            "min_speech_duration_ms": 200,
            "max_speech_duration_s": 25,
            # "threshold": 0.5,
            # "neg_threshold": 0.15,
        },
        "aggressive": {
            "speech_pad_ms": 150,
            "min_silence_duration_ms": 300,
            "min_speech_duration_ms": 120,
            "max_speech_duration_s": 20,
            # "threshold": 0.55,
            # "neg_threshold": 0.2,
        },
    }
    return presets.get(preset, presets["balanced"])


def normalize_text(s: str) -> str:
    """Simple normalizer used for WER eval; not applied to printed transcript."""
    import re

    s = s.lower()
    s = re.sub(r"[^\w\s']", " ", s)
    s = re.sub(r"\s+", " ", s).strip()
    return s


def transcribe_final(
    audio_path: str, model_dir: str = "nemo_global_onnx", vad_preset: str = "balanced"
):
    audio_file = Path(audio_path)
    model_path = Path(model_dir)

    # if not model_path.exists():
    #     print(f"Error: Run the export script first to create {model_dir}")
    #     return

    vad_params = get_vad_params(vad_preset)
    print(f"Using VAD preset: {vad_preset} -> {vad_params}")

    # 1. Performance Tuning for M3
    opts = rt.SessionOptions()
    opts.intra_op_num_threads = 8

    print(f"Initializing VAD + Global Parakeet Pipeline...")
    try:
        # Load VAD to handle the chunking automatically
        vad = onnx_asr.load_vad("silero")

        # Load our Global Model (with the config.json we made)
        # We wrap it in .with_vad(...) and pass tuned VAD params.
        asr = onnx_asr.load_model(
            model="nemo-parakeet-tdt-0.6b-v2",
            # path=model_path,
            providers=["CPUExecutionProvider"],
            sess_options=opts,
        ).with_vad(
            vad,
            # core tuning knobs:
            speech_pad_ms=vad_params["speech_pad_ms"],
            min_silence_duration_ms=vad_params["min_silence_duration_ms"],
            min_speech_duration_ms=vad_params["min_speech_duration_ms"],
            max_speech_duration_s=vad_params["max_speech_duration_s"],
            # if your VAD backend supports thresholds, you can add them here
            # threshold=vad_params.get("threshold"),
            # neg_threshold=vad_params.get("neg_threshold"),
        )

        print("✓ Pipeline initialized. Memory-safe Global Attention active.")
    except Exception as e:
        print(f"✗ Load Error: {e}")
        return

    # 2. Load Audio
    print(f"Loading: {audio_file.name}")
    audio, sr = sf.read(str(audio_file), dtype="float32", always_2d=False)
    if audio.ndim == 2:
        audio = np.mean(audio, axis=1)
    if sr != 16000:
        try:
            import librosa  # optional fallback for resampling

            audio = librosa.resample(audio, orig_sr=sr, target_sr=16000)
        except Exception as e:
            raise RuntimeError(
                f"Audio sample rate is {sr} Hz; expected 16000 Hz. "
                "Resampling requires librosa or an external tool."
            ) from e

    # 3. Transcribe
    print("Transcribing (Segmented Mode)...")
    full_text_segments = []
    try:
        # recognize() now returns an iterator of segment results
        results = asr.recognize(audio, sample_rate=16000)

        # Collect and apply tiny postprocessing:
        # - merge very short segments into previous segment (helps boundary artifacts)
        prev_seg = None
        for segment in results:
            seg_text = segment.text.strip()
            seg_dur = getattr(segment, "duration", None)  # not guaranteed; defensive
            # If segment is extremely short (<0.25s) or text length is tiny, attach to previous
            attach_threshold_seconds = 0.25
            attach_by_text_length = len(seg_text.split()) <= 2

            if prev_seg and (
                (seg_dur is not None and seg_dur < attach_threshold_seconds)
                or attach_by_text_length
            ):
                # merge into previous
                prev_seg = prev_seg + " " + seg_text
                full_text_segments[-1] = prev_seg
            else:
                full_text_segments.append(seg_text)
                prev_seg = seg_text

        print("\n" + "=" * 60)
        print("FINAL CLEAN TRANSCRIPT:")
        print("=" * 60)
        for s in full_text_segments:
            print(s, end=" ", flush=True)
        print("\n" + "=" * 60 + "\n")

    except Exception as e:
        print(f"\n✗ Transcription Error: {e}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python final_transcribe.py <audio_file> [preset]")
        print("Presets: balanced (default), conservative, aggressive")
    else:
        audio_file = sys.argv[1]
        preset = sys.argv[2] if len(sys.argv) > 2 else "balanced"
        transcribe_final(audio_file, vad_preset=preset)
