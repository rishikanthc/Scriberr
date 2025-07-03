#!/usr/bin/env python3
"""
Simple test script to verify WebSocket server functionality
"""

import asyncio
import json
import websockets
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def test_websocket_connection():
    """Test the WebSocket server connection"""
    uri = "ws://localhost:9090"

    try:
        async with websockets.connect(uri) as websocket:
            logger.info("Connected to WebSocket server")

            # Send init message
            init_message = {
                "type": "init",
                "client_id": "test_client_123",
                "model_size": "small",
                "language": "en",
                "translate": False,
            }

            await websocket.send(json.dumps(init_message))
            logger.info("Sent init message")

            # Wait for ready response
            response = await websocket.recv()
            data = json.loads(response)
            logger.info(f"Received response: {data}")

            if data.get("type") == "ready":
                logger.info("✅ WebSocket server is working correctly!")

                # Send a test audio message
                test_audio = "test_audio_data_base64_encoded"
                audio_message = {"type": "audio_data", "audio": test_audio}

                await websocket.send(json.dumps(audio_message))
                logger.info("Sent test audio data")

                # Wait for transcription response
                transcription_response = await websocket.recv()
                transcription_data = json.loads(transcription_response)
                logger.info(f"Received transcription: {transcription_data}")

                # Send stop message
                stop_message = {"type": "stop"}
                await websocket.send(json.dumps(stop_message))
                logger.info("Sent stop message")

            else:
                logger.error("❌ Unexpected response type")

    except Exception as e:
        logger.error(f"❌ WebSocket test failed: {e}")
        return False

    return True


if __name__ == "__main__":
    asyncio.run(test_websocket_connection())
