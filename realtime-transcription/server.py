import asyncio
import json
import logging
from contextlib import asynccontextmanager
from typing import Dict, Optional

from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from whisperlivekit import AudioProcessor, TranscriptionEngine

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("ScriberrRealtime")

# Global cache for transcription engines to avoid reloading models
# Key: "{model_size}_{device}"
engine_cache: Dict[str, TranscriptionEngine] = {}

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    logger.info("Starting Scriberr Real-time Transcription Server")
    yield
    # Shutdown
    logger.info("Shutting down server")
    engine_cache.clear()

app = FastAPI(lifespan=lifespan)

def get_or_create_engine(model_size: str, device: str) -> TranscriptionEngine:
    """
    Get an existing engine from cache or create a new one.
    """
    cache_key = f"{model_size}_{device}"
    
    if cache_key in engine_cache:
        logger.info(f"Using cached engine for {cache_key}")
        return engine_cache[cache_key]
    
    logger.info(f"Initializing new engine for {cache_key}")
    # Note: WhisperLiveKit's TranscriptionEngine handles device selection internally based on availability
    # but we can pass arguments if needed. For now, we trust its default or explicit args if supported.
    # The current version of WLK might not expose 'device' directly in __init__ in a simple way 
    # without looking deep, but usually it uses 'cuda' if available.
    # We will pass the model size.
    
    # TODO: If WLK supports explicit device selection in Init, add it here.
    # Based on research, it uses faster-whisper or openai-whisper which auto-detects.
    
    engine = TranscriptionEngine(model=model_size, diarization=False)
    engine_cache[cache_key] = engine
    return engine

async def handle_websocket_results(websocket: WebSocket, results_generator):
    """
    Iterate over results from the audio processor and send them to the client.
    """
    try:
        async for response in results_generator:
            # response is typically a dict with 'text', 'start', 'end', 'is_final' etc.
            # WLK sends partial results with is_final=False (gray) and final with is_final=True (black)
            logger.info(f"Sending to client: {response}")
            await websocket.send_json(response)
    except Exception as e:
        logger.error(f"Error sending results: {e}")

@app.websocket("/asr")
async def websocket_endpoint(websocket: WebSocket):
    await websocket.accept()
    logger.info("New WebSocket connection accepted")
    
    audio_processor: Optional[AudioProcessor] = None
    processing_task: Optional[asyncio.Task] = None
    
    try:
        # 1. Wait for configuration message
        # Client should send: {"model": "base", "device": "cpu"}
        config_data = await websocket.receive_json()
        logger.info(f"Received config: {config_data}")
        
        model_size = config_data.get("model", "base")
        device = config_data.get("device", "cpu") # 'cpu' or 'cuda'
        
        # Initialize engine
        engine = get_or_create_engine(model_size, device)
        
        # Create AudioProcessor for this session
        audio_processor = AudioProcessor(transcription_engine=engine)
        
        # Create tasks for processing
        results_generator = await audio_processor.create_tasks()
        
        # Start sending results in background
        results_task = asyncio.create_task(handle_websocket_results(websocket, results_generator))
        
        # 2. Loop to receive audio data
        while True:
            message = await websocket.receive_bytes()
            # Feed audio to processor
            await audio_processor.process_audio(message)
            
    except WebSocketDisconnect:
        logger.info("WebSocket disconnected")
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        try:
            await websocket.close(code=1011, reason=str(e))
        except:
            pass
    finally:
        # Cleanup
        if audio_processor:
            # There isn't an explicit 'stop' or 'close' on AudioProcessor in the snippet,
            # but we should cancel the results task if it's running.
            if results_task and not results_task.done():
                results_task.cancel()
