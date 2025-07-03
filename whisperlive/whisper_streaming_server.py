#!/usr/bin/env python3
"""
Real-time transcription server using faster-whisper with FastAPI
Based on whisper_streaming approach for robust audio handling
"""

import json
import logging
import tempfile
import os
from typing import Dict, Optional

import numpy as np
import librosa
from faster_whisper import WhisperModel
from fastapi import FastAPI, WebSocket, WebSocketDisconnect, WebSocketException
from fastapi.middleware.cors import CORSMiddleware
import base64
import io

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Create FastAPI app
app = FastAPI(title="Live Transcription Server", version="1.0.0")

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


class StreamingWhisperASR:
    """Wrapper for faster-whisper with streaming capabilities"""

    def __init__(
        self, model_size="small", language="en", device="cpu", compute_type="int8"
    ):
        self.model = WhisperModel(model_size, device=device, compute_type=compute_type)
        self.language = language
        self.task = "transcribe"

    def transcribe_streaming(self, audio_chunk: np.ndarray):
        """Transcribe audio chunk with streaming approach"""
        try:
            # Preprocess audio: normalize and apply some gain
            audio_chunk = (
                audio_chunk / np.max(np.abs(audio_chunk))
                if np.max(np.abs(audio_chunk)) > 0
                else audio_chunk
            )
            # Apply slight gain to make audio more audible
            audio_chunk = audio_chunk * 1.5
            # Clip to prevent distortion
            audio_chunk = np.clip(audio_chunk, -1.0, 1.0)

            # Process the audio chunk with anti-repetition settings
            segments, info = self.model.transcribe(
                audio_chunk,
                language=self.language,
                task=self.task,
                beam_size=1,  # Reduced for faster processing
                best_of=1,  # Reduced for faster processing
                temperature=0.0,
                compression_ratio_threshold=1.8,  # More strict (was 2.4)
                log_prob_threshold=-0.7,  # More strict (was -1.0)
                no_speech_threshold=0.3,  # More sensitive (was 0.6)
                condition_on_previous_text=False,  # Critical for live transcription
                repetition_penalty=1.2,  # Increased penalty to prevent repetition
                initial_prompt=None,  # Remove initial prompt to avoid bias
                word_timestamps=False,
            )

            # Convert generator to list
            segments_list = list(segments)

            if segments_list:
                # Combine all segments and clean up repetition
                transcription = " ".join(
                    [segment.text.strip() for segment in segments_list]
                )

                # Clean up common repetition patterns
                transcription = self._clean_repetition(transcription)

                logger.info(f"Generated transcription: '{transcription}'")
                return transcription.strip()
            else:
                logger.info("No segments generated - audio might be too short or quiet")
                return ""

        except Exception as e:
            logger.error(f"Transcription error: {e}")
            return ""

    def _clean_repetition(self, text: str) -> str:
        """Clean up repetitive text patterns"""
        if not text:
            return text

        # Split into sentences
        sentences = text.split(".")
        if len(sentences) <= 1:
            return text

        # Remove duplicate consecutive sentences
        cleaned_sentences = []
        for i, sentence in enumerate(sentences):
            sentence = sentence.strip()
            if sentence and (i == 0 or sentence != sentences[i - 1].strip()):
                cleaned_sentences.append(sentence)

        # If we have too many similar sentences, take only the first few
        if len(cleaned_sentences) > 3:
            # Check if sentences are very similar
            first_sentence = cleaned_sentences[0]
            similar_count = 1
            for sentence in cleaned_sentences[1:]:
                if self._similarity(first_sentence, sentence) > 0.8:
                    similar_count += 1
                else:
                    break

            if similar_count > 2:
                cleaned_sentences = cleaned_sentences[:2]

        return ". ".join(cleaned_sentences)

    def _similarity(self, text1: str, text2: str) -> float:
        """Calculate similarity between two text strings"""
        if not text1 or not text2:
            return 0.0

        # Simple word overlap similarity
        words1 = set(text1.lower().split())
        words2 = set(text2.lower().split())

        if not words1 or not words2:
            return 0.0

        intersection = words1.intersection(words2)
        union = words1.union(words2)

        return len(intersection) / len(union) if union else 0.0


def process_audio_chunk(
    audio_bytes: bytes, format_type: str = "audio/wav"
) -> Optional[np.ndarray]:
    """Process audio chunk and convert to numpy array"""
    try:
        # For browser audio formats, try multiple approaches
        if format_type.startswith("audio/mp4") or format_type.startswith("audio/m4a"):
            # Try pydub first for MP4
            try:
                from pydub import AudioSegment

                with tempfile.NamedTemporaryFile(
                    suffix=".mp4", delete=False
                ) as temp_file:
                    temp_file.write(audio_bytes)
                    temp_file_path = temp_file.name

                # Load with pydub
                audio = AudioSegment.from_file(temp_file_path, format="mp4")
                audio = audio.set_channels(1).set_frame_rate(16000)
                audio_array = (
                    np.array(audio.get_array_of_samples(), dtype=np.float32) / 32768.0
                )
                os.unlink(temp_file_path)
                logger.info(
                    f"Successfully processed MP4 with pydub: {len(audio_array)} samples"
                )
                return audio_array
            except Exception as e:
                logger.warning(f"Failed to load MP4 with pydub: {e}")
                if "temp_file_path" in locals() and os.path.exists(temp_file_path):
                    os.unlink(temp_file_path)

                # Fallback: try to extract raw audio from MP4 fragment
                try:
                    # Try multiple approaches to extract audio
                    audio_array = None

                    # Method 1: Skip MP4 headers and try to extract raw audio
                    if len(audio_bytes) > 200:
                        # Skip first 200 bytes (MP4 headers)
                        audio_data = audio_bytes[200:]
                        # Try to interpret as 16-bit PCM
                        if len(audio_data) % 2 == 0:
                            audio_array = np.frombuffer(audio_data, dtype=np.int16)
                            audio_array = audio_array.astype(np.float32) / 32768.0
                            logger.info(
                                f"Extracted raw audio from MP4 fragment (method 1): {len(audio_array)} samples"
                            )
                            return audio_array

                    # Method 2: Try to find audio data after 'mdat' box
                    if audio_array is None and len(audio_bytes) > 100:
                        # Look for 'mdat' box which contains the actual audio data
                        mdat_pos = audio_bytes.find(b"mdat")
                        if mdat_pos != -1 and mdat_pos + 8 < len(audio_bytes):
                            # Skip 'mdat' + 4-byte size + actual audio data starts
                            audio_data = audio_bytes[mdat_pos + 8 :]
                            if len(audio_data) % 2 == 0:
                                audio_array = np.frombuffer(audio_data, dtype=np.int16)
                                audio_array = audio_array.astype(np.float32) / 32768.0
                                logger.info(
                                    f"Extracted raw audio from MP4 fragment (method 2): {len(audio_array)} samples"
                                )
                                return audio_array

                    # Method 3: Try to interpret entire buffer as raw PCM
                    if audio_array is None and len(audio_bytes) % 2 == 0:
                        audio_array = np.frombuffer(audio_bytes, dtype=np.int16)
                        audio_array = audio_array.astype(np.float32) / 32768.0
                        logger.info(
                            f"Extracted raw audio from MP4 fragment (method 3): {len(audio_array)} samples"
                        )
                        return audio_array

                except Exception as e2:
                    logger.warning(f"Failed to extract raw audio from MP4: {e2}")

        elif format_type.startswith("audio/webm"):
            # Try pydub for WebM
            try:
                from pydub import AudioSegment

                with tempfile.NamedTemporaryFile(
                    suffix=".webm", delete=False
                ) as temp_file:
                    temp_file.write(audio_bytes)
                    temp_file_path = temp_file.name

                # Load with pydub
                audio = AudioSegment.from_file(temp_file_path, format="webm")
                audio = audio.set_channels(1).set_frame_rate(16000)
                audio_array = (
                    np.array(audio.get_array_of_samples(), dtype=np.float32) / 32768.0
                )
                os.unlink(temp_file_path)
                logger.info(
                    f"Successfully processed WebM with pydub: {len(audio_array)} samples"
                )
                return audio_array
            except Exception as e:
                logger.warning(f"Failed to load WebM with pydub: {e}")
                if "temp_file_path" in locals() and os.path.exists(temp_file_path):
                    os.unlink(temp_file_path)

        # Try librosa as fallback for any format
        try:
            with tempfile.NamedTemporaryFile(
                suffix=".audio", delete=False
            ) as temp_file:
                temp_file.write(audio_bytes)
                temp_file_path = temp_file.name

            audio_array, sample_rate = librosa.load(temp_file_path, sr=16000, mono=True)
            os.unlink(temp_file_path)
            logger.info(
                f"Successfully processed with librosa: {len(audio_array)} samples"
            )
            return audio_array
        except Exception as e:
            logger.warning(f"Failed to load with librosa: {e}")
            if "temp_file_path" in locals() and os.path.exists(temp_file_path):
                os.unlink(temp_file_path)

        # Last resort: try to interpret as raw PCM
        try:
            if len(audio_bytes) % 2 == 0:
                audio_array = np.frombuffer(audio_bytes, dtype=np.int16)
                audio_array = audio_array.astype(np.float32) / 32768.0
                logger.info(f"Processed as raw PCM: {len(audio_array)} samples")
                return audio_array
        except Exception as e:
            logger.warning(f"Failed to process as raw PCM: {e}")

    except Exception as e:
        logger.error(f"Error processing audio chunk: {e}")
        return None

    return None


@app.websocket("/ws/transcribe")
async def websocket_endpoint(websocket: WebSocket):
    """WebSocket endpoint for live transcription"""
    try:
        await websocket.accept()
        logger.info(f"WebSocket connection accepted from {websocket.client}")
    except WebSocketException as e:
        logger.error(f"WebSocket exception: {e}")
        return
    except Exception as e:
        logger.error(f"Failed to accept WebSocket connection: {e}")
        return
    client_id = None
    whisper_model = None
    audio_buffer = []
    last_transcription_time = 0
    transcription_history = []

    try:
        # Wait for client initialization
        init_message = await websocket.receive_text()
        init_data = json.loads(init_message)

        if init_data.get("type") != "init":
            await websocket.send_json(
                {"type": "error", "message": "Expected init message"}
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
            whisper_model = StreamingWhisperASR(
                model_size=actual_model,
                language=language,
                device="cpu",
                compute_type="int8",
            )

            logger.info(f"Created Whisper model {actual_model} for client {client_id}")
        except Exception as e:
            logger.error(f"Failed to create Whisper model for {client_id}: {e}")
            await websocket.send_json(
                {
                    "type": "error",
                    "message": f"Failed to initialize model: {str(e)}",
                }
            )
            return

        # Send ready message
        await websocket.send_json(
            {
                "type": "ready",
                "client_id": client_id,
                "model_size": model_size,
                "language": language,
                "translate": translate,
            }
        )

        # Handle audio data and transcription
        while True:
            try:
                message = await websocket.receive_text()
                data = json.loads(message)

                if data["type"] == "audio_data":
                    # Process audio chunk
                    audio_data = data["audio"]
                    audio_format = data.get("format", "audio/wav")
                    logger.info(
                        f"Client {client_id} sent audio data of length {len(audio_data)}, format: {audio_format}"
                    )

                    try:
                        # Decode base64 audio data
                        audio_bytes = base64.b64decode(audio_data)
                        logger.info(
                            f"Client {client_id} decoded audio bytes: {len(audio_bytes)} bytes"
                        )

                        # Process audio based on format
                        audio_array = process_audio_chunk(audio_bytes, audio_format)

                        if audio_array is not None and len(audio_array) > 0:
                            logger.info(
                                f"Client {client_id} converted to numpy array: {len(audio_array)} samples"
                            )

                            # Check if audio has meaningful content (lowered threshold)
                            if np.abs(audio_array).max() < 0.0001:
                                logger.info(f"Client {client_id} skipping silence")
                                continue

                            # Add to buffer
                            audio_buffer.extend(audio_array)
                            logger.info(
                                f"Client {client_id} buffer size: {len(audio_buffer)} samples"
                            )

                            # Process buffer when it's large enough (about 1 second of audio)
                            if len(audio_buffer) > 16000:  # 16kHz * 1 second
                                import time

                                current_time = time.time()

                                # Throttle transcription to avoid overwhelming the model
                                if (
                                    current_time - last_transcription_time > 0.3
                                ):  # Max ~3 transcriptions per second
                                    # Reset history if there's been a long pause (helps break repetition)
                                    if current_time - last_transcription_time > 5.0:
                                        transcription_history.clear()
                                        logger.info(
                                            f"Client {client_id} reset transcription history after pause"
                                        )
                                    logger.info(
                                        f"Client {client_id} processing audio buffer of {len(audio_buffer)} samples"
                                    )

                                    # Get the audio to process
                                    audio_to_process = np.array(audio_buffer[:16000])
                                    audio_buffer = audio_buffer[
                                        16000:
                                    ]  # Keep remaining audio

                                    logger.info(
                                        f"Audio duration: {len(audio_to_process) / 16000:.2f} seconds"
                                    )

                                    # Run transcription in a thread to avoid blocking
                                    def transcribe_audio():
                                        try:
                                            logger.info(
                                                f"Client {client_id} starting transcription..."
                                            )
                                            result = whisper_model.transcribe_streaming(
                                                audio_to_process
                                            )
                                            logger.info(
                                                f"Client {client_id} transcription result: '{result}'"
                                            )
                                            return result
                                        except Exception as e:
                                            logger.error(f"Transcription error: {e}")
                                            return None

                                    # Run transcription in thread pool
                                    import asyncio

                                    loop = asyncio.get_event_loop()
                                    transcription_text = await loop.run_in_executor(
                                        None, transcribe_audio
                                    )

                                    if (
                                        transcription_text
                                        and transcription_text.strip()
                                    ):
                                        # Check if this is new content (not repetitive)
                                        is_repetitive = False

                                        # Check against recent history
                                        for prev_text in transcription_history[
                                            -3:
                                        ]:  # Check last 3
                                            if (
                                                transcription_text.strip()
                                                == prev_text.strip()
                                            ):
                                                is_repetitive = True
                                                break

                                            # Check for high similarity
                                            if (
                                                whisper_model._similarity(
                                                    transcription_text, prev_text
                                                )
                                                > 0.7
                                            ):
                                                is_repetitive = True
                                                break

                                        if (
                                            not is_repetitive
                                            and transcription_text.strip()
                                        ):
                                            transcription_history.append(
                                                transcription_text
                                            )
                                            if (
                                                len(transcription_history) > 10
                                            ):  # Keep last 10 transcriptions
                                                transcription_history.pop(0)

                                            response = {
                                                "type": "transcription",
                                                "text": transcription_text,
                                                "timestamp": current_time,
                                                "final": True,
                                            }

                                            logger.info(
                                                f"Client {client_id} sending transcription: {transcription_text}"
                                            )
                                            await websocket.send_json(response)
                                        else:
                                            logger.info(
                                                f"Client {client_id} skipping repetitive text: {transcription_text[:50]}..."
                                            )
                                    else:
                                        logger.info(
                                            f"Client {client_id} no transcription text generated"
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
                        await websocket.send_json(error_response)

                elif data["type"] == "stop":
                    logger.info(f"Client {client_id} stopping transcription")
                    break

            except json.JSONDecodeError as e:
                logger.error(f"Invalid JSON from client {client_id}: {e}")
                await websocket.send_json(
                    {"type": "error", "message": "Invalid JSON format"}
                )
            except Exception as e:
                logger.error(f"Error processing message from {client_id}: {e}")
                await websocket.send_json({"type": "error", "message": str(e)})
                break

    except WebSocketDisconnect:
        logger.info(f"Client {client_id} disconnected")
    except json.JSONDecodeError as e:
        logger.error(f"Invalid JSON in init message: {e}")
        await websocket.send_json(
            {"type": "error", "message": "Invalid JSON in init message"}
        )
    except Exception as e:
        logger.error(f"WebSocket error: {e}")
        await websocket.send_json({"type": "error", "message": str(e)})


@app.get("/")
async def root():
    """Root endpoint for health check"""
    return {
        "message": "Live Transcription Server is running",
        "websocket_endpoint": "/ws/transcribe",
        "status": "ready",
    }


@app.get("/test")
async def test():
    """Test endpoint"""
    return {"status": "ok", "message": "Server is working"}


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "service": "live-transcription"}


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=9090)
