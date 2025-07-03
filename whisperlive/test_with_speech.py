#!/usr/bin/env python3
"""
Test WhisperLiveKit with actual speech audio
"""

import asyncio
import json
import logging
import tempfile
import os
from whisperlivekit import TranscriptionEngine, AudioProcessor

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def test_with_speech():
    """Test with actual speech audio"""

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

        # Create a simple sine wave that might trigger transcription
        logger.info("🎵 Creating test audio with speech-like content...")

        # Create a simple audio pattern that might be detected as speech
        import numpy as np

        sample_rate = 16000
        duration = 2.0  # 2 seconds
        t = np.linspace(0, duration, int(sample_rate * duration), False)

        # Create a complex waveform that might be detected as speech
        frequency = 440  # A4 note
        audio = np.sin(2 * np.pi * frequency * t) * 0.3
        audio += np.sin(2 * np.pi * frequency * 1.5 * t) * 0.2  # Add harmonics

        # Convert to 16-bit PCM
        audio_int16 = (audio * 32767).astype(np.int16)
        test_audio = audio_int16.tobytes()

        logger.info(f"🎵 Created test audio: {len(test_audio)} bytes")

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

                # Check if we got any transcription
                if response.get("lines"):
                    lines = response["lines"]
                    for line in lines:
                        text = line.get("text", "").strip()
                        if text:
                            logger.info(f"🎤 TRANSCRIPTION FOUND: {text}")

                if response_count >= 10:  # Limit to 10 responses
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
    asyncio.run(test_with_speech())
