# Realtime ASR Demo (onnx-asr + Parakeet TDT)

This prototype streams browser audio to a FastAPI backend over WSS, uses onnx-asr with the Parakeet TDT model, and segments by the built-in Silero VAD (300ms silence).

## Setup (UV)

```bash
cd /root/asr/realtime-asr-demo
uv sync
```

Notes:
- `onnx-asr[gpu,hub]` pulls Hugging Face Hub support and GPU extras.
- TensorRT + CUDA must already be installed on the system for `TensorrtExecutionProvider` to be available.

## HTTPS cert (self-signed)

```bash
./scripts/gen_certs.sh
```

## Run (HTTPS)

```bash
uv run uvicorn app.main:app \
  --host 0.0.0.0 --port 8443 \
  --ssl-keyfile certs/key.pem \
  --ssl-certfile certs/cert.pem
```

Open `https://<server-ip>:8443` from another machine and accept the self-signed cert warning.
