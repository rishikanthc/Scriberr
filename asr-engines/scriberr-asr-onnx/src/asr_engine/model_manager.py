from __future__ import annotations

from dataclasses import dataclass
import threading
import time
from typing import Any, Callable

import onnxruntime as rt
import onnx_asr


@dataclass
class ModelSpec:
    model_id: str
    model_name: str
    model_path: str | None = None
    providers: list[str] | None = None
    intra_op_threads: int = 8
    vad_backend: str = "silero"


@dataclass
class LoadedModel:
    spec: ModelSpec
    asr_base: Any
    loaded_at: float


class ModelManager:
    def __init__(self, loader: Callable[[ModelSpec], LoadedModel] | None = None) -> None:
        self._loader = loader or self._default_loader
        self._loaded: LoadedModel | None = None
        self._lock = threading.Lock()

    def load(self, spec: ModelSpec) -> LoadedModel:
        with self._lock:
            self._loaded = self._loader(spec)
            return self._loaded

    def unload(self, model_id: str | None = None) -> bool:
        with self._lock:
            if self._loaded is None:
                return False
            if model_id and self._loaded.spec.model_id != model_id:
                return False
            self._loaded = None
            return True

    def get_loaded(self) -> LoadedModel | None:
        with self._lock:
            return self._loaded

    def _default_loader(self, spec: ModelSpec) -> LoadedModel:
        opts = rt.SessionOptions()
        opts.intra_op_num_threads = spec.intra_op_threads

        providers = spec.providers
        if not providers:
            available = rt.get_available_providers()
            if "CUDAExecutionProvider" in available:
                providers = ["CUDAExecutionProvider", "CPUExecutionProvider"]
            else:
                providers = ["CPUExecutionProvider"]

        load_kwargs: dict[str, Any] = {
            "model": spec.model_name,
            "providers": providers,
            "sess_options": opts,
        }
        if spec.model_path:
            load_kwargs["path"] = spec.model_path

        asr_base = onnx_asr.load_model(**load_kwargs)
        return LoadedModel(spec=spec, asr_base=asr_base, loaded_at=time.time())
