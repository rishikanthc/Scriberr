#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
import time
import uuid
from pathlib import Path

import grpc

try:
    from asr_engine.proto import asr_engine_pb2 as pb2
    from asr_engine.proto import asr_engine_pb2_grpc as pb2_grpc
except ModuleNotFoundError:
    import sys

    repo_root = Path(__file__).resolve().parents[1]
    asr_src = repo_root / "asr-engines" / "scriberr-asr-onnx" / "src"
    sys.path.append(str(asr_src))
    from asr_engine.proto import asr_engine_pb2 as pb2
    from asr_engine.proto import asr_engine_pb2_grpc as pb2_grpc


def _dial(socket_path: str) -> pb2_grpc.AsrEngineStub:
    target = f"unix:{socket_path}"
    channel = grpc.insecure_channel(target)
    return pb2_grpc.AsrEngineStub(channel)


def _wait_for_engine(stub: pb2_grpc.AsrEngineStub, label: str, timeout_s: float = 15.0) -> None:
    deadline = time.time() + timeout_s
    last_err: Exception | None = None
    while time.time() < deadline:
        try:
            stub.GetEngineInfo(pb2.GetEngineInfoRequest())
            return
        except Exception as exc:  # noqa: BLE001
            last_err = exc
            time.sleep(0.3)
    raise RuntimeError(f"{label} engine not ready: {last_err}")


def _run_job(
    stub: pb2_grpc.AsrEngineStub,
    job_id: str,
    input_path: str,
    output_dir: str,
    params: dict[str, str],
) -> pb2.JobStatus:
    stub.StartJob(
        pb2.StartJobRequest(
            job_id=job_id,
            input_path=input_path,
            output_dir=output_dir,
            params=params,
        )
    )
    stream = stub.StreamJobStatus(pb2.StreamJobStatusRequest(job_id=job_id))
    final = None
    for status in stream:
        final = status
        if status.state in (
            pb2.JobState.Value("JOB_STATE_COMPLETED"),
            pb2.JobState.Value("JOB_STATE_FAILED"),
            pb2.JobState.Value("JOB_STATE_CANCELLED"),
        ):
            break
    if final is None:
        raise RuntimeError("job stream ended without status")
    if final.state != pb2.JobState.Value("JOB_STATE_COMPLETED"):
        raise RuntimeError(f"job failed: {final.message}")
    return final


def _load_words(path: Path) -> list[dict]:
    words = []
    with path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            words.append(json.loads(line))
    return words


def _load_diarization(path: Path) -> list[dict]:
    data = json.loads(path.read_text(encoding="utf-8"))
    return data.get("segments", [])


def _estimate_offset(words: list[dict], diar_segments: list[dict]) -> float:
    if not words or not diar_segments:
        return 0.0

    def coverage(offset: float) -> int:
        count = 0
        for w in words:
            start = float(w.get("start", 0))
            end = float(w.get("end", start))
            mid = (start + end) / 2 if end > start else start
            for seg in diar_segments:
                if mid >= float(seg["start"]) + offset and mid <= float(seg["end"]) + offset:
                    count += 1
                    break
        return count

    base = coverage(0.0)
    best_offset = 0.0
    best = base
    offset = -2.0
    while offset <= 2.0001:
        score = coverage(offset)
        if score > best:
            best = score
            best_offset = offset
        offset += 0.05

    min_gain = max(2, int(len(words) * 0.05))
    if best - base < min_gain:
        return 0.0
    return round(best_offset, 3)


def _assign_speakers(words: list[dict], diar_segments: list[dict], offset: float = 0.0) -> list[dict]:
    def best_speaker(start: float, end: float) -> str | None:
        best = None
        max_overlap = 0.0
        for seg in diar_segments:
            overlap_start = max(start, float(seg["start"]) + offset)
            overlap_end = min(end, float(seg["end"]) + offset)
            overlap = max(0.0, overlap_end - overlap_start)
            if overlap > max_overlap:
                max_overlap = overlap
                best = str(seg["speaker"])
        return best

    out = []
    for word in words:
        start = float(word.get("start", 0))
        end = float(word.get("end", start))
        speaker = best_speaker(start, end)
        entry = dict(word)
        entry["speaker"] = speaker
        out.append(entry)
    return out


def _write_alignment(path: Path, words_with_speakers: list[dict], offset: float) -> None:
    stats = {
        "word_count": len(words_with_speakers),
        "speaker_counts": {},
        "unassigned": 0,
        "offset_s": offset,
    }
    for w in words_with_speakers:
        spk = w.get("speaker")
        if not spk:
            stats["unassigned"] += 1
            continue
        stats["speaker_counts"][spk] = stats["speaker_counts"].get(spk, 0) + 1

    payload = {
        "stats": stats,
        "words": words_with_speakers,
    }
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")


def _write_alignment_text(path: Path, words_with_speakers: list[dict]) -> None:
    lines = []
    for w in words_with_speakers:
        speaker = w.get("speaker") or "UNKNOWN"
        word = str(w.get("word", "")).strip()
        if not word:
            continue
        lines.append(f"{speaker} - {word}")
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--audio", required=True, help="Path to audio file")
    parser.add_argument("--out", required=True, help="Output directory")
    parser.add_argument("--asr-socket", default="/tmp/scriberr-asr.sock")
    parser.add_argument("--diar-socket", default="/tmp/scriberr-diar.sock")
    parser.add_argument("--asr-model", default="nemo-parakeet-tdt-0.6b-v3")
    parser.add_argument("--diar-model", default="nvidia/diar_sortformer_4spk-v1")
    parser.add_argument("--run-pyannote", action="store_true")
    parser.add_argument("--sortformer-streaming", action="store_true")
    parser.add_argument("--sortformer-batch", type=int, default=1)
    parser.add_argument("--chunk-len", type=int, default=340)
    parser.add_argument("--chunk-right-context", type=int, default=40)
    parser.add_argument("--fifo-len", type=int, default=40)
    parser.add_argument("--spkcache-update-period", type=int, default=300)
    parser.add_argument("--pyannote-seg-batch", type=int, default=None)
    parser.add_argument("--pyannote-emb-batch", type=int, default=None)
    parser.add_argument("--pyannote-exclude-overlap", action="store_true")
    parser.add_argument("--torch-threads", type=int, default=None)
    parser.add_argument("--torch-interop-threads", type=int, default=None)
    args = parser.parse_args()

    audio = Path(args.audio).resolve()
    out_dir = Path(args.out).resolve()
    out_dir.mkdir(parents=True, exist_ok=True)

    asr_stub = _dial(args.asr_socket)
    diar_stub = _dial(args.diar_socket)

    _wait_for_engine(asr_stub, "ASR")
    _wait_for_engine(diar_stub, "Diarization")

    asr_spec = pb2.ModelSpec(model_id="parakeet", model_name=args.asr_model)
    asr_stub.LoadModel(pb2.LoadModelRequest(spec=asr_spec))

    diar_spec = pb2.ModelSpec(model_id="sortformer", model_name=args.diar_model)
    diar_stub.LoadModel(pb2.LoadModelRequest(spec=diar_spec))

    asr_job = f"asr-{uuid.uuid4()}"
    diar_job = f"diar-{uuid.uuid4()}"

    asr_dir = out_dir / "asr"
    diar_dir = out_dir / "diar"
    asr_dir.mkdir(parents=True, exist_ok=True)
    diar_dir.mkdir(parents=True, exist_ok=True)

    asr_start = time.time()
    asr_status = _run_job(
        asr_stub,
        asr_job,
        str(audio),
        str(asr_dir),
        {
            "include_segments": "true",
            "include_words": "true",
            "vad_preset": "balanced",
            "vad_enabled": "true",
        },
    )
    asr_elapsed = time.time() - asr_start

    diar_params = {
        "output_format": "json",
        "device": "auto",
        "max_speakers": "4",
        "batch_size": str(args.sortformer_batch),
    }
    if args.sortformer_streaming:
        diar_params["streaming_mode"] = "true"
        diar_params["chunk_len"] = str(args.chunk_len)
        diar_params["chunk_right_context"] = str(args.chunk_right_context)
        diar_params["fifo_len"] = str(args.fifo_len)
        diar_params["spkcache_update_period"] = str(args.spkcache_update_period)

    diar_start = time.time()
    diar_status = _run_job(
        diar_stub,
        diar_job,
        str(audio),
        str(diar_dir),
        diar_params,
    )
    diar_elapsed = time.time() - diar_start

    words_path = Path(asr_status.outputs["words"])
    diar_path = Path(diar_status.outputs["diarization"])

    words = _load_words(words_path)
    diar_segments = _load_diarization(diar_path)
    aligned = _assign_speakers(words, diar_segments)
    offset = _estimate_offset(words, diar_segments)
    aligned = _assign_speakers(words, diar_segments, offset=offset)
    _write_alignment(out_dir / "align_sortformer.json", aligned, offset)
    _write_alignment_text(out_dir / "align_sortformer.txt", aligned)

    if args.run_pyannote:
        diar_spec = pb2.ModelSpec(model_id="pyannote", model_name="pyannote/speaker-diarization-community-1")
        diar_stub.LoadModel(pb2.LoadModelRequest(spec=diar_spec))
        diar_job = f"diar-pyannote-{uuid.uuid4()}"
        diar_dir = out_dir / "diar_pyannote"
        diar_dir.mkdir(parents=True, exist_ok=True)
        pyannote_params = {
            "output_format": "json",
            "device": "auto",
            "max_speakers": "4",
            "exclusive": "true",
        }
        if args.pyannote_seg_batch:
            pyannote_params["segmentation_batch_size"] = str(args.pyannote_seg_batch)
        if args.pyannote_emb_batch:
            pyannote_params["embedding_batch_size"] = str(args.pyannote_emb_batch)
        if args.pyannote_exclude_overlap:
            pyannote_params["embedding_exclude_overlap"] = "true"
        if args.torch_threads:
            pyannote_params["torch_threads"] = str(args.torch_threads)
        if args.torch_interop_threads:
            pyannote_params["torch_interop_threads"] = str(args.torch_interop_threads)

        diar_status = _run_job(
            diar_stub,
            diar_job,
            str(audio),
            str(diar_dir),
            pyannote_params,
        )
        diar_path = Path(diar_status.outputs["diarization"])
        diar_segments = _load_diarization(diar_path)
        offset = _estimate_offset(words, diar_segments)
        aligned = _assign_speakers(words, diar_segments, offset=offset)
        _write_alignment(out_dir / "align_pyannote.json", aligned, offset)
        _write_alignment_text(out_dir / "align_pyannote.txt", aligned)

    print(f"ASR outputs: {asr_dir} (elapsed {asr_elapsed:.2f}s)")
    print(f"Diar outputs: {out_dir} (elapsed {diar_elapsed:.2f}s)")


if __name__ == "__main__":
    main()
