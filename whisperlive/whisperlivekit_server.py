#!/usr/bin/env python3
"""
WhisperLiveKit-based streaming transcription server
Based on https://github.com/QuentinFuxa/WhisperLiveKit
"""

import asyncio
import base64
import json
import logging
import time
from contextlib import asynccontextmanager
from typing import Optional

from fastapi import FastAPI, WebSocket, WebSocketDisconnect, WebSocketException
from fastapi.middleware.cors import CORSMiddleware
from whisperlivekit import TranscriptionEngine, AudioProcessor

# Configure logging
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

# Global transcription engine
transcription_engine = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Initialize and cleanup the transcription engine"""
    global transcription_engine
    try:
        logger.info("🚀 Initializing WhisperLiveKit transcription engine...")

        # Initialize the transcription engine with recommended settings
        transcription_engine = TranscriptionEngine(
            model="tiny",  # Use tiny model for fastest processing
            diarization=False,  # Disable diarization for simplicity
            lan="en",  # English language
            backend="faster-whisper",  # Use faster-whisper backend
            confidence_validation=True,  # Enable confidence validation for faster results
            min_chunk_size=0.5,  # Shorter chunks for more responsive transcription
            buffer_trimming="segment",  # Trim on segments for better performance
            no_vad=False,  # Enable Voice Activity Detection
        )

        logger.info("✅ Transcription engine initialized successfully")

        # Test the engine with a simple check
        try:
            logger.info("🧪 Testing transcription engine...")
            test_processor = AudioProcessor(transcription_engine=transcription_engine)
            test_results = await test_processor.create_tasks()
            test_processor.cleanup()
            logger.info("🧪 Transcription engine test passed")
        except Exception as e:
            logger.error(f"❌ Transcription engine test failed: {e}")
            raise

    except Exception as e:
        logger.error(f"❌ Failed to initialize transcription engine: {e}")
        raise
    yield
    # Cleanup
    if transcription_engine:
        logger.info("🧹 Cleaning up transcription engine...")
        transcription_engine.cleanup()
        logger.info("✅ Transcription engine cleaned up")


app = FastAPI(title="WhisperLiveKit Transcription Server", lifespan=lifespan)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # In production, specify your frontend domain
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


async def handle_websocket_results(
    websocket: WebSocket, results_generator, client_id: str
):
    """Handle transcription results and send them to the client"""
    try:
        logger.info(f"🔍 Starting to listen for results from {client_id}")
        response_count = 0

        async for response in results_generator:
            response_count += 1
            logger.info(f"📥 Received response #{response_count} for {client_id}")

            # Add client ID to response for tracking
            response["client_id"] = client_id
            response["timestamp"] = time.time()

            # Log the response type
            response_type = response.get("type", "unknown")
            logger.info(f"Client {client_id} received result type: {response_type}")

            # Handle different response types
            if response_type == "transcription":
                # Extract transcription text
                lines = response.get("lines", [])
                if lines:
                    # Get the latest transcription line
                    latest_line = lines[-1]
                    text = latest_line.get("text", "").strip()
                    if text:
                        logger.info(f"🎤 LIVE TRANSCRIPTION [{client_id}]: '{text}'")

                        # Send transcription to frontend in expected format
                        transcription_response = {
                            "type": "transcription",
                            "text": text,
                            "timestamp": time.time(),
                            "final": True,
                            "client_id": client_id,
                        }
                        await websocket.send_json(transcription_response)
                        logger.info(f"📤 Sent transcription to frontend: '{text}'")

            elif response_type == "buffer_transcription":
                # Handle buffer transcription (partial results)
                buffer_text = response.get("buffer_transcription", "").strip()
                if buffer_text:
                    logger.info(
                        f"📝 BUFFER TRANSCRIPTION [{client_id}]: '{buffer_text}'"
                    )

                    # Send as partial transcription
                    transcription_response = {
                        "type": "transcription",
                        "text": buffer_text,
                        "timestamp": time.time(),
                        "final": False,
                        "client_id": client_id,
                    }
                    await websocket.send_json(transcription_response)

            elif response.get("status") == "no_audio_detected":
                logger.info(f"🔇 No audio detected for client {client_id}")
                # Send this to frontend so it knows no audio was detected
                await websocket.send_json(
                    {
                        "type": "status",
                        "status": "no_audio_detected",
                        "client_id": client_id,
                        "timestamp": time.time(),
                    }
                )

            else:
                # Send other response types as-is, but also send a transcription version for WhisperLiveKit format
                logger.info(f"📨 Other response for client {client_id}: {response}")

                # If this is a WhisperLiveKit active_transcription response, also send it as a transcription
                if response.get("status") == "active_transcription":
                    lines = response.get("lines", [])
                    if lines:
                        latest_line = lines[-1]
                        text = latest_line.get("text", "").strip()
                        if text:
                            # Send both the original response and a simplified transcription
                            await websocket.send_json(response)

                            # Also send a simplified transcription format
                            transcription_response = {
                                "type": "transcription",
                                "text": text,
                                "timestamp": time.time(),
                                "final": True,
                                "client_id": client_id,
                            }
                            await websocket.send_json(transcription_response)
                        else:
                            await websocket.send_json(response)
                    else:
                        await websocket.send_json(response)
                else:
                    await websocket.send_json(response)

        # Send ready to stop signal
        await websocket.send_json(
            {"type": "ready_to_stop", "client_id": client_id, "timestamp": time.time()}
        )
        logger.info(
            f"🏁 Finished processing results for {client_id}, total responses: {response_count}"
        )

    except Exception as e:
        logger.error(f"❌ Error handling results for client {client_id}: {e}")
        import traceback

        logger.error(f"🔍 Full traceback for {client_id}: {traceback.format_exc()}")


@app.websocket("/ws/transcribe")
async def websocket_endpoint(websocket: WebSocket):
    """WebSocket endpoint for live transcription using WhisperLiveKit"""
    global transcription_engine

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
    audio_processor = None
    results_task = None

    try:
        # Wait for client initialization
        init_message = await websocket.receive_text()
        init_data = json.loads(init_message)

        if init_data.get("type") != "init":
            await websocket.send_json(
                {"type": "error", "message": "Expected init message"}
            )
            return

        client_id = init_data.get("client_id", f"client_{int(time.time() * 1000)}")
        model_size = init_data.get("model_size", "tiny")
        language = init_data.get("language", "en")

        logger.info(f"New client {client_id} connecting with model {model_size}")

        # Check if transcription engine is available
        if transcription_engine is None:
            await websocket.send_json(
                {"type": "error", "message": "Transcription engine not initialized"}
            )
            return

        # Create a new AudioProcessor for this connection (following official pattern)
        try:
            logger.info(f"🎯 Creating AudioProcessor for {client_id}")
            audio_processor = AudioProcessor(transcription_engine=transcription_engine)
            results_generator = await audio_processor.create_tasks()

            logger.info(
                f"🎯 Created AudioProcessor and results_generator for {client_id}"
            )

            # Start handling results
            results_task = asyncio.create_task(
                handle_websocket_results(websocket, results_generator, client_id)
            )
            logger.info(f"🚀 Started results task for {client_id}")

            # Send initialization confirmation
            await websocket.send_json(
                {
                    "type": "init_success",
                    "client_id": client_id,
                    "model_size": model_size,
                    "language": language,
                }
            )

        except Exception as e:
            logger.error(
                f"❌ Failed to initialize audio processor for {client_id}: {e}"
            )
            await websocket.send_json(
                {
                    "type": "error",
                    "message": f"Failed to initialize audio processor: {str(e)}",
                }
            )
            return

        # Handle audio data
        while True:
            try:
                message = await websocket.receive_text()
                data = json.loads(message)

                if data["type"] == "audio_data":
                    # Process audio chunk
                    audio_data = data["audio"]
                    audio_format = data.get("format", "audio/webm")

                    logger.info(
                        f"Client {client_id} sent audio data of length {len(audio_data)}, format: {audio_format}"
                    )

                    try:
                        # Decode base64 audio data
                        audio_bytes = base64.b64decode(audio_data)
                        logger.info(
                            f"Client {client_id} decoded audio bytes: {len(audio_bytes)} bytes"
                        )

                        # Process audio with WhisperLiveKit (following official pattern)
                        await audio_processor.process_audio(audio_bytes)
                        logger.info(
                            f"✅ Audio processed successfully for client {client_id}"
                        )

                    except Exception as e:
                        logger.error(f"❌ Error processing audio for {client_id}: {e}")
                        # Send error but don't break the connection
                        error_response = {
                            "type": "error",
                            "message": f"Audio processing error: {str(e)}",
                            "client_id": client_id,
                        }
                        await websocket.send_json(error_response)

                elif data["type"] == "stop":
                    logger.info(f"Client {client_id} stopping transcription")
                    break

            except json.JSONDecodeError as e:
                logger.error(f"Invalid JSON from client {client_id}: {e}")
                await websocket.send_json(
                    {
                        "type": "error",
                        "message": "Invalid JSON format",
                        "client_id": client_id,
                    }
                )
            except Exception as e:
                logger.error(f"Error processing message from {client_id}: {e}")
                await websocket.send_json(
                    {"type": "error", "message": str(e), "client_id": client_id}
                )
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
    finally:
        # Cleanup
        if results_task:
            results_task.cancel()
        if audio_processor:
            audio_processor.cleanup()


@app.get("/")
async def root():
    """Root endpoint for health check"""
    return {
        "message": "WhisperLiveKit Transcription Server is running",
        "websocket_endpoint": "/ws/transcribe",
        "status": "ready",
        "version": "whisperlivekit-based",
    }


@app.get("/test")
async def test():
    """Test endpoint"""
    return {"status": "ok", "message": "WhisperLiveKit server is working"}


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "service": "whisperlivekit-transcription"}


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=9090)
