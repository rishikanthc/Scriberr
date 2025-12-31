#!/usr/bin/env python3
"""
Voxtral buffered inference for long audio files.
Splits audio into chunks to avoid context window limits.
"""

import argparse
import json
import sys
import os
import librosa
import soundfile as sf
import torch
from pathlib import Path
from transformers import VoxtralForConditionalGeneration, AutoProcessor


def split_audio_file(audio_path, chunk_duration_secs=1500):
    """Split audio file into chunks of specified duration.

    Default: 1500 seconds (25 minutes) to stay safely within Voxtral's 30-40 min limit.
    """
    audio, sr = librosa.load(audio_path, sr=None, mono=True)
    total_duration = len(audio) / sr
    chunk_samples = int(chunk_duration_secs * sr)

    chunks = []
    for start_sample in range(0, len(audio), chunk_samples):
        end_sample = min(start_sample + chunk_samples, len(audio))
        chunk_audio = audio[start_sample:end_sample]
        start_time = start_sample / sr
        chunks.append(
            {
                "audio": chunk_audio,
                "start_time": start_time,
                "duration": len(chunk_audio) / sr,
            }
        )

    return chunks, sr


def transcribe_buffered(
    audio_path: str,
    output_file: str,
    language: str = "en",
    model_id: str = "mistralai/Voxtral-mini",
    device: str = "auto",
    max_new_tokens: int = 8192,
    chunk_duration_secs: float = 1500,  # 25 minutes default
):
    """
    Transcribe long audio by splitting into chunks and merging results.
    """
    # Determine device
    # if device == "auto":
    #     device = "cuda" if torch.cuda.is_available() else "cpu"
    device = "cuda" if torch.cuda.is_available() else "cpu"

    print(f"Loading Voxtral model on {device}...", file=sys.stderr)

    # Load processor and model
    processor = AutoProcessor.from_pretrained(model_id)

    # Use appropriate dtype based on device
    dtype = torch.bfloat16 if device == "cuda" else torch.float32

    model = VoxtralForConditionalGeneration.from_pretrained(
        model_id,
        torch_dtype=dtype,
        device_map=device,
    )

    print(f"Model loaded successfully", file=sys.stderr)
    print(f"Splitting audio into {chunk_duration_secs}s chunks...", file=sys.stderr)

    chunks, sr = split_audio_file(audio_path, chunk_duration_secs)
    print(f"Created {len(chunks)} chunks", file=sys.stderr)

    full_text = []

    for i, chunk_info in enumerate(chunks):
        print(
            f"Transcribing chunk {i + 1}/{len(chunks)} (duration: {chunk_info['duration']:.1f}s)...",
            file=sys.stderr,
        )

        # Save chunk to temporary file
        chunk_path = f"/tmp/voxtral_chunk_{i}.wav"
        sf.write(chunk_path, chunk_info["audio"], sr)

        try:
            # Prepare transcription request for this chunk
            inputs = processor.apply_transcription_request(
                language=language, audio=chunk_path, model_id=model_id
            )

            # Move inputs to device with correct dtype
            inputs = inputs.to(device, dtype=dtype)

            # Generate transcription
            with torch.no_grad():
                outputs = model.generate(
                    **inputs,
                    max_new_tokens=max_new_tokens,
                )

            # Decode only the newly generated tokens (skip the input prompt)
            decoded_outputs = processor.batch_decode(
                outputs[:, inputs.input_ids.shape[1] :], skip_special_tokens=True
            )

            chunk_text = decoded_outputs[0]
            full_text.append(chunk_text)

            print(
                f"Chunk {i + 1} complete: {len(chunk_text)} characters", file=sys.stderr
            )

        finally:
            # Clean up temp file
            if os.path.exists(chunk_path):
                os.remove(chunk_path)

    # Concatenate all chunks
    final_text = " ".join(full_text)
    print(
        f"Transcription complete: {len(final_text)} characters total", file=sys.stderr
    )

    # Prepare output in Scriberr format
    # Note: Voxtral doesn't provide word-level timestamps
    result = {
        "text": final_text,
        "segments": [
            {
                "id": 0,
                "start": 0.0,
                "end": 0.0,  # Duration unknown without audio analysis
                "text": final_text,
                "words": [],  # Voxtral doesn't provide word-level timestamps
            }
        ],
        "language": language,
        "model": model_id,
        "has_word_timestamps": False,
        "buffered": True,
        "chunk_duration_secs": chunk_duration_secs,
        "num_chunks": len(chunks),
    }

    # Write output
    output_file_path = Path(output_file)
    with output_file_path.open("w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)

    print(f"Results written to {output_file}", file=sys.stderr)

    return result


def main():
    parser = argparse.ArgumentParser(
        description="Transcribe long audio using Voxtral with chunking"
    )
    parser.add_argument("audio_path", type=str, help="Path to input audio file")
    parser.add_argument("output_path", type=str, help="Path to output JSON file")
    parser.add_argument(
        "--language", type=str, default="en", help="Language code (default: en)"
    )
    parser.add_argument(
        "--model-id",
        type=str,
        default="mistralai/Voxtral-mini",
        help="HuggingFace model ID (default: mistralai/Voxtral-mini)",
    )
    parser.add_argument(
        "--device",
        type=str,
        default="auto",
        choices=["cpu", "cuda", "auto"],
        help="Device to use (default: auto)",
    )
    parser.add_argument(
        "--max-new-tokens",
        type=int,
        default=8192,
        help="Maximum number of tokens to generate per chunk (default: 8192)",
    )
    parser.add_argument(
        "--chunk-len",
        type=float,
        default=1500,
        help="Chunk duration in seconds (default: 1500 = 25 minutes)",
    )

    args = parser.parse_args()

    if not os.path.exists(args.audio_path):
        print(f"Error: Audio file not found: {args.audio_path}", file=sys.stderr)
        sys.exit(1)

    try:
        transcribe_buffered(
            audio_path=args.audio_path,
            output_file=args.output_path,
            language=args.language,
            model_id=args.model_id,
            device=args.device,
            max_new_tokens=args.max_new_tokens,
            chunk_duration_secs=args.chunk_len,
        )
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        import traceback

        traceback.print_exc(file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
