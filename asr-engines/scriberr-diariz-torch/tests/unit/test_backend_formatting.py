from diar_engine.backend import (
    _build_json_payload,
    _sortformer_segments_to_dicts,
    _write_rttm,
)


def test_sortformer_segment_normalization():
    segments = ["0.0 1.5 speaker_1", "1.5 2.0 speaker_2"]
    normalized = _sortformer_segments_to_dicts(segments)
    assert normalized[0]["speaker"] == "speaker_1"
    assert normalized[1]["speaker"] == "speaker_2"
    assert normalized[0]["duration"] == 1.5


def test_json_payload_structure():
    segments = [
        {"start": 0.0, "end": 1.0, "speaker": "s1", "duration": 1.0, "confidence": 1.0},
        {"start": 1.0, "end": 2.5, "speaker": "s2", "duration": 1.5, "confidence": 1.0},
    ]
    payload = _build_json_payload(
        input_path="audio.wav",
        model_id="pyannote",
        model_name="pyannote/speaker-diarization-community-1",
        segments=segments,
        audio_seconds=2.5,
    )
    assert payload["speaker_count"] == 2
    assert payload["total_duration"] == 2.5
    assert payload["segments"][0]["speaker"] == "s1"


def test_write_rttm(tmp_path):
    segments = [
        {"start": 0.0, "end": 1.0, "speaker": "spk1", "duration": 1.0},
        {"start": 1.0, "end": 2.0, "speaker": "spk2", "duration": 1.0},
    ]
    output = tmp_path / "out.rttm"
    _write_rttm(str(output), "audio.wav", segments)
    data = output.read_text().strip().splitlines()
    assert len(data) == 2
    assert data[0].startswith("SPEAKER audio 1 0.000 1.000")
