# Plan: Add Linux desktop clients (tray recorder + local live dictation) under `linux/`

## Context
The desktop clients for this project were developed in a standalone repo
(`~/projects/live-transcription-client`, 3 commits) and are working in
production on this box. The user wants them preserved inside the established
Scriberr repo so they aren't lost, under a platform-specific `linux/` directory
(X11/PipeWire specifics — not portable to other OSes).

## Source material
`~/projects/live-transcription-client` — tracked files only:
- `tray_recorder.py` — X11 tray daemon: F9/F10 hotkeys, pw-record → Flask `/api/record` → clipboard
- `live_dictate.py` — X11 tray daemon: F8 toggle, pw-record → faster-whisper (local/offline) → xdotool types at cursor, desktop notifications
- `test_api_roundtrip.py` — standalone API roundtrip test against the Flask service
- `live-transcription-client.service` / `live-dictation.service` — systemd user units
- `requirements.txt`, `README.md`, `.gitignore`
- EXCLUDED: `last-take.wav` (2.8MB debug artifact), `.venv/`, `__pycache__/`

## Changes (branch: feature/linux-dictation-client)
1. Create `linux/` at repo root containing the 8 files above (README.md → `linux/README.md`, .gitignore → `linux/.gitignore`).
2. Update absolute install paths from `/home/atlas/projects/live-transcription-client` to `/home/atlas/projects/Scriberr/linux` in:
   - both `.service` files (WorkingDirectory + ExecStart)
   - `linux/README.md` (manual-run + setup instructions)
3. No code logic changes — the Python scripts derive their last-take path from `__file__`, so no other path references exist (verified).
4. Add `work-logs/WORK LOG - linux-dictation-client.md`.
5. Do NOT touch: the uncommitted `web/frontend/vite.config.ts` WIP, untracked `copy-transcript-to-clipboard.md`, or the existing untracked `work-logs/WORK LOG - copy-transcript-to-clipboard.md` (left uncommitted on purpose per earlier user request — stage ONLY the new work log file explicitly).

## Acceptance criteria
- `git status` shows new files only under `linux/` + the new work log; nothing else staged.
- File contents byte-identical to source except the two path substitutions.
- `cd ~/projects/Scriberr/linux && ~/projects/live-transcription-client/.venv/bin/python live_dictate.py --test ~/projects/live-transcription-client/last-take.wav` prints a transcript and exits 0 (proves the moved script runs from its new home).
- `python -m py_compile` passes on all three .py files.

## Gotchas
- `git add linux/ "work-logs/WORK LOG - linux-dictation-client.md"` — stage explicitly, never `git add -A` (would sweep in the floating WIP).
- A 41MB binary named `main` exists at repo root — always use `git diff main --` style path separation when diffing.
