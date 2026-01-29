from asr_engine.params import JobParams, get_vad_params


def test_get_vad_params_defaults():
    params = get_vad_params("balanced")
    assert params["speech_pad_ms"] == 300
    assert params["min_silence_duration_ms"] == 600


def test_job_params_override():
    job = JobParams.from_kv(
        {
            "vad_preset": "aggressive",
            "vad_speech_pad_ms": "999",
            "include_words": "false",
            "merge_attach_threshold_s": "0.5",
            "merge_attach_max_words": "4",
            "sample_rate": "8000",
        }
    )
    vad = job.resolved_vad_params()
    assert vad["speech_pad_ms"] == 999
    assert job.include_words is False
    assert job.merge_attach_threshold_s == 0.5
    assert job.merge_attach_max_words == 4
    assert job.sample_rate == 8000
