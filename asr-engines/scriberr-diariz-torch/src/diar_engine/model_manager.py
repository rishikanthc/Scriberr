from __future__ import annotations

from dataclasses import dataclass
import os
import threading
import time
from typing import Any, Callable

from huggingface_hub import hf_hub_download


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
    kind: str
    model: Any
    loaded_at: float
    token: str | None = None


class ModelManager:
    def __init__(self, loader: Callable[[ModelSpec, str | None], LoadedModel] | None = None) -> None:
        self._loader = loader or self._default_loader
        self._loaded: LoadedModel | None = None
        self._lock = threading.Lock()

    def load(self, spec: ModelSpec, hf_token: str | None = None) -> LoadedModel:
        with self._lock:
            self._loaded = self._loader(spec, hf_token)
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

    def ensure_loaded(self, spec: ModelSpec, hf_token: str | None = None) -> LoadedModel:
        with self._lock:
            if self._loaded is None:
                self._loaded = self._loader(spec, hf_token)
            elif self._loaded.spec.model_id != spec.model_id:
                self._loaded = self._loader(spec, hf_token)
            elif spec.model_id == "pyannote" and hf_token and self._loaded.token != hf_token:
                self._loaded = self._loader(spec, hf_token)
            return self._loaded

    def _default_loader(self, spec: ModelSpec, hf_token: str | None) -> LoadedModel:
        if spec.intra_op_threads:
            try:
                import torch

                torch.set_num_threads(spec.intra_op_threads)
            except Exception:
                pass

        if spec.model_id == "pyannote":
            token = hf_token or os.environ.get("HF_TOKEN") or os.environ.get("PYANNOTE_TOKEN")
            return _load_pyannote(spec, token)
        if spec.model_id == "sortformer":
            return _load_sortformer(spec)
        raise ValueError(f"Unsupported diarization model_id: {spec.model_id}")


def _load_pyannote(spec: ModelSpec, token: str | None) -> LoadedModel:
    from pyannote.audio import Pipeline
    import torch

    _allowlist_pyannote_safe_globals()

    model_ref = spec.model_path or spec.model_name
    if not model_ref:
        raise ValueError("model_name is required for pyannote")

    try:
        if token:
            pipeline = Pipeline.from_pretrained(model_ref, token=token)
        else:
            pipeline = Pipeline.from_pretrained(model_ref)
    except TypeError:
        if token:
            pipeline = Pipeline.from_pretrained(model_ref, use_auth_token=token)
        else:
            pipeline = Pipeline.from_pretrained(model_ref)

    device = _resolve_device(spec.providers)
    if device == "cuda" and torch.cuda.is_available():
        pipeline = pipeline.to(torch.device("cuda"))
    else:
        pipeline = pipeline.to(torch.device("cpu"))

    return LoadedModel(spec=spec, kind="pyannote", model=pipeline, loaded_at=time.time(), token=token)


def _load_sortformer(spec: ModelSpec) -> LoadedModel:
    import torch

    try:
        from nemo.collections.asr.models import SortformerEncLabelModel
    except Exception as exc:
        raise RuntimeError("nemo_toolkit[asr] is required for Sortformer diarization") from exc

    model_path = _resolve_sortformer_model_path(spec)
    device = _resolve_device(spec.providers)
    if device == "cuda" and torch.cuda.is_available():
        map_location = torch.device("cuda")
    else:
        map_location = torch.device("cpu")

    diar_model = SortformerEncLabelModel.restore_from(
        restore_path=model_path,
        map_location=map_location,
        strict=False,
    )
    diar_model.eval()
    return LoadedModel(spec=spec, kind="sortformer", model=diar_model, loaded_at=time.time())


def _resolve_sortformer_model_path(spec: ModelSpec) -> str:
    if spec.model_path:
        return spec.model_path

    model_name = spec.model_name or "nvidia/diar_sortformer_4spk-v1"
    if model_name.endswith(".nemo") and os.path.exists(model_name):
        return model_name

    if "streaming" in model_name:
        filename = "diar_streaming_sortformer_4spk-v2.nemo"
    else:
        filename = "diar_sortformer_4spk-v1.nemo"
    return hf_hub_download(repo_id=model_name, filename=filename)


def _resolve_device(providers: list[str] | None) -> str:
    if not providers:
        return "auto"
    for provider in providers:
        lower = provider.lower()
        if "cuda" in lower or "tensorrt" in lower:
            return "cuda"
    return "cpu"


def _allowlist_pyannote_safe_globals() -> None:
    try:
        from pyannote.audio.core.task import Specifications, Problem, Resolution
        import torch

        if hasattr(torch.serialization, "add_safe_globals"):
            torch.serialization.add_safe_globals([Specifications, Problem, Resolution])
    except Exception:
        return
