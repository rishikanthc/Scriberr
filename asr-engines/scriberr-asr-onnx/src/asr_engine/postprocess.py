from __future__ import annotations

from dataclasses import dataclass
from typing import Iterable


@dataclass
class Segment:
    text: str
    start: float | None = None
    end: float | None = None

    @property
    def duration(self) -> float | None:
        if self.start is None or self.end is None:
            return None
        return max(0.0, self.end - self.start)


def merge_short_segments(
    segments: Iterable[Segment],
    attach_threshold_s: float = 0.25,
    attach_max_words: int = 2,
) -> list[Segment]:
    merged: list[Segment] = []
    prev: Segment | None = None

    for seg in segments:
        text = seg.text.strip()
        if not text:
            continue
        seg = Segment(text=text, start=seg.start, end=seg.end)
        seg_dur = seg.duration
        attach_by_duration = seg_dur is not None and seg_dur < attach_threshold_s
        attach_by_words = len(text.split()) <= attach_max_words

        if prev is not None and (attach_by_duration or attach_by_words):
            prev.text = f"{prev.text} {seg.text}".strip()
            prev.end = seg.end if seg.end is not None else prev.end
            merged[-1] = prev
        else:
            merged.append(seg)
            prev = seg

    return merged
