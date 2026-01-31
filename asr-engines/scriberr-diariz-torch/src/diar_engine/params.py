from dataclasses import dataclass
from typing import Any


def _parse_bool(value: str | None, default: bool) -> bool:
    if value is None:
        return default
    return value.strip().lower() in {"1", "true", "yes", "y", "on"}


def _parse_int(value: str | None, default: int | None) -> int | None:
    if value is None:
        return default
    try:
        return int(value)
    except ValueError:
        return default


def _parse_float(value: str | None, default: float | None) -> float | None:
    if value is None:
        return default
    try:
        return float(value)
    except ValueError:
        return default


@dataclass
class JobParams:
    output_format: str = "rttm"
    device: str = "auto"
    hf_token: str | None = None
    model: str | None = None

    min_speakers: int | None = None
    max_speakers: int | None = None

    segmentation_onset: float | None = 0.5
    segmentation_offset: float | None = 0.363

    batch_size: int = 1
    streaming_mode: bool = False
    chunk_length_s: float = 30.0

    chunk_len: int = 340
    chunk_right_context: int = 40
    fifo_len: int = 40
    spkcache_update_period: int = 300
    exclusive: bool = True
    segmentation_batch_size: int | None = None
    embedding_batch_size: int | None = None
    embedding_exclude_overlap: bool | None = None
    torch_threads: int | None = None
    torch_interop_threads: int | None = None

    @classmethod
    def from_kv(cls, params: dict[str, str]) -> "JobParams":
        return cls(
            output_format=params.get("output_format", "rttm"),
            device=params.get("device", "auto"),
            hf_token=params.get("hf_token") or None,
            model=params.get("model") or None,
            min_speakers=_parse_int(params.get("min_speakers"), None),
            max_speakers=_parse_int(params.get("max_speakers"), None),
            segmentation_onset=_parse_float(params.get("segmentation_onset"), 0.5),
            segmentation_offset=_parse_float(params.get("segmentation_offset"), 0.363),
            batch_size=_parse_int(params.get("batch_size"), 1) or 1,
            streaming_mode=_parse_bool(params.get("streaming_mode"), False),
            chunk_length_s=_parse_float(params.get("chunk_length_s"), 30.0) or 30.0,
            chunk_len=_parse_int(params.get("chunk_len"), 340) or 340,
            chunk_right_context=_parse_int(params.get("chunk_right_context"), 40) or 40,
            fifo_len=_parse_int(params.get("fifo_len"), 40) or 40,
            spkcache_update_period=_parse_int(params.get("spkcache_update_period"), 300)
            or 300,
            exclusive=_parse_bool(params.get("exclusive"), False),
            segmentation_batch_size=_parse_int(params.get("segmentation_batch_size"), None),
            embedding_batch_size=_parse_int(params.get("embedding_batch_size"), None),
            embedding_exclude_overlap=(
                _parse_bool(params.get("embedding_exclude_overlap"), False)
                if params.get("embedding_exclude_overlap") is not None
                else None
            ),
            torch_threads=_parse_int(params.get("torch_threads"), None),
            torch_interop_threads=_parse_int(params.get("torch_interop_threads"), None),
        )


def ensure_output_dir(path: str) -> None:
    import os

    os.makedirs(path, exist_ok=True)


def to_dict(obj: Any) -> dict[str, Any]:
    if hasattr(obj, "__dict__"):
        return dict(obj.__dict__)
    return dict(obj)
