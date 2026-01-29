#!/usr/bin/env python3
import sys
import argparse
import logging
from pathlib import Path
import json

import numpy as np
import librosa
import onnxruntime as rt
import onnx_asr

logging.basicConfig(level=logging.ERROR)


def format_ts(seconds: float) -> str:
    """HH:MM:SS.mmm"""
    if seconds is None:
        return "??:??:??.???"
    h = int(seconds // 3600)
    m = int((seconds % 3600) // 60)
    s = seconds % 60
    return f"{h:02d}:{m:02d}:{s:06.3f}"


def word_timestamps_from_segment(text: str, start: float, end: float):
    """
    Heuristic word timestamps within a segment.

    Strategy:
      - split on whitespace
      - allocate duration proportionally to word length (chars) to avoid giving equal time to "a" and "velociraptor"
    This is NOT forced alignment and will not match word boundaries perfectly.
    """
    words = [w for w in text.strip().split() if w]
    if start is None or end is None or end <= start or not words:
        return []

    dur = end - start
    lengths = [max(1, len(w)) for w in words]
    total = float(sum(lengths))

    out = []
    t = start
    for w, L in zip(words, lengths):
        w_dur = dur * (L / total)
        w_start = t
        w_end = t + w_dur
        out.append({"word": w, "start": w_start, "end": w_end})
        t = w_end

    # ensure last word ends exactly at segment end (numerical stability)
    out[-1]["end"] = end
    return out


def transcribe_and_dump(
    audio_path: str, model_dir: str, out_prefix: str | None, use_local_export: bool
):
    audio_file = Path(audio_path)
    model_path = Path(model_dir)

    if use_local_export and not model_path.exists():
        print(f"Error: model_dir '{model_dir}' not found.")
        return 2

    if out_prefix is None:
        out_prefix = str(audio_file.with_suffix(""))

    out_transcript = Path(out_prefix + ".transcript.txt")
    out_segments = Path(out_prefix + ".segments.jsonl")
    out_words = Path(out_prefix + ".words.jsonl")

    # ONNXRuntime session opts
    opts = rt.SessionOptions()
    opts.intra_op_num_threads = 8

    print("Initializing VAD + Parakeet pipeline...")
    try:
        vad = onnx_asr.load_vad("silero")

        load_kwargs = dict(
            model="nemo-parakeet-tdt-0.6b-v2",
            providers=["CPUExecutionProvider"],
            sess_options=opts,
        )
        if use_local_export:
            load_kwargs["path"] = model_path

        asr = onnx_asr.load_model(**load_kwargs).with_vad(vad)

        print("✓ Pipeline initialized.")
    except Exception as e:
        print(f"✗ Load Error: {e}")
        return 2

    print(f"Loading audio: {audio_file.name}")
    audio, _ = librosa.load(str(audio_file), sr=16000, mono=True)
    audio = audio.astype(np.float32)

    print("Transcribing...")
    segments = []
    try:
        results = asr.recognize(audio, sample_rate=16000)
        for seg in results:
            text = seg.text.strip()
            start = getattr(seg, "start", None)
            end = getattr(seg, "end", None)
            segments.append({"text": text, "start": start, "end": end})
    except Exception as e:
        print(f"✗ Transcription Error: {e}")
        return 2

    # 1) Transcript file (no timestamps)
    with open(out_transcript, "w", encoding="utf-8") as f:
        f.write(" ".join(s["text"] for s in segments if s["text"]).strip() + "\n")

    # 2) Segment-level timestamps (real)
    with open(out_segments, "w", encoding="utf-8") as f:
        for i, s in enumerate(segments, start=1):
            rec = {
                "segment_index": i,
                "start": s["start"],
                "end": s["end"],
                "start_hhmmss": format_ts(s["start"]),
                "end_hhmmss": format_ts(s["end"]),
                "text": s["text"],
            }
            f.write(json.dumps(rec, ensure_ascii=False) + "\n")

    # 3) Word-level timestamps (heuristic)
    with open(out_words, "w", encoding="utf-8") as f:
        global_word_index = 0
        for si, s in enumerate(segments, start=1):
            wt = word_timestamps_from_segment(s["text"], s["start"], s["end"])
            for wi, wrec in enumerate(wt, start=1):
                global_word_index += 1
                rec = {
                    "global_word_index": global_word_index,
                    "segment_index": si,
                    "word_index_in_segment": wi,
                    "word": wrec["word"],
                    "start": wrec["start"],
                    "end": wrec["end"],
                    "start_hhmmss": format_ts(wrec["start"]),
                    "end_hhmmss": format_ts(wrec["end"]),
                }
                f.write(json.dumps(rec, ensure_ascii=False) + "\n")

    print("✓ Wrote:")
    print(f"  - {out_transcript}")
    print(f"  - {out_segments}")
    print(f"  - {out_words}")

    return 0


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("audio_file", help="Path to audio file")
    ap.add_argument(
        "--model-dir", default="nemo_global_onnx", help="Local ONNX export directory"
    )
    ap.add_argument(
        "--out-prefix",
        default=None,
        help="Prefix for output files (default: audio path without extension)",
    )
    ap.add_argument(
        "--use-local-export",
        action="store_true",
        help="Use local exported ONNX model from --model-dir. If not set, loads the HF model by name.",
    )
    args = ap.parse_args()

    return transcribe_and_dump(
        audio_path=args.audio_file,
        model_dir=args.model_dir,
        out_prefix=args.out_prefix,
        use_local_export=args.use_local_export,
    )


if __name__ == "__main__":
    raise SystemExit(main())
