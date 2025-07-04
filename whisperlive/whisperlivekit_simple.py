#!/usr/bin/env python3
"""
Robust WhisperLiveKit-based live transcription server
Enhanced with better error handling, connection management, and performance monitoring
"""

import asyncio
import logging
import time
import json
import base64
from contextlib import asynccontextmanager
from typing import Dict, Optional, Set
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

# Connection management
active_connections: Dict[str, Dict] = {}
MAX_CONCURRENT_CONNECTIONS = 5
CONNECTION_TIMEOUT = 300  # 5 minutes

# Performance monitoring
performance_stats = {
    "total_connections": 0,
    "active_connections": 0,
    "total_audio_processed": 0,
    "total_transcriptions": 0,
    "errors": 0,
    "ffmpeg_restarts": 0,
    "start_time": time.time(),
}


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Initialize and cleanup the transcription engine"""
    global transcription_engine

    try:
        logger.info("🚀 Initializing WhisperLiveKit transcription engine...")

        # Initialize with more robust settings for live transcription
        transcription_engine = TranscriptionEngine(
            model="tiny.en",  # Fastest model for English
            diarization=False,  # Disable for simplicity and performance
            lan="en",  # English language
            backend="faster-whisper",  # Fastest backend
            confidence_validation=False,  # Disable for simplicity
            min_chunk_size=0.5,  # Smaller chunks for lower latency
            buffer_trimming="segment",  # Trim on segments
            no_vad=False,  # Enable VAD for better performance
            # Additional robust settings
            max_audio_length=30.0,  # Limit audio length to prevent memory issues
            chunk_length=1.0,  # Process in 1-second chunks
            stride_length=0.1,  # Small stride for overlap
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


app = FastAPI(title="WhisperLiveKit Robust Server", lifespan=lifespan)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


def cleanup_connection(client_id: str):
    """Clean up connection resources"""
    if client_id in active_connections:
        conn_info = active_connections[client_id]
        if conn_info.get("audio_processor"):
            try:
                asyncio.create_task(conn_info["audio_processor"].cleanup())
            except Exception as e:
                logger.warning(
                    f"Error cleaning up audio processor for {client_id}: {e}"
                )

        if conn_info.get("results_task") and not conn_info["results_task"].done():
            conn_info["results_task"].cancel()

        del active_connections[client_id]
        performance_stats["active_connections"] = len(active_connections)
        logger.info(f"🧹 Cleaned up connection for {client_id}")


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
                        performance_stats["total_transcriptions"] += 1

            # Send response to client
            try:
                await websocket.send_json(response)
            except WebSocketDisconnect:
                logger.info(
                    f"WebSocket disconnected for {client_id} during results sending"
                )
                break
            except Exception as e:
                logger.error(f"Error sending response to {client_id}: {e}")
                break

        # Send ready to stop signal
        logger.info(f"🏁 Results generator finished for {client_id}")
        try:
            await websocket.send_json({"type": "ready_to_stop", "client_id": client_id})
        except WebSocketDisconnect:
            logger.info(f"WebSocket disconnected for {client_id} before ready_to_stop")
        except Exception as e:
            logger.warning(f"Error sending ready_to_stop to {client_id}: {e}")

    except WebSocketDisconnect:
        logger.info(f"WebSocket disconnected for {client_id} during results handling")
    except Exception as e:
        logger.error(f"❌ Error in results handler for {client_id}: {e}")
        performance_stats["errors"] += 1
    finally:
        cleanup_connection(client_id)


async def monitor_connection_health(client_id: str):
    """Monitor connection health and cleanup if needed"""
    try:
        await asyncio.sleep(CONNECTION_TIMEOUT)
        if client_id in active_connections:
            logger.warning(f"Connection timeout for {client_id}, cleaning up")
            cleanup_connection(client_id)
    except asyncio.CancelledError:
        pass
    except Exception as e:
        logger.error(f"Error in connection health monitor for {client_id}: {e}")


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
    """WebSocket endpoint for live transcription with enhanced robustness"""
    global transcription_engine

    # Check connection limit
    if len(active_connections) >= MAX_CONCURRENT_CONNECTIONS:
        logger.warning(
            f"Connection limit reached ({MAX_CONCURRENT_CONNECTIONS}), rejecting new connection"
        )
        await websocket.close(code=1008, reason="Too many connections")
        return

    try:
        await websocket.accept()
        logger.info(f"WebSocket connection accepted from {websocket.client}")
    except Exception as e:
        logger.error(f"Failed to accept WebSocket connection: {e}")
        return

    client_id = None
    audio_processor = None
    results_task = None
    health_monitor_task = None

    try:
        # Wait for client initialization with timeout
        try:
            init_message = await asyncio.wait_for(
                websocket.receive_text(), timeout=10.0
            )
        except asyncio.TimeoutError:
            logger.warning("Client initialization timeout")
            await websocket.close(code=1000, reason="Initialization timeout")
            return

        try:
            init_data = json.loads(init_message)
        except json.JSONDecodeError as e:
            logger.error(f"Invalid JSON in init message: {e}")
            await websocket.send_json({"type": "error", "message": "Invalid JSON"})
            return

        if init_data.get("type") != "init":
            await websocket.send_json(
                {"type": "error", "message": "Expected init message"}
            )
            return

        client_id = init_data.get(
            "client_id", f"client_{int(asyncio.get_event_loop().time() * 1000)}"
        )

        # Check if client_id already exists
        if client_id in active_connections:
            logger.warning(f"Client {client_id} already connected, generating new ID")
            client_id = f"{client_id}_{int(time.time())}"

        logger.info(f"New client {client_id} connecting")

        # Check if transcription engine is available
        if transcription_engine is None:
            await websocket.send_json(
                {"type": "error", "message": "Transcription engine not initialized"}
            )
            return

        # Create AudioProcessor for this connection
        try:
            audio_processor = AudioProcessor(transcription_engine=transcription_engine)
            results_generator = await audio_processor.create_tasks()
        except Exception as e:
            logger.error(f"Failed to create audio processor for {client_id}: {e}")
            await websocket.send_json(
                {"type": "error", "message": "Failed to initialize audio processor"}
            )
            return

        # Start results handler
        results_task = asyncio.create_task(
            handle_websocket_results(websocket, results_generator, client_id)
        )

        # Start health monitor
        health_monitor_task = asyncio.create_task(monitor_connection_health(client_id))

        # Register connection
        active_connections[client_id] = {
            "websocket": websocket,
            "audio_processor": audio_processor,
            "results_task": results_task,
            "health_monitor_task": health_monitor_task,
            "start_time": time.time(),
            "audio_chunks_processed": 0,
            "last_activity": time.time(),
        }
        performance_stats["total_connections"] += 1
        performance_stats["active_connections"] = len(active_connections)

        # Send initialization confirmation
        await websocket.send_json(
            {
                "type": "init_success",
                "client_id": client_id,
            }
        )

        logger.info(f"✅ Client {client_id} initialized successfully")

        # Handle audio data with enhanced error handling
        while True:
            try:
                # Try to receive as text first (JSON message)
                try:
                    message = await asyncio.wait_for(
                        websocket.receive_text(), timeout=30.0
                    )
                    logger.debug(
                        f"Received text message from {client_id}: {message[:100]}..."
                    )

                    try:
                        data = json.loads(message)
                    except json.JSONDecodeError as e:
                        logger.warning(f"Invalid JSON from {client_id}: {e}")
                        continue

                    if data.get("type") == "audio_data":
                        # Decode base64 audio data
                        try:
                            audio_bytes = base64.b64decode(data["audio"])
                            logger.info(
                                f"🎵 Processing {len(audio_bytes)} bytes of audio from {client_id}"
                            )

                            # Update connection stats
                            active_connections[client_id]["audio_chunks_processed"] += 1
                            active_connections[client_id]["last_activity"] = time.time()
                            performance_stats["total_audio_processed"] += len(
                                audio_bytes
                            )

                            await audio_processor.process_audio(audio_bytes)

                        except base64.binascii.Error as e:
                            logger.error(
                                f"Invalid base64 audio data from {client_id}: {e}"
                            )
                            continue
                        except Exception as e:
                            logger.error(
                                f"Error processing audio from {client_id}: {e}"
                            )
                            performance_stats["errors"] += 1
                            continue

                    elif data.get("type") == "stop":
                        logger.info(f"Client {client_id} stopping transcription")
                        break
                    elif data.get("type") == "ping":
                        # Handle ping for connection health
                        await websocket.send_json(
                            {"type": "pong", "client_id": client_id}
                        )
                        active_connections[client_id]["last_activity"] = time.time()
                    else:
                        logger.warning(
                            f"Unknown message type from {client_id}: {data.get('type')}"
                        )

                except asyncio.TimeoutError:
                    # Send ping to check connection health
                    try:
                        await websocket.send_json(
                            {"type": "ping", "client_id": client_id}
                        )
                    except Exception:
                        logger.warning(f"Connection timeout for {client_id}")
                        break
                except json.JSONDecodeError:
                    # If not JSON, try as raw bytes
                    logger.debug(f"Received raw bytes from {client_id}")
                    try:
                        await audio_processor.process_audio(message.encode())
                        active_connections[client_id]["last_activity"] = time.time()
                    except Exception as e:
                        logger.error(
                            f"Error processing raw audio from {client_id}: {e}"
                        )
                        performance_stats["errors"] += 1

            except WebSocketDisconnect:
                logger.info(f"Client {client_id} disconnected")
                break
            except Exception as e:
                logger.error(f"Error processing message from {client_id}: {e}")
                performance_stats["errors"] += 1
                import traceback

                logger.error(f"Full traceback: {traceback.format_exc()}")
                break

    except Exception as e:
        logger.error(f"WebSocket error for {client_id}: {e}")
        performance_stats["errors"] += 1

    finally:
        # Cleanup
        logger.info(f"🧹 Cleaning up connection for {client_id}")

        if health_monitor_task and not health_monitor_task.done():
            health_monitor_task.cancel()
            try:
                await health_monitor_task
            except asyncio.CancelledError:
                pass

        if results_task and not results_task.done():
            results_task.cancel()
            try:
                await results_task
            except asyncio.CancelledError:
                pass

        if audio_processor:
            try:
                await audio_processor.cleanup()
            except Exception as e:
                logger.warning(
                    f"Error cleaning up audio processor for {client_id}: {e}"
                )

        cleanup_connection(client_id)


@app.get("/")
async def root():
    """Root endpoint with web interface"""
    return HTMLResponse(get_web_interface_html())


@app.get("/health")
async def health_check():
    """Enhanced health check endpoint with performance stats"""
    uptime = time.time() - performance_stats["start_time"]

    return {
        "status": "healthy" if transcription_engine else "initializing",
        "service": "whisperlivekit-robust",
        "uptime_seconds": int(uptime),
        "performance": {
            "total_connections": performance_stats["total_connections"],
            "active_connections": performance_stats["active_connections"],
            "total_audio_processed_bytes": performance_stats["total_audio_processed"],
            "total_transcriptions": performance_stats["total_transcriptions"],
            "errors": performance_stats["errors"],
            "ffmpeg_restarts": performance_stats["ffmpeg_restarts"],
            "connections": [
                {
                    "client_id": client_id,
                    "uptime": int(time.time() - conn_info["start_time"]),
                    "audio_chunks_processed": conn_info["audio_chunks_processed"],
                    "last_activity": int(conn_info["last_activity"]),
                }
                for client_id, conn_info in active_connections.items()
            ],
        },
    }


@app.get("/stats")
async def get_stats():
    """Get detailed performance statistics"""
    return performance_stats


if __name__ == "__main__":
    import uvicorn

    logger.info("🚀 Starting WhisperLiveKit Robust Server...")
    uvicorn.run(app, host="0.0.0.0", port=9090)
