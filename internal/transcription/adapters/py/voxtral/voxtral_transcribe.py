#!/usr/bin/env python3
"""
Voxtral-mini transcription script for Scriberr
Transcribes audio using Mistral's Voxtral-mini model
"""

import argparse
import json
import sys
import torch
from pathlib import Path
from transformers import VoxtralForConditionalGeneration, AutoProcessor


def transcribe_audio(
    audio_path: str,
    output_path: str,
    language: str = "en",
    model_id: str = "mistralai/Voxtral-mini",
    device: str = "auto",
    max_new_tokens: int = 8192,
) -> dict:
    """
    Transcribe audio using Voxtral-mini model.

    Args:
        audio_path: Path to input audio file
        output_path: Path to output JSON file
        language: Language code (e.g., 'en', 'es', 'fr')
        model_id: HuggingFace model ID
        device: Device to use ('cpu', 'cuda', or 'auto')
        max_new_tokens: Maximum number of tokens to generate

    Returns:
        Dictionary containing transcription results
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
    print(f"Processing audio: {audio_path}", file=sys.stderr)

    # Prepare transcription request using the proper method
    inputs = processor.apply_transcription_request(
        language=language, audio=audio_path, model_id=model_id
    )

    # Move inputs to device with correct dtype
    inputs = inputs.to(device, dtype=dtype)

    print(f"Generating transcription...", file=sys.stderr)

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

    transcription_text = decoded_outputs[0]

    print(f"Transcription completed ({len(transcription_text)} chars)", file=sys.stderr)

    # Prepare output in Scriberr format
    # Note: Voxtral doesn't provide word-level timestamps, so we create a single segment
    result = {
        "text": transcription_text,
        "segments": [
            {
                "id": 0,
                "start": 0.0,
                "end": 0.0,  # Duration unknown without audio analysis
                "text": transcription_text,
                "words": [],  # Voxtral doesn't provide word-level timestamps
            }
        ],
        "language": language,
        "model": model_id,
        "has_word_timestamps": False,  # Important: Voxtral doesn't support timestamps
    }

    # Write output
    output_file = Path(output_path)
    with output_file.open("w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)

    print(f"Results written to {output_path}", file=sys.stderr)

    return result


def main():
    parser = argparse.ArgumentParser(
        description="Transcribe audio using Voxtral-mini model"
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
        help="Maximum number of tokens to generate (default: 8192)",
    )

    args = parser.parse_args()

    try:
        transcribe_audio(
            audio_path=args.audio_path,
            output_path=args.output_path,
            language=args.language,
            model_id=args.model_id,
            device=args.device,
            max_new_tokens=args.max_new_tokens,
        )
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        import traceback

        traceback.print_exc(file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
