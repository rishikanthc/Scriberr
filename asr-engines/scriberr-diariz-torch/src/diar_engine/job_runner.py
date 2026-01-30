from __future__ import annotations

import threading
import time

from .backend import DiarBackend, CancelToken, CancelledError
from .params import JobParams
from .status_store import JobState, JobStatus, JobStatusStore


class JobRunner:
    def __init__(self, backend: DiarBackend, status_store: JobStatusStore) -> None:
        self._backend = backend
        self._status_store = status_store
        self._lock = threading.Lock()
        self._active_job_id: str | None = None
        self._cancel_token: CancelToken | None = None

    def active_job_id(self) -> str | None:
        with self._lock:
            return self._active_job_id

    def start_job(
        self,
        job_id: str,
        input_path: str,
        output_dir: str,
        params: JobParams,
    ) -> bool:
        with self._lock:
            if self._active_job_id is not None:
                return False
            self._active_job_id = job_id
            self._cancel_token = CancelToken()

        status = JobStatus(job_id=job_id, state=JobState.QUEUED, started_unix_ms=_now_ms())
        self._status_store.set(status)

        thread = threading.Thread(
            target=self._run_job,
            args=(job_id, input_path, output_dir, params, self._cancel_token),
            daemon=True,
        )
        thread.start()
        return True

    def stop_job(self, job_id: str) -> bool:
        with self._lock:
            if self._active_job_id != job_id or self._cancel_token is None:
                return False
            self._cancel_token.cancel()
            return True

    def _run_job(
        self,
        job_id: str,
        input_path: str,
        output_dir: str,
        params: JobParams,
        cancel_token: CancelToken,
    ) -> None:
        def progress_cb(progress: float, message: str) -> None:
            status = self._status_store.get(job_id)
            if not status:
                return
            status.progress = progress
            status.state = JobState.RUNNING
            status.message = message
            self._status_store.set(status)

        status = JobStatus(job_id=job_id, state=JobState.RUNNING, started_unix_ms=_now_ms())
        self._status_store.set(status)

        try:
            result = self._backend.diarize(
                input_path=input_path,
                output_dir=output_dir,
                params=params,
                cancel_token=cancel_token,
                progress_cb=progress_cb,
            )
            status = JobStatus(
                job_id=job_id,
                state=JobState.COMPLETED,
                progress=1.0,
                outputs={
                    "diarization": result.diarization_path,
                    "rttm": result.rttm_path or "",
                    "result": result.result_path,
                },
                started_unix_ms=status.started_unix_ms,
                finished_unix_ms=_now_ms(),
            )
            self._status_store.set(status)
        except CancelledError:
            status = JobStatus(
                job_id=job_id,
                state=JobState.CANCELLED,
                message="cancelled",
                progress=0.0,
                started_unix_ms=status.started_unix_ms,
                finished_unix_ms=_now_ms(),
            )
            self._status_store.set(status)
        except Exception as exc:
            status = JobStatus(
                job_id=job_id,
                state=JobState.FAILED,
                message=str(exc),
                progress=0.0,
                started_unix_ms=status.started_unix_ms,
                finished_unix_ms=_now_ms(),
            )
            self._status_store.set(status)
        finally:
            with self._lock:
                self._active_job_id = None
                self._cancel_token = None


def _now_ms() -> int:
    return int(time.time() * 1000)
