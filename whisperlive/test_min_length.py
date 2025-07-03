#!/usr/bin/env python3
"""
Test minimum audio length for faster-whisper
"""

import numpy as np
from faster_whisper import WhisperModel


def test_min_length():
    """Test different audio lengths to find minimum"""

    # Load the model
    print("Loading Whisper model...")
    model = WhisperModel("tiny", device="cpu", compute_type="int8")

    # Test different durations
    durations = [0.1, 0.2, 0.5, 1.0, 2.0, 3.0]  # seconds

    for duration in durations:
        print(f"\n=== Testing {duration}s audio ===")

        # Generate test audio
        sample_rate = 16000
        samples = int(sample_rate * duration)

        # Create a simple sine wave
        t = np.linspace(0, duration, samples, False)
        audio = np.sin(2 * np.pi * 440 * t)  # 440Hz tone
        audio = audio.astype(np.float32)

        print(f"Audio: {len(audio)} samples, {duration}s")

        try:
            # Test transcription
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

            segments_list = list(segments)
            print(f"Segments: {len(segments_list)}")

            for i, segment in enumerate(segments_list):
                print(
                    f"  Segment {i}: '{segment.text}' ({segment.start:.2f}s - {segment.end:.2f}s)"
                )

        except Exception as e:
            print(f"Error: {e}")


if __name__ == "__main__":
    test_min_length()
