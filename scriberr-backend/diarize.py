#!/usr/bin/env python3
"""
Dedicated speaker diarization script using whisperx
This script provides more robust diarization than the built-in whisperx diarization
"""

import whisperx
import gc
import json
import sys
import os
import argparse
from pathlib import Path
import torch


def diarize_audio(
    audio_file,
    model_size="large-v2",
    min_speakers=1,
    max_speakers=2,
    batch_size=16,
    compute_type="float16",
    output_file=None,
    hf_token=None,
):
    """
    Perform robust speaker diarization using whisperx

    Args:
        audio_file: Path to the audio file
        model_size: Whisper model size (tiny, base, small, medium, large-v2, large-v3)
        min_speakers: Minimum number of speakers
        max_speakers: Maximum number of speakers
        batch_size: Batch size for transcription (reduce if low on GPU mem)
        compute_type: Compute type (float16, int8)
        output_file: Output JSON file path
        hf_token: HuggingFace token for pyannote.audio

    Returns:
        dict: Processed transcript with speaker diarization
    """

    device = "cuda" if torch.cuda.is_available() else "cpu"
    print(f"Using device: {device}")

    if device == "cpu":
        compute_type = "int8"  # Force int8 for CPU
        batch_size = 1  # Reduce batch size for CPU

    try:
        # 1. Transcribe with original whisper (batched)
        print("Loading whisper model...")
        model = whisperx.load_model(model_size, device, compute_type=compute_type)

        print("Loading audio...")
        audio = whisperx.load_audio(audio_file)

        print("Transcribing audio...")
        result = model.transcribe(audio, batch_size=batch_size)
        print(f"Initial transcription complete. Segments: {len(result['segments'])}")

        # Clean up model to free GPU memory
        del model
        gc.collect()
        if device == "cuda":
            torch.cuda.empty_cache()

        # 2. Align whisper output
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

        # 3. Assign speaker labels
        print("Loading diarization model...")
        if hf_token:
            diarize_model = whisperx.diarize.DiarizationPipeline(
                use_auth_token=hf_token, device=device
            )
        else:
            # Try to use environment variable
            hf_token = os.getenv("HF_TOKEN")
            if hf_token:
                diarize_model = whisperx.diarize.DiarizationPipeline(
                    use_auth_token=hf_token, device=device
                )
            else:
                print("Warning: No HF token provided. Using default diarization.")
                diarize_model = whisperx.diarize.DiarizationPipeline(device=device)

        print("Performing speaker diarization...")
        diarize_segments = diarize_model(
            audio, min_speakers=min_speakers, max_speakers=max_speakers
        )
        print(f"Diarization complete. Speaker segments: {len(diarize_segments)}")

        print("Assigning speaker labels to words...")
        result = whisperx.assign_word_speakers(diarize_segments, result)
        print(f"Speaker assignment complete. Final segments: {len(result['segments'])}")

        # Clean up diarization model
        del diarize_model
        gc.collect()
        if device == "cuda":
            torch.cuda.empty_cache()

        # 4. Post-process the result for better robustness
        result = post_process_diarization(result, min_speakers, max_speakers)

        # 5. Save to file if specified
        if output_file:
            print(f"Saving transcript to {output_file}")
            with open(output_file, "w", encoding="utf-8") as f:
                json.dump(result, f, indent=2, ensure_ascii=False)

        return result

    except Exception as e:
        print(f"Error during diarization: {e}")
        raise


def post_process_diarization(result, min_speakers, max_speakers):
    """
    Post-process diarization results for better robustness
    """
    segments = result.get("segments", [])
    if not segments:
        return result

    print("Post-processing diarization results...")

    # 1. Ensure all segments have speaker labels
    segments = assign_missing_speakers(segments)

    # 2. Split segments by speaker changes
    segments = split_segments_by_speaker(segments)

    # 3. Fill gaps between segments
    segments = fill_gaps(segments)

    # 4. Merge adjacent segments from same speaker
    segments = merge_adjacent_segments(segments)

    result["segments"] = segments
    print(f"Post-processing complete. Final segments: {len(segments)}")

    return result


def assign_missing_speakers(segments):
    """Assign speaker labels to segments that don't have them"""
    if not segments:
        return segments

    # Find the most common speaker
    speaker_counts = {}
    for segment in segments:
        speaker = segment.get("speaker")
        if speaker:
            speaker_counts[speaker] = speaker_counts.get(speaker, 0) + 1

    # Find dominant speaker
    dominant_speaker = "SPEAKER_00"
    max_count = 0
    for speaker, count in speaker_counts.items():
        if count > max_count:
            dominant_speaker = speaker
            max_count = count

    # Assign missing speakers
    for segment in segments:
        if not segment.get("speaker"):
            segment["speaker"] = dominant_speaker

    return segments


def split_segments_by_speaker(segments):
    """Split segments that contain multiple speakers"""
    if not segments:
        return segments

    new_segments = []

    for segment in segments:
        words = segment.get("words", [])
        if not words:
            new_segments.append(segment)
            continue

        # Group words by speaker
        speaker_groups = {}
        for word in words:
            speaker = word.get("speaker", segment.get("speaker", "SPEAKER_00"))
            if speaker not in speaker_groups:
                speaker_groups[speaker] = []
            speaker_groups[speaker].append(word)

        # Create separate segments for each speaker
        for speaker, speaker_words in speaker_groups.items():
            if not speaker_words:
                continue

            # Sort words by start time
            speaker_words.sort(key=lambda w: w.get("start", 0))

            # Create text from words
            text = " ".join(word.get("word", "") for word in speaker_words)

            new_segment = {
                "start": speaker_words[0].get("start", segment.get("start", 0)),
                "end": speaker_words[-1].get("end", segment.get("end", 0)),
                "text": text,
                "words": speaker_words,
                "speaker": speaker,
            }

            new_segments.append(new_segment)

    # Sort segments by start time
    new_segments.sort(key=lambda s: s.get("start", 0))

    return new_segments


def fill_gaps(segments):
    """Fill gaps between segments with silence markers"""
    if len(segments) < 2:
        return segments

    new_segments = []
    gap_threshold = 0.5  # 500ms
    gaps_added = 0

    for i, segment in enumerate(segments):
        new_segments.append(segment)

        # Check for gap to next segment
        if i < len(segments) - 1:
            next_segment = segments[i + 1]
            gap = next_segment.get("start", 0) - segment.get("end", 0)

            if gap > gap_threshold:
                # Determine speaker for gap
                current_duration = segment.get("end", 0) - segment.get("start", 0)
                next_duration = next_segment.get("end", 0) - next_segment.get(
                    "start", 0
                )

                gap_speaker = segment.get("speaker", "SPEAKER_00")
                if next_duration > current_duration:
                    gap_speaker = next_segment.get("speaker", "SPEAKER_00")

                gap_segment = {
                    "start": segment.get("end", 0),
                    "end": next_segment.get("start", 0),
                    "text": "[silence]",
                    "words": [],
                    "speaker": gap_speaker,
                }

                new_segments.append(gap_segment)
                gaps_added += 1

    if gaps_added > 0:
        print(f"Added {gaps_added} gap segments")

    return new_segments


def merge_adjacent_segments(segments):
    """Merge adjacent segments from the same speaker"""
    if len(segments) < 2:
        return segments

    new_segments = []
    merge_threshold = 0.1  # 100ms

    i = 0
    while i < len(segments):
        current_segment = segments[i].copy()

        # Try to merge with next segments
        while i < len(segments) - 1:
            next_segment = segments[i + 1]

            if (
                current_segment.get("speaker") == next_segment.get("speaker")
                and (next_segment.get("start", 0) - current_segment.get("end", 0))
                <= merge_threshold
            ):
                # Merge segments
                current_segment["end"] = next_segment.get("end", 0)
                current_segment["text"] = (
                    current_segment.get("text", "") + " " + next_segment.get("text", "")
                )
                current_segment["words"].extend(next_segment.get("words", []))
                i += 1
            else:
                break

        new_segments.append(current_segment)
        i += 1

    return new_segments


def main():
    parser = argparse.ArgumentParser(
        description="Robust speaker diarization using whisperx"
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
        result = diarize_audio(
            audio_file=args.audio_file,
            model_size=args.model,
            min_speakers=args.min_speakers,
            max_speakers=args.max_speakers,
            batch_size=args.batch_size,
            compute_type=args.compute_type,
            output_file=args.output,
            hf_token=args.hf_token,
        )

        print(f"Diarization completed successfully!")
        print(f"Output saved to: {args.output}")
        print(f"Total segments: {len(result.get('segments', []))}")

    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
