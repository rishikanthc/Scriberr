#!/usr/bin/env python3
"""
Enhanced speaker diarization script using whisperx with multiple model options
This script provides robust diarization with improved transcript coverage and speaker alignment
"""

import whisperx
import gc
import json
import sys
import os
import argparse
from pathlib import Path
import torch
import numpy as np
import pandas as pd
from typing import List, Dict, Any, Optional

# Try to import pyannote.audio for enhanced diarization
try:
    from pyannote.audio import Pipeline
    from pyannote.audio.pipelines.utils.hook import ProgressHook

    PYANNOTE_AVAILABLE = True
except ImportError:
    PYANNOTE_AVAILABLE = False
    print("Warning: pyannote.audio not available. Using WhisperX diarization only.")

# Available diarization models
DIARIZATION_MODELS = {
    "pyannote/speaker-diarization-3.1": {
        "name": "Pyannote 3.1 (Recommended)",
        "description": "Latest pyannote model with improved accuracy",
        "requires_token": True,
        "type": "pyannote",
    },
    "pyannote/speaker-diarization-3.0": {
        "name": "Pyannote 3.0",
        "description": "Previous generation pyannote model",
        "requires_token": True,
        "type": "pyannote",
    },
    "pyannote/speaker-diarization-2.1": {
        "name": "Pyannote 2.1",
        "description": "Older but stable pyannote model",
        "requires_token": True,
        "type": "pyannote",
    },
    "whisperx-default": {
        "name": "WhisperX Default",
        "description": "Built-in WhisperX diarization",
        "requires_token": False,
        "type": "whisperx",
    },
}


def create_speaker_based_segments(
    result: Dict[str, Any], min_segment_duration: float = 0.5
) -> Dict[str, Any]:
    """
    Create segments based on speaker changes rather than time-based segmentation.
    Combines contiguous words with the same speaker into segments.

    Args:
        result: WhisperX result with word-level speaker assignments
        min_segment_duration: Minimum duration for a segment (seconds)

    Returns:
        Dict with speaker-based segments
    """
    if "segments" not in result or not result["segments"]:
        return result

    new_segments = []
    current_segment = None

    for segment in result["segments"]:
        if "words" not in segment or not segment["words"]:
            # If no word-level data, keep the original segment
            new_segments.append(segment)
            continue

        words = segment["words"]

        for word in words:
            if "speaker" not in word:
                continue

            speaker = word["speaker"]
            start = word["start"]
            end = word["end"]
            text = word.get("word", "")

            # Initialize first segment
            if current_segment is None:
                current_segment = {
                    "start": start,
                    "end": end,
                    "speaker": speaker,
                    "text": text,
                    "words": [word],
                }
            elif current_segment["speaker"] == speaker:
                # Same speaker, extend current segment
                current_segment["end"] = end
                current_segment["text"] += " " + text
                current_segment["words"].append(word)
            else:
                # Different speaker, finalize current segment and start new one
                if (
                    current_segment["end"] - current_segment["start"]
                    >= min_segment_duration
                ):
                    new_segments.append(current_segment)

                current_segment = {
                    "start": start,
                    "end": end,
                    "speaker": speaker,
                    "text": text,
                    "words": [word],
                }

    # Add the last segment if it exists
    if (
        current_segment
        and current_segment["end"] - current_segment["start"] >= min_segment_duration
    ):
        new_segments.append(current_segment)

    # Update the result with new segments
    result["segments"] = new_segments
    return result


def merge_short_segments(
    segments: List[Dict], min_duration: float = 1.0, max_gap: float = 0.5
) -> List[Dict]:
    """
    Merge short segments with adjacent segments to improve readability.

    Args:
        segments: List of segments
        min_duration: Minimum duration for a segment to be kept separate
        max_gap: Maximum gap between segments to merge

    Returns:
        List of merged segments
    """
    if not segments:
        return segments

    merged = []
    current = segments[0].copy()

    for next_segment in segments[1:]:
        gap = next_segment["start"] - current["end"]

        # Merge if current segment is too short and gap is small
        if (
            current["end"] - current["start"] < min_duration
            and gap <= max_gap
            and current.get("speaker") == next_segment.get("speaker")
        ):
            current["end"] = next_segment["end"]
            current["text"] += " " + next_segment["text"]
            if "words" in current and "words" in next_segment:
                current["words"].extend(next_segment["words"])
        else:
            merged.append(current)
            current = next_segment.copy()

    merged.append(current)
    return merged


def diarize_audio(
    audio_file,
    model_size="large-v2",
    min_speakers=1,
    max_speakers=2,
    batch_size=16,
    compute_type="float16",
    output_file=None,
    hf_token=None,
    diarization_model="pyannote/speaker-diarization-3.1",
    # Additional transcription parameters
    vad_onset=0.5,
    vad_offset=0.5,
    condition_on_previous_text=True,
    compression_ratio_threshold=2.4,
    logprob_threshold=-1.0,
    no_speech_threshold=0.6,
    temperature=0.0,
    best_of=5,
    beam_size=5,
    patience=1.0,
    length_penalty=1.0,
    suppress_numerals=False,
    initial_prompt="",
    temperature_increment_on_fallback=0.2,
    # New parameters for improved diarization
    min_segment_duration=0.5,
    merge_short_segments_enabled=True,
    merge_min_duration=1.0,
    merge_max_gap=0.5,
):
    """
    Perform robust speaker diarization using whisperx with enhanced transcript coverage

    Args:
        audio_file: Path to the audio file
        model_size: Whisper model size (tiny, base, small, medium, large-v2, large-v3)
        min_speakers: Minimum number of speakers
        max_speakers: Maximum number of speakers
        batch_size: Batch size for transcription (reduce if low on GPU mem)
        compute_type: Compute type (float16, int8)
        output_file: Output JSON file path
        hf_token: HuggingFace token for pyannote.audio
        diarization_model: Diarization model to use
        vad_onset: VAD onset threshold
        vad_offset: VAD offset threshold
        condition_on_previous_text: Condition on previous text
        compression_ratio_threshold: Compression ratio threshold
        logprob_threshold: Log probability threshold
        no_speech_threshold: No speech threshold
        temperature: Temperature for sampling
        best_of: Number of best candidates to consider
        beam_size: Beam size for beam search
        patience: Patience for beam search
        length_penalty: Length penalty for beam search
        suppress_numerals: Suppress numerals in output
        initial_prompt: Initial prompt for transcription
        temperature_increment_on_fallback: Temperature increment on fallback
        min_segment_duration: Minimum duration for speaker-based segments
        merge_short_segments_enabled: Whether to merge short segments
        merge_min_duration: Minimum duration for segments to be kept separate
        merge_max_gap: Maximum gap between segments to merge

    Returns:
        dict: Processed transcript with robust speaker diarization
    """

    device = "cuda" if torch.cuda.is_available() else "cpu"
    print(f"Using device: {device}")

    if device == "cpu":
        compute_type = "int8"  # Force int8 for CPU
        batch_size = 1  # Reduce batch size for CPU

    try:
        # 1. Load and validate audio
        print("Loading audio...")
        audio = whisperx.load_audio(audio_file)
        audio_duration = len(audio) / 16000  # Assuming 16kHz sample rate
        print(f"Audio duration: {audio_duration:.2f} seconds")

        # 2. Transcribe with original whisper (batched)
        print("Loading whisper model...")
        model = whisperx.load_model(model_size, device, compute_type=compute_type)

        print("Transcribing audio...")
        # Use only the most basic parameters that are definitely supported by WhisperX
        result = model.transcribe(
            audio,
            batch_size=batch_size,
        )
        print(f"Transcription complete. Segments: {len(result['segments'])}")

        # Clean up model to free GPU memory
        del model
        gc.collect()
        if device == "cuda":
            torch.cuda.empty_cache()

        # 3. Align whisper output with enhanced error handling
        print("Loading alignment model...")
        model_a, metadata = whisperx.load_align_model(
            language_code=result["language"], device=device
        )

        print("Aligning transcript...")
        result = whisperx.align(
            result["segments"],
            model_a,
            metadata,
            audio,
            device,
            return_char_alignments=False,
        )
        print(f"Alignment complete. Segments: {len(result['segments'])}")

        # Clean up alignment model
        del model_a
        gc.collect()
        if device == "cuda":
            torch.cuda.empty_cache()

        # 4. Speaker diarization
        print(f"Loading diarization model: {diarization_model}")

        # Check if the model requires a token
        model_info = DIARIZATION_MODELS.get(diarization_model, {})
        requires_token = model_info.get("requires_token", True)

        # --- Token handling: always check env if CLI arg is missing or empty ---
        if not hf_token:
            hf_token = os.getenv("HF_TOKEN")
        if requires_token and not hf_token:
            print(
                f"Warning: {diarization_model} requires HF token but none provided. Set the HF_TOKEN environment variable or use --hf-token. Falling back to WhisperX default."
            )
            diarization_model = "whisperx-default"

        # Try to use pyannote.audio for better results
        diarize_segments = None
        if (
            diarization_model.startswith("pyannote/")
            and PYANNOTE_AVAILABLE
            and hf_token
        ):
            print("Using pyannote.audio for enhanced diarization...")
            diarize_segments = enhanced_pyannote_diarization(
                audio_file,
                hf_token,
                device,
                min_speakers,
                max_speakers,
                diarization_model,
            )

        # Fall back to WhisperX diarization if pyannote fails or is not available
        if diarize_segments is None:
            print("Using WhisperX diarization...")
            if hf_token and diarization_model.startswith("pyannote/"):
                diarize_model = whisperx.diarize.DiarizationPipeline(
                    use_auth_token=hf_token, device=device, model_name=diarization_model
                )
            else:
                # Use default WhisperX diarization
                diarize_model = whisperx.diarize.DiarizationPipeline(device=device)

            print("Performing WhisperX speaker diarization...")
            # Use only supported parameters for WhisperX diarization
            diarize_segments = diarize_model(
                audio,
                min_speakers=min_speakers,
                max_speakers=max_speakers,
            )

        print(f"Diarization complete. Speaker segments: {len(diarize_segments)}")

        # 5. Assign speaker labels using WhisperX built-in function
        print("Assigning speaker labels to words...")
        result = whisperx.assign_word_speakers(diarize_segments, result)
        print(f"Speaker assignment complete. Final segments: {len(result['segments'])}")

        # 6. Create speaker-based segments
        print("Creating speaker-based segments...")
        result = create_speaker_based_segments(result, min_segment_duration)
        print(
            f"Speaker-based segmentation complete. Segments: {len(result['segments'])}"
        )

        # 7. Merge short segments if enabled
        if merge_short_segments_enabled:
            print("Merging short segments...")
            result["segments"] = merge_short_segments(
                result["segments"], merge_min_duration, merge_max_gap
            )
            print(
                f"Segment merging complete. Final segments: {len(result['segments'])}"
            )

        # Clean up diarization model (only if it was created)
        if "diarize_model" in locals():
            del diarize_model
        gc.collect()
        if device == "cuda":
            torch.cuda.empty_cache()

        # 8. Save to file if specified
        if output_file:
            print(f"Saving transcript to {output_file}")
            with open(output_file, "w", encoding="utf-8") as f:
                json.dump(result, f, indent=2, ensure_ascii=False)

        return result

    except Exception as e:
        print(f"Error during diarization: {e}")
        raise


def enhanced_pyannote_diarization(
    audio_file: str,
    hf_token: str,
    device: str,
    min_speakers: int,
    max_speakers: int,
    model_name: str,
) -> Optional[Any]:
    """Enhanced diarization using pyannote.audio directly for better results"""
    try:
        # Initialize the pipeline with the specified model
        pipeline = Pipeline.from_pretrained(model_name, use_auth_token=hf_token)

        # Move to device if available
        if device == "cuda":
            pipeline = pipeline.to(torch.device("cuda"))

        print(f"Running pyannote.audio diarization with {model_name}...")

        # Run diarization with only supported parameters
        with ProgressHook() as hook:
            diarization = pipeline(
                audio_file,
                hook=hook,
                min_speakers=min_speakers,
                max_speakers=max_speakers,
            )

        # Convert pyannote format to pandas DataFrame format expected by WhisperX
        segments_data = []
        for turn, _, speaker in diarization.itertracks(yield_label=True):
            segments_data.append(
                {"start": turn.start, "end": turn.end, "speaker": speaker}
            )

        # Create DataFrame in the format expected by WhisperX
        diarize_df = pd.DataFrame(segments_data)

        print(f"Pyannote diarization complete: {len(diarize_df)} segments")
        return diarize_df

    except Exception as e:
        print(f"Pyannote diarization failed: {e}")
        print("Falling back to WhisperX diarization...")
        return None


def list_available_models():
    """List all available diarization models"""
    print("Available diarization models:")
    print("-" * 50)
    for model_id, info in DIARIZATION_MODELS.items():
        print(f"ID: {model_id}")
        print(f"Name: {info['name']}")
        print(f"Description: {info['description']}")
        print(f"Requires HF Token: {info['requires_token']}")
        print(f"Type: {info['type']}")
        print("-" * 50)


def main():
    parser = argparse.ArgumentParser(
        description="Enhanced robust speaker diarization using whisperx and pyannote.audio"
    )
    parser.add_argument(
        "--list-models", action="store_true", help="List available diarization models"
    )
    parser.add_argument("audio_file", nargs="?", help="Path to audio file")
    parser.add_argument("--model", default="large-v2", help="Whisper model size")
    parser.add_argument(
        "--min-speakers", type=int, default=1, help="Minimum number of speakers"
    )
    parser.add_argument(
        "--max-speakers", type=int, default=2, help="Maximum number of speakers"
    )
    parser.add_argument(
        "--batch-size", type=int, default=16, help="Batch size for transcription"
    )
    parser.add_argument(
        "--compute-type", default="float16", help="Compute type (float16, int8)"
    )
    parser.add_argument("--output", help="Output JSON file path")
    parser.add_argument("--hf-token", help="HuggingFace token for pyannote.audio")
    parser.add_argument(
        "--diarization-model",
        default="pyannote/speaker-diarization-3.1",
        choices=list(DIARIZATION_MODELS.keys()),
        help="Diarization model to use",
    )

    # New diarization parameters
    parser.add_argument(
        "--min-segment-duration",
        type=float,
        default=0.5,
        help="Minimum duration for speaker-based segments (seconds)",
    )
    parser.add_argument(
        "--merge-short-segments",
        action="store_true",
        default=True,
        help="Merge short segments with adjacent segments",
    )
    parser.add_argument(
        "--merge-min-duration",
        type=float,
        default=1.0,
        help="Minimum duration for segments to be kept separate (seconds)",
    )
    parser.add_argument(
        "--merge-max-gap",
        type=float,
        default=0.5,
        help="Maximum gap between segments to merge (seconds)",
    )

    # Additional transcription parameters
    parser.add_argument(
        "--vad-onset",
        type=float,
        default=0.5,
        help="VAD onset threshold",
    )
    parser.add_argument(
        "--vad-offset",
        type=float,
        default=0.5,
        help="VAD offset threshold",
    )
    parser.add_argument(
        "--condition-on-previous-text",
        type=str,
        default="True",
        choices=["True", "False"],
        help="Condition on previous text (True/False)",
    )
    parser.add_argument(
        "--compression-ratio-threshold",
        type=float,
        default=2.4,
        help="Compression ratio threshold",
    )
    parser.add_argument(
        "--logprob-threshold",
        type=float,
        default=-1.0,
        help="Log probability threshold",
    )
    parser.add_argument(
        "--no-speech-threshold",
        type=float,
        default=0.6,
        help="No speech threshold",
    )
    parser.add_argument(
        "--temperature",
        type=float,
        default=0.0,
        help="Temperature for sampling",
    )
    parser.add_argument(
        "--best-of",
        type=int,
        default=5,
        help="Number of best candidates to consider",
    )
    parser.add_argument(
        "--beam-size",
        type=int,
        default=5,
        help="Beam size for beam search",
    )
    parser.add_argument(
        "--patience",
        type=float,
        default=1.0,
        help="Patience for beam search",
    )
    parser.add_argument(
        "--length-penalty",
        type=float,
        default=1.0,
        help="Length penalty for beam search",
    )
    parser.add_argument(
        "--suppress-numerals",
        action="store_true",
        help="Suppress numerals in output",
    )
    parser.add_argument(
        "--initial-prompt",
        type=str,
        default="",
        help="Initial prompt for transcription",
    )
    parser.add_argument(
        "--temperature-increment-on-fallback",
        type=float,
        default=0.2,
        help="Temperature increment on fallback",
    )

    args = parser.parse_args()

    # Handle list models command
    if args.list_models:
        list_available_models()
        return

    # Validate input file
    if not args.audio_file:
        print("Error: Audio file is required")
        sys.exit(1)

    if not os.path.exists(args.audio_file):
        print(f"Error: Audio file '{args.audio_file}' not found")
        sys.exit(1)

    # Set default output file if not specified
    if not args.output:
        audio_path = Path(args.audio_file)
        args.output = audio_path.with_suffix(".json")

    try:
        # Convert string "True"/"False" to boolean
        args.condition_on_previous_text = (
            args.condition_on_previous_text.lower() == "true"
        )

        result = diarize_audio(
            audio_file=args.audio_file,
            model_size=args.model,
            min_speakers=args.min_speakers,
            max_speakers=args.max_speakers,
            batch_size=args.batch_size,
            compute_type=args.compute_type,
            output_file=args.output,
            hf_token=args.hf_token,
            diarization_model=args.diarization_model,
            vad_onset=args.vad_onset,
            vad_offset=args.vad_offset,
            condition_on_previous_text=args.condition_on_previous_text,
            compression_ratio_threshold=args.compression_ratio_threshold,
            logprob_threshold=args.logprob_threshold,
            no_speech_threshold=args.no_speech_threshold,
            temperature=args.temperature,
            best_of=args.best_of,
            beam_size=args.beam_size,
            patience=args.patience,
            length_penalty=args.length_penalty,
            suppress_numerals=args.suppress_numerals,
            initial_prompt=args.initial_prompt,
            temperature_increment_on_fallback=args.temperature_increment_on_fallback,
            min_segment_duration=args.min_segment_duration,
            merge_short_segments_enabled=args.merge_short_segments,
            merge_min_duration=args.merge_min_duration,
            merge_max_gap=args.merge_max_gap,
        )

        print(f"Enhanced diarization completed successfully!")
        print(f"Output saved to: {args.output}")
        print(f"Total segments: {len(result.get('segments', []))}")

    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
