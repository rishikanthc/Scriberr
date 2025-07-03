#!/usr/bin/env python3
"""
Real-time transcription server using faster-whisper
This version provides actual transcription with minimal latency
"""

import asyncio
import json
import logging
import websockets
import threading
import time
import base64
import io
import wave
import numpy as np
import tempfile
import os
from typing import Dict, Optional
from faster_whisper import WhisperModel

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class LiveTranscriptionServer:
    def __init__(self, host="localhost", port=9090):
        self.host = host
        self.port = port
        self.clients: Dict[str, Dict] = {}
        self.models: Dict[str, WhisperModel] = {}

    def start(self):
        """Start the WebSocket server"""
        logger.info(f"Starting Live Transcription server on {self.host}:{self.port}")

    def stop(self):
        """Stop the server"""
        logger.info("Live Transcription server stopped")
        # Clean up models
        for client_id, model in self.models.items():
            try:
                del model
            except Exception as e:
                logger.error(f"Error cleaning up model for {client_id}: {e}")


# Global server instance
transcription_server = None


async def start_websocket_server():
    """Start the WebSocket server"""
    async with websockets.serve(websocket_handler, "localhost", 9090):
        logger.info("WebSocket server started on ws://localhost:9090")
        await asyncio.Future()  # run forever


def audio_chunk_to_numpy(audio_bytes: bytes) -> np.ndarray:
    """Convert audio bytes to numpy array for faster-whisper"""
    try:
        # Try to read as WAV first
        with io.BytesIO(audio_bytes) as audio_io:
            with wave.open(audio_io, "rb") as wav_file:
                # Read audio data
                frames = wav_file.readframes(wav_file.getnframes())
                # Convert to numpy array
                audio_array = np.frombuffer(frames, dtype=np.int16)
                # Convert to float32 and normalize
                audio_array = audio_array.astype(np.float32) / 32768.0
                return audio_array
    except Exception as e:
        logger.warning(f"Could not read as WAV, trying raw audio: {e}")
        try:
            # Try as raw PCM data (16-bit, 16kHz, mono)
            audio_array = np.frombuffer(audio_bytes, dtype=np.int16)
            audio_array = audio_array.astype(np.float32) / 32768.0
            print(
                f"DEBUG: Converted {len(audio_bytes)} bytes to {len(audio_array)} samples"
            )
            return audio_array
        except Exception as e2:
            logger.error(f"Failed to convert audio to numpy array: {e2}")
            print(f"DEBUG: Audio bytes length: {len(audio_bytes)}")
            print(f"DEBUG: Audio bytes sample: {audio_bytes[:20]}")
            return None


def process_webm_audio(audio_bytes: bytes) -> np.ndarray:
    """Process WebM audio using pydub for better compatibility"""
    try:
        # Try using pydub if available
        from pydub import AudioSegment
        import io

        # Load audio from bytes
        audio = AudioSegment.from_file(io.BytesIO(audio_bytes), format="webm")

        # Convert to mono and 16kHz
        audio = audio.set_channels(1).set_frame_rate(16000)

        # Export as raw PCM
        raw_audio = audio.raw_data

        # Convert to numpy array
        audio_array = np.frombuffer(raw_audio, dtype=np.int16)
        audio_array = audio_array.astype(np.float32) / 32768.0

        return audio_array

    except ImportError:
        logger.warning("pydub not available, trying ffmpeg fallback")
        return process_webm_audio_ffmpeg(audio_bytes)
    except Exception as e:
        logger.error(f"Error processing WebM audio with pydub: {e}")
        return process_webm_audio_ffmpeg(audio_bytes)


def process_mp4_audio(audio_bytes: bytes) -> np.ndarray:
    """Process MP4 audio using pydub for better compatibility"""
    try:
        # Try using pydub if available
        from pydub import AudioSegment
        import io

        print(f"DEBUG: Processing MP4 audio with pydub, {len(audio_bytes)} bytes")

        # Check if this looks like a complete MP4 file or a fragment
        if audio_bytes.startswith(b"\x00\x00\x00") and b"moof" in audio_bytes[:20]:
            print(f"DEBUG: Detected MP4 fragment, trying to process as fragment")
            # This is likely a fragmented MP4, try to handle it differently
            return process_mp4_fragment(audio_bytes)

        # Load audio from bytes
        audio = AudioSegment.from_file(io.BytesIO(audio_bytes), format="mp4")

        print(
            f"DEBUG: Loaded audio: {len(audio)}ms, {audio.channels} channels, {audio.frame_rate}Hz"
        )

        # Convert to mono and 16kHz
        audio = audio.set_channels(1).set_frame_rate(16000)

        print(
            f"DEBUG: Converted audio: {len(audio)}ms, {audio.channels} channels, {audio.frame_rate}Hz"
        )

        # Export as raw PCM
        raw_audio = audio.raw_data

        print(f"DEBUG: Raw audio data: {len(raw_audio)} bytes")

        # Convert to numpy array
        audio_array = np.frombuffer(raw_audio, dtype=np.int16)
        audio_array = audio_array.astype(np.float32) / 32768.0

        print(f"DEBUG: MP4 audio processed successfully: {len(audio_array)} samples")
        print(f"DEBUG: Audio range: {audio_array.min():.4f} to {audio_array.max():.4f}")
        return audio_array

    except ImportError:
        logger.warning("pydub not available, trying ffmpeg fallback")
        return process_mp4_audio_ffmpeg(audio_bytes)
    except Exception as e:
        logger.error(f"Error processing MP4 audio with pydub: {e}")
        print(f"DEBUG: Pydub error details: {str(e)}")
        return process_mp4_fragment(audio_bytes)


def process_mp4_fragment(audio_bytes: bytes) -> np.ndarray:
    """Process MP4 fragment by extracting raw audio data"""
    try:
        print(f"DEBUG: Processing MP4 fragment, {len(audio_bytes)} bytes")

        # For fragmented MP4, we'll try a simpler approach
        # Skip the first 200 bytes (MP4 headers) and try to process the rest as audio
        if len(audio_bytes) > 200:
            audio_data = audio_bytes[200:]
            print(
                f"DEBUG: Skipping first 200 bytes, processing {len(audio_data)} bytes as audio"
            )

            # Try to interpret as raw PCM (16-bit, 16kHz, mono)
            # Make sure we have an even number of bytes for 16-bit samples
            if len(audio_data) % 2 == 0:
                try:
                    audio_array = np.frombuffer(audio_data, dtype=np.int16)
                    audio_array = audio_array.astype(np.float32) / 32768.0
                    print(f"DEBUG: Extracted {len(audio_array)} samples from fragment")
                    return audio_array
                except Exception as e:
                    print(f"DEBUG: Failed to process as PCM: {e}")
            else:
                # Remove the last byte to make it even
                audio_data = audio_data[:-1]
                print(
                    f"DEBUG: Removed last byte to make even, processing {len(audio_data)} bytes"
                )
                try:
                    audio_array = np.frombuffer(audio_data, dtype=np.int16)
                    audio_array = audio_array.astype(np.float32) / 32768.0
                    print(f"DEBUG: Extracted {len(audio_array)} samples from fragment")
                    return audio_array
                except Exception as e:
                    print(f"DEBUG: Failed to process as PCM after trimming: {e}")

        # If that fails, try processing the entire fragment
        print(f"DEBUG: Trying to process entire fragment as raw audio")
        try:
            audio_array = np.frombuffer(audio_bytes, dtype=np.int16)
            audio_array = audio_array.astype(np.float32) / 32768.0
            print(f"DEBUG: Processed entire fragment: {len(audio_array)} samples")
            return audio_array
        except Exception as e:
            print(f"DEBUG: Failed to process entire fragment: {e}")
            return None

    except Exception as e:
        logger.error(f"Error processing MP4 fragment: {e}")
        return None


def process_mp4_audio_ffmpeg(audio_bytes: bytes) -> np.ndarray:
    """Process MP4 audio using ffmpeg to convert to WAV"""
    try:
        # Create a temporary file for the MP4 audio
        with tempfile.NamedTemporaryFile(suffix=".mp4", delete=False) as temp_mp4:
            temp_mp4.write(audio_bytes)
            temp_mp4_path = temp_mp4.name

        # Create a temporary file for the WAV output
        with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as temp_wav:
            temp_wav_path = temp_wav.name

        try:
            # Use ffmpeg to convert MP4 to WAV
            import subprocess

            cmd = [
                "ffmpeg",
                "-i",
                temp_mp4_path,
                "-acodec",
                "pcm_s16le",
                "-ar",
                "16000",
                "-ac",
                "1",
                "-y",  # Overwrite output file
                temp_wav_path,
            ]

            result = subprocess.run(cmd, capture_output=True, text=True)

            if result.returncode != 0:
                logger.error(f"ffmpeg MP4 conversion failed: {result.stderr}")
                return None

            # Read the converted WAV file
            with wave.open(temp_wav_path, "rb") as wav_file:
                frames = wav_file.readframes(wav_file.getnframes())
                audio_array = np.frombuffer(frames, dtype=np.int16)
                audio_array = audio_array.astype(np.float32) / 32768.0
                return audio_array

        finally:
            # Clean up temporary files
            try:
                os.unlink(temp_mp4_path)
                os.unlink(temp_wav_path)
            except:
                pass

    except Exception as e:
        logger.error(f"Error processing MP4 audio with ffmpeg: {e}")
        return None


def process_webm_audio_ffmpeg(audio_bytes: bytes) -> np.ndarray:
    """Process WebM audio using ffmpeg to convert to WAV"""
    try:
        # Create a temporary file for the WebM audio
        with tempfile.NamedTemporaryFile(suffix=".webm", delete=False) as temp_webm:
            temp_webm.write(audio_bytes)
            temp_webm_path = temp_webm.name

        # Create a temporary file for the WAV output
        with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as temp_wav:
            temp_wav_path = temp_wav.name

        try:
            # Use ffmpeg to convert WebM to WAV
            import subprocess

            cmd = [
                "ffmpeg",
                "-i",
                temp_webm_path,
                "-acodec",
                "pcm_s16le",
                "-ar",
                "16000",
                "-ac",
                "1",
                "-y",  # Overwrite output file
                temp_wav_path,
            ]

            result = subprocess.run(cmd, capture_output=True, text=True)

            if result.returncode != 0:
                logger.error(f"ffmpeg conversion failed: {result.stderr}")
                return None

            # Read the converted WAV file
            with wave.open(temp_wav_path, "rb") as wav_file:
                frames = wav_file.readframes(wav_file.getnframes())
                audio_array = np.frombuffer(frames, dtype=np.int16)
                audio_array = audio_array.astype(np.float32) / 32768.0
                return audio_array

        finally:
            # Clean up temporary files
            try:
                os.unlink(temp_webm_path)
                os.unlink(temp_wav_path)
            except:
                pass

    except Exception as e:
        logger.error(f"Error processing WebM audio with ffmpeg: {e}")
        return None


async def websocket_handler(websocket):
    """Handle WebSocket connections for live transcription"""
    client_id = None
    whisper_model = None
    audio_buffer = []
    last_transcription_time = 0

    # Log the connection
    logger.info(f"New WebSocket connection from {websocket.remote_address}")

    try:
        # Wait for client initialization
        init_message = await websocket.recv()
        init_data = json.loads(init_message)

        if init_data.get("type") != "init":
            logger.error(f"Expected init message, got: {init_data.get('type')}")
            await websocket.send(
                json.dumps({"type": "error", "message": "Expected init message"})
            )
            return

        client_id = init_data.get("client_id")
        model_size = init_data.get("model_size", "small")
        language = init_data.get("language", "en")
        translate = init_data.get("translate", False)

        logger.info(f"New client {client_id} connecting with model {model_size}")

        # Create Whisper model for this client
        try:
            # Map model sizes to actual model names
            model_map = {
                "tiny": "tiny",
                "base": "base",
                "small": "small",
                "medium": "medium",
                "large": "large-v2",
            }

            actual_model = model_map.get(model_size, "small")
            # Use CPU for better compatibility, int8 for speed
            whisper_model = WhisperModel(
                actual_model, device="cpu", compute_type="int8"
            )
            transcription_server.models[client_id] = whisper_model

            logger.info(f"Created Whisper model {actual_model} for client {client_id}")
        except Exception as e:
            logger.error(f"Failed to create Whisper model for {client_id}: {e}")
            await websocket.send(
                json.dumps(
                    {
                        "type": "error",
                        "message": f"Failed to initialize model: {str(e)}",
                    }
                )
            )
            return

        # Send ready message
        await websocket.send(
            json.dumps(
                {
                    "type": "ready",
                    "client_id": client_id,
                    "model_size": model_size,
                    "language": language,
                    "translate": translate,
                }
            )
        )

        # Handle audio data and transcription
        async def process_audio():
            nonlocal audio_buffer, last_transcription_time

            while True:
                try:
                    message = await websocket.recv()
                    data = json.loads(message)

                    if data["type"] == "audio_data":
                        # Process audio chunk
                        audio_data = data["audio"]
                        audio_format = data.get("format", "unknown")
                        logger.info(
                            f"Client {client_id} sent audio data of length {len(audio_data)}, format: {audio_format}"
                        )
                        print(f"DEBUG: Received audio format: {audio_format}")

                        try:
                            # Decode base64 audio data
                            audio_bytes = base64.b64decode(audio_data)
                            logger.info(
                                f"Client {client_id} decoded audio bytes: {len(audio_bytes)} bytes"
                            )

                            # Debug: Check the first few bytes to see what format we're getting
                            if len(audio_bytes) > 10:
                                print(
                                    f"DEBUG: First 10 bytes: {audio_bytes[:10].hex()}"
                                )
                                print(
                                    f"DEBUG: Audio data starts with: {audio_bytes[:20]}"
                                )

                            # Process audio based on format
                            if audio_format.startswith(
                                "audio/mp4"
                            ) or audio_format.startswith("audio/m4a"):
                                print(
                                    f"DEBUG: Processing MP4 audio format: {audio_format}"
                                )
                                audio_array = process_mp4_audio(audio_bytes)
                            elif audio_format.startswith("audio/webm"):
                                print(
                                    f"DEBUG: Processing WebM audio format: {audio_format}"
                                )
                                audio_array = process_webm_audio(audio_bytes)
                            else:
                                print(
                                    f"DEBUG: Unknown format {audio_format}, trying WebM first"
                                )
                                audio_array = process_webm_audio(audio_bytes)

                            if audio_array is None:
                                print(
                                    f"DEBUG: Format-specific processing failed, trying fallback"
                                )
                                # Fallback to direct conversion
                                audio_array = audio_chunk_to_numpy(audio_bytes)

                            # Additional validation
                            if audio_array is not None and len(audio_array) > 0:
                                # Check if audio has any non-zero values (actual audio content)
                                if np.abs(audio_array).max() < 0.001:
                                    print(
                                        f"DEBUG: Audio appears to be silence, skipping"
                                    )
                                    continue

                            if audio_array is not None:
                                logger.info(
                                    f"Client {client_id} converted to numpy array: {len(audio_array)} samples"
                                )
                                # Add to buffer
                                audio_buffer.extend(audio_array)
                                logger.info(
                                    f"Client {client_id} buffer size: {len(audio_buffer)} samples"
                                )
                                print(
                                    f"DEBUG: Added {len(audio_array)} samples to buffer, total: {len(audio_buffer)}"
                                )

                                # Process buffer when it's large enough (about 1 second of audio)
                                # Reduced for more responsive transcription
                                print(
                                    f"DEBUG: Buffer size: {len(audio_buffer)}, threshold: 16000"
                                )
                                if len(audio_buffer) > 16000:  # 16kHz * 1 second
                                    current_time = time.time()

                                    # Throttle transcription to avoid overwhelming the model
                                    if (
                                        current_time - last_transcription_time > 0.2
                                    ):  # Max 5 transcriptions per second
                                        logger.info(
                                            f"Client {client_id} processing audio buffer of {len(audio_buffer)} samples"
                                        )
                                        # Get the audio to process
                                        audio_to_process = np.array(
                                            audio_buffer[:16000]
                                        )
                                        audio_buffer = audio_buffer[
                                            16000:
                                        ]  # Keep remaining audio

                                        # Debug: Check audio characteristics
                                        print(
                                            f"DEBUG: Audio stats - min: {audio_to_process.min():.4f}, max: {audio_to_process.max():.4f}, mean: {audio_to_process.mean():.4f}, std: {audio_to_process.std():.4f}"
                                        )
                                        print(
                                            f"DEBUG: Audio duration: {len(audio_to_process) / 16000:.2f} seconds"
                                        )

                                        # Run transcription in a thread to avoid blocking
                                        def transcribe_audio():
                                            try:
                                                logger.info(
                                                    f"Client {client_id} starting transcription..."
                                                )
                                                print(
                                                    f"DEBUG: Starting transcription for client {client_id}"
                                                )
                                                print(
                                                    f"DEBUG: Audio array shape: {audio_to_process.shape}"
                                                )
                                                print(
                                                    f"DEBUG: Audio array dtype: {audio_to_process.dtype}"
                                                )
                                                print(
                                                    f"DEBUG: Audio array min/max: {audio_to_process.min():.4f}/{audio_to_process.max():.4f}"
                                                )
                                                segments, info = (
                                                    whisper_model.transcribe(
                                                        audio_to_process,
                                                        language=language,
                                                        task="translate"
                                                        if translate
                                                        else "transcribe",
                                                        beam_size=3,  # Reduced for speed
                                                        best_of=3,  # Reduced for speed
                                                        temperature=0.0,
                                                        compression_ratio_threshold=2.4,
                                                        log_prob_threshold=-1.0,
                                                        no_speech_threshold=0.6,  # Back to default to avoid hallucinations
                                                        condition_on_previous_text=False,
                                                        repetition_penalty=1.2,  # Penalize repetitive text
                                                        initial_prompt=None,
                                                        word_timestamps=False,
                                                    )
                                                )

                                                # Collect transcription results
                                                transcription_text = ""
                                                segments_list = list(
                                                    segments
                                                )  # Convert generator to list
                                                print(
                                                    f"DEBUG: Number of segments returned: {len(segments_list)}"
                                                )
                                                print(
                                                    f"DEBUG: Transcription info: {info}"
                                                )

                                                for i, segment in enumerate(
                                                    segments_list
                                                ):
                                                    print(
                                                        f"DEBUG: Segment {i}: '{segment.text}' (start: {segment.start:.2f}s, end: {segment.end:.2f}s)"
                                                    )
                                                    transcription_text += (
                                                        segment.text + " "
                                                    )

                                                result = transcription_text.strip()
                                                print(
                                                    f"DEBUG: Final transcription result: '{result}'"
                                                )
                                                logger.info(
                                                    f"Client {client_id} transcription result: '{result}'"
                                                )

                                                # Return empty string if no meaningful transcription
                                                if (
                                                    not result
                                                    or len(result.strip()) == 0
                                                ):
                                                    print(
                                                        f"DEBUG: Empty transcription result, returning None"
                                                    )
                                                    return None

                                                return result
                                            except Exception as e:
                                                logger.error(
                                                    f"Transcription error: {e}"
                                                )
                                                return None

                                        # Run transcription in thread pool
                                        loop = asyncio.get_event_loop()
                                        transcription_text = await loop.run_in_executor(
                                            None, transcribe_audio
                                        )

                                        if transcription_text:
                                            print(
                                                f"DEBUG: Processing transcription text: '{transcription_text}'"
                                            )
                                            # Check for repetitive text (simple heuristic)
                                            if (
                                                len(transcription_text.split()) > 3
                                            ):  # Only check longer phrases
                                                words = transcription_text.split()
                                                # Check if the same word appears too many times
                                                word_counts = {}
                                                for word in words:
                                                    word_counts[word.lower()] = (
                                                        word_counts.get(word.lower(), 0)
                                                        + 1
                                                    )

                                                # If any word appears more than 50% of the time, it's likely repetitive
                                                max_repetition = (
                                                    max(word_counts.values())
                                                    if word_counts
                                                    else 0
                                                )
                                                if max_repetition > len(words) * 0.5:
                                                    logger.info(
                                                        f"Client {client_id} skipping repetitive text: {transcription_text}"
                                                    )
                                                    print(
                                                        f"DEBUG: Skipping repetitive text: {transcription_text}"
                                                    )
                                                    continue

                                            response = {
                                                "type": "transcription",
                                                "text": transcription_text,
                                                "timestamp": time.time(),
                                                "final": True,
                                            }

                                            logger.info(
                                                f"Client {client_id} sending transcription: {transcription_text}"
                                            )
                                            print(
                                                f"DEBUG: Sending transcription to frontend: '{transcription_text}'"
                                            )
                                            await websocket.send(json.dumps(response))
                                            print(
                                                f"DEBUG: Transcription sent successfully"
                                            )
                                        else:
                                            logger.info(
                                                f"Client {client_id} no transcription text generated"
                                            )
                                            print(
                                                f"DEBUG: No transcription text generated"
                                            )

                                        last_transcription_time = current_time
                                    else:
                                        logger.info(
                                            f"Client {client_id} throttling transcription (too soon)"
                                        )
                            else:
                                logger.warning(
                                    f"Client {client_id} failed to convert audio to numpy array"
                                )

                        except Exception as e:
                            logger.error(f"Error processing audio for {client_id}: {e}")
                            # Send error but don't break the connection
                            error_response = {
                                "type": "error",
                                "message": f"Audio processing error: {str(e)}",
                            }
                            await websocket.send(json.dumps(error_response))

                    elif data["type"] == "stop":
                        logger.info(f"Client {client_id} stopping transcription")
                        break

                except websockets.exceptions.ConnectionClosed:
                    logger.info(f"Client {client_id} disconnected")
                    break
                except json.JSONDecodeError as e:
                    logger.error(f"Invalid JSON from client {client_id}: {e}")
                    await websocket.send(
                        json.dumps({"type": "error", "message": "Invalid JSON format"})
                    )
                except Exception as e:
                    logger.error(f"Error processing message from {client_id}: {e}")
                    await websocket.send(
                        json.dumps({"type": "error", "message": str(e)})
                    )
                    break

        await process_audio()

    except json.JSONDecodeError as e:
        logger.error(f"Invalid JSON in init message: {e}")
        await websocket.send(
            json.dumps({"type": "error", "message": "Invalid JSON in init message"})
        )
    except Exception as e:
        logger.error(f"WebSocket error: {e}")
    finally:
        # Cleanup
        if client_id and whisper_model:
            try:
                if client_id in transcription_server.models:
                    del transcription_server.models[client_id]
            except Exception as e:
                logger.error(f"Error cleaning up model for {client_id}: {e}")
        logger.info(f"Client {client_id} cleanup complete")


def main():
    """Main function to start the WebSocket server"""
    global transcription_server

    # Start transcription server
    transcription_server = LiveTranscriptionServer()
    transcription_server.start()

    # Start WebSocket server
    try:
        asyncio.run(start_websocket_server())
    except KeyboardInterrupt:
        logger.info("Shutting down server...")
    finally:
        if transcription_server:
            transcription_server.stop()


if __name__ == "__main__":
    main()
