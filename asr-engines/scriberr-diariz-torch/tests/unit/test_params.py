from diar_engine.params import JobParams


def test_job_params_parsing():
    params = JobParams.from_kv(
        {
            "output_format": "json",
            "device": "cuda",
            "hf_token": "hf_test",
            "min_speakers": "2",
            "max_speakers": "4",
            "segmentation_onset": "0.4",
            "segmentation_offset": "0.2",
            "batch_size": "3",
            "streaming_mode": "true",
            "chunk_length_s": "22.5",
            "chunk_len": "320",
            "chunk_right_context": "32",
            "fifo_len": "28",
            "spkcache_update_period": "200",
        }
    )

    assert params.output_format == "json"
    assert params.device == "cuda"
    assert params.hf_token == "hf_test"
    assert params.min_speakers == 2
    assert params.max_speakers == 4
    assert abs(params.segmentation_onset - 0.4) < 1e-6
    assert abs(params.segmentation_offset - 0.2) < 1e-6
    assert params.batch_size == 3
    assert params.streaming_mode is True
    assert abs(params.chunk_length_s - 22.5) < 1e-6
    assert params.chunk_len == 320
    assert params.chunk_right_context == 32
    assert params.fifo_len == 28
    assert params.spkcache_update_period == 200
