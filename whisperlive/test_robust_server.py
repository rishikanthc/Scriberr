#!/usr/bin/env python3
"""
Test script for the robust WhisperLiveKit server
Tests connection management, error handling, and performance monitoring
"""

import asyncio
import websockets
import json
import base64
import time
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def test_connection_management():
    """Test connection limits and management"""
    logger.info("🧪 Testing connection management...")

    # Test multiple connections
    connections = []
    try:
        for i in range(7):  # Try to create more than the limit (5)
            try:
                uri = "ws://localhost:9090/ws/transcribe"
                websocket = await websockets.connect(uri)

                # Send init message
                init_message = {
                    "type": "init",
                    "client_id": f"test_client_{i}",
                    "model_size": "tiny.en",
                    "language": "en",
                    "translate": False,
                }
                await websocket.send(json.dumps(init_message))

                # Wait for response
                response = await websocket.recv()
                data = json.loads(response)

                if data.get("type") == "init_success":
                    logger.info(f"✅ Connection {i} established successfully")
                    connections.append(websocket)
                elif data.get("type") == "error":
                    logger.info(f"❌ Connection {i} rejected: {data.get('message')}")
                    await websocket.close()
                    break

            except Exception as e:
                logger.error(f"❌ Failed to create connection {i}: {e}")
                break

        logger.info(f"📊 Created {len(connections)} connections")

        # Test ping/pong
        if connections:
            logger.info("🧪 Testing ping/pong...")
            await connections[0].send(
                json.dumps({"type": "ping", "client_id": "test_client_0"})
            )
            response = await connections[0].recv()
            data = json.loads(response)
            if data.get("type") == "pong":
                logger.info("✅ Ping/pong working correctly")
            else:
                logger.warning("⚠️ Unexpected ping/pong response")

    finally:
        # Clean up connections
        for websocket in connections:
            try:
                await websocket.close()
            except:
                pass


async def test_health_endpoint():
    """Test the health endpoint"""
    logger.info("🧪 Testing health endpoint...")

    try:
        import aiohttp

        async with aiohttp.ClientSession() as session:
            async with session.get("http://localhost:9090/health") as response:
                if response.status == 200:
                    data = await response.json()
                    logger.info("✅ Health endpoint working")
                    logger.info(f"📊 Server uptime: {data.get('uptime_seconds', 0)}s")
                    logger.info(
                        f"📊 Active connections: {data.get('performance', {}).get('active_connections', 0)}"
                    )
                    logger.info(
                        f"📊 Total connections: {data.get('performance', {}).get('total_connections', 0)}"
                    )
                    logger.info(
                        f"📊 Total audio processed: {data.get('performance', {}).get('total_audio_processed_bytes', 0)} bytes"
                    )
                    logger.info(
                        f"📊 Total transcriptions: {data.get('performance', {}).get('total_transcriptions', 0)}"
                    )
                    logger.info(
                        f"📊 Errors: {data.get('performance', {}).get('errors', 0)}"
                    )
                else:
                    logger.error(f"❌ Health endpoint failed: {response.status}")
    except ImportError:
        logger.warning("⚠️ aiohttp not available, skipping health endpoint test")
    except Exception as e:
        logger.error(f"❌ Health endpoint test failed: {e}")


async def test_stats_endpoint():
    """Test the stats endpoint"""
    logger.info("🧪 Testing stats endpoint...")

    try:
        import aiohttp

        async with aiohttp.ClientSession() as session:
            async with session.get("http://localhost:9090/stats") as response:
                if response.status == 200:
                    data = await response.json()
                    logger.info("✅ Stats endpoint working")
                    logger.info(f"📊 Performance stats: {json.dumps(data, indent=2)}")
                else:
                    logger.error(f"❌ Stats endpoint failed: {response.status}")
    except ImportError:
        logger.warning("⚠️ aiohttp not available, skipping stats endpoint test")
    except Exception as e:
        logger.error(f"❌ Stats endpoint test failed: {e}")


async def test_audio_processing():
    """Test audio processing with error handling"""
    logger.info("🧪 Testing audio processing...")

    try:
        uri = "ws://localhost:9090/ws/transcribe"
        websocket = await websockets.connect(uri)

        # Send init message
        init_message = {
            "type": "init",
            "client_id": "test_audio_client",
            "model_size": "tiny.en",
            "language": "en",
            "translate": False,
        }
        await websocket.send(json.dumps(init_message))

        # Wait for init success
        response = await websocket.recv()
        data = json.loads(response)

        if data.get("type") != "init_success":
            logger.error(f"❌ Init failed: {data}")
            return

        logger.info("✅ Audio processing test initialized")

        # Test invalid audio data
        logger.info("🧪 Testing invalid audio data handling...")
        invalid_audio = base64.b64encode(b"invalid audio data").decode()
        await websocket.send(json.dumps({"type": "audio_data", "audio": invalid_audio}))

        # Wait a bit for processing
        await asyncio.sleep(2)

        # Test valid audio data (silence)
        logger.info("🧪 Testing valid audio data...")
        silence_audio = base64.b64encode(
            b"\x00" * 16000
        ).decode()  # 1 second of silence at 16kHz
        await websocket.send(json.dumps({"type": "audio_data", "audio": silence_audio}))

        # Wait for processing
        await asyncio.sleep(3)

        # Send stop message
        await websocket.send(json.dumps({"type": "stop"}))

        # Wait for ready_to_stop
        try:
            response = await asyncio.wait_for(websocket.recv(), timeout=5.0)
            data = json.loads(response)
            if data.get("type") == "ready_to_stop":
                logger.info("✅ Audio processing test completed successfully")
            else:
                logger.warning(f"⚠️ Unexpected response: {data}")
        except asyncio.TimeoutError:
            logger.warning("⚠️ Timeout waiting for ready_to_stop")

        await websocket.close()

    except Exception as e:
        logger.error(f"❌ Audio processing test failed: {e}")


async def main():
    """Run all tests"""
    logger.info("🚀 Starting robust server tests...")

    # Wait for server to be ready
    logger.info("⏳ Waiting for server to be ready...")
    await asyncio.sleep(2)

    try:
        await test_health_endpoint()
        await test_stats_endpoint()
        await test_connection_management()
        await test_audio_processing()

        logger.info("✅ All tests completed!")

    except Exception as e:
        logger.error(f"❌ Test suite failed: {e}")


if __name__ == "__main__":
    asyncio.run(main())
