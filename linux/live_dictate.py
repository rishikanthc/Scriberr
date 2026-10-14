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
TYPE_DELAY_MS_DEFAULT = "0"
TARGET_DEFAULT = ""
CHUNK_MODE_DEFAULT = "vad"
MAX_CHUNK_SECONDS_DEFAULT = "4"
MIN_CHUNK_SECONDS_DEFAULT = "0.5"
SILENCE_FLUSH_MS_DEFAULT = "300"

AUDIO_RATE = 16000

HOTKEY_RAW = os.environ.get("LT_DICTATE_HOTKEY", HOTKEY_DEFAULT)
MODEL_NAME = os.environ.get("LT_DICTATE_MODEL", MODEL_DEFAULT)
DEVICE = os.environ.get("LT_DICTATE_DEVICE", DEVICE_DEFAULT)
COMPUTE_TYPE = os.environ.get("LT_DICTATE_COMPUTE", COMPUTE_DEFAULT)
CHUNK_SECONDS = float(os.environ.get("LT_DICTATE_CHUNK_SECONDS", CHUNK_SECONDS_DEFAULT))
TYPE_DELAY_MS = int(os.environ.get("LT_DICTATE_TYPE_DELAY_MS", TYPE_DELAY_MS_DEFAULT))
TARGET = os.environ.get("LT_DICTATE_TARGET", TARGET_DEFAULT).strip()
MAX_CHUNK_SECONDS = float(os.environ.get("LT_DICTATE_MAX_CHUNK_SECONDS", MAX_CHUNK_SECONDS_DEFAULT))
MIN_CHUNK_SECONDS = float(os.environ.get("LT_DICTATE_MIN_CHUNK_SECONDS", MIN_CHUNK_SECONDS_DEFAULT))
SILENCE_FLUSH_MS = int(os.environ.get("LT_DICTATE_SILENCE_FLUSH_MS", SILENCE_FLUSH_MS_DEFAULT))

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

CHUNK_MODE_RAW = os.environ.get("LT_DICTATE_CHUNK_MODE", CHUNK_MODE_DEFAULT).strip().lower()
if CHUNK_MODE_RAW in ("vad", "fixed"):
    CHUNK_MODE = CHUNK_MODE_RAW
else:
    logger.warning(f"Unknown LT_DICTATE_CHUNK_MODE '{CHUNK_MODE_RAW}'; defaulting to 'vad'")
    CHUNK_MODE = "vad"


class AdaptiveChunker:
    """Phrase-aware audio chunker using Silero VAD to detect phrase boundaries."""

    def __init__(
        self,
        min_chunk_seconds: float = MIN_CHUNK_SECONDS,
        max_chunk_seconds: float = MAX_CHUNK_SECONDS,
        silence_flush_ms: int = SILENCE_FLUSH_MS,
    ):
        self.min_chunk_seconds = min_chunk_seconds
        self.max_chunk_seconds = max(max_chunk_seconds, min_chunk_seconds)
        self.silence_flush_ms = silence_flush_ms
        self.buffer = bytearray()
        self._vad_get_timestamps = None
        self._vad_opts = None
        self._vad_error_warned = False
        self.last_speech_end_samples: Optional[int] = None

    def _ensure_vad(self):
        if self._vad_get_timestamps is None:
            from faster_whisper.vad import get_speech_timestamps, VadOptions

            self._vad_get_timestamps = get_speech_timestamps
            self._vad_opts = VadOptions(
                threshold=0.5,
                min_silence_duration_ms=250,
                speech_pad_ms=100,
            )

    def feed(self, data: bytes) -> list[tuple[bytes, str]]:
        """Accumulates audio bytes and flushes chunks on phrase boundaries or max duration."""
        self.buffer.extend(data)
        flushed: list[tuple[bytes, str]] = []

        min_bytes = int(self.min_chunk_seconds * AUDIO_RATE * 2)
        max_bytes = int(self.max_chunk_seconds * AUDIO_RATE * 2)
        silence_samples = int(self.silence_flush_ms / 1000.0 * AUDIO_RATE)

        if len(self.buffer) >= min_bytes:
            phrase_end = False
            last_segment_end = None
            try:
                self._ensure_vad()
                usable_bytes = (len(self.buffer) // 2) * 2
                audio_f32 = (
                    np.frombuffer(self.buffer[:usable_bytes], dtype=np.int16).astype(np.float32)
                    / 32768.0
                )
                buffer_samples = len(audio_f32)
                segments = self._vad_get_timestamps(audio_f32, self._vad_opts)
                if segments:
                    last_segment_end = segments[-1]["end"]
                    if (buffer_samples - last_segment_end) >= silence_samples:
                        phrase_end = True
            except Exception as e:
                if not self._vad_error_warned:
                    logger.warning(f"VAD probe failed; falling back to max-chunk flush: {e}")
                    self._vad_error_warned = True
                phrase_end = False
                last_segment_end = None

            if phrase_end:
                chunk = bytes(self.buffer)
                self.buffer.clear()
                self.last_speech_end_samples = last_segment_end
                flushed.append((chunk, "phrase-end"))
            elif len(self.buffer) >= max_bytes:
                chunk = bytes(self.buffer)
                self.buffer.clear()
                self.last_speech_end_samples = last_segment_end
                flushed.append((chunk, "max"))

        return flushed

    def drain(self, min_bytes: int = 0) -> Optional[bytes]:
        """Flushes remaining audio in buffer if >= min_bytes, else discards."""
        if len(self.buffer) >= min_bytes:
            chunk = bytes(self.buffer)
            self.buffer.clear()
            return chunk
        self.buffer.clear()
        return None


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

                # Model warmup: transcribe 0.5s silence before setting dictating state
                t_warmup = time.time()
                warmup_audio = np.zeros(int(0.5 * AUDIO_RATE), dtype=np.float32)
                transcribe_kwargs = {"beam_size": 1, "vad_filter": True}
                if MODEL_NAME.endswith(".en"):
                    transcribe_kwargs["language"] = "en"
                segs, _ = self.model.transcribe(warmup_audio, **transcribe_kwargs)
                list(segs)
                elapsed = time.time() - t_warmup
                logger.info(f"Model warmed up in {elapsed*1000:.0f}ms")
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
        if CHUNK_MODE == "fixed":
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
        else:
            chunker = AdaptiveChunker(
                min_chunk_seconds=MIN_CHUNK_SECONDS,
                max_chunk_seconds=MAX_CHUNK_SECONDS,
                silence_flush_ms=SILENCE_FLUSH_MS,
            )
            try:
                while True:
                    data = proc.stdout.read(4096)
                    if not data:
                        break
                    for chunk, reason in chunker.feed(data):
                        logger.info(f"VAD flush: {len(chunk)/(AUDIO_RATE*2):.2f}s chunk ({reason})")
                        chunk_queue.put(chunk)
            except Exception as e:
                logger.error(f"Reader thread exception: {e}")
            finally:
                min_partial_bytes = int(0.5 * AUDIO_RATE * 2)
                buf_len = len(chunker.buffer)
                final_chunk = chunker.drain(min_bytes=min_partial_bytes)
                if final_chunk is not None:
                    logger.info(
                        f"Flushing final partial chunk ({len(final_chunk)} bytes, {len(final_chunk)/(AUDIO_RATE*2):.2f}s)"
                    )
                    chunk_queue.put(final_chunk)
                else:
                    logger.info(f"Discarding partial buffer under 0.5s ({buf_len} bytes)")
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
        if CHUNK_MODE == "fixed":
            chunk_info = f"CHUNK={CHUNK_SECONDS}s (fixed)"
        else:
            chunk_info = f"MODE=vad (min={MIN_CHUNK_SECONDS}s, max={MAX_CHUNK_SECONDS}s, silence={SILENCE_FLUSH_MS}ms)"
        logger.info(
            f"Tray application running ({chunk_info}, MODEL={MODEL_NAME}, DEVICE={DEVICE}, COMPUTE={COMPUTE_TYPE}, HOTKEY={self.hotkey_normalized})."
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


def run_test_chunks_mode(file_path: Optional[str] = None):
    """Dev simulation mode: feed audio through AdaptiveChunker in 4096-byte steps and transcribe."""
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

    if np.abs(audio).max() <= 1.5:
        raw_s16 = (np.clip(audio, -1.0, 1.0) * 32767.0).astype(np.int16).tobytes()
    else:
        raw_s16 = audio.astype(np.int16).tobytes()

    logger.info(f"Loading Whisper model '{MODEL_NAME}' on {DEVICE} ({COMPUTE_TYPE})...")
    from faster_whisper import WhisperModel

    t_load = time.time()
    model = WhisperModel(MODEL_NAME, device=DEVICE, compute_type=COMPUTE_TYPE)
    load_time = time.time() - t_load
    logger.info(f"Model loaded in {load_time:.2f}s.")

    chunker = AdaptiveChunker(
        min_chunk_seconds=MIN_CHUNK_SECONDS,
        max_chunk_seconds=MAX_CHUNK_SECONDS,
        silence_flush_ms=SILENCE_FLUSH_MS,
    )

    transcribe_kwargs = {"beam_size": 1, "vad_filter": True}
    if MODEL_NAME.endswith(".en"):
        transcribe_kwargs["language"] = "en"

    flushed_records = []
    step = 4096

    for i in range(0, len(raw_s16), step):
        step_data = raw_s16[i : i + step]
        file_time = min(len(raw_s16), i + step) / (AUDIO_RATE * 2)
        flushed = chunker.feed(step_data)
        for chunk, reason in flushed:
            chunk_len = len(chunk) / (AUDIO_RATE * 2)
            chunk_start_time = file_time - chunk_len
            if reason == "phrase-end" and chunker.last_speech_end_samples is not None:
                chunk_audio_end_time = chunk_start_time + (chunker.last_speech_end_samples / AUDIO_RATE)
            else:
                chunk_audio_end_time = file_time
            flushed_records.append((file_time, chunk_len, reason, chunk, chunk_audio_end_time))

    final_chunk = chunker.drain(min_bytes=int(MIN_CHUNK_SECONDS * AUDIO_RATE * 2))
    if final_chunk is not None:
        file_time = len(raw_s16) / (AUDIO_RATE * 2)
        chunk_len = len(final_chunk) / (AUDIO_RATE * 2)
        flushed_records.append((file_time, chunk_len, "drain", final_chunk, file_time))

    print("\n--- Chunks ---")
    total_chars = 0
    latencies = []
    phrase_end_count = 0
    max_count = 0

    for file_time, chunk_len, reason, chunk, audio_end_time in flushed_records:
        if reason == "phrase-end":
            phrase_end_count += 1
            latencies.append(file_time - audio_end_time)
        elif reason == "max":
            max_count += 1

        a = np.frombuffer(chunk, dtype=np.int16).astype(np.float32) / 32768.0
        segments, _ = model.transcribe(a, **transcribe_kwargs)
        text = " ".join(s.text.strip() for s in segments).strip()
        total_chars += len(text)
        print(f"[flush at file-time {file_time:5.1f}s | chunk {chunk_len:4.1f}s | {reason}] {text}")

    total_chunks = len(flushed_records)
    avg_latency = (sum(latencies) / len(latencies)) if latencies else 0.0
    max_latency = max(latencies) if latencies else 0.0

    print("--------------")
    print(
        f"Total chunks: {total_chunks} ({phrase_end_count} phrase-end, {max_count} max) | "
        f"Total chars: {total_chars} | "
        f"Latency (flush - audio_end): avg {avg_latency:.2f}s, max {max_latency:.2f}s\n"
    )
    sys.exit(0)


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
    if "--test-chunks" in sys.argv:
        idx = sys.argv.index("--test-chunks")
        test_file = None
        if idx + 1 < len(sys.argv) and not sys.argv[idx + 1].startswith("-"):
            test_file = sys.argv[idx + 1]
        run_test_chunks_mode(test_file)

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
