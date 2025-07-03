#!/usr/bin/env python3
"""
Test script to verify audio processing works correctly
"""

import numpy as np
import base64
from faster_whisper import WhisperModel


def test_audio_processing():
    """Test that we can process audio and get transcription"""

    # Create a simple test audio (sine wave)
    sample_rate = 16000
    duration = 2.0  # 2 seconds
    frequency = 440  # A4 note

    # Generate a sine wave
    t = np.linspace(0, duration, int(sample_rate * duration), False)
    audio = np.sin(2 * np.pi * frequency * t)

    # Convert to float32 (faster-whisper expects this)
    audio = audio.astype(np.float32)

    print(f"Test audio: {len(audio)} samples, {duration}s, {sample_rate}Hz")
    print(f"Audio range: {audio.min():.4f} to {audio.max():.4f}")

    # Load the model
    print("Loading Whisper model...")
    model = WhisperModel("tiny", device="cpu", compute_type="int8")

    # Test transcription
    print("Testing transcription...")
    segments, info = model.transcribe(
        audio,
        language="en",
        task="transcribe",
        beam_size=1,
        best_of=1,
        temperature=0.0,
        no_speech_threshold=0.3,
        condition_on_previous_text=False,
        word_timestamps=False,
    )

    print(f"Transcription info: {info}")
    print(f"Number of segments: {len(list(segments))}")

    # Try again to get the actual segments
    segments, info = model.transcribe(
        audio,
        language="en",
        task="transcribe",
        beam_size=1,
        best_of=1,
        temperature=0.0,
        no_speech_threshold=0.3,
        condition_on_previous_text=False,
        word_timestamps=False,
    )

    for i, segment in enumerate(segments):
        print(
            f"Segment {i}: '{segment.text}' (start: {segment.start:.2f}s, end: {segment.end:.2f}s)"
        )

    print("✅ Audio processing test completed!")


if __name__ == "__main__":
    test_audio_processing()
