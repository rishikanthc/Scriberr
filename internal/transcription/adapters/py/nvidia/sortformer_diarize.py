#!/usr/bin/env python3
"""
NVIDIA Sortformer speaker diarization script.
Uses diar_streaming_sortformer_4spk-v2 for optimized 4-speaker diarization.
"""

import argparse
import json
import sys
import os
from pathlib import Path
import torch

try:
    from nemo.collections.asr.models import SortformerEncLabelModel
except ImportError:
    print("Error: NeMo not found. Please install nemo_toolkit[asr]")
    sys.exit(1)


def diarize_audio(
    audio_path: str,
    output_file: str,
    batch_size: int = 1,
    device: str = None,
    max_speakers: int = 4,
    output_format: str = "rttm",
    streaming_mode: bool = False,
    chunk_length_s: float = 30.0,
):
    """
    Perform speaker diarization using NVIDIA's Sortformer model.
    """
    if device is None or device == "auto":
        if torch.cuda.is_available():
            device = "cuda"

        else:
            device = "cpu"

    print(f"Using device: {device}")
    print(f"Loading NVIDIA Sortformer diarization model...")

    # Determine model path
    model_filename = "diar_streaming_sortformer_4spk-v2.nemo"
    model_path = None

    # Locate project root: derived from VIRTUAL_ENV, which is set by `uv run` to path/.venv
    virtual_env = os.environ.get("VIRTUAL_ENV")
    if not virtual_env:
        print("Error: VIRTUAL_ENV environment variable not set. Script must be run with 'uv run'.")
        sys.exit(1)

    project_root = os.path.dirname(virtual_env)
    model_path = os.path.join(project_root, model_filename)

    try:
        if not os.path.exists(model_path):
            print(f"Error: Model file not found: {model_filename} in project root: {project_root}")
            sys.exit(1)

        # Load from local file
        print(f"Loading model from path: {model_path}")
        diar_model = SortformerEncLabelModel.restore_from(
            restore_path=model_path,
            map_location=device,
            strict=False,
        )

        # Switch to inference mode
        diar_model.eval()
        print("Model loaded successfully")

    except Exception as e:
        print(f"Error loading model: {e}")
        sys.exit(1)

    print(f"Processing audio file: {audio_path}")

    # Verify audio file exists
    if not os.path.exists(audio_path):
        print(f"Error: Audio file not found: {audio_path}")
        sys.exit(1)

    try:
        # Run diarization
        print(f"Running diarization with batch_size={batch_size}, max_speakers={max_speakers}")

        if streaming_mode:
            print(f"Using streaming mode with chunk_length_s={chunk_length_s}")
            # Note: Streaming mode implementation would go here
            # For now, use standard diarization
            predicted_segments = diar_model.diarize(audio=audio_path, batch_size=batch_size)
        else:
            predicted_segments = diar_model.diarize(audio=audio_path, batch_size=batch_size)

        print(f"Diarization completed. Found segments: {len(predicted_segments)}")

        # Process and save results
        save_results(predicted_segments, output_file, audio_path, output_format)

    except Exception as e:
        print(f"Error during diarization: {e}")
        sys.exit(1)


def save_results(segments, output_file: str, audio_path: str, output_format: str):
    """
    Save diarization results to output file.
    Supports both JSON and RTTM formats based on output_format parameter.
    """
    output_path = Path(output_file)

    if output_format == "rttm":
        save_rttm_format(segments, output_file, audio_path)
    else:
        save_json_format(segments, output_file, audio_path)


def save_json_format(segments, output_file: str, audio_path: str):
    """Save results in JSON format."""
    results = {
        "audio_file": audio_path,
        "model": "nvidia/diar_streaming_sortformer_4spk-v2",
        "segments": [],
    }

    # Handle the case where segments is a list containing a single list of string entries
    if len(segments) == 1 and isinstance(segments[0], list):
        segments = segments[0]

    # Convert segments to JSON format
    speakers = set()
    for i, segment in enumerate(segments):
        try:
            # Handle different possible segment formats
            if isinstance(segment, str):
                # String format: "start end speaker_id"
                parts = segment.strip().split()
                if len(parts) >= 3:
                    segment_data = {
                        "start": float(parts[0]),
                        "end": float(parts[1]),
                        "speaker": str(parts[2]),
                        "duration": float(parts[1]) - float(parts[0]),
                        "confidence": 1.0,
                    }
                else:
                    print(f"Warning: Invalid string segment format: {segment}")
                    continue
            elif hasattr(segment, 'start') and hasattr(segment, 'end') and hasattr(segment, 'label'):
                # Standard pyannote-like format
                segment_data = {
                    "start": float(segment.start),
                    "end": float(segment.end),
                    "speaker": str(segment.label),
                    "duration": float(segment.end - segment.start),
                    "confidence": getattr(segment, 'confidence', 1.0),
                }
            elif isinstance(segment, (list, tuple)) and len(segment) >= 3:
                # List/tuple format: [start, end, speaker]
                segment_data = {
                    "start": float(segment[0]),
                    "end": float(segment[1]),
                    "speaker": str(segment[2]),
                    "duration": float(segment[1] - segment[0]),
                    "confidence": 1.0,
                }
            elif isinstance(segment, dict):
                # Dictionary format
                segment_data = {
                    "start": float(segment.get('start', 0)),
                    "end": float(segment.get('end', 0)),
                    "speaker": str(segment.get('speaker', segment.get('label', f'speaker_{i}'))),
                    "duration": float(segment.get('end', 0) - segment.get('start', 0)),
                    "confidence": float(segment.get('confidence', 1.0)),
                }
            else:
                # Fallback: try to extract attributes dynamically
                segment_data = {
                    "start": float(getattr(segment, 'start', 0)),
                    "end": float(getattr(segment, 'end', 0)),
                    "speaker": str(getattr(segment, 'label', getattr(segment, 'speaker', f'speaker_{i}'))),
                    "duration": float(getattr(segment, 'end', 0) - getattr(segment, 'start', 0)),
                    "confidence": float(getattr(segment, 'confidence', 1.0)),
                }

            results["segments"].append(segment_data)
            speakers.add(segment_data["speaker"])

        except Exception as e:
            print(f"Warning: Could not process segment {i}: {e}")
            print(f"Segment: {segment}")

    # Sort by start time
    if results["segments"]:
        results["segments"].sort(key=lambda x: x["start"])

    # Add summary statistics
    results["speakers"] = sorted(speakers)
    results["speaker_count"] = len(speakers)
    results["total_segments"] = len(results["segments"])
    results["total_duration"] = max(seg["end"] for seg in results["segments"]) if results["segments"] else 0

    with open(output_file, "w") as f:
        json.dump(results, f, indent=2)

    print(f"Results saved to: {output_file}")
    print(f"Found {len(speakers)} speakers: {', '.join(sorted(speakers))}")


def save_rttm_format(segments, output_file: str, audio_path: str):
    """Save results in RTTM (Rich Transcription Time Marked) format."""
    audio_filename = Path(audio_path).stem
    speakers = set()

    # Handle the case where segments is a list containing a single list of string entries
    if len(segments) == 1 and isinstance(segments[0], list):
        segments = segments[0]

    with open(output_file, "w") as f:
        for i, segment in enumerate(segments):
            try:
                # Handle different possible segment formats
                if isinstance(segment, str):
                    # String format: "start end speaker_id"
                    parts = segment.strip().split()
                    if len(parts) >= 3:
                        start = float(parts[0])
                        end = float(parts[1])
                        speaker = str(parts[2])
                    else:
                        print(f"Warning: Invalid string segment format: {segment}")
                        continue
                elif hasattr(segment, 'start') and hasattr(segment, 'end') and hasattr(segment, 'label'):
                    # Standard pyannote-like format
                    start = float(segment.start)
                    end = float(segment.end)
                    speaker = str(segment.label)
                elif isinstance(segment, (list, tuple)) and len(segment) >= 3:
                    # List/tuple format: [start, end, speaker]
                    start = float(segment[0])
                    end = float(segment[1])
                    speaker = str(segment[2])
                elif isinstance(segment, dict):
                    # Dictionary format
                    start = float(segment.get('start', 0))
                    end = float(segment.get('end', 0))
                    speaker = str(segment.get('speaker', segment.get('label', f'speaker_{i}')))
                else:
                    # Fallback: try to extract attributes dynamically
                    start = float(getattr(segment, 'start', 0))
                    end = float(getattr(segment, 'end', 0))
                    speaker = str(getattr(segment, 'label', getattr(segment, 'speaker', f'speaker_{i}')))

                duration = end - start
                speakers.add(speaker)

                # RTTM format: SPEAKER <filename> <channel> <start> <duration> <NA> <NA> <speaker_id> <NA> <NA>
                line = f"SPEAKER {audio_filename} 1 {start:.3f} {duration:.3f} <NA> <NA> {speaker} <NA> <NA>\n"
                f.write(line)

            except Exception as e:
                print(f"Warning: Could not process segment {i} for RTTM: {e}")
                print(f"Segment: {segment}")

    print(f"RTTM results saved to: {output_file}")
    print(f"Found {len(speakers)} speakers: {', '.join(sorted(speakers))}")


def main():
    parser = argparse.ArgumentParser(
        description="Speaker diarization using NVIDIA Sortformer model (local model only)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
    # Basic diarization with JSON output
    python sortformer_diarize.py samples/sample.wav output.json

    # Generate RTTM format output
    python sortformer_diarize.py samples/sample.wav output.rttm

    # Specify device and batch size
    python sortformer_diarize.py --device cuda --batch-size 2 samples/sample.wav output.json

Note: This script requires diar_streaming_sortformer_4spk-v2.nemo to be in the same directory.
        """,
    )

    parser.add_argument("audio_file", help="Path to input audio file (WAV, FLAC, etc.)")
    parser.add_argument("output_file", help="Path to output file (.json for JSON format, .rttm for RTTM format)")
    parser.add_argument("--batch-size", type=int, default=1, help="Batch size for processing (default: 1)")
    parser.add_argument("--device", choices=["cuda", "cpu", "auto"], default="auto", help="Device to use for inference (default: auto-detect)")
    parser.add_argument("--max-speakers", type=int, default=4, help="Maximum number of speakers (default: 4, optimized for this model)")
    parser.add_argument("--output-format", choices=["json", "rttm"], help="Output format (auto-detected from file extension if not specified)")
    parser.add_argument("--streaming", action="store_true", help="Enable streaming mode")
    parser.add_argument("--chunk-length-s", type=float, default=30.0, help="Chunk length in seconds for streaming mode (default: 30.0)")

    args = parser.parse_args()

    # Validate inputs
    if not os.path.exists(args.audio_file):
        print(f"Error: Audio file not found: {args.audio_file}")
        sys.exit(1)

    # Auto-detect output format from file extension if not specified
    if args.output_format is None:
        if args.output_file.lower().endswith('.rttm'):
            output_format = "rttm"
        else:
            output_format = "json"
    else:
        output_format = args.output_format

    # Create output directory if it doesn't exist
    output_dir = Path(args.output_file).parent
    output_dir.mkdir(parents=True, exist_ok=True)

    device = None if args.device == "auto" else args.device

    # Run diarization
    diarize_audio(
        audio_path=args.audio_file,
        output_file=args.output_file,
        batch_size=args.batch_size,
        device=device,
        max_speakers=args.max_speakers,
        output_format=output_format,
        streaming_mode=args.streaming,
        chunk_length_s=args.chunk_length_s,
    )


if __name__ == "__main__":
    main()
