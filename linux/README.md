# Live Transcription Desktop Client

A lightweight Linux (X11) system tray daemon that records microphone audio via global hotkeys or tray menu, transcribes it through the local `live-transcription` Flask backend, and copies the resulting transcript directly to your clipboard.

---

## Live Dictation (Local / Offline)

In addition to the server-backed transcription client (`tray_recorder.py`), this repository includes a standalone local dictation daemon (`live_dictate.py`). It captures audio directly from PipeWire (`pw-record`), transcribes speech locally in real-time using `faster-whisper` directly on this machine, and types the resulting text at the cursor of the currently focused X11 window using `xdotool`. No network service or server is required.

- **Toggle Hotkey:** Press `F8` (default, configurable via `LT_DICTATE_HOTKEY`) once to start dictation, and press it again to stop and flush the final partial audio chunk.
- **Pipeline:** Records audio in streaming chunks (default 3 seconds), transcribes each chunk via `faster-whisper`, and types the text snippet at the active cursor with automatic space separation between chunks.
- **Tray Icon Indicator:**
  - Gray: Idle
  - Red: Dictating
  - Yellow: Thinking / Transcribing chunk
  - Green: Brief "done" notification on stop
- **Desktop Notifications:** Shows desktop notifications via `notify-send` when dictation starts ("good to go"), stops (with typed character count), and on errors.
- **Lazy Model Loading:** The Whisper model is loaded lazily on the first dictation start (displaying "loading model…" in the tray).
- **Prerequisites:** `xdotool` (`sudo apt install xdotool`), `pw-record`, and an active X11 desktop session.

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `LT_DICTATE_HOTKEY` | `<f8>` | Global toggle hotkey (e.g. `<f8>`, `<ctrl>+<alt>+d`) |
| `LT_DICTATE_MODEL` | `base.en` | faster-whisper model name (e.g. `base.en`, `small.en`) |
| `LT_DICTATE_DEVICE` | `cpu` | Device to run transcription on (`cpu` or `cuda`) |
| `LT_DICTATE_COMPUTE` | `int8` | Model quantization / compute type (`int8`, `float16`, etc.) |
| `LT_DICTATE_CHUNK_SECONDS` | `3` | Duration of each streaming chunk in seconds |
| `LT_DICTATE_TYPE_DELAY_MS` | `10` | Milliseconds delay between typed keystrokes in xdotool |
| `LT_DICTATE_TARGET` | *(empty)* | Optional PipeWire node target serial or name for pw-record |

> [!NOTE]
> GPU acceleration is optional: setting `LT_DICTATE_DEVICE=cuda` requires `pip install nvidia-cublas-cu12 nvidia-cudnn-cu12`. On modern multi-core CPUs, `base.en` with `int8` compute runs ~25–35x faster than realtime, so a GPU is not needed for base.en.

### Running Live Dictation

#### 1. Test Mode (`--test`)
You can verify audio decoding and local Whisper transcription against an audio file without starting the tray daemon or hotkeys:
```bash
.venv/bin/python live_dictate.py --test [FILE]
```
If `FILE` is omitted, it defaults to `last-take.wav` in the repository root. This mode decodes the audio via an `ffmpeg` subprocess directly to 16kHz mono float32, transcribes it, and outputs the full transcript along with execution timing and realtime multiplier.

#### 2. Running Manually
```bash
DISPLAY=:1 .venv/bin/python live_dictate.py
```

#### 3. Systemd User Service Setup
To run the live dictation daemon automatically in your desktop session:
```bash
mkdir -p ~/.config/systemd/user
cp live-dictation.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now live-dictation.service
```
Check status or logs:
```bash
systemctl --user status live-dictation.service
journalctl --user -u live-dictation.service -f
```

---


## Features

- **Global Hotkeys:** Start recording with `F9`, stop and transcribe with `F10`.
- **System Tray Icon:** Visual status indicator generated dynamically via Pillow (no static image files needed):
  - Gray: Idle
  - Red: Recording
  - Yellow/Orange: Transcribing
  - Green: Copied to clipboard
- **Clipboard Integration:** Automatically copies finished transcript using `xclip`.
- **Debugging & Audit:** Saves the most recent audio recording to `last-take.wav` in the project root.
- **Single Instance:** Enforces one running instance using `/tmp/live-transcription-client.lock`.
- **Clean Shutdown:** Stops any active audio recording safely before exiting.

---

## Hotkeys & Configuration

Default hotkeys:
- **Start recording:** `<f9>`
- **Stop & transcribe:** `<f10>`

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `LT_SERVICE_URL` | `http://10.25.0.112:11060` | Base URL of the live-transcription Flask service |
| `LT_START_HOTKEY` | `<f9>` | Global hotkey to start recording (e.g. `<f9>`, `<ctrl>+<alt>+r`) |
| `LT_STOP_HOTKEY` | `<f10>` | Global hotkey to stop recording and submit for transcription |

---

## Setup & Running Manually

### Prerequisites
- Python 3.10+
- `pw-record` (provided by PipeWire)
- `xclip` (for clipboard access under X11)
- Active X11 graphical session (`DISPLAY` set)

### Installation
```bash
cd /home/atlas/projects/Scriberr/linux
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
```

### Running Manually
```bash
.venv/bin/python tray_recorder.py
```

---

## Systemd User Service Setup

To run the daemon automatically in your desktop session:

1. Copy or link the service unit to your user systemd directory:
   ```bash
   mkdir -p ~/.config/systemd/user
   cp live-transcription-client.service ~/.config/systemd/user/
   ```

2. Reload systemd daemon and enable the service:
   ```bash
   systemctl --user daemon-reload
   systemctl --user enable --now live-transcription-client.service
   ```

3. Check service status and logs:
   ```bash
   systemctl --user status live-transcription-client.service
   journalctl --user -u live-transcription-client.service -f
   ```

---

## Backend API Specification

The client communicates with the Flask backend via:
1. `POST /api/record`
   - Form-data multipart upload with `audio` (WAV file) and optional `title`.
   - Returns HTTP 201: `{"conversation_id": <int>, "scriberr_job_id": "...", "status": "processing"}`.
2. `GET /api/conversations/<id>/status`
   - Polled every ~1.5s until completion (up to 300s).
   - Returns JSON with `status` (`processing` | `done` | `failed`), `transcript` (string), or `error`.

---

## Testing & Verification

Run the standalone API roundtrip test:
```bash
.venv/bin/python test_api_roundtrip.py
```
This script generates a 3-second 16kHz sine-wave audio sample in memory, submits it to `/api/record`, and polls until status is `done`.

---

## Troubleshooting

- **No tray icon appears:** Ensure you are running under an X11 desktop session and `DISPLAY` is exported (e.g., `export DISPLAY=:1`).
- **pw-record error:** The client relies on PipeWire's `pw-record` binary to capture microphone input. Ensure PipeWire is active (`systemctl --user status pipewire`).
- **Clipboard not updating:** Ensure `xclip` is installed (`which xclip`).
- **Global hotkeys not firing:** Under X11, ensure the window manager is not intercepting `F9`/`F10` exclusively, or customize the hotkeys using `LT_START_HOTKEY` and `LT_STOP_HOTKEY`.
