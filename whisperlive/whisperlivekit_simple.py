#!/usr/bin/env python3
"""
Simple WhisperLiveKit-based live transcription server
Based on the official WhisperLiveKit implementation
"""

import asyncio
import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import HTMLResponse
from whisperlivekit import TranscriptionEngine, AudioProcessor, get_web_interface_html

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

        # Initialize with optimal settings for live transcription
        transcription_engine = TranscriptionEngine(
            model="tiny.en",  # Fastest model for English
            diarization=False,  # Disable for simplicity
            lan="en",  # English language
            backend="faster-whisper",  # Fastest backend
            confidence_validation=False,  # Disable for simplicity
            min_chunk_size=1.0,  # 1 second chunks
            buffer_trimming="segment",  # Trim on segments
            no_vad=False,  # Enable VAD for better performance
        )

        logger.info("✅ Transcription engine initialized successfully")

    except Exception as e:
        logger.error(f"❌ Failed to initialize transcription engine: {e}")
        raise

    yield

    # Cleanup
    if transcription_engine:
        logger.info("🧹 Cleaning up transcription engine...")
        transcription_engine.cleanup()
        logger.info("✅ Transcription engine cleaned up")


app = FastAPI(title="WhisperLiveKit Simple Server", lifespan=lifespan)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


async def handle_websocket_results(
    websocket: WebSocket, results_generator, client_id: str
):
    """Consumes results from the audio processor and sends them via WebSocket."""
    try:
        logger.info(f"🔍 Starting results handler for {client_id}")

        async for response in results_generator:
            # Add client ID to response for tracking
            response["client_id"] = client_id

            # Log transcription results
            if response.get("type") == "transcription":
                lines = response.get("lines", [])
                if lines:
                    latest_line = lines[-1]
                    text = latest_line.get("text", "").strip()
                    if text:
                        logger.info(f"🎤 TRANSCRIPTION [{client_id}]: '{text}'")

            # Send response to client
            await websocket.send_json(response)

        # Send ready to stop signal
        logger.info(f"🏁 Results generator finished for {client_id}")
        await websocket.send_json({"type": "ready_to_stop", "client_id": client_id})

    except WebSocketDisconnect:
        logger.info(f"WebSocket disconnected for {client_id} during results handling")
    except Exception as e:
        logger.error(f"❌ Error in results handler for {client_id}: {e}")


@app.websocket("/asr")
async def asr_endpoint(websocket: WebSocket):
    """Official WhisperLiveKit /asr endpoint"""
    global transcription_engine

    audio_processor = AudioProcessor(transcription_engine=transcription_engine)
    await websocket.accept()
    logger.info("WebSocket connection opened.")

    results_generator = await audio_processor.create_tasks()
    websocket_task = asyncio.create_task(
        handle_websocket_results(websocket, results_generator, "asr_client")
    )

    try:
        while True:
            message = await websocket.receive_bytes()
            await audio_processor.process_audio(message)
    except KeyError as e:
        if "bytes" in str(e):
            logger.warning(f"Client has closed the connection.")
        else:
            logger.error(
                f"Unexpected KeyError in websocket_endpoint: {e}", exc_info=True
            )
    except WebSocketDisconnect:
        logger.info("WebSocket disconnected by client during message receiving loop.")
    except Exception as e:
        logger.error(
            f"Unexpected error in websocket_endpoint main loop: {e}", exc_info=True
        )
    finally:
        logger.info("Cleaning up WebSocket endpoint...")
        if not websocket_task.done():
            websocket_task.cancel()
        try:
            await websocket_task
        except asyncio.CancelledError:
            logger.info("WebSocket results handler task was cancelled.")
        except Exception as e:
            logger.warning(f"Exception while awaiting websocket_task completion: {e}")

        await audio_processor.cleanup()
        logger.info("WebSocket endpoint cleaned up successfully.")


@app.websocket("/ws/transcribe")
async def websocket_endpoint(websocket: WebSocket):
    """WebSocket endpoint for live transcription"""
    global transcription_engine

    try:
        await websocket.accept()
        logger.info(f"WebSocket connection accepted from {websocket.client}")
    except Exception as e:
        logger.error(f"Failed to accept WebSocket connection: {e}")
        return

    client_id = None
    audio_processor = None
    results_task = None

    try:
        # Wait for client initialization
        init_message = await websocket.receive_text()
        import json

        init_data = json.loads(init_message)

        if init_data.get("type") != "init":
            await websocket.send_json(
                {"type": "error", "message": "Expected init message"}
            )
            return

        client_id = init_data.get(
            "client_id", f"client_{int(asyncio.get_event_loop().time() * 1000)}"
        )
        logger.info(f"New client {client_id} connecting")

        # Check if transcription engine is available
        if transcription_engine is None:
            await websocket.send_json(
                {"type": "error", "message": "Transcription engine not initialized"}
            )
            return

        # Create AudioProcessor for this connection
        audio_processor = AudioProcessor(transcription_engine=transcription_engine)
        results_generator = await audio_processor.create_tasks()

        # Start results handler
        results_task = asyncio.create_task(
            handle_websocket_results(websocket, results_generator, client_id)
        )

        # Send initialization confirmation
        await websocket.send_json(
            {
                "type": "init_success",
                "client_id": client_id,
            }
        )

        logger.info(f"✅ Client {client_id} initialized successfully")

        # Handle audio data - expect JSON messages with base64 audio
        while True:
            try:
                # Try to receive as text first (JSON message)
                try:
                    message = await websocket.receive_text()
                    logger.debug(
                        f"Received text message from {client_id}: {message[:100]}..."
                    )
                    data = json.loads(message)

                    if data.get("type") == "audio_data":
                        # Decode base64 audio data
                        import base64

                        audio_bytes = base64.b64decode(data["audio"])
                        logger.info(
                            f"🎵 Processing {len(audio_bytes)} bytes of audio from {client_id}"
                        )
                        await audio_processor.process_audio(audio_bytes)
                    elif data.get("type") == "stop":
                        logger.info(f"Client {client_id} stopping transcription")
                        break
                    else:
                        logger.warning(
                            f"Unknown message type from {client_id}: {data.get('type')}"
                        )

                except json.JSONDecodeError:
                    # If not JSON, try as raw bytes
                    logger.debug(f"Received raw bytes from {client_id}")
                    await audio_processor.process_audio(message.encode())

            except WebSocketDisconnect:
                logger.info(f"Client {client_id} disconnected")
                break
            except Exception as e:
                logger.error(f"Error processing message from {client_id}: {e}")
                import traceback

                logger.error(f"Full traceback: {traceback.format_exc()}")
                break

    except Exception as e:
        logger.error(f"WebSocket error for {client_id}: {e}")

    finally:
        # Cleanup
        logger.info(f"🧹 Cleaning up connection for {client_id}")

        if results_task and not results_task.done():
            results_task.cancel()
            try:
                await results_task
            except asyncio.CancelledError:
                pass

        if audio_processor:
            await audio_processor.cleanup()


@app.get("/")
async def root():
    """Root endpoint with web interface"""
    return HTMLResponse(get_web_interface_html())


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {
        "status": "healthy" if transcription_engine else "initializing",
        "service": "whisperlivekit-simple",
    }


if __name__ == "__main__":
    import uvicorn

    logger.info("🚀 Starting WhisperLiveKit Simple Server...")
    uvicorn.run(app, host="0.0.0.0", port=9090)
