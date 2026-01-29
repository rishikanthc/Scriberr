#!/usr/bin/env python3
import argparse
import logging
import time
from pathlib import Path

import numpy as np
import onnxruntime as rt
import librosa
import onnx_asr

# Silence the noise
logging.basicConfig(level=logging.ERROR)


def get_vad_params(preset: str = "balanced"):
    presets = {
        "conservative": {
            "speech_pad_ms": 400,
            "min_silence_duration_ms": 800,
            "min_speech_duration_ms": 300,
            "max_speech_duration_s": 30,
        },
        "balanced": {
            "speech_pad_ms": 300,
            "min_silence_duration_ms": 600,
            "min_speech_duration_ms": 200,
            "max_speech_duration_s": 25,
        },
        "aggressive": {
            "speech_pad_ms": 150,
            "min_silence_duration_ms": 300,
            "min_speech_duration_ms": 120,
            "max_speech_duration_s": 20,
        },
    }
    return presets.get(preset, presets["aggressive"])


def build_asr_cpu(vad_preset: str, intra_threads: int = 8, inter_threads: int = 1):
    vad_params = get_vad_params(vad_preset)

    # ORT options (CPU)
    opts = rt.SessionOptions()
    opts.intra_op_num_threads = intra_threads
    opts.inter_op_num_threads = inter_threads
    opts.graph_optimization_level = rt.GraphOptimizationLevel.ORT_ENABLE_ALL

    # VAD + ASR
    vad = onnx_asr.load_vad("silero")
    asr = onnx_asr.load_model(
        model="nemo-parakeet-tdt-0.6b-v2",
        providers=["CPUExecutionProvider"],
        sess_options=opts,
    ).with_vad(
        vad,
        speech_pad_ms=vad_params["speech_pad_ms"],
        min_silence_duration_ms=vad_params["min_silence_duration_ms"],
        min_speech_duration_ms=vad_params["min_speech_duration_ms"],
        max_speech_duration_s=vad_params["max_speech_duration_s"],
    )

    return asr, vad_params


def run_once(asr, audio: np.ndarray, sr: int, print_text: bool):
    results = asr.recognize(audio, sample_rate=sr)
    segments = []
    for seg in results:
        text = seg.text.strip()
        if text:
            segments.append(text)

    if print_text:
        print("\n" + "=" * 60)
        print("TRANSCRIPT:")
        print("=" * 60)
        print(" ".join(segments))
        print("=" * 60 + "\n")

    return segments


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("audio_file")
    ap.add_argument(
        "--preset",
        default="aggressive",
        choices=["aggressive", "balanced", "conservative"],
    )
    ap.add_argument("--sr", type=int, default=16000)

    # Benchmark knobs
    ap.add_argument("--warmup", type=int, default=1, help="Warmup runs (not timed)")
    ap.add_argument("--iters", type=int, default=3, help="Timed runs to average")

    # ORT threading
    ap.add_argument("--intra", type=int, default=8)
    ap.add_argument("--inter", type=int, default=1)

    # Output control
    ap.add_argument(
        "--no-print",
        action="store_true",
        help="Do not print transcript (reduces IO noise)",
    )

    args = ap.parse_args()

    audio_path = Path(args.audio_file)
    if not audio_path.exists():
        print(f"Error: audio file not found: {audio_path}")
        return 2

    # Load audio (outside timed region)
    audio, _ = librosa.load(str(audio_path), sr=args.sr, mono=True)
    audio = audio.astype(np.float32)
    audio_seconds = len(audio) / float(args.sr)

    print(f"Audio: {audio_path.name} | {audio_seconds:.2f}s @ {args.sr}Hz")
    print(f"Preset: {args.preset}")
    print(f"Threads: intra={args.intra}, inter={args.inter}")

    print("\nInitializing CPU pipeline...")
    asr, vad_params = build_asr_cpu(
        args.preset, intra_threads=args.intra, inter_threads=args.inter
    )
    print(f"✓ Initialized. VAD params: {vad_params}")

    # Warmup
    for _ in range(args.warmup):
        _ = run_once(asr, audio, args.sr, print_text=False)

    # Timed runs
    times = []
    for i in range(args.iters):
        t0 = time.perf_counter()
        _ = run_once(
            asr, audio, args.sr, print_text=(not args.no_print and i == args.iters - 1)
        )
        t1 = time.perf_counter()
        times.append(t1 - t0)

    avg = sum(times) / len(times)
    rtf = audio_seconds / avg if avg > 0 else 0.0

    print("\n=== CPU Benchmark ===")
    print(f"Runs timed: {args.iters} (warmup: {args.warmup})")
    print(f"Avg time/run: {avg:.3f} sec")
    print(f"RTF (audio_sec / run_sec): {rtf:.2f}x realtime")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
