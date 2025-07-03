#!/usr/bin/env python3
"""
Test script for WhisperLiveKit server
"""

import asyncio
import base64
import json
import websockets
import time


async def test_whisperlivekit_server():
    """Test the WhisperLiveKit server"""
    uri = "ws://localhost:9090/ws/transcribe"

    try:
        async with websockets.connect(uri) as websocket:
            print("Connected to WhisperLiveKit server")

            # Send initialization message
            init_message = {
                "type": "init",
                "client_id": f"test_client_{int(time.time() * 1000)}",
                "model_size": "small",
                "language": "en",
            }

            await websocket.send(json.dumps(init_message))
            print("Sent init message:", init_message)

            # Wait for init response
            response = await websocket.recv()
            response_data = json.loads(response)
            print("Received response:", response_data)

            if response_data.get("type") == "init_success":
                print("✅ Server initialization successful!")

                # Send a test audio message (empty for now)
                test_audio = base64.b64encode(b"test_audio_data").decode()
                audio_message = {"type": "audio_data", "audio": test_audio}

                await websocket.send(json.dumps(audio_message))
                print("Sent test audio message")

                # Wait for any response
                try:
                    response = await asyncio.wait_for(websocket.recv(), timeout=5.0)
                    response_data = json.loads(response)
                    print("Received audio response:", response_data)
                except asyncio.TimeoutError:
                    print("No response to audio message (this is normal for test data)")

                # Send stop message
                stop_message = {"type": "stop"}
                await websocket.send(json.dumps(stop_message))
                print("Sent stop message")

            else:
                print("❌ Server initialization failed:", response_data)

    except Exception as e:
        print(f"❌ Error testing server: {e}")


if __name__ == "__main__":
    asyncio.run(test_whisperlivekit_server())
