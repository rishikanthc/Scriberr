import json
import logging
import os
from dataclasses import dataclass, field
from typing import Any

import numpy as np
import onnxruntime as rt
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.responses import HTMLResponse
from fastapi.staticfiles import StaticFiles

import onnx_asr

APP_DIR = os.path.dirname(os.path.abspath(__file__))
STATIC_DIR = os.path.join(os.path.dirname(APP_DIR), "static")

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("realtime-asr")

SAMPLE_RATE = 16_000
MAX_AUDIO_SECONDS = 25
MIN_SEGMENT_SECONDS = 0.2


@dataclass
class StreamSession:
    asr: Any
    vad: Any
    sample_rate: int = SAMPLE_RATE
    buffer: np.ndarray = field(default_factory=lambda: np.zeros((0,), dtype=np.float32))
    base_offset: int = 0
    processed_abs_end: int = 0

    def reset(self) -> None:
        self.buffer = np.zeros((0,), dtype=np.float32)
        self.base_offset = 0
        self.processed_abs_end = 0

    def add_audio(self, chunk: np.ndarray) -> list[dict[str, Any]]:
        if chunk.size == 0:
            return []
        if chunk.dtype != np.float32:
            chunk = chunk.astype(np.float32)
        self.buffer = np.concatenate([self.buffer, chunk])
        return self._segment_and_transcribe()

    def flush(self) -> list[dict[str, Any]]:
        if self.buffer.size == 0:
            return []
        text = self.asr.recognize(self.buffer, sample_rate=self.sample_rate)
        result = {
            "start": self.base_offset / self.sample_rate,
            "end": (self.base_offset + self.buffer.size) / self.sample_rate,
            "text": text,
            "final": True,
        }
        self.reset()
        return [result]

    def _segment_and_transcribe(self) -> list[dict[str, Any]]:
        results: list[dict[str, Any]] = []
        buf_len = self.buffer.size
        if buf_len == 0:
            return results

        if buf_len > int(MAX_AUDIO_SECONDS * self.sample_rate):
            text = self.asr.recognize(self.buffer, sample_rate=self.sample_rate)
            results.append(
                {
                    "start": self.base_offset / self.sample_rate,
                    "end": (self.base_offset + buf_len) / self.sample_rate,
                    "text": text,
                    "final": True,
                    "forced": True,
                }
            )
            self.reset()
            return results

        waveforms = self.buffer[None, :]
        lengths = np.asarray([buf_len], dtype=np.int64)
        segments_iter = next(
            self.vad.segment_batch(
                waveforms,
                lengths,
                self.sample_rate,
                min_silence_duration_ms=300,
            )
        )
        segments = list(segments_iter)

        last_commit_end = None
        for start, end in segments:
            abs_start = self.base_offset + start
            abs_end = self.base_offset + end

            if abs_end <= self.processed_abs_end:
                continue

            if end >= buf_len:
                # Likely an open segment (no long silence yet).
                continue

            rel_start = max(start, self.processed_abs_end - self.base_offset)
            if end - rel_start < int(MIN_SEGMENT_SECONDS * self.sample_rate):
                continue

            chunk = self.buffer[rel_start:end]
            text = self.asr.recognize(chunk, sample_rate=self.sample_rate)
            results.append(
                {
                    "start": abs_start / self.sample_rate,
                    "end": abs_end / self.sample_rate,
                    "text": text,
                    "final": True,
                }
            )
            self.processed_abs_end = abs_end
            last_commit_end = end

        if last_commit_end is not None:
            self.buffer = self.buffer[last_commit_end:]
            self.base_offset += last_commit_end

        return results


app = FastAPI()
app.mount("/static", StaticFiles(directory=STATIC_DIR), name="static")

MODEL_NAME = "nemo-parakeet-tdt-0.6b-v3"


@app.on_event("startup")
def _startup() -> None:
    providers = rt.get_available_providers()
    if "CUDAExecutionProvider" in providers:
        asr_providers = ["CUDAExecutionProvider", "CPUExecutionProvider"]
        logger.info("Using CUDAExecutionProvider for ASR")
    else:
        asr_providers = ["CPUExecutionProvider"]
        logger.warning("CUDAExecutionProvider not available, using CPU")

    vad_providers = [p for p in ["CUDAExecutionProvider", "CPUExecutionProvider"] if p in providers]
    if not vad_providers:
        vad_providers = ["CPUExecutionProvider"]

    app.state.asr = onnx_asr.load_model(MODEL_NAME, providers=asr_providers)
    app.state.vad = onnx_asr.load_vad("silero", providers=vad_providers)


@app.get("/")
async def index() -> HTMLResponse:
    with open(os.path.join(STATIC_DIR, "index.html"), "r", encoding="utf-8") as f:
        return HTMLResponse(f.read())


@app.websocket("/ws")
async def websocket_endpoint(ws: WebSocket) -> None:
    await ws.accept()
    session = StreamSession(app.state.asr, app.state.vad)

    try:
        while True:
            message = await ws.receive()
            if "text" in message and message["text"] is not None:
                payload = json.loads(message["text"])
                msg_type = payload.get("type")
                if msg_type == "config":
                    sr = int(payload.get("sample_rate", SAMPLE_RATE))
                    if sr != SAMPLE_RATE:
                        await ws.send_text(
                            json.dumps(
                                {
                                    "type": "error",
                                    "message": f"Server expects {SAMPLE_RATE} Hz audio.",
                                }
                            )
                        )
                        await ws.close()
                        return
                    await ws.send_text(json.dumps({"type": "ready"}))
                elif msg_type == "reset":
                    session.reset()
                elif msg_type == "stop":
                    for res in session.flush():
                        await ws.send_text(json.dumps({"type": "result", **res}))
                continue

            if "bytes" in message and message["bytes"] is not None:
                chunk = np.frombuffer(message["bytes"], dtype=np.float32)
                for res in session.add_audio(chunk):
                    await ws.send_text(json.dumps({"type": "result", **res}))

    except WebSocketDisconnect:
        logger.info("WebSocket disconnected")


@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}
