from asr_engine.timestamps import format_ts, word_timestamps_from_segment


def test_format_ts():
    assert format_ts(0.0) == "00:00:00.000"
    assert format_ts(61.5) == "00:01:01.500"


def test_word_timestamps():
    words = word_timestamps_from_segment("hello world", 0.0, 2.0)
    assert len(words) == 2
    assert words[0]["word"] == "hello"
    assert words[-1]["end"] == 2.0
