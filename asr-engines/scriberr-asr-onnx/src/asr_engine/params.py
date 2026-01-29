from dataclasses import dataclass
from typing import Any


def get_vad_params(preset: str = "balanced") -> dict[str, int | float]:
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
    return presets.get(preset, presets["balanced"]).copy()


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
    vad_preset: str = "balanced"
    vad_speech_pad_ms: int | None = None
    vad_min_silence_ms: int | None = None
    vad_min_speech_ms: int | None = None
    vad_max_speech_s: int | None = None

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
            vad_preset=params.get("vad_preset", "balanced"),
            vad_speech_pad_ms=_parse_int(params.get("vad_speech_pad_ms"), None),
            vad_min_silence_ms=_parse_int(params.get("vad_min_silence_ms"), None),
            vad_min_speech_ms=_parse_int(params.get("vad_min_speech_ms"), None),
            vad_max_speech_s=_parse_int(params.get("vad_max_speech_s"), None),
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

    def resolved_vad_params(self) -> dict[str, int | float]:
        vad_params = get_vad_params(self.vad_preset)
        if self.vad_speech_pad_ms is not None:
            vad_params["speech_pad_ms"] = self.vad_speech_pad_ms
        if self.vad_min_silence_ms is not None:
            vad_params["min_silence_duration_ms"] = self.vad_min_silence_ms
        if self.vad_min_speech_ms is not None:
            vad_params["min_speech_duration_ms"] = self.vad_min_speech_ms
        if self.vad_max_speech_s is not None:
            vad_params["max_speech_duration_s"] = self.vad_max_speech_s
        return vad_params


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
