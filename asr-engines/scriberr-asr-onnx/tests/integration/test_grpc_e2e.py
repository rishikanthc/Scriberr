from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
import time

import grpc
import pytest

from asr_engine.job_runner import JobRunner
from asr_engine.model_manager import ModelManager
from asr_engine.status_store import JobStatusStore
from asr_engine.backend import AsrBackend
from asr_engine.proto import asr_engine_pb2 as pb2
from asr_engine.proto import asr_engine_pb2_grpc as pb2_grpc
from asr_engine.server import AsrEngineService

FIXTURE_AUDIO = Path(__file__).resolve().parents[1] / "fixtures" / "jfk.wav"
EXPECTED_DIR = Path(__file__).resolve().parents[1] / "fixtures" / "expected"

MODELS = [
    "nemo-parakeet-tdt-0.6b-v2",
    "nemo-parakeet-tdt-0.6b-v3",
    # "nemo-canary-1b-v2",
]


def _start_server():
    model_manager = ModelManager()
    status_store = JobStatusStore()
    backend = AsrBackend(model_manager)
    runner = JobRunner(backend, status_store)

    server = grpc.server(ThreadPoolExecutor(max_workers=4))
    pb2_grpc.add_AsrEngineServicer_to_server(
        AsrEngineService(model_manager, runner, status_store), server
    )
    port = server.add_insecure_port("127.0.0.1:0")
    server.start()
    return server, port


@pytest.mark.parametrize("model_name", MODELS)
def test_grpc_transcription_e2e(tmp_path: Path, model_name: str):
    if not FIXTURE_AUDIO.exists():
        pytest.fail(f"Missing fixture audio: {FIXTURE_AUDIO}")

    expected_path = EXPECTED_DIR / f"{model_name}.txt"
    if not expected_path.exists():
        pytest.fail(
            f"Missing expected transcript: {expected_path}. "
            "Generate goldens before running integration tests."
        )

    server, port = _start_server()
    channel = grpc.insecure_channel(f"127.0.0.1:{port}")
    stub = pb2_grpc.AsrEngineStub(channel)

    model_id = f"{model_name}-id"
    stub.LoadModel(
        pb2.LoadModelRequest(
            spec=pb2.ModelSpec(model_id=model_id, model_name=model_name)
        )
    )

    job_id = f"job-{model_name}"
    output_dir = tmp_path / model_name

    stub.StartJob(
        pb2.StartJobRequest(
            job_id=job_id,
            input_path=str(FIXTURE_AUDIO),
            output_dir=str(output_dir),
            model_id=model_id,
            params={"vad_preset": "balanced"},
        )
    )

    final = None
    deadline = time.time() + 600
    for status in stub.StreamJobStatus(pb2.StreamJobStatusRequest(job_id=job_id)):
        final = status
        if status.state == pb2.JobState.Value("JOB_STATE_COMPLETED"):
            break
        if time.time() > deadline:
            pytest.fail("Timed out waiting for transcription")

    assert final is not None
    assert final.state == pb2.JobState.Value("JOB_STATE_COMPLETED")

    transcript_path = Path(final.outputs["transcript"])
    assert transcript_path.exists()

    expected = expected_path.read_text(encoding="utf-8").strip()
    got = transcript_path.read_text(encoding="utf-8").strip()
    assert got == expected

    server.stop(0)
