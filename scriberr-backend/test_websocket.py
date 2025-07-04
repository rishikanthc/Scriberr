#!/usr/bin/env python3
"""
Simple WebSocket test client for WhisperLiveKit
"""

import asyncio
import json
import websockets
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def test_websocket_connection():
    """Test WebSocket connection to the WhisperLiveKit server"""

    uri = "ws://localhost:9090/ws/transcribe"

    try:
        logger.info(f"Connecting to {uri}...")

        async with websockets.connect(uri) as websocket:
            logger.info("✅ WebSocket connection established")

            # Send initialization message
            init_message = {
                "type": "init",
                "client_id": "test_client_123",
                "model_size": "tiny.en",
                "language": "en",
                "translate": False,
            }

            logger.info(f"Sending init message: {init_message}")
            await websocket.send(json.dumps(init_message))

            # Wait for response
            response = await websocket.recv()
            data = json.loads(response)
            logger.info(f"Received response: {data}")

            if data.get("type") == "init_success":
                logger.info("✅ Initialization successful!")

                # Send a test audio message (empty for testing)
                test_audio_message = {
                    "type": "audio_data",
                    "audio": "",  # Empty for testing
                    "format": "audio/webm",
                }

                logger.info("Sending test audio message...")
                await websocket.send(json.dumps(test_audio_message))

                # Wait a bit for any response
                try:
                    response = await asyncio.wait_for(websocket.recv(), timeout=5.0)
                    data = json.loads(response)
                    logger.info(f"Received audio response: {data}")
                except asyncio.TimeoutError:
                    logger.info("No immediate response to audio message (expected)")

                # Send stop message
                stop_message = {"type": "stop"}
                await websocket.send(json.dumps(stop_message))
                logger.info("✅ Test completed successfully")

            else:
                logger.error(f"❌ Initialization failed: {data}")
                return False

    except Exception as e:
        logger.error(f"❌ WebSocket test failed: {e}")
        return False

    return True


async def test_asr_endpoint():
    """Test the /asr endpoint (WhisperLiveKit standard format)"""

    uri = "ws://localhost:9090/asr"

    try:
        logger.info(f"Testing /asr endpoint at {uri}...")

        async with websockets.connect(uri) as websocket:
            logger.info("✅ /asr WebSocket connection established")

            # Send empty bytes (standard format)
            await websocket.send(b"")
            logger.info("✅ /asr endpoint test completed")

    except Exception as e:
        logger.error(f"❌ /asr endpoint test failed: {e}")
        return False

    return True


async def main():
    """Run all WebSocket tests"""
    logger.info("🧪 Starting WebSocket connection tests...")

    # Test 1: Custom /ws/transcribe endpoint
    test1_result = await test_websocket_connection()

    # Test 2: Standard /asr endpoint
    test2_result = await test_asr_endpoint()

    # Summary
    if test1_result and test2_result:
        logger.info("🎉 All WebSocket tests passed!")
        return True
    else:
        logger.error("❌ Some WebSocket tests failed.")
        return False


if __name__ == "__main__":
    success = asyncio.run(main())
    exit(0 if success else 1)
