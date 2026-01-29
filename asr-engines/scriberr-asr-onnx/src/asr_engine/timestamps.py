from __future__ import annotations

import json
from pathlib import Path

from .postprocess import Segment


def format_ts(seconds: float | None) -> str:
    if seconds is None:
        return "??:??:??.???"
    h = int(seconds // 3600)
    m = int((seconds % 3600) // 60)
    s = seconds % 60
    return f"{h:02d}:{m:02d}:{s:06.3f}"


def word_timestamps_from_segment(text: str, start: float | None, end: float | None) -> list[dict[str, float | str]]:
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

    out[-1]["end"] = end
    return out


def write_transcript(path: str | Path, segments: list[Segment]) -> None:
    output = " ".join(s.text for s in segments if s.text).strip()
    Path(path).write_text(output + "\n", encoding="utf-8")


def write_segments_jsonl(path: str | Path, segments: list[Segment]) -> None:
    with open(path, "w", encoding="utf-8") as f:
        for idx, s in enumerate(segments, start=1):
            rec = {
                "segment_index": idx,
                "start": s.start,
                "end": s.end,
                "start_hhmmss": format_ts(s.start),
                "end_hhmmss": format_ts(s.end),
                "text": s.text,
            }
            f.write(json.dumps(rec, ensure_ascii=False) + "\n")


def write_words_jsonl(path: str | Path, segments: list[Segment]) -> None:
    with open(path, "w", encoding="utf-8") as f:
        global_word_index = 0
        for si, s in enumerate(segments, start=1):
            wt = word_timestamps_from_segment(s.text, s.start, s.end)
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
