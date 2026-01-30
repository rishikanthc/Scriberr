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

        if params.vad_enabled:
            vad_params = params.resolved_vad_params()
            asr = loaded.asr_base.with_vad(loaded.vad, **vad_params)
        else:
            asr = loaded.asr_base

        if params.include_words and hasattr(asr, "with_timestamps"):
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

        results = asr.recognize(audio, **recognize_kwargs)
        for seg in results:
            if cancel_token and cancel_token.is_cancelled():
                raise CancelledError("Job cancelled")
            text = getattr(seg, "text", "").strip()
            start = getattr(seg, "start", None)
            end = getattr(seg, "end", None)
            segments.append(Segment(text=text, start=start, end=end))
            segment_index += 1

            if params.include_words:
                tokens = getattr(seg, "tokens", None)
                timestamps = getattr(seg, "timestamps", None)
                words = word_timestamps_from_tokens(tokens, timestamps, start, end)
                if not words:
                    words = word_timestamps_from_segment(text, start, end)
                for wi, wrec in enumerate(words, start=1):
                    rec = {
                        "global_word_index": len(word_entries) + 1,
                        "segment_index": segment_index,
                        "word_index_in_segment": wi,
                        "word": wrec["word"],
                        "start": wrec["start"],
                        "end": wrec["end"],
                    }
                    word_entries.append(rec)
            if progress_cb and end is not None and audio_seconds > 0:
                progress = min(1.0, max(0.0, float(end) / audio_seconds))
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
            "vad_enabled": params.vad_enabled,
            "vad_preset": params.vad_preset,
                "vad_speech_pad_ms": params.vad_speech_pad_ms,
                "vad_min_silence_ms": params.vad_min_silence_ms,
                "vad_min_speech_ms": params.vad_min_speech_ms,
                "vad_max_speech_s": params.vad_max_speech_s,
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
