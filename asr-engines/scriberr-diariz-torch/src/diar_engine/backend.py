from __future__ import annotations

from dataclasses import dataclass
import json
import time
from pathlib import Path
from typing import Any, Callable

import soundfile as sf

from .model_manager import ModelManager, ModelSpec, LoadedModel
from .params import JobParams, ensure_output_dir


class CancelledError(RuntimeError):
    pass


@dataclass
class DiarizeResult:
    diarization_path: str
    rttm_path: str | None
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


class DiarBackend:
    def __init__(self, model_manager: ModelManager) -> None:
        self._model_manager = model_manager

    def diarize(
        self,
        input_path: str,
        output_dir: str,
        params: JobParams,
        cancel_token: CancelToken | None = None,
        progress_cb: Callable[[float, str], None] | None = None,
    ) -> DiarizeResult:
        ensure_output_dir(output_dir)
        loaded = self._model_manager.get_loaded()
        if not loaded:
            raise RuntimeError("No model loaded")

        if progress_cb:
            progress_cb(0.0, "RUNNING")

        spec = loaded.spec
        if params.model and params.model != spec.model_name:
            spec = ModelSpec(
                model_id=spec.model_id,
                model_name=params.model,
                model_path=spec.model_path,
                providers=spec.providers,
                intra_op_threads=spec.intra_op_threads,
                vad_backend=spec.vad_backend,
            )
        loaded = self._model_manager.ensure_loaded(spec, params.hf_token)

        audio_seconds = _get_audio_duration(input_path)

        segments = _run_diarization(loaded, input_path, params)
        if cancel_token and cancel_token.is_cancelled():
            raise CancelledError("Job cancelled")

        diarization_path = str(Path(output_dir) / "diarization.json")
        rttm_path = str(Path(output_dir) / "diarization.rttm")
        result_path = str(Path(output_dir) / "result.json")

        diarization_payload = _build_json_payload(
            input_path=input_path,
            model_id=loaded.spec.model_id,
            model_name=loaded.spec.model_name,
            segments=segments,
            audio_seconds=audio_seconds,
        )

        with open(diarization_path, "w", encoding="utf-8") as f:
            json.dump(diarization_payload, f, ensure_ascii=False, indent=2)

        rttm_written = None
        if params.output_format.lower() == "rttm":
            _write_rttm(rttm_path, input_path, segments)
            rttm_written = rttm_path
        else:
            rttm_path = None

        result = {
            "model_id": loaded.spec.model_id,
            "model_name": loaded.spec.model_name,
            "audio_path": input_path,
            "output_dir": output_dir,
            "segment_count": len(segments),
            "audio_seconds": audio_seconds,
            "created_unix_ms": int(time.time() * 1000),
            "params": {
                "output_format": params.output_format,
                "min_speakers": params.min_speakers,
                "max_speakers": params.max_speakers,
                "segmentation_onset": params.segmentation_onset,
                "segmentation_offset": params.segmentation_offset,
                "batch_size": params.batch_size,
                "streaming_mode": params.streaming_mode,
                "chunk_length_s": params.chunk_length_s,
                "chunk_len": params.chunk_len,
                "chunk_right_context": params.chunk_right_context,
                "fifo_len": params.fifo_len,
                "spkcache_update_period": params.spkcache_update_period,
            },
            "outputs": {
                "diarization": diarization_path,
                "rttm": rttm_written or "",
            },
        }
        with open(result_path, "w", encoding="utf-8") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)

        if progress_cb:
            progress_cb(1.0, "COMPLETED")

        return DiarizeResult(
            diarization_path=diarization_path,
            rttm_path=rttm_written,
            result_path=result_path,
            segment_count=len(segments),
            audio_seconds=audio_seconds,
            model_id=loaded.spec.model_id,
        )


def _run_diarization(loaded: LoadedModel, input_path: str, params: JobParams) -> list[dict[str, Any]]:
    if loaded.kind == "pyannote":
        return _run_pyannote(loaded, input_path, params)
    if loaded.kind == "sortformer":
        return _run_sortformer(loaded, input_path, params)
    raise RuntimeError(f"Unknown diarization model kind: {loaded.kind}")


def _run_pyannote(loaded: LoadedModel, input_path: str, params: JobParams) -> list[dict[str, Any]]:
    import torch

    pipeline = loaded.model

    device = _resolve_device(params.device)
    if device == "auto":
        device = "cuda" if torch.cuda.is_available() else "cpu"
    if device == "cuda" and torch.cuda.is_available():
        pipeline = pipeline.to(torch.device("cuda"))
    else:
        pipeline = pipeline.to(torch.device("cpu"))

    if params.torch_threads:
        torch.set_num_threads(params.torch_threads)
    if params.torch_interop_threads:
        torch.set_num_interop_threads(params.torch_interop_threads)

    _apply_pyannote_segmentation_thresholds(
        pipeline, params.segmentation_onset, params.segmentation_offset
    )

    diarization_params: dict[str, Any] = {}
    if params.min_speakers is not None:
        diarization_params["min_speakers"] = params.min_speakers
    if params.max_speakers is not None:
        diarization_params["max_speakers"] = params.max_speakers
    if params.segmentation_batch_size is not None:
        diarization_params["segmentation_batch_size"] = params.segmentation_batch_size
    if params.embedding_batch_size is not None:
        diarization_params["embedding_batch_size"] = params.embedding_batch_size
    if params.embedding_exclude_overlap is not None:
        diarization_params["embedding_exclude_overlap"] = params.embedding_exclude_overlap

    if params.exclusive:
        diarization_params["exclusive"] = True

    diarization = pipeline(input_path, **diarization_params) if diarization_params else pipeline(input_path)
    return _pyannote_segments_to_dicts(diarization)


def _run_sortformer(loaded: LoadedModel, input_path: str, params: JobParams) -> list[dict[str, Any]]:
    diar_model = loaded.model

    if params.streaming_mode and hasattr(diar_model, "setup_streaming_params"):
        diar_model.setup_streaming_params(
            chunk_len=params.chunk_len,
            chunk_right_context=params.chunk_right_context,
            fifo_len=params.fifo_len,
            spkcache_update_period=params.spkcache_update_period,
        )

    predicted_segments = diar_model.diarize(audio=input_path, batch_size=params.batch_size)
    return _sortformer_segments_to_dicts(predicted_segments)


def _apply_pyannote_segmentation_thresholds(pipeline, onset: float | None, offset: float | None) -> None:
    if onset is None and offset is None:
        return
    try:
        params = pipeline.parameters(instantiated=True)
        if "segmentation" in params:
            if onset is not None:
                params["segmentation"]["threshold"] = onset
            if offset is not None:
                params["segmentation"]["min_duration_off"] = offset
            pipeline.instantiate(params)
    except Exception:
        return


def _pyannote_segments_to_dicts(diarization) -> list[dict[str, Any]]:
    segments: list[dict[str, Any]] = []
    if hasattr(diarization, "speaker_diarization"):
        for turn, speaker in diarization.speaker_diarization:
            segments.append(
                {
                    "start": float(turn.start),
                    "end": float(turn.end),
                    "speaker": str(speaker),
                    "duration": float(turn.duration),
                    "confidence": 1.0,
                }
            )
    elif hasattr(diarization, "itertracks"):
        for segment, _, speaker in diarization.itertracks(yield_label=True):
            segments.append(
                {
                    "start": float(segment.start),
                    "end": float(segment.end),
                    "speaker": str(speaker),
                    "duration": float(segment.end - segment.start),
                    "confidence": 1.0,
                }
            )
    segments.sort(key=lambda s: s["start"])
    return segments


def _sortformer_segments_to_dicts(segments: Any) -> list[dict[str, Any]]:
    entries: list[dict[str, Any]] = []

    if isinstance(segments, list) and len(segments) == 1 and isinstance(segments[0], list):
        segments = segments[0]

    for i, segment in enumerate(segments):
        if isinstance(segment, str):
            parts = segment.strip().split()
            if len(parts) < 3:
                continue
            start = float(parts[0])
            end = float(parts[1])
            speaker = str(parts[2])
            confidence = 1.0
        elif hasattr(segment, "start") and hasattr(segment, "end") and hasattr(segment, "label"):
            start = float(segment.start)
            end = float(segment.end)
            speaker = str(segment.label)
            confidence = float(getattr(segment, "confidence", 1.0))
        elif isinstance(segment, (list, tuple)) and len(segment) >= 3:
            start = float(segment[0])
            end = float(segment[1])
            speaker = str(segment[2])
            confidence = 1.0
        elif isinstance(segment, dict):
            start = float(segment.get("start", 0))
            end = float(segment.get("end", 0))
            speaker = str(segment.get("speaker", segment.get("label", f"speaker_{i}")))
            confidence = float(segment.get("confidence", 1.0))
        else:
            start = float(getattr(segment, "start", 0))
            end = float(getattr(segment, "end", 0))
            speaker = str(getattr(segment, "label", getattr(segment, "speaker", f"speaker_{i}")))
            confidence = float(getattr(segment, "confidence", 1.0))

        entries.append(
            {
                "start": start,
                "end": end,
                "speaker": speaker,
                "duration": end - start,
                "confidence": confidence,
            }
        )

    entries.sort(key=lambda s: s["start"])
    return entries


def _build_json_payload(
    input_path: str,
    model_id: str,
    model_name: str,
    segments: list[dict[str, Any]],
    audio_seconds: float,
) -> dict[str, Any]:
    speakers = sorted({segment["speaker"] for segment in segments})
    return {
        "audio_file": input_path,
        "model": model_name,
        "model_id": model_id,
        "segments": segments,
        "speakers": speakers,
        "speaker_count": len(speakers),
        "total_duration": audio_seconds,
        "processing_info": {
            "total_segments": len(segments),
            "total_speech_time": sum(segment["duration"] for segment in segments),
        },
    }


def _write_rttm(path: str, audio_path: str, segments: list[dict[str, Any]]) -> None:
    audio_filename = Path(audio_path).stem
    with open(path, "w", encoding="utf-8") as f:
        for segment in segments:
            start = float(segment["start"])
            end = float(segment["end"])
            duration = end - start
            speaker = segment["speaker"]
            line = f"SPEAKER {audio_filename} 1 {start:.3f} {duration:.3f} <NA> <NA> {speaker} <NA> <NA>\n"
            f.write(line)


def _resolve_device(device: str) -> str:
    if not device:
        return "auto"
    return device.lower()


def _get_audio_duration(path: str) -> float:
    try:
        info = sf.info(path)
        if info.frames and info.samplerate:
            return float(info.frames) / float(info.samplerate)
        return float(info.duration) if info.duration else 0.0
    except Exception:
        return 0.0
