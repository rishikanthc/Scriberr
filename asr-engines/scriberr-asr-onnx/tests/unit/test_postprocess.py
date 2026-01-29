from asr_engine.postprocess import Segment, merge_short_segments


def test_merge_short_segments():
    segments = [
        Segment(text="hello", start=0.0, end=0.5),
        Segment(text="world", start=0.5, end=0.6),
        Segment(text="this is long", start=0.6, end=2.0),
    ]
    merged = merge_short_segments(segments, attach_threshold_s=0.25, attach_max_words=2)
    assert len(merged) == 2
    assert merged[0].text == "hello world"
    assert merged[1].text == "this is long"
