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


def _parse_optional_float(value: str | None) -> float | None:
    if value is None:
        return None
    if value.strip() == "":
        return None
    try:
        return float(value)
    except ValueError:
        return None


def _parse_pnc(value: str | None) -> str | bool | None:
    if value is None:
        return None
    val = value.strip().lower()
    if val in {"pnc", "nopnc"}:
        return val
    if val in {"1", "true", "yes", "y", "on"}:
        return True
    if val in {"0", "false", "no", "n", "off"}:
        return False
    return None


@dataclass
class JobParams:
    chunk_len_s: float = 300.0
    chunk_batch_size: int = 8
    segment_gap_s: float | None = None

    include_segments: bool = True
    include_words: bool = True
    merge_short_segments: bool = True
    merge_attach_threshold_s: float = 0.25
    merge_attach_max_words: int = 2

    sample_rate: int = 16000
    language: str | None = None
    target_language: str | None = None
    pnc: str | bool | None = None

    @classmethod
    def from_kv(cls, params: dict[str, str]) -> "JobParams":
        return cls(
            chunk_len_s=_parse_float(params.get("chunk_len_s"), 300.0) or 300.0,
            chunk_batch_size=_parse_int(params.get("chunk_batch_size"), 8) or 8,
            segment_gap_s=_parse_optional_float(params.get("segment_gap_s")),
            include_segments=_parse_bool(params.get("include_segments"), True),
            include_words=_parse_bool(params.get("include_words"), True),
            merge_short_segments=_parse_bool(params.get("merge_short_segments"), True),
            merge_attach_threshold_s=_parse_float(params.get("merge_attach_threshold_s"), 0.25)
            or 0.25,
            merge_attach_max_words=_parse_int(params.get("merge_attach_max_words"), 2) or 2,
            sample_rate=_parse_int(params.get("sample_rate"), 16000) or 16000,
            language=params.get("language") or None,
            target_language=params.get("target_language") or None,
            pnc=_parse_pnc(params.get("pnc")),
        )


def normalize_text(text: str) -> str:
    import re

    text = text.lower()
    text = re.sub(r"[^\w\s']", " ", text)
    text = re.sub(r"\s+", " ", text).strip()
    return text


def ensure_output_dir(path: str) -> None:
    import os

    os.makedirs(path, exist_ok=True)


def to_dict(obj: Any) -> dict[str, Any]:
    if hasattr(obj, "__dict__"):
        return dict(obj.__dict__)
    return dict(obj)
