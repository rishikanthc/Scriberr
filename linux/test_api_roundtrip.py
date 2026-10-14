#!/usr/bin/env python3
"""
Test script for verifying round-trip transcription against the live Flask service.
Generates a 3s 16kHz mono sine-wave WAV using stdlib wave, POSTs to /api/record,
and polls /api/conversations/<id>/status until done or failed.
"""

import io
import math
import os
import struct
import sys
import time
import wave
import requests

SERVICE_URL = os.environ.get("LT_SERVICE_URL", "http://10.25.0.112:11060")
TIMEOUT_SECONDS = 120
POLL_INTERVAL = 1.5


def generate_sine_wav(duration_sec: float = 3.0, sample_rate: int = 16000, frequency: float = 440.0) -> bytes:
    """Generate a valid 16kHz mono 16-bit PCM WAV in-memory using stdlib wave."""
    buf = io.BytesIO()
    num_samples = int(duration_sec * sample_rate)
    amplitude = 16000  # moderate volume for 16-bit signed PCM
    with wave.open(buf, "wb") as wf:
        wf.setnchannels(1)      # mono
        wf.setsampwidth(2)      # 16-bit (2 bytes per sample)
        wf.setframerate(sample_rate)
        # Pack frames
        frames = bytearray()
        for i in range(num_samples):
            sample = int(amplitude * math.sin(2.0 * math.pi * frequency * (i / sample_rate)))
            frames.extend(struct.pack("<h", sample))
        wf.writeframes(frames)
    return buf.getvalue()


def test_roundtrip():
    print(f"Connecting to service at: {SERVICE_URL}")
    wav_bytes = generate_sine_wav()
    print(f"Generated sine wave WAV: {len(wav_bytes)} bytes")

    files = {
        "audio": ("test_sine.wav", wav_bytes, "audio/wav")
    }
    data = {
        "title": "API Roundtrip Test Sine Wave"
    }

    print("POSTing audio to /api/record...")
    resp = requests.post(f"{SERVICE_URL}/api/record", files=files, data=data, timeout=60)
    print(f"POST response code: {resp.status_code}")
    if resp.status_code != 201:
        print(f"Error: expected 201 Created, got {resp.status_code}: {resp.text}")
        sys.exit(1)

    record_data = resp.json()
    print(f"Record response JSON: {record_data}")
    conv_id = record_data.get("conversation_id")
    if not conv_id:
        print("Error: No conversation_id in response!")
        sys.exit(1)

    print(f"Polling status for conversation {conv_id} every {POLL_INTERVAL}s (timeout {TIMEOUT_SECONDS}s)...")
    start_time = time.time()
    while time.time() - start_time < TIMEOUT_SECONDS:
        status_resp = requests.get(f"{SERVICE_URL}/api/conversations/{conv_id}/status", timeout=30)
        if status_resp.status_code != 200:
            print(f"Warning: Status returned code {status_resp.status_code}: {status_resp.text}")
        else:
            conv_status = status_resp.json()
            status = conv_status.get("status")
            print(f"[{time.time() - start_time:.1f}s] status: {status}")
            if status == "done":
                print("Success! Conversation status is 'done'.")
                print(f"Final response JSON:\n{conv_status}")
                transcript = conv_status.get("transcript", "")
                print(f"Transcript received: {transcript!r}")
                sys.exit(0)
            elif status == "failed":
                err = conv_status.get("error", "Unknown error")
                print(f"Failure: Conversation status is 'failed': {err}")
                print(f"Final response JSON:\n{conv_status}")
                sys.exit(1)

        time.sleep(POLL_INTERVAL)

    print(f"Error: Timed out waiting for conversation {conv_id} after {TIMEOUT_SECONDS}s.")
    sys.exit(1)


if __name__ == "__main__":
    test_roundtrip()
