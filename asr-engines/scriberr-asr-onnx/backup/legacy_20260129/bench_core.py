#!/usr/bin/env python3
import argparse
import logging
import time
from pathlib import Path
import shutil

import numpy as np
import onnxruntime as rt
import librosa
import onnx_asr

logging.basicConfig(level=logging.ERROR)

DEFAULT_REPO_ID = "istupakov/parakeet-tdt-0.6b-v2-onnx"


def now():
    return time.strftime("%Y-%m-%d %H:%M:%S")


def log(msg: str):
    print(f"[{now()}] {msg}", flush=True)


def get_vad_params(preset: str = "aggressive"):
    presets = {
        "conservative": dict(
            speech_pad_ms=400,
            min_silence_duration_ms=800,
            min_speech_duration_ms=300,
            max_speech_duration_s=30,
        ),
        "balanced": dict(
            speech_pad_ms=300,
            min_silence_duration_ms=600,
            min_speech_duration_ms=200,
            max_speech_duration_s=25,
        ),
        "aggressive": dict(
            speech_pad_ms=150,
            min_silence_duration_ms=300,
            min_speech_duration_ms=120,
            max_speech_duration_s=20,
        ),
    }
    return presets.get(preset, presets["aggressive"])


def ensure_hf_snapshot(repo_id: str) -> Path:
    from huggingface_hub import snapshot_download

    local_dir = snapshot_download(
        repo_id=repo_id,
        allow_patterns=["*.onnx", "*.onnx.data", "*.json", "*.txt", "*.model"],
    )
    return Path(local_dir)


def ensure_coreml_int8_shim(snapshot_dir: Path) -> Path:
    enc_int8 = snapshot_dir / "encoder-model.int8.onnx"
    dec_int8 = snapshot_dir / "decoder_joint-model.int8.onnx"
    vocab = snapshot_dir / "vocab.txt"
    cfg = snapshot_dir / "config.json"
    missing = [p for p in [enc_int8, dec_int8, vocab, cfg] if not p.exists()]
    if missing:
        raise FileNotFoundError(f"Missing required files in snapshot: {missing}")

    shim_dir = (
        Path.home() / ".cache" / "onnx_asr_shims" / "parakeet-tdt-0.6b-v2-int8-coreml"
    )
    shim_dir.mkdir(parents=True, exist_ok=True)

    shutil.copy2(vocab, shim_dir / "vocab.txt")
    shutil.copy2(cfg, shim_dir / "config.json")
    shutil.copy2(enc_int8, shim_dir / "encoder-model.onnx")
    shutil.copy2(dec_int8, shim_dir / "decoder_joint-model.onnx")

    return shim_dir


def make_sess_options(intra: int, inter: int) -> rt.SessionOptions:
    so = rt.SessionOptions()
    so.intra_op_num_threads = intra
    so.inter_op_num_threads = inter
    so.graph_optimization_level = rt.GraphOptimizationLevel.ORT_ENABLE_ALL
    return so


def build_asr(model_dir: Path, providers, so: rt.SessionOptions, vad_preset: str):
    vad = onnx_asr.load_vad("silero")
    vad_kwargs = get_vad_params(vad_preset)

    log(f"Creating model sessions with providers={providers} ...")
    asr = onnx_asr.load_model(
        model="nemo-parakeet-tdt-0.6b-v2",
        path=model_dir,
        providers=providers,
        sess_options=so,
    ).with_vad(vad, **vad_kwargs)

    log(f"✓ Model created. VAD params: {vad_kwargs}")
    return asr


def recognize_once(asr, audio: np.ndarray, sr: int) -> int:
    # Return number of segments produced (forces full iteration)
    n = 0
    for _seg in asr.recognize(audio, sample_rate=sr):
        n += 1
    return n


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("audio_file")
    ap.add_argument("--repo-id", default=DEFAULT_REPO_ID)
    ap.add_argument(
        "--preset",
        default="aggressive",
        choices=["aggressive", "balanced", "conservative"],
    )
    ap.add_argument("--sr", type=int, default=16000)
    ap.add_argument("--warmup", type=int, default=1)
    ap.add_argument("--iters", type=int, default=1)  # keep small while debugging
    ap.add_argument("--intra", type=int, default=8)
    ap.add_argument("--inter", type=int, default=1)
    ap.add_argument(
        "--no-coreml", action="store_true", help="Force CPU only (sanity check)."
    )
    args = ap.parse_args()

    audio_path = Path(args.audio_file)
    if not audio_path.exists():
        raise SystemExit(f"Audio not found: {audio_path}")

    log(f"ORT providers: {rt.get_available_providers()}")

    # Load audio outside timing
    log("Loading audio...")
    audio, _ = librosa.load(str(audio_path), sr=args.sr, mono=True)
    audio = audio.astype(np.float32)
    audio_seconds = len(audio) / float(args.sr)
    log(f"Audio loaded: {audio_seconds:.2f}s")

    log(f"Downloading/caching model: {args.repo_id}")
    snap = ensure_hf_snapshot(args.repo_id)
    log(f"Snapshot: {snap}")

    log("Preparing INT8 shim dir for CoreML safety...")
    shim = ensure_coreml_int8_shim(snap)
    log(f"Shim: {shim}")

    so = make_sess_options(args.intra, args.inter)

    if args.no_coreml:
        providers = ["CPUExecutionProvider"]
    else:
        if "CoreMLExecutionProvider" not in rt.get_available_providers():
            raise SystemExit(
                "CoreMLExecutionProvider not available in this onnxruntime build."
            )
        providers = ["CoreMLExecutionProvider", "CPUExecutionProvider"]

    asr = build_asr(shim, providers, so, args.preset)

    # Warmup + timing with explicit progress
    for i in range(args.warmup):
        log(f"Warmup {i + 1}/{args.warmup}: starting recognize() ...")
        t0 = time.perf_counter()
        nseg = recognize_once(asr, audio, args.sr)
        t1 = time.perf_counter()
        log(f"Warmup {i + 1}/{args.warmup}: done ({nseg} segments) in {(t1 - t0):.2f}s")

    times = []
    for i in range(args.iters):
        log(f"Timed run {i + 1}/{args.iters}: starting recognize() ...")
        t0 = time.perf_counter()
        nseg = recognize_once(asr, audio, args.sr)
        t1 = time.perf_counter()
        dt = t1 - t0
        times.append(dt)
        log(f"Timed run {i + 1}/{args.iters}: done ({nseg} segments) in {dt:.2f}s")

    avg = sum(times) / len(times)
    rtf = audio_seconds / avg if avg > 0 else 0.0
    log(f"RESULT: avg={avg:.2f}s/run  RTF={rtf:.2f}x realtime")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
