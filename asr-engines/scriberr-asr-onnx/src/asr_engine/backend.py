from __future__ import annotations

from dataclasses import dataclass
import inspect
import json
import time
from pathlib import Path
from typing import Any, Callable

import numpy as np

from .audio_io import load_audio
from .model_manager import ModelManager
from .params import JobParams, ensure_output_dir
from .postprocess import Segment, merge_short_segments
from .timestamps import (
    split_segments_from_words,
    word_timestamps_from_segment,
    word_timestamps_from_tokens,
    write_segments_jsonl,
    write_transcript,
    write_words_jsonl_from_entries,
)


class CancelledError(RuntimeError):
    pass


@dataclass
class TranscribeResult:
    transcript_path: str
    segments_path: str | None
    words_path: str | None
    result_path: str
    segment_count: int
    audio_seconds: float
    model_id: str


class CancelToken:
    def __init__(self) -> None:
        import threading

        self._event = threading.Event()

    def cancel(self) -> None:
        self._event.set()

    def is_cancelled(self) -> bool:
        return self._event.is_set()


class AsrBackend:
    def __init__(self, model_manager: ModelManager) -> None:
        self._model_manager = model_manager

    def transcribe(
        self,
        input_path: str,
        output_dir: str,
        params: JobParams,
        cancel_token: CancelToken | None = None,
        progress_cb: Callable[[float, str], None] | None = None,
    ) -> TranscribeResult:
        ensure_output_dir(output_dir)
        loaded = self._model_manager.get_loaded()
        if not loaded:
            raise RuntimeError("No model loaded")

        audio, sr = load_audio(input_path, sample_rate=params.sample_rate)
        audio = audio.astype(np.float32)
        audio_seconds = len(audio) / float(sr) if sr else 0.0
        asr = loaded.asr_base
        if (params.include_words or params.include_segments) and hasattr(asr, "with_timestamps"):
            asr = asr.with_timestamps()

        segments: list[Segment] = []
        word_entries: list[dict[str, float | str | int]] = []
        segment_index = 0
        recognize_kwargs: dict[str, Any] = {"sample_rate": sr}
        if params.language:
            recognize_kwargs["language"] = params.language
        if params.target_language:
            recognize_kwargs["target_language"] = params.target_language
        if params.pnc is not None:
            recognize_kwargs["pnc"] = params.pnc

        sig = inspect.signature(asr.recognize)
        has_kwargs = any(
            p.kind == inspect.Parameter.VAR_KEYWORD for p in sig.parameters.values()
        )
        if not has_kwargs:
            recognize_kwargs = {
                k: v for k, v in recognize_kwargs.items() if k in sig.parameters
            }

        chunk_len_s = max(1.0, float(params.chunk_len_s))
        batch_size = max(1, int(params.chunk_batch_size))
        chunk_samples = int(chunk_len_s * sr)
        chunks: list[tuple[np.ndarray, float, float]] = []
        for start in range(0, len(audio), chunk_samples):
            end = min(start + chunk_samples, len(audio))
            if end <= start:
                continue
            chunk = np.ascontiguousarray(audio[start:end], dtype=np.float32)
            start_s = start / float(sr)
            end_s = end / float(sr)
            chunks.append((chunk, start_s, end_s))

        for batch_start in range(0, len(chunks), batch_size):
            batch = chunks[batch_start : batch_start + batch_size]
            batch_audio = [c[0] for c in batch]
            results = asr.recognize(batch_audio, **recognize_kwargs)
            if not isinstance(results, list):
                results = [results]

            for (chunk_audio, start_s, end_s), res in zip(batch, results, strict=False):
                if cancel_token and cancel_token.is_cancelled():
                    raise CancelledError("Job cancelled")

                if isinstance(res, str):
                    text = res.strip()
                    tokens = None
                    timestamps = None
                else:
                    text = getattr(res, "text", "").strip()
                    tokens = getattr(res, "tokens", None)
                    timestamps = getattr(res, "timestamps", None)

                if not text:
                    continue

                seg_start = start_s
                seg_end = end_s
                words_local: list[dict[str, float | str]] = []

                if tokens and timestamps:
                    try:
                        seg_start = start_s + float(min(timestamps))
                        seg_end = start_s + float(max(timestamps))
                    except Exception:
                        seg_start = start_s
                        seg_end = end_s
                    words_local = word_timestamps_from_tokens(tokens, timestamps, seg_start, seg_end)

                if not words_local:
                    words_local = word_timestamps_from_segment(text, seg_start, seg_end)

                segments_local = split_segments_from_words(
                    words_local,
                    gap_s=params.segment_gap_s,
                )
                if not segments_local:
                    segments_local = [{"text": text, "start": seg_start, "end": seg_end, "words": words_local}]

                if params.include_words:
                    for seg in segments_local:
                        segments.append(
                            Segment(
                                text=str(seg["text"]),
                                start=float(seg["start"]),
                                end=float(seg["end"]),
                            )
                        )
                        segment_index += 1
                        for wi, wrec in enumerate(seg["words"], start=1):
                            rec = {
                                "global_word_index": len(word_entries) + 1,
                                "segment_index": segment_index,
                                "word_index_in_segment": wi,
                                "word": wrec["word"],
                                "start": wrec["start"],
                                "end": wrec["end"],
                            }
                            word_entries.append(rec)
                else:
                    for seg in segments_local:
                        segments.append(
                            Segment(
                                text=str(seg["text"]),
                                start=float(seg["start"]),
                                end=float(seg["end"]),
                            )
                        )
                        segment_index += 1

                if progress_cb and audio_seconds > 0:
                    progress = min(1.0, max(0.0, float(end_s) / audio_seconds))
                    progress_cb(progress, "RUNNING")

        if params.merge_short_segments:
            segments = merge_short_segments(
                segments,
                attach_threshold_s=params.merge_attach_threshold_s,
                attach_max_words=params.merge_attach_max_words,
            )

        transcript_path = str(Path(output_dir) / "transcript.txt")
        segments_path = str(Path(output_dir) / "segments.jsonl")
        words_path = str(Path(output_dir) / "words.jsonl")
        result_path = str(Path(output_dir) / "result.json")

        write_transcript(transcript_path, segments)

        if params.include_segments:
            write_segments_jsonl(segments_path, segments)
        else:
            segments_path = None

        if params.include_words:
            write_words_jsonl_from_entries(words_path, word_entries)
        else:
            words_path = None

        result = {
            "model_id": loaded.spec.model_id,
            "model_name": loaded.spec.model_name,
            "audio_path": input_path,
            "output_dir": output_dir,
            "segment_count": len(segments),
            "audio_seconds": audio_seconds,
            "created_unix_ms": int(time.time() * 1000),
            "params": {
                "language": params.language,
                "target_language": params.target_language,
                "pnc": params.pnc,
                "chunk_len_s": params.chunk_len_s,
                "chunk_batch_size": params.chunk_batch_size,
                "segment_gap_s": params.segment_gap_s,
            },
            "outputs": {
                "transcript": transcript_path,
                "segments": segments_path,
                "words": words_path,
            },
        }
        with open(result_path, "w", encoding="utf-8") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)

        return TranscribeResult(
            transcript_path=transcript_path,
            segments_path=segments_path,
            words_path=words_path,
            result_path=result_path,
            segment_count=len(segments),
            audio_seconds=audio_seconds,
            model_id=loaded.spec.model_id,
        )
