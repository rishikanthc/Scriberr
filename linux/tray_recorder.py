#!/usr/bin/env python3
"""
Live Transcription Desktop Client (X11 System Tray Daemon)
Records audio via PipeWire, uploads to Flask service, copies transcript to clipboard.
"""

import atexit
import fcntl
import logging
import os
import shutil
import signal
import subprocess
import sys
import tempfile
import threading
import time
from typing import Optional

from PIL import Image, ImageDraw
from pynput import keyboard
import pystray
import requests

# Configuration & environment defaults
SERVICE_URL = os.environ.get("LT_SERVICE_URL", "http://10.25.0.112:11060").rstrip("/")
START_HOTKEY = os.environ.get("LT_START_HOTKEY", "<f9>")
STOP_HOTKEY = os.environ.get("LT_STOP_HOTKEY", "<f10>")
AUDIO_RATE = 16000

LOCK_PATH = "/tmp/live-transcription-client.lock"
PROJECT_DIR = os.path.dirname(os.path.abspath(__file__))
LAST_TAKE_PATH = os.path.join(PROJECT_DIR, "last-take.wav")

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    stream=sys.stdout,
)
logger = logging.getLogger("live-transcription-client")
if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(line_buffering=True)


def acquire_single_instance_lock(lock_path: str = LOCK_PATH):
    """Enforces a single running instance. If lock is held, exit 0 silently."""
    try:
        lock_file = open(lock_path, "w")
        fcntl.flock(lock_file, fcntl.LOCK_EX | fcntl.LOCK_NB)
        return lock_file
    except (BlockingIOError, OSError):
        sys.exit(0)


def format_hotkey_label(hotkey: str) -> str:
    """Formats '<f9>' or '<ctrl>+<alt>+r' into friendly display text like 'F9'."""
    clean = hotkey.replace("<", "").replace(">", "")
    return "+".join(p.capitalize() if len(p) > 1 else p.upper() for p in clean.split("+"))


def create_circle_icon(color_rgba: tuple) -> Image.Image:
    """Generates a 64x64 RGBA anti-aliased circle icon at runtime."""
    img = Image.new("RGBA", (128, 128), (0, 0, 0, 0))
    ImageDraw.Draw(img).ellipse((8, 8, 120, 120), fill=color_rgba)
    return img.resize((64, 64), Image.Resampling.LANCZOS)


def copy_to_clipboard(text: str) -> bool:
    """Copies text to the system clipboard using xclip."""
    try:
        proc = subprocess.Popen(["xclip", "-selection", "clipboard"], stdin=subprocess.PIPE, close_fds=True)
        proc.communicate(input=text.encode("utf-8"), timeout=5)
        if proc.returncode == 0:
            logger.info("Transcript copied to clipboard.")
            return True
        logger.error(f"xclip failed with code {proc.returncode}")
        return False
    except Exception as e:
        logger.error(f"Failed to copy to clipboard: {e}")
        return False


class TrayRecorderApp:
    def __init__(self):
        self.lock = threading.Lock()
        self.state = "idle"  # idle | recording | transcribing | done | error
        self.status_desc = "idle"
        self.is_shutting_down = False
        self.record_proc: Optional[subprocess.Popen] = None
        self.current_temp_path: Optional[str] = None
        self.hotkeys: Optional[keyboard.GlobalHotKeys] = None

        # Pre-generate 64x64 runtime icons
        self.icons = {
            "idle": create_circle_icon((128, 128, 128, 255)),          # Gray
            "recording": create_circle_icon((239, 68, 68, 255)),       # Red
            "transcribing": create_circle_icon((245, 158, 11, 255)),   # Yellow/Orange
            "done": create_circle_icon((34, 197, 94, 255)),            # Green
        }

        self.start_label = format_hotkey_label(START_HOTKEY)
        self.stop_label = format_hotkey_label(STOP_HOTKEY)

        # Build pystray tray menu
        menu = pystray.Menu(
            pystray.MenuItem(lambda item: f"Status: {self.status_desc}", None, enabled=False),
            pystray.MenuItem(f"Start ({self.start_label})", lambda icon, item: self.start_recording()),
            pystray.MenuItem(f"Stop ({self.stop_label})", lambda icon, item: self.stop_recording()),
            pystray.Menu.SEPARATOR,
            pystray.MenuItem("Quit", lambda icon, item: self.quit()),
        )
        self.icon = pystray.Icon(
            name="live-transcription-client",
            icon=self.icons["idle"],
            title="Transcription: idle",
            menu=menu,
        )

    def _set_state_unlocked(self, state: str, tooltip: str, icon_key: Optional[str] = None, desc: Optional[str] = None):
        """Update app state, tray icon image, and tooltip while holding self.lock."""
        self.state = state
        self.status_desc = desc or state
        try:
            self.icon.icon = self.icons.get(icon_key or state, self.icons["idle"])
            self.icon.title = tooltip
        except Exception as e:
            logger.debug(f"Failed to update tray icon UI: {e}")

    def _revert_to_idle_after(self, delay: float, expected_state: str):
        """Timer callback that reverts state to idle after transient messages (done/error)."""
        time.sleep(delay)
        with self.lock:
            if not self.is_shutting_down and self.state == expected_state:
                self._set_state_unlocked("idle", "Transcription: idle")

    def _set_error_transient(self, err_text: str):
        """Display short error in tooltip for ~8s, then return to idle."""
        short_err = err_text[:60]
        logger.error(f"Error state: {err_text}")
        with self.lock:
            self._set_state_unlocked("error", short_err, icon_key="idle", desc="error")
        threading.Thread(target=self._revert_to_idle_after, args=(8.0, "error"), daemon=True).start()

    def start_recording(self):
        """Start audio recording. Starting while busy (recording/transcribing) is ignored."""
        with self.lock:
            if self.state in ("recording", "transcribing"):
                logger.info(f"Start requested while '{self.state}'; ignoring.")
                return

            if not shutil.which("pw-record"):
                self._set_error_transient("Error: pw-record not found in PATH")
                return

            try:
                fd, tmp_path = tempfile.mkstemp(suffix=".wav", prefix="live-take-")
                os.close(fd)
                self.current_temp_path = tmp_path

                cmd = ["pw-record", f"--rate={AUDIO_RATE}", "--channels=1", tmp_path]
                logger.info(f"Starting pw-record: {' '.join(cmd)}")
                self.record_proc = subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)

                # Verify pw-record started successfully
                time.sleep(0.15)
                if self.record_proc.poll() is not None:
                    _, err_bytes = self.record_proc.communicate()
                    err = err_bytes.decode("utf-8", errors="replace").strip() or "pw-record failed to start"
                    self.record_proc = None
                    self._set_error_transient(f"Error: {err}")
                    return

                self._set_state_unlocked("recording", f"recording… press {self.stop_label} to stop")
                logger.info("Recording started.")
            except Exception as e:
                self._set_error_transient(f"Error: {e}")

    def stop_recording(self):
        """Stop audio recording and launch asynchronous upload to transcription service."""
        with self.lock:
            if self.state != "recording":
                logger.info(f"Stop requested while '{self.state}'; ignoring.")
                return

            proc = self.record_proc
            self.record_proc = None
            temp_path = self.current_temp_path
            self.current_temp_path = None
            self._set_state_unlocked("transcribing", "transcribing…")

        # Stop pw-record: send SIGINT, fallback to SIGTERM, then SIGKILL
        if proc and proc.poll() is None:
            logger.info(f"Stopping pw-record (PID {proc.pid}) with SIGINT...")
            proc.send_signal(signal.SIGINT)
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.terminate()
                try:
                    proc.wait(timeout=2)
                except subprocess.TimeoutExpired:
                    proc.kill()
                    proc.wait()

        # Save last take in project dir for debugging
        if temp_path and os.path.exists(temp_path):
            try:
                shutil.copyfile(temp_path, LAST_TAKE_PATH)
                logger.info(f"Saved take to {LAST_TAKE_PATH} ({os.path.getsize(LAST_TAKE_PATH)} bytes)")
            except Exception as e:
                logger.warning(f"Could not copy last-take.wav: {e}")

        # Run upload & polling in background thread so tray icon never freezes
        threading.Thread(target=self._upload_and_poll, args=(temp_path,), daemon=True).start()

    def _upload_and_poll(self, temp_wav_path: Optional[str]):
        """Uploads take to backend service and polls until done or failed."""
        try:
            if not temp_wav_path or not os.path.exists(temp_wav_path) or os.path.getsize(temp_wav_path) == 0:
                self._set_error_transient("Error: Audio recording is empty")
                return

            logger.info(f"Uploading recording to {SERVICE_URL}/api/record...")
            with open(temp_wav_path, "rb") as f:
                files = {"audio": ("take.wav", f, "audio/wav")}
                resp = requests.post(f"{SERVICE_URL}/api/record", files=files, data={"title": "Desktop Hotkey Take"}, timeout=60)

            if resp.status_code != 201:
                err_msg = f"HTTP {resp.status_code}"
                try:
                    err_json = resp.json()
                    if err_json.get("error"):
                        err_msg = str(err_json["error"])
                except Exception:
                    pass
                self._set_error_transient(f"Error: {err_msg}")
                return

            conv_id = resp.json().get("conversation_id")
            if not conv_id:
                self._set_error_transient("Error: Missing conversation_id in response")
                return

            logger.info(f"Uploaded conversation {conv_id}; polling status...")
            start_poll = time.time()
            while time.time() - start_poll < 300.0:
                if self.is_shutting_down:
                    return

                try:
                    poll = requests.get(f"{SERVICE_URL}/api/conversations/{conv_id}/status", timeout=15)
                    if poll.status_code == 200:
                        data = poll.json()
                        status = data.get("status")
                        if status == "done":
                            transcript = data.get("transcript") or ""
                            if not transcript.strip():
                                transcript = "[no speech detected]"
                            logger.info(f"Transcription complete: {transcript[:80]!r}")
                            copy_to_clipboard(transcript)

                            with self.lock:
                                self._set_state_unlocked("done", "✓ copied to clipboard")
                            threading.Thread(target=self._revert_to_idle_after, args=(4.0, "done"), daemon=True).start()
                            return
                        elif status == "failed":
                            err = data.get("error") or "Transcription failed"
                            self._set_error_transient(f"Error: {err}")
                            return
                except requests.RequestException as e:
                    logger.warning(f"Polling warning: {e}")

                time.sleep(1.5)

            self._set_error_transient("Error: Transcription timed out")
        except requests.RequestException as e:
            self._set_error_transient(f"Network error: {e}")
        except Exception as e:
            logger.error(f"Upload flow error: {e}", exc_info=True)
            self._set_error_transient(f"Error: {e}")
        finally:
            if temp_wav_path and os.path.exists(temp_wav_path):
                try:
                    os.remove(temp_wav_path)
                except Exception:
                    pass

    def start_hotkeys(self):
        """Binds global hotkeys in a daemon thread via pynput."""
        def normalize_key(k: str) -> str:
            k = k.strip().lower()
            return f"<{k}>" if k in (f"f{i}" for i in range(1, 13)) else k

        start_key = normalize_key(START_HOTKEY)
        stop_key = normalize_key(STOP_HOTKEY)
        try:
            self.hotkeys = keyboard.GlobalHotKeys({
                start_key: self.start_recording,
                stop_key: self.stop_recording,
            })
            self.hotkeys.daemon = True
            self.hotkeys.start()
            logger.info(f"Bound global hotkeys: Start={start_key}, Stop={stop_key}")
        except Exception as e:
            logger.warning(f"Global hotkeys binding warning ({e}); menu controls remain active.")

    def run(self):
        """Runs the tray icon event loop in the main thread."""
        self.start_hotkeys()
        logger.info(f"Tray application running (SERVICE_URL={SERVICE_URL}).")
        self.icon.run()

    def quit(self):
        """Stops active recording, shuts down hotkeys and stops tray icon cleanly."""
        if self.is_shutting_down:
            return
        self.is_shutting_down = True
        logger.info("Shutting down live transcription client...")

        with self.lock:
            if self.record_proc and self.record_proc.poll() is None:
                try:
                    self.record_proc.send_signal(signal.SIGINT)
                    self.record_proc.wait(timeout=2)
                except Exception:
                    try:
                        self.record_proc.kill()
                    except Exception:
                        pass
                self.record_proc = None

            if self.current_temp_path and os.path.exists(self.current_temp_path):
                try:
                    os.remove(self.current_temp_path)
                except Exception:
                    pass

        if self.hotkeys:
            try:
                self.hotkeys.stop()
            except Exception:
                pass

        if self.icon:
            try:
                self.icon.stop()
            except Exception:
                pass


def main():
    lock_file = acquire_single_instance_lock()
    atexit.register(lambda: (fcntl.flock(lock_file, fcntl.LOCK_UN), lock_file.close()))

    app = TrayRecorderApp()

    def handle_signal(signum, frame):
        app.quit()

    signal.signal(signal.SIGINT, handle_signal)
    signal.signal(signal.SIGTERM, handle_signal)

    try:
        app.run()
    except Exception as e:
        logger.error(f"Fatal exception: {e}", exc_info=True)
        app.quit()


if __name__ == "__main__":
    main()
