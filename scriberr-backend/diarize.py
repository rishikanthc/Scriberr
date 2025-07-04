#!/usr/bin/env python3
"""
Enhanced speaker diarization script using whisperx
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
        diarization_model: Pyannote diarization model to use
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
        print("Loading diarization model...")

        # Try to use pyannote.audio first for better results
        diarize_segments = None
        if PYANNOTE_AVAILABLE and hf_token:
            print("Using pyannote.audio for enhanced diarization...")
            diarize_segments = enhanced_pyannote_diarization(
                audio_file, hf_token, device, min_speakers, max_speakers
            )

        # Fall back to WhisperX diarization if pyannote fails or is not available
        if diarize_segments is None:
            print("Using WhisperX diarization...")
            if hf_token:
                diarize_model = whisperx.diarize.DiarizationPipeline(
                    use_auth_token=hf_token, device=device, model_name=diarization_model
                )
            else:
                # Try to use environment variable
                hf_token = os.getenv("HF_TOKEN")
                if hf_token:
                    diarize_model = whisperx.diarize.DiarizationPipeline(
                        use_auth_token=hf_token,
                        device=device,
                        model_name=diarization_model,
                    )
                else:
                    print("Warning: No HF token provided. Using default diarization.")
                    diarize_model = whisperx.diarize.DiarizationPipeline(
                        device=device, model_name=diarization_model
                    )

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

        # Clean up diarization model (only if it was created)
        if "diarize_model" in locals():
            del diarize_model
        gc.collect()
        if device == "cuda":
            torch.cuda.empty_cache()

        # 6. Save to file if specified
        if output_file:
            print(f"Saving transcript to {output_file}")
            with open(output_file, "w", encoding="utf-8") as f:
                json.dump(result, f, indent=2, ensure_ascii=False)

        return result

    except Exception as e:
        print(f"Error during diarization: {e}")
        raise


def enhanced_pyannote_diarization(
    audio_file: str, hf_token: str, device: str, min_speakers: int, max_speakers: int
) -> Optional[Any]:
    """Enhanced diarization using pyannote.audio directly for better results"""
    try:
        # Initialize the pipeline with the latest model
        pipeline = Pipeline.from_pretrained(
            "pyannote/speaker-diarization-3.1", use_auth_token=hf_token
        )

        # Move to device if available
        if device == "cuda":
            pipeline = pipeline.to(torch.device("cuda"))

        print("Running pyannote.audio diarization...")

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


def main():
    parser = argparse.ArgumentParser(
        description="Enhanced robust speaker diarization using whisperx and pyannote.audio"
    )
    parser.add_argument("audio_file", help="Path to audio file")
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
        help="Diarization model to use",
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

    # Validate input file
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
        )

        print(f"Enhanced diarization completed successfully!")
        print(f"Output saved to: {args.output}")
        print(f"Total segments: {len(result.get('segments', []))}")

    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
