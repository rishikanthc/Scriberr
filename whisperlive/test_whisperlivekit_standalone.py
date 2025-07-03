#!/usr/bin/env python3
"""
Standalone test for WhisperLiveKit transcription
"""

import asyncio
import base64
import json
import logging
import tempfile
import time
from whisperlivekit import TranscriptionEngine, AudioProcessor

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def test_whisperlivekit_transcription():
    """Test WhisperLiveKit transcription directly"""

    try:
        # Initialize transcription engine
        logger.info("🔧 Initializing WhisperLiveKit transcription engine...")
        transcription_engine = TranscriptionEngine(
            model="tiny",
            diarization=False,
            lan="en",
            backend="faster-whisper",
            confidence_validation=True,
            min_chunk_size=0.5,
            buffer_trimming="segment",
            no_vad=False,
        )
        logger.info("✅ Transcription engine initialized")

        # Create audio processor
        logger.info("🎯 Creating AudioProcessor...")
        audio_processor = AudioProcessor(transcription_engine=transcription_engine)
        results_generator = await audio_processor.create_tasks()
        logger.info("✅ AudioProcessor created")

        # Create some test audio (silence)
        logger.info("🎵 Creating test audio...")
        test_audio = b"\x00\x00" * 8000  # 1 second of silence at 16kHz

        # Process audio
        logger.info("🔄 Processing test audio...")
        await audio_processor.process_audio(test_audio)
        logger.info("✅ Audio processed")

        # Try to get results
        logger.info("📥 Trying to get results...")
        response_count = 0

        try:
            async for response in results_generator:
                response_count += 1
                logger.info(
                    f"📨 Response #{response_count}: {json.dumps(response, indent=2)}"
                )

                if response_count >= 5:  # Limit to 5 responses
                    break

        except Exception as e:
            logger.error(f"❌ Error getting results: {e}")

        logger.info(f"🏁 Test completed. Total responses: {response_count}")

        # Cleanup
        audio_processor.cleanup()
        transcription_engine.cleanup()

    except Exception as e:
        logger.error(f"❌ Test failed: {e}")


if __name__ == "__main__":
    asyncio.run(test_whisperlivekit_transcription())
