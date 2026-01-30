from __future__ import annotations

import argparse
import logging
import os
import queue
import time

import grpc
from concurrent.futures import ThreadPoolExecutor

from .backend import DiarBackend
from .job_runner import JobRunner
from .metrics import get_rss_bytes
from .model_manager import ModelManager, ModelSpec
from .params import JobParams
from .status_store import JobState, JobStatusStore
from .proto import asr_engine_pb2 as pb2
from .proto import asr_engine_pb2_grpc as pb2_grpc


LOG = logging.getLogger("diar-engine")


class DiarEngineService(pb2_grpc.AsrEngineServicer):
    def __init__(self, model_manager: ModelManager, runner: JobRunner, status_store: JobStatusStore) -> None:
        self._model_manager = model_manager
        self._runner = runner
        self._status_store = status_store

    def LoadModel(self, request, context):
        spec = request.spec
        if not spec.model_id:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "model_id is required")
        if not spec.model_name:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "model_name is required")

        loaded = self._model_manager.load(
            ModelSpec(
                model_id=spec.model_id,
                model_name=spec.model_name,
                model_path=spec.model_path or None,
                providers=list(spec.providers),
                intra_op_threads=spec.intra_op_threads or 8,
                vad_backend=spec.vad_backend or "silero",
            )
        )
        return pb2.LoadModelResponse(model_id=loaded.spec.model_id, ok=True, message="loaded")

    def UnloadModel(self, request, context):
        ok = self._model_manager.unload(request.model_id or None)
        if not ok:
            return pb2.UnloadModelResponse(ok=False, message="not_loaded")
        return pb2.UnloadModelResponse(ok=True, message="unloaded")

    def StartJob(self, request, context):
        loaded = self._model_manager.get_loaded()
        if not loaded:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, "no model loaded")
        if request.model_id and request.model_id != loaded.spec.model_id:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "model_id mismatch")

        params = JobParams.from_kv(dict(request.params))
        accepted = self._runner.start_job(
            job_id=request.job_id,
            input_path=request.input_path,
            output_dir=request.output_dir,
            params=params,
        )
        if not accepted:
            context.abort(grpc.StatusCode.RESOURCE_EXHAUSTED, "engine busy")

        return pb2.StartJobResponse(job_id=request.job_id, accepted=True, message="started")

    def StopJob(self, request, context):
        ok = self._runner.stop_job(request.job_id)
        if not ok:
            return pb2.StopJobResponse(ok=False, message="not_running")
        return pb2.StopJobResponse(ok=True, message="stopping")

    def GetJobStatus(self, request, context):
        status = self._status_store.get(request.job_id)
        if not status:
            context.abort(grpc.StatusCode.NOT_FOUND, "job not found")
        return _status_to_pb(status)

    def StreamJobStatus(self, request, context):
        q = self._status_store.subscribe(request.job_id)
        try:
            while True:
                try:
                    status = q.get(timeout=1.0)
                except queue.Empty:
                    if context.is_active():
                        continue
                    break
                yield _status_to_pb(status)
                if status.state in {JobState.COMPLETED, JobState.FAILED, JobState.CANCELLED}:
                    break
        finally:
            self._status_store.unsubscribe(request.job_id, q)

    def ListLoadedModels(self, request, context):
        loaded = self._model_manager.get_loaded()
        if not loaded:
            return pb2.ListLoadedModelsResponse(models=[])
        spec = loaded.spec
        return pb2.ListLoadedModelsResponse(
            models=[
                pb2.ModelSpec(
                    model_id=spec.model_id,
                    model_name=spec.model_name,
                    model_path=spec.model_path or "",
                    providers=list(spec.providers or []),
                    intra_op_threads=spec.intra_op_threads,
                    vad_backend=spec.vad_backend,
                )
            ]
        )

    def GetEngineInfo(self, request, context):
        active_job_id = self._runner.active_job_id() or ""
        loaded = self._model_manager.get_loaded()
        return pb2.GetEngineInfoResponse(
            busy=bool(active_job_id),
            active_job_id=active_job_id,
            loaded_model_id=loaded.spec.model_id if loaded else "",
            rss_bytes=get_rss_bytes(),
        )


def _status_to_pb(status) -> pb2.JobStatus:
    state_map = {
        JobState.QUEUED: pb2.JobState.Value("JOB_STATE_QUEUED"),
        JobState.RUNNING: pb2.JobState.Value("JOB_STATE_RUNNING"),
        JobState.COMPLETED: pb2.JobState.Value("JOB_STATE_COMPLETED"),
        JobState.FAILED: pb2.JobState.Value("JOB_STATE_FAILED"),
        JobState.CANCELLED: pb2.JobState.Value("JOB_STATE_CANCELLED"),
    }
    return pb2.JobStatus(
        job_id=status.job_id,
        state=state_map.get(status.state, pb2.JobState.Value("JOB_STATE_UNSPECIFIED")),
        message=status.message,
        progress=status.progress,
        outputs=status.outputs,
        started_unix_ms=status.started_unix_ms,
        finished_unix_ms=status.finished_unix_ms,
    )


def serve(socket_path: str | None, host: str, port: int) -> None:
    model_manager = ModelManager()
    status_store = JobStatusStore()
    backend = DiarBackend(model_manager)
    runner = JobRunner(backend, status_store)

    server = grpc.server(ThreadPoolExecutor(max_workers=8))
    pb2_grpc.add_AsrEngineServicer_to_server(
        DiarEngineService(model_manager, runner, status_store), server
    )

    if socket_path:
        if os.path.exists(socket_path):
            os.remove(socket_path)
        server.add_insecure_port(f"unix:{socket_path}")
        LOG.info("Listening on unix:%s", socket_path)
    else:
        server.add_insecure_port(f"{host}:{port}")
        LOG.info("Listening on %s:%s", host, port)

    server.start()
    try:
        while True:
            time.sleep(3600)
    except KeyboardInterrupt:
        LOG.info("Shutting down")
        server.stop(0)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--socket", default=None, help="Unix domain socket path")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=50052)
    parser.add_argument("--log-level", default="INFO")
    args = parser.parse_args()

    logging.basicConfig(level=getattr(logging, args.log_level.upper(), logging.INFO))
    serve(args.socket, args.host, args.port)


if __name__ == "__main__":
    main()
