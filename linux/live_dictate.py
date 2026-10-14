#!/usr/bin/env python3
"""
Live Dictation Desktop Client (Local / Offline X11 System Tray Daemon)
Records audio via PipeWire, transcribes locally with faster-whisper,
and types transcribed text directly at the cursor using xdotool.
"""

import atexit
import fcntl
import logging
import os
import queue
import shutil
import signal
import subprocess
import sys
import threading
import time
from typing import Optional

import numpy as np
from PIL import Image, ImageDraw
from pynput import keyboard
import pystray

# Environment variables & configuration defaults
HOTKEY_DEFAULT = "<f8>"
MODEL_DEFAULT = "base.en"
DEVICE_DEFAULT = "cpu"
COMPUTE_DEFAULT = "int8"
CHUNK_SECONDS_DEFAULT = "3"
TYPE_DELAY_MS_DEFAULT = "10"
TARGET_DEFAULT = ""

AUDIO_RATE = 16000

HOTKEY_RAW = os.environ.get("LT_DICTATE_HOTKEY", HOTKEY_DEFAULT)
MODEL_NAME = os.environ.get("LT_DICTATE_MODEL", MODEL_DEFAULT)
DEVICE = os.environ.get("LT_DICTATE_DEVICE", DEVICE_DEFAULT)
COMPUTE_TYPE = os.environ.get("LT_DICTATE_COMPUTE", COMPUTE_DEFAULT)
CHUNK_SECONDS = float(os.environ.get("LT_DICTATE_CHUNK_SECONDS", CHUNK_SECONDS_DEFAULT))
TYPE_DELAY_MS = int(os.environ.get("LT_DICTATE_TYPE_DELAY_MS", TYPE_DELAY_MS_DEFAULT))
TARGET = os.environ.get("LT_DICTATE_TARGET", TARGET_DEFAULT).strip()

LOCK_PATH = "/tmp/live-dictation-client.lock"
PROJECT_DIR = os.path.dirname(os.path.abspath(__file__))
DEFAULT_TEST_WAV = os.path.join(PROJECT_DIR, "last-take.wav")

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    stream=sys.stdout,
)
logger = logging.getLogger("live-dictate")
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


def normalize_key(k: str) -> str:
    """Normalizes key names like 'f8' into '<f8>'."""
    k = k.strip().lower()
    return f"<{k}>" if k in (f"f{i}" for i in range(1, 13)) else k


def format_hotkey_label(hotkey: str) -> str:
    """Formats '<f8>' or '<ctrl>+<alt>+d' into friendly display text like 'F8'."""
    clean = hotkey.replace("<", "").replace(">", "")
    return "+".join(p.capitalize() if len(p) > 1 else p.upper() for p in clean.split("+"))


def create_circle_icon(color_rgba: tuple) -> Image.Image:
    """Generates a 64x64 RGBA anti-aliased circle icon at runtime."""
    img = Image.new("RGBA", (128, 128), (0, 0, 0, 0))
    ImageDraw.Draw(img).ellipse((8, 8, 120, 120), fill=color_rgba)
    return img.resize((64, 64), Image.Resampling.LANCZOS)


def notify(title: str, message: str, urgency: str = "normal"):
    """Shows a desktop notification via notify-send; no-op with a debug log if missing."""
    if not shutil.which("notify-send"):
        logger.debug("notify-send not found in PATH; skipping notification")
        return
    try:
        subprocess.run(
            ["notify-send", "-a", "Live Dictation", "-u", urgency, title, message],
            timeout=5,
            check=False,
        )
    except Exception as e:
        logger.debug(f"notify-send failed: {e}")


class LiveDictateApp:
    def __init__(self):
        self.lock = threading.Lock()
        self.state = "idle"  # idle | loading | dictating | stopping | done | error
        self.status_desc = "idle"
        self.is_shutting_down = False

        self.model = None  # Loaded lazily on first dictation start
        self.record_proc: Optional[subprocess.Popen] = None
        self.reader_thread: Optional[threading.Thread] = None
        self.worker_thread: Optional[threading.Thread] = None
        self.chunk_queue: Optional[queue.Queue] = None

        self.hotkey_raw = HOTKEY_RAW
        self.hotkey_normalized = normalize_key(self.hotkey_raw)
        self.hotkey_label = format_hotkey_label(self.hotkey_normalized)

        self.last_typed_char: Optional[str] = None
        self.chars_typed = 0
        self.hotkeys: Optional[keyboard.GlobalHotKeys] = None

        # Check for xdotool availability
        if not shutil.which("xdotool"):
            logger.warning("xdotool not found in PATH; typing will be a no-op.")

        # 64x64 runtime circle icons
        self.icons = {
            "idle": create_circle_icon((128, 128, 128, 255)),        # Gray
            "dictating": create_circle_icon((239, 68, 68, 255)),     # Red
            "thinking": create_circle_icon((245, 158, 11, 255)),      # Yellow
            "done": create_circle_icon((34, 197, 94, 255)),          # Green
        }

        # Build pystray tray menu
        menu = pystray.Menu(
            pystray.MenuItem(lambda item: f"Status: {self.status_desc}", None, enabled=False),
            pystray.MenuItem(
                lambda item: f"Dictate ({'Stop' if self.state == 'dictating' else 'Start'}) ({self.hotkey_label})",
                lambda icon, item: self.toggle_dictation(),
            ),
            pystray.Menu.SEPARATOR,
            pystray.MenuItem("Quit", lambda icon, item: self.quit()),
        )
        self.icon = pystray.Icon(
            name="live-dictate",
            icon=self.icons["idle"],
            title=f"Dictation: idle ({self.hotkey_label})",
            menu=menu,
        )

    def _set_state_unlocked(
        self,
        state: str,
        tooltip: str,
        icon_key: Optional[str] = None,
        desc: Optional[str] = None,
    ):
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
                self._set_state_unlocked("idle", f"Dictation: idle ({self.hotkey_label})")

    def _set_error_transient(self, err_text: str):
        """Display short error in tooltip for ~8s, then return to idle."""
        short_err = err_text[:60]
        logger.error(f"Error state: {err_text}")
        with self.lock:
            self._set_state_unlocked("error", short_err, icon_key="idle", desc="error")
        notify("Dictation error", err_text, urgency="critical")
        threading.Thread(target=self._revert_to_idle_after, args=(8.0, "error"), daemon=True).start()

    def toggle_dictation(self):
        """Toggle dictation session on/off. Ignored while loading or stopping."""
        with self.lock:
            curr = self.state

        if curr in ("idle", "done"):
            self.start_dictation()
        elif curr == "dictating":
            self.stop_dictation()
        else:
            logger.info(f"Hotkey toggle ignored while state is '{curr}'.")

    def start_dictation(self):
        """Start audio dictation. Starting while busy is ignored."""
        with self.lock:
            if self.state in ("loading", "dictating", "stopping"):
                logger.info(f"Start requested while '{self.state}'; ignoring.")
                return

            if not shutil.which("pw-record"):
                self._set_error_transient("Error: pw-record not found in PATH")
                return

            self.last_typed_char = None
            self.chars_typed = 0

            if self.model is None:
                self._set_state_unlocked(
                    "loading",
                    f"loading model… ({self.hotkey_label})",
                    icon_key="thinking",
                    desc="loading model…",
                )
                threading.Thread(target=self._start_pipeline_thread, args=(True,), daemon=True).start()
            else:
                self._set_state_unlocked(
                    "dictating",
                    f"dictating… press {self.hotkey_label} to stop",
                    icon_key="dictating",
                    desc="dictating",
                )
                threading.Thread(target=self._start_pipeline_thread, args=(False,), daemon=True).start()

    def _start_pipeline_thread(self, load_model: bool):
        """Loads model if needed and spawns pw-record and worker threads."""
        if load_model:
            try:
                from faster_whisper import WhisperModel

                logger.info(f"Loading Whisper model '{MODEL_NAME}' on {DEVICE} ({COMPUTE_TYPE})...")
                self.model = WhisperModel(MODEL_NAME, device=DEVICE, compute_type=COMPUTE_TYPE)
                logger.info(f"Whisper model '{MODEL_NAME}' loaded successfully.")
            except Exception as e:
                logger.error(f"Failed to load Whisper model: {e}", exc_info=True)
                self._set_error_transient(f"Error loading model: {e}")
                return

            with self.lock:
                if self.is_shutting_down:
                    return
                self._set_state_unlocked(
                    "dictating",
                    f"dictating… press {self.hotkey_label} to stop",
                    icon_key="dictating",
                    desc="dictating",
                )

        cmd = ["pw-record", f"--rate={AUDIO_RATE}", "--channels=1", "--format=s16", "-"]
        if TARGET:
            cmd.extend(["--target", TARGET])

        logger.info(f"Starting pw-record: {' '.join(cmd)}")
        try:
            proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        except Exception as e:
            logger.error(f"Failed to spawn pw-record: {e}")
            self._set_error_transient(f"Error: {e}")
            return

        time.sleep(0.15)
        if proc.poll() is not None:
            _, err_bytes = proc.communicate()
            err = err_bytes.decode("utf-8", errors="replace").strip() or "pw-record failed to start"
            logger.error(f"pw-record failed immediately: {err}")
            self._set_error_transient(f"Error: {err}")
            return

        chunk_queue: queue.Queue = queue.Queue()
        with self.lock:
            if self.is_shutting_down or self.state != "dictating":
                try:
                    proc.send_signal(signal.SIGINT)
                    proc.wait(timeout=2)
                except Exception:
                    proc.kill()
                return

            self.record_proc = proc
            self.chunk_queue = chunk_queue

        reader_thread = threading.Thread(
            target=self._reader_worker,
            args=(proc, chunk_queue),
            daemon=True,
        )
        worker_thread = threading.Thread(
            target=self._transcribe_worker,
            args=(chunk_queue,),
            daemon=True,
        )
        self.reader_thread = reader_thread
        self.worker_thread = worker_thread

        reader_thread.start()
        worker_thread.start()
        logger.info("Dictation pipeline active.")
        notify("Dictation started", f"You're good to go — start talking. Press {self.hotkey_label} to stop.")
        if shutil.which("xdotool") is None:
            notify(
                "Dictation: no text output",
                "xdotool is not installed; transcriptions will not be typed. Run: sudo apt install xdotool",
                urgency="critical",
            )

    def _reader_worker(self, proc: subprocess.Popen, chunk_queue: queue.Queue):
        """Reads raw s16 mono audio bytes from pw-record stdout and enqueues chunks."""
        chunk_bytes_needed = int(CHUNK_SECONDS * AUDIO_RATE * 2)
        accumulated = bytearray()
        try:
            while True:
                data = proc.stdout.read(4096)
                if not data:
                    break
                accumulated.extend(data)
                while len(accumulated) >= chunk_bytes_needed:
                    chunk = bytes(accumulated[:chunk_bytes_needed])
                    accumulated = accumulated[chunk_bytes_needed:]
                    chunk_queue.put(chunk)
        except Exception as e:
            logger.error(f"Reader thread exception: {e}")
        finally:
            min_partial_bytes = int(0.5 * AUDIO_RATE * 2)
            if len(accumulated) >= min_partial_bytes:
                logger.info(
                    f"Flushing final partial chunk ({len(accumulated)} bytes, {len(accumulated)/(AUDIO_RATE*2):.2f}s)"
                )
                chunk_queue.put(bytes(accumulated))
            else:
                logger.info(f"Discarding partial buffer under 0.5s ({len(accumulated)} bytes)")
            chunk_queue.put(None)

    def _transcribe_worker(self, chunk_queue: queue.Queue):
        """Processes chunks from queue, transcribes with Whisper, and types at cursor."""
        while True:
            chunk = chunk_queue.get()
            if chunk is None:
                chunk_queue.task_done()
                break

            if self.is_shutting_down:
                chunk_queue.task_done()
                break

            try:
                # Indicate active transcription via yellow icon
                with self.lock:
                    if not self.is_shutting_down:
                        try:
                            self.icon.icon = self.icons["thinking"]
                        except Exception:
                            pass

                audio = np.frombuffer(chunk, dtype=np.int16).astype(np.float32) / 32768.0
                transcribe_kwargs = {"beam_size": 1, "vad_filter": True}
                if MODEL_NAME.endswith(".en"):
                    transcribe_kwargs["language"] = "en"

                segments, _ = self.model.transcribe(audio, **transcribe_kwargs)
                text = " ".join(s.text.strip() for s in segments).strip()

                if text:
                    if (
                        self.last_typed_char is not None
                        and not self.last_typed_char.isspace()
                        and not text[0].isspace()
                    ):
                        text = " " + text
                    self.type_text(text)
                    self.last_typed_char = text[-1]
            except Exception as e:
                logger.error(f"Error during chunk transcription/typing: {e}", exc_info=True)
            finally:
                with self.lock:
                    if not self.is_shutting_down and self.state == "dictating":
                        try:
                            self.icon.icon = self.icons["dictating"]
                        except Exception:
                            pass
                chunk_queue.task_done()

        # Session finished: mark as done and schedule revert to idle
        with self.lock:
            if not self.is_shutting_down and self.state in ("stopping", "dictating"):
                self._set_state_unlocked(
                    "done",
                    f"✓ Done ({self.hotkey_label})",
                    icon_key="done",
                    desc="done",
                )
                notify(
                    "Dictation stopped",
                    f"Typed {self.chars_typed} characters at your cursor."
                    if self.chars_typed
                    else "Stopped — no speech detected.",
                )
                threading.Thread(target=self._revert_to_idle_after, args=(2.5, "done"), daemon=True).start()

    def stop_dictation(self):
        """Stop audio recording and flush final partial chunk."""
        with self.lock:
            if self.state != "dictating":
                logger.info(f"Stop requested while '{self.state}'; ignoring.")
                return

            self._set_state_unlocked(
                "stopping",
                f"transcribing… ({self.hotkey_label})",
                icon_key="thinking",
                desc="transcribing…",
            )
            proc = self.record_proc
            self.record_proc = None

        threading.Thread(target=self._stop_pipeline_thread, args=(proc,), daemon=True).start()

    def _stop_pipeline_thread(self, proc: Optional[subprocess.Popen]):
        """Sends SIGINT to pw-record with SIGTERM/SIGKILL fallback."""
        if proc and proc.poll() is None:
            logger.info(f"Stopping pw-record (PID {proc.pid}) with SIGINT...")
            proc.send_signal(signal.SIGINT)
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                logger.warning("pw-record did not exit after SIGINT; sending SIGTERM...")
                proc.terminate()
                try:
                    proc.wait(timeout=2)
                except subprocess.TimeoutExpired:
                    logger.warning("pw-record did not exit after SIGTERM; killing...")
                    proc.kill()
                    proc.wait()
            logger.info("pw-record stopped.")

    def type_text(self, text: str):
        """Types text at cursor using xdotool."""
        if not shutil.which("xdotool"):
            logger.warning(f"xdotool not found in PATH; skipping typing: {text!r}")
            return
        try:
            cmd = ["xdotool", "type", "--clearmodifiers", f"--delay={TYPE_DELAY_MS}", "--", text]
            subprocess.run(cmd, timeout=30, check=True)
            self.chars_typed += len(text)
            logger.info(f"Typed: {text!r}")
        except Exception as e:
            logger.error(f"Failed to type text via xdotool: {e}")

    def start_hotkeys(self):
        """Binds global toggle hotkey via pynput."""
        try:
            self.hotkeys = keyboard.GlobalHotKeys({
                self.hotkey_normalized: self.toggle_dictation,
            })
            self.hotkeys.daemon = True
            self.hotkeys.start()
            logger.info(f"Bound global hotkey: {self.hotkey_normalized}")
        except Exception as e:
            logger.warning(f"Global hotkey binding warning ({e}); menu controls remain active.")

    def run(self):
        """Runs the tray icon event loop in the main thread."""
        self.start_hotkeys()
        logger.info(
            f"Tray application running (MODEL={MODEL_NAME}, DEVICE={DEVICE}, COMPUTE={COMPUTE_TYPE}, CHUNK={CHUNK_SECONDS}s, HOTKEY={self.hotkey_normalized})."
        )
        self.icon.run()

    def quit(self):
        """Stops active dictation session, shuts down hotkeys and stops tray cleanly."""
        if self.is_shutting_down:
            return
        self.is_shutting_down = True
        logger.info("Shutting down live dictation daemon...")

        with self.lock:
            proc = self.record_proc
            self.record_proc = None

        if proc and proc.poll() is None:
            try:
                proc.send_signal(signal.SIGINT)
                proc.wait(timeout=2)
            except Exception:
                try:
                    proc.kill()
                    proc.wait()
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


def run_test_mode(file_path: Optional[str] = None):
    """Test mode: decode audio via ffmpeg, transcribe with faster-whisper, print timing and transcript."""
    if not file_path:
        file_path = DEFAULT_TEST_WAV

    if not os.path.exists(file_path):
        logger.error(f"Test audio file not found: {file_path}")
        sys.exit(1)

    logger.info(f"Decoding {file_path} via ffmpeg...")
    cmd = ["ffmpeg", "-i", file_path, "-f", "f32le", "-ar", f"{AUDIO_RATE}", "-ac", "1", "-"]
    try:
        proc = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, check=True)
    except subprocess.CalledProcessError as e:
        logger.error(f"ffmpeg decoding failed: {e}")
        sys.exit(1)

    audio = np.frombuffer(proc.stdout, dtype=np.float32)
    duration = len(audio) / float(AUDIO_RATE)
    logger.info(f"Decoded {len(audio)} samples ({duration:.2f}s of audio).")

    logger.info(f"Loading Whisper model '{MODEL_NAME}' on {DEVICE} ({COMPUTE_TYPE})...")
    from faster_whisper import WhisperModel

    t_load = time.time()
    model = WhisperModel(MODEL_NAME, device=DEVICE, compute_type=COMPUTE_TYPE)
    load_time = time.time() - t_load
    logger.info(f"Model loaded in {load_time:.2f}s.")

    logger.info("Transcribing audio...")
    t_start = time.time()
    transcribe_kwargs = {"beam_size": 1, "vad_filter": True}
    if MODEL_NAME.endswith(".en"):
        transcribe_kwargs["language"] = "en"
    segments, info = model.transcribe(audio, **transcribe_kwargs)
    transcript = " ".join(s.text.strip() for s in segments).strip()
    elapsed = time.time() - t_start
    speedup = (duration / elapsed) if elapsed > 0 else 0.0

    print("\n--- Transcript ---")
    print(transcript)
    print("------------------")
    print(f"Audio duration:     {duration:.2f}s")
    print(f"Transcription time: {elapsed:.2f}s ({speedup:.1f}x realtime)\n")
    sys.exit(0)


def main():
    if "--test" in sys.argv:
        idx = sys.argv.index("--test")
        test_file = None
        if idx + 1 < len(sys.argv) and not sys.argv[idx + 1].startswith("-"):
            test_file = sys.argv[idx + 1]
        run_test_mode(test_file)

    lock_file = acquire_single_instance_lock()
    atexit.register(lambda: (fcntl.flock(lock_file, fcntl.LOCK_UN), lock_file.close()))

    app = LiveDictateApp()

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
