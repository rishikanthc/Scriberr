from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
import queue
import threading
from typing import Any


class JobState(str, Enum):
    QUEUED = "QUEUED"
    RUNNING = "RUNNING"
    COMPLETED = "COMPLETED"
    FAILED = "FAILED"
    CANCELLED = "CANCELLED"


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
        self._statuses: dict[str, JobStatus] = {}
        self._subscribers: dict[str, list[queue.Queue[JobStatus]]] = {}

    def set(self, status: JobStatus) -> None:
        with self._lock:
            self._statuses[status.job_id] = status
            for q in self._subscribers.get(status.job_id, []):
                q.put(status)

    def get(self, job_id: str) -> JobStatus | None:
        with self._lock:
            return self._statuses.get(job_id)

    def subscribe(self, job_id: str) -> queue.Queue[JobStatus]:
        q: queue.Queue[JobStatus] = queue.Queue()
        with self._lock:
            self._subscribers.setdefault(job_id, []).append(q)
            if job_id in self._statuses:
                q.put(self._statuses[job_id])
        return q

    def unsubscribe(self, job_id: str, q: queue.Queue[JobStatus]) -> None:
        with self._lock:
            subs = self._subscribers.get(job_id, [])
            if q in subs:
                subs.remove(q)

    def reset(self) -> None:
        with self._lock:
            self._statuses.clear()
            self._subscribers.clear()
