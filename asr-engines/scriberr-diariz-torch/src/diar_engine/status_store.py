from __future__ import annotations

from dataclasses import dataclass, field
import enum
import queue
import threading


class JobState(enum.Enum):
    QUEUED = "queued"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


@dataclass
class JobStatus:
    job_id: str
    state: JobState
    message: str = ""
    progress: float = 0.0
    outputs: dict[str, str] = field(default_factory=dict)
    started_unix_ms: int = 0
    finished_unix_ms: int = 0


class JobStatusStore:
    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._status: dict[str, JobStatus] = {}
        self._subscribers: dict[str, list[queue.Queue[JobStatus]]] = {}

    def set(self, status: JobStatus) -> None:
        with self._lock:
            self._status[status.job_id] = status
            for q in self._subscribers.get(status.job_id, []):
                q.put(status)

    def get(self, job_id: str) -> JobStatus | None:
        with self._lock:
            return self._status.get(job_id)

    def subscribe(self, job_id: str) -> queue.Queue[JobStatus]:
        with self._lock:
            q: queue.Queue[JobStatus] = queue.Queue()
            self._subscribers.setdefault(job_id, []).append(q)
            return q

    def unsubscribe(self, job_id: str, q: queue.Queue[JobStatus]) -> None:
        with self._lock:
            if job_id in self._subscribers:
                self._subscribers[job_id] = [
                    existing for existing in self._subscribers[job_id] if existing is not q
                ]
                if not self._subscribers[job_id]:
                    self._subscribers.pop(job_id, None)
