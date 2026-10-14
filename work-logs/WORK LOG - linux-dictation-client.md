# WORK LOG: Linux dictation client

**Date:** 2026-10-14
**Status:** Complete
**Commit:** to be committed

## WHAT WAS ASKED
Move the standalone Linux desktop clients (tray recorder and local Whisper dictation daemons) from the separate repository `~/projects/live-transcription-client` into `linux/` in this repository so that desktop client work lives alongside the Scriberr server.

## BEFORE
Desktop clients were developed and tracked in a standalone repository (`~/projects/live-transcription-client`). The systemd user services and documentation pointed directly to `/home/atlas/projects/live-transcription-client`.

## AFTER
Tracked desktop client files live under `linux/` in the Scriberr repository:
- `linux/tray_recorder.py`: X11 tray daemon using F9/F10 global hotkeys, `pw-record` audio capture, Flask `/api/record` submission, and clipboard copy.
- `linux/live_dictate.py`: X11 tray daemon using F8 toggle hotkey, `pw-record` audio capture, offline `faster-whisper` transcription, `xdotool` typing at active cursor, and desktop notifications.
- `linux/test_api_roundtrip.py`: Standalone API roundtrip verification script.
- `linux/live-transcription-client.service` & `linux/live-dictation.service`: systemd user service units configured with `WorkingDirectory=/home/atlas/projects/Scriberr/linux` and ExecStart paths pointing to `.venv/bin/python`.
- `linux/requirements.txt`, `linux/.gitignore`, and `linux/README.md` (with updated paths).
Excluded runtime/build artifacts: `last-take.wav`, `.venv/`, `__pycache__/`, `.git/`.

## WHAT WAS DONE
- Copied tracked files from `~/projects/live-transcription-client` to `linux/`:
  - `tray_recorder.py`
  - `live_dictate.py`
  - `test_api_roundtrip.py`
  - `live-transcription-client.service`
  - `live-dictation.service`
  - `requirements.txt`
  - `.gitignore`
  - `README.md`
- Performed path substitutions (`/home/atlas/projects/live-transcription-client` → `/home/atlas/projects/Scriberr/linux`) in:
  - `linux/live-transcription-client.service`: `WorkingDirectory=` and `ExecStart=` lines.
  - `linux/live-dictation.service`: `WorkingDirectory=` and `ExecStart=` lines.
  - `linux/README.md`: Setup/installation instructions.
- Python files left untouched (they compute paths dynamically from `__file__`).
- Both daemons verified working in production on the dev box:
  - F9/F10 recorder → `/api/record` → clipboard
  - F8 toggle → `faster-whisper` local STT → `xdotool` types at cursor + `notify-send` notifications.

## VERIFICATION
- Checked `git status --short`: clean except new untracked files in `linux/` and this work log; pre-existing modified/untracked files untouched.
- `diff -r` comparison against source files: Python files, requirements.txt, and .gitignore are byte-identical; service units and README differ only by path substitution.
- Bytecode compilation: `python3 -m py_compile` passed for `tray_recorder.py`, `live_dictate.py`, and `test_api_roundtrip.py`.
- Execution test: Ran `live_dictate.py --test /home/atlas/projects/live-transcription-client/last-take.wav` using the venv Python binary; verified it transcribes successfully and exits 0.
- Stale path check: Verified no references to `/home/atlas/projects/live-transcription-client` remain in `linux/`.

## FOLLOW-UPS / NOTES
- Optional follow-up: user may migrate the live systemd install to the new path (`~/.config/systemd/user/`) after merge.
- Do not commit changes; user will commit.

---

## UPDATE (2026-10-14): Phrase-aware dictation chunking

**Problem:** User reported dictation text arriving in 3-second bursts and always
feeling like it was falling behind.

**Investigation:** Inference was not the bottleneck (faster-whisper base.en CPU
int8 = ~25-30x realtime on the dev box). The latency came from (1) fixed 3s
chunks — text can only appear once a chunk fills, even mid-sentence and even
after the speaker stops — and (2) `xdotool --delay 10ms/char` adding 0.5-0.8s
per burst.

**Change (linux/live_dictate.py):**
- New default chunking mode `vad`: the streaming audio buffer is probed with
  the bundled Silero VAD (~3-6ms per probe via `faster_whisper.vad.get_speech_timestamps`)
  and a chunk is flushed the moment the speaker pauses (>= 300ms trailing
  silence) or at the 4s cap — text now flows phrase by phrase.
- `LT_DICTATE_CHUNK_MODE=fixed` restores the legacy fixed-chunk behavior
  (`LT_DICTATE_CHUNK_SECONDS`). New env vars: `LT_DICTATE_CHUNK_MODE`,
  `LT_DICTATE_MAX_CHUNK_SECONDS` (4), `LT_DICTATE_MIN_CHUNK_SECONDS` (0.5),
  `LT_DICTATE_SILENCE_FLUSH_MS` (300). `LT_DICTATE_TYPE_DELAY_MS` default 10 -> 0.
- Model warmup (0.5s dummy transcribe) right after model load so the first
  chunk is responsive.
- New dev mode `--test-chunks [FILE]`: simulates the adaptive chunker over a
  WAV file and prints per-chunk flush time/reason/text plus latency stats.
- README updated (env table + how chunking works + --test-chunks).

**Verification:** `--test-chunks last-take.wav` (89s real speech): 27 chunks
(15 phrase-end / 12 max), avg flush latency 0.36s, max 0.41s (was up to ~3s+
per burst). Live daemon lifecycle (F8 start/stop under DISPLAY=:1) clean,
VAD phrase-end flushes observed in real time, no tracebacks.

Files changed: `linux/live_dictate.py`, `linux/README.md`.
