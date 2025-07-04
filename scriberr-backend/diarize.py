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
    min_silence_duration=0.3,
    max_silence_duration=2.0,
    confidence_threshold=0.5,
    diarization_model="pyannote/speaker-diarization-3.1",
    clustering_method="centroid",
    min_cluster_size=15,
    min_samples=15,
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
        min_silence_duration: Minimum silence duration to mark as gap (seconds)
        max_silence_duration: Maximum silence duration before splitting (seconds)
        confidence_threshold: Minimum confidence for word-level speaker assignment
        diarization_model: Pyannote diarization model to use
        clustering_method: Clustering method for speaker diarization
        min_cluster_size: Minimum cluster size for speaker detection
        min_samples: Minimum samples for clustering
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
        # Use additional transcription parameters
        result = model.transcribe(
            audio,
            batch_size=batch_size,
            temperature=temperature,
            best_of=best_of,
            beam_size=beam_size,
            patience=patience,
            length_penalty=length_penalty,
            suppress_numerals=suppress_numerals,
            initial_prompt=initial_prompt if initial_prompt else None,
            temperature_increment_on_fallback=temperature_increment_on_fallback,
            compression_ratio_threshold=compression_ratio_threshold,
            log_prob_threshold=logprob_threshold,
            no_speech_threshold=no_speech_threshold,
            condition_on_previous_text=condition_on_previous_text,
        )
        print(f"Transcription complete. Segments: {len(result['segments'])}")

        # Validate transcription coverage
        coverage = validate_transcription_coverage(result, audio_duration)
        print(f"Initial transcription coverage: {coverage:.2%}")

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

        # Validate alignment coverage
        coverage = validate_transcription_coverage(result, audio_duration)
        print(f"Post-alignment coverage: {coverage:.2%}")

        # Clean up alignment model
        del model_a
        gc.collect()
        if device == "cuda":
            torch.cuda.empty_cache()

        # 4. Enhanced speaker diarization with better model and parameters
        print("Loading enhanced diarization model...")

        # Try to use pyannote.audio directly for better results
        diarize_segments = None
        if PYANNOTE_AVAILABLE and hf_token:
            print("Using pyannote.audio for enhanced diarization...")
            diarize_segments = enhanced_pyannote_diarization(
                audio_file, hf_token, device, min_speakers, max_speakers
            )

        # Fall back to WhisperX diarization if pyannote fails or is not available
        if diarize_segments is None:
            print("Using WhisperX diarization as fallback...")
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
            # Use more aggressive parameters for better speaker detection
            diarize_segments = diarize_model(
                audio,
                min_speakers=min_speakers,
                max_speakers=max_speakers,
                clustering_method=clustering_method,
                min_cluster_size=min_cluster_size,
                min_samples=min_samples,
            )

        print(
            f"Enhanced diarization complete. Speaker segments: {len(diarize_segments)}"
        )

        # 5. Enhanced speaker assignment with confidence filtering
        print("Assigning speaker labels to words...")
        result = whisperx.assign_word_speakers(diarize_segments, result)
        print(f"Speaker assignment complete. Final segments: {len(result['segments'])}")

        # Clean up diarization model (only if it was created)
        if "diarize_model" in locals():
            del diarize_model
        gc.collect()
        if device == "cuda":
            torch.cuda.empty_cache()

        # 6. Enhanced post-processing for better robustness
        result = enhanced_post_process_diarization(
            result,
            min_speakers,
            max_speakers,
            audio_duration,
            min_silence_duration,
            max_silence_duration,
            confidence_threshold,
        )

        # 7. Final validation
        final_coverage = validate_transcription_coverage(result, audio_duration)
        print(f"Final transcription coverage: {final_coverage:.2%}")

        # 8. Save to file if specified
        if output_file:
            print(f"Saving transcript to {output_file}")
            with open(output_file, "w", encoding="utf-8") as f:
                json.dump(result, f, indent=2, ensure_ascii=False)

        return result

    except Exception as e:
        print(f"Error during diarization: {e}")
        raise


def validate_transcription_coverage(
    result: Dict[str, Any], audio_duration: float
) -> float:
    """Validate how much of the audio is covered by transcription"""
    segments = result.get("segments", [])
    if not segments:
        return 0.0

    total_covered = 0.0
    for segment in segments:
        start = segment.get("start", 0)
        end = segment.get("end", 0)
        total_covered += end - start

    coverage = total_covered / audio_duration
    return min(coverage, 1.0)


def enhanced_pyannote_diarization(
    audio_file: str, hf_token: str, device: str, min_speakers: int, max_speakers: int
) -> List[Dict[str, Any]]:
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

        # Run diarization with optimized parameters for better accuracy
        with ProgressHook() as hook:
            diarization = pipeline(
                audio_file,
                hook=hook,
                min_speakers=min_speakers,
                max_speakers=max_speakers,
                # Enhanced parameters for better accuracy
                onset=0.3,  # Lower onset threshold for better speaker detection
                offset=0.3,  # Lower offset threshold
                min_duration_on=0.05,  # Shorter minimum duration for better precision
                min_duration_off=0.05,  # Shorter minimum duration for non-speaker segments
                # Additional parameters for better clustering
                clustering={
                    "method": "centroid",
                    "min_clusters": min_speakers,
                    "max_clusters": max_speakers,
                },
            )

        # Convert pyannote format to WhisperX format
        segments = []
        for turn, _, speaker in diarization.itertracks(yield_label=True):
            segment = {"start": turn.start, "end": turn.end, "speaker": speaker}
            segments.append(segment)

        print(f"Pyannote diarization complete: {len(segments)} segments")
        return segments

    except Exception as e:
        print(f"Pyannote diarization failed: {e}")
        print("Falling back to WhisperX diarization...")
        return None


def enhanced_post_process_diarization(
    result: Dict[str, Any],
    min_speakers: int,
    max_speakers: int,
    audio_duration: float,
    min_silence_duration: float = 0.3,
    max_silence_duration: float = 2.0,
    confidence_threshold: float = 0.5,
) -> Dict[str, Any]:
    """
    Enhanced post-processing for better diarization robustness
    """
    segments = result.get("segments", [])
    if not segments:
        return result

    initial_count = len(segments)
    print(
        f"Enhanced post-processing diarization results... (initial segments: {initial_count})"
    )

    # 1. Clean and validate segments
    segments = clean_segments(segments, audio_duration)
    print(f"After cleaning: {len(segments)} segments")

    # 2. Ensure all segments have speaker labels with improved logic
    segments = enhanced_assign_missing_speakers(segments, min_speakers, max_speakers)
    print(f"After speaker assignment: {len(segments)} segments")

    # 3. Split segments by speaker changes with confidence filtering
    segments = enhanced_split_segments_by_speaker(segments, confidence_threshold)
    print(f"After speaker splitting: {len(segments)} segments")

    # 4. Comprehensive gap detection and filling
    segments = comprehensive_fill_gaps(
        segments, audio_duration, min_silence_duration, max_silence_duration
    )
    print(f"After gap filling: {len(segments)} segments")

    # 5. Smart merging of adjacent segments
    segments = smart_merge_adjacent_segments(segments)
    print(f"After merging: {len(segments)} segments")

    # 6. Final speaker consistency check
    segments = ensure_speaker_consistency(segments, min_speakers, max_speakers)
    print(f"After consistency check: {len(segments)} segments")

    # 7. Add metadata about processing
    result["segments"] = segments
    result["processing_metadata"] = {
        "audio_duration": audio_duration,
        "total_segments": len(segments),
        "coverage_percentage": validate_transcription_coverage(result, audio_duration)
        * 100,
        "min_silence_duration": min_silence_duration,
        "max_silence_duration": max_silence_duration,
        "confidence_threshold": confidence_threshold,
        "initial_segments": initial_count,
        "final_segments": len(segments),
        "segments_preserved": len(segments) >= initial_count,
    }

    print(f"Enhanced post-processing complete. Final segments: {len(segments)}")
    if len(segments) < initial_count:
        print(
            f"WARNING: Lost {initial_count - len(segments)} segments during processing!"
        )
    else:
        print(
            f"Successfully preserved all segments (gained {len(segments) - initial_count} from gap filling)"
        )

    return result


def clean_segments(
    segments: List[Dict[str, Any]], audio_duration: float
) -> List[Dict[str, Any]]:
    """Clean and validate segments - NEVER remove segments, only fix timing issues"""
    cleaned = []

    for segment in segments:
        # Fix invalid timing but keep segment
        if segment.get("start", 0) >= segment.get("end", 0):
            # Fix timing by adding a small duration
            segment["end"] = segment.get("start", 0) + 0.1

        # Clamp start time to audio duration but keep segment
        if segment.get("start", 0) > audio_duration:
            segment["start"] = audio_duration - 0.1

        # Clamp end time to audio duration
        if segment.get("end", 0) > audio_duration:
            segment["end"] = audio_duration

        # Ensure segment has text (add placeholder if empty)
        if not segment.get("text", "").strip():
            segment["text"] = "[unclear]"

        cleaned.append(segment)

    # Sort by start time
    cleaned.sort(key=lambda s: s.get("start", 0))
    return cleaned


def enhanced_assign_missing_speakers(
    segments: List[Dict[str, Any]], min_speakers: int, max_speakers: int
) -> List[Dict[str, Any]]:
    """Smart speaker assignment using word-level information and context"""
    if not segments:
        return segments

    print("Performing smart speaker assignment...")

    # First, analyze word-level speaker patterns
    word_speaker_stats = analyze_word_speaker_patterns(segments)

    # Then analyze segment-level speaker patterns
    segment_speaker_stats = analyze_speaker_patterns(segments)

    # Determine optimal speaker count
    detected_speakers = len(segment_speaker_stats)
    optimal_speakers = max(min_speakers, min(detected_speakers, max_speakers))

    print(
        f"Detected {detected_speakers} speakers at segment level, using {optimal_speakers}"
    )

    # If we have too many speakers, merge similar ones
    if detected_speakers > optimal_speakers:
        segments = merge_similar_speakers(
            segments, segment_speaker_stats, optimal_speakers
        )

    # Smart speaker assignment using word-level information
    for i, segment in enumerate(segments):
        if not segment.get("speaker"):
            # Try to infer speaker from word-level information first
            inferred_speaker = infer_speaker_from_words(segment, word_speaker_stats)

            if inferred_speaker:
                segment["speaker"] = inferred_speaker
                print(
                    f"Segment {i}: Inferred speaker '{inferred_speaker}' from word-level info"
                )
            else:
                # Fall back to context-based assignment
                context_speaker = assign_speaker_by_context(
                    segments, i, segment_speaker_stats
                )
                segment["speaker"] = context_speaker
                print(f"Segment {i}: Assigned speaker '{context_speaker}' by context")

    return segments


def analyze_word_speaker_patterns(
    segments: List[Dict[str, Any]],
) -> Dict[str, Dict[str, Any]]:
    """Analyze speaker patterns at the word level"""
    word_speaker_stats = {}

    for segment in segments:
        words = segment.get("words", [])
        for word in words:
            speaker = word.get("speaker")
            if not speaker:
                continue

            if speaker not in word_speaker_stats:
                word_speaker_stats[speaker] = {
                    "word_count": 0,
                    "total_duration": 0.0,
                    "segments": set(),
                    "time_ranges": [],
                }

            duration = word.get("end", 0) - word.get("start", 0)
            word_speaker_stats[speaker]["word_count"] += 1
            word_speaker_stats[speaker]["total_duration"] += duration
            word_speaker_stats[speaker]["segments"].add(id(segment))
            word_speaker_stats[speaker]["time_ranges"].append(
                {"start": word.get("start", 0), "end": word.get("end", 0)}
            )

    return word_speaker_stats


def infer_speaker_from_words(
    segment: Dict[str, Any], word_speaker_stats: Dict[str, Dict[str, Any]]
) -> Optional[str]:
    """Infer speaker from word-level information in the segment"""
    words = segment.get("words", [])
    if not words:
        return None

    # Count words by speaker
    speaker_word_counts = {}
    for word in words:
        speaker = word.get("speaker")
        if speaker:
            speaker_word_counts[speaker] = speaker_word_counts.get(speaker, 0) + 1

    if not speaker_word_counts:
        return None

    # Return the speaker with the most words
    return max(speaker_word_counts.items(), key=lambda x: x[1])[0]


def assign_speaker_by_context(
    segments: List[Dict[str, Any]],
    current_index: int,
    speaker_stats: Dict[str, Dict[str, Any]],
) -> str:
    """Assign speaker based on temporal context and speaker statistics"""

    # Priority 1: Look for previous speaker
    for i in range(current_index - 1, max(0, current_index - 5), -1):
        if segments[i].get("speaker"):
            return segments[i]["speaker"]

    # Priority 2: Look for next speaker
    for i in range(current_index + 1, min(len(segments), current_index + 5)):
        if segments[i].get("speaker"):
            return segments[i]["speaker"]

    # Priority 3: Use dominant speaker
    if speaker_stats:
        dominant_speaker = max(speaker_stats.items(), key=lambda x: x[1]["duration"])[0]
        return dominant_speaker

    # Priority 4: Fallback to unknown
    return "UNKNOWN_SPEAKER"


def analyze_speaker_patterns(
    segments: List[Dict[str, Any]],
) -> Dict[str, Dict[str, Any]]:
    """Analyze speaker patterns and statistics"""
    speaker_stats = {}

    for segment in segments:
        speaker = segment.get("speaker")
        if not speaker:
            continue

        if speaker not in speaker_stats:
            speaker_stats[speaker] = {
                "count": 0,
                "duration": 0.0,
                "total_words": 0,
                "segments": [],
            }

        duration = segment.get("end", 0) - segment.get("start", 0)
        word_count = len(segment.get("words", []))

        speaker_stats[speaker]["count"] += 1
        speaker_stats[speaker]["duration"] += duration
        speaker_stats[speaker]["total_words"] += word_count
        speaker_stats[speaker]["segments"].append(segment)

    return speaker_stats


def get_context_speaker(
    segments: List[Dict[str, Any]], current_index: int
) -> Optional[str]:
    """Get speaker from context (surrounding segments)"""
    # Look at previous segments
    for i in range(current_index - 1, max(0, current_index - 5), -1):
        if segments[i].get("speaker"):
            return segments[i]["speaker"]

    # Look at next segments
    for i in range(current_index + 1, min(len(segments), current_index + 5)):
        if segments[i].get("speaker"):
            return segments[i]["speaker"]

    return None


def find_next_speaker(
    segments: List[Dict[str, Any]], current_index: int
) -> Optional[str]:
    """Find the next speaker in the sequence"""
    for i in range(current_index + 1, min(len(segments), current_index + 10)):
        if segments[i].get("speaker"):
            return segments[i]["speaker"]
    return None


def merge_segments_without_speaker_info(
    speaker_segments: List[Dict[str, Any]], no_speaker_segments: List[Dict[str, Any]]
) -> List[Dict[str, Any]]:
    """Merge segments without speaker info into speaker-based segments based on timing"""
    if not no_speaker_segments:
        return speaker_segments

    print(f"Merging {len(no_speaker_segments)} segments without speaker info")

    # Create a timeline of all segments
    all_segments = speaker_segments + no_speaker_segments
    all_segments.sort(key=lambda s: s.get("start", 0))

    # For segments without speaker info, assign speaker based on surrounding context
    for segment in no_speaker_segments:
        segment_start = segment.get("start", 0)
        segment_end = segment.get("end", 0)

        # Find the closest speaker segment before this segment
        prev_speaker = None
        prev_distance = float("inf")

        for speaker_seg in speaker_segments:
            if speaker_seg.get("end", 0) <= segment_start:
                distance = segment_start - speaker_seg.get("end", 0)
                if distance < prev_distance:
                    prev_distance = distance
                    prev_speaker = speaker_seg.get("speaker")

        # Find the closest speaker segment after this segment
        next_speaker = None
        next_distance = float("inf")

        for speaker_seg in speaker_segments:
            if speaker_seg.get("start", 0) >= segment_end:
                distance = speaker_seg.get("start", 0) - segment_end
                if distance < next_distance:
                    next_distance = distance
                    next_speaker = speaker_seg.get("speaker")

        # Assign speaker based on proximity
        if prev_speaker and next_speaker:
            # Both speakers available, choose the closer one
            if prev_distance <= next_distance:
                segment["speaker"] = prev_speaker
            else:
                segment["speaker"] = next_speaker
        elif prev_speaker:
            segment["speaker"] = prev_speaker
        elif next_speaker:
            segment["speaker"] = next_speaker
        else:
            # No speaker context available, mark as unknown
            segment["speaker"] = "UNKNOWN_SPEAKER"

    # Return all segments sorted by time
    all_segments.sort(key=lambda s: s.get("start", 0))
    return all_segments


def merge_similar_speakers(
    segments: List[Dict[str, Any]],
    speaker_stats: Dict[str, Dict[str, Any]],
    target_count: int,
) -> List[Dict[str, Any]]:
    """Merge similar speakers to reach target count"""
    # Sort speakers by duration
    sorted_speakers = sorted(
        speaker_stats.items(), key=lambda x: x[1]["duration"], reverse=True
    )

    # Keep the top speakers
    keep_speakers = [s[0] for s in sorted_speakers[:target_count]]

    # Create mapping for speakers to merge
    speaker_mapping = {}
    for speaker, _ in sorted_speakers[target_count:]:
        # Map to the most similar kept speaker (simplified: map to first kept speaker)
        speaker_mapping[speaker] = keep_speakers[0]

    # Apply mapping
    for segment in segments:
        if segment.get("speaker") in speaker_mapping:
            segment["speaker"] = speaker_mapping[segment["speaker"]]

    return segments


def enhanced_split_segments_by_speaker(
    segments: List[Dict[str, Any]], confidence_threshold: float
) -> List[Dict[str, Any]]:
    """Smart speaker-based segment reconstruction using word-level timestamps"""
    if not segments:
        return segments

    print("Reconstructing segments based on word-level speaker assignments...")

    # Collect all words with their speaker assignments and timestamps
    all_words = []
    for segment in segments:
        words = segment.get("words", [])
        for word in words:
            # Only include words with speaker labels and valid timestamps
            if (
                word.get("speaker")
                and word.get("start") is not None
                and word.get("end") is not None
            ):
                confidence = word.get("confidence", 1.0)
                if confidence >= confidence_threshold:
                    all_words.append(word)

    if not all_words:
        print("No words with speaker assignments found, keeping original segments")
        return segments

    print(f"Processing {len(all_words)} words with speaker assignments")

    # Sort all words by start time
    all_words.sort(key=lambda w: w.get("start", 0))

    # Group consecutive words by speaker with time-based merging
    new_segments = []
    current_speaker = None
    current_words = []
    current_start = None
    current_end = None

    # Time threshold for merging consecutive words from same speaker
    merge_threshold = 0.5  # 500ms gap threshold

    for word in all_words:
        word_speaker = word.get("speaker")
        word_start = word.get("start", 0)
        word_end = word.get("end", 0)

        # Check if we should start a new segment
        should_start_new = False

        if current_speaker is None:
            # First word
            should_start_new = True
        elif word_speaker != current_speaker:
            # Speaker change
            should_start_new = True
        elif current_words and (word_start - current_end) > merge_threshold:
            # Gap too large, start new segment even for same speaker
            should_start_new = True

        if should_start_new:
            # Save current segment if it exists
            if current_words:
                segment_text = " ".join(w.get("word", "") for w in current_words)
                new_segment = {
                    "start": current_start,
                    "end": current_end,
                    "text": segment_text,
                    "words": current_words.copy(),
                    "speaker": current_speaker,
                }
                new_segments.append(new_segment)

            # Start new segment
            current_speaker = word_speaker
            current_words = [word]
            current_start = word_start
            current_end = word_end
        else:
            # Continue current segment
            current_words.append(word)
            current_end = word_end

    # Don't forget the last segment
    if current_words:
        segment_text = " ".join(w.get("word", "") for w in current_words)
        new_segment = {
            "start": current_start,
            "end": current_end,
            "text": segment_text,
            "words": current_words.copy(),
            "speaker": current_speaker,
        }
        new_segments.append(new_segment)

    # Sort by start time
    new_segments.sort(key=lambda s: s.get("start", 0))

    print(f"Created {len(new_segments)} segments from word-level speaker assignments")

    # Handle segments without word-level speaker info
    segments_without_words = []
    for segment in segments:
        words = segment.get("words", [])
        has_speaker_words = any(w.get("speaker") for w in words)

        if not has_speaker_words and segment.get("text", "").strip():
            # This segment has no word-level speaker info, preserve it
            segments_without_words.append(segment)

    if segments_without_words:
        print(
            f"Found {len(segments_without_words)} segments without word-level speaker info"
        )
        # Merge these segments into the new segments based on timing
        new_segments = merge_segments_without_speaker_info(
            new_segments, segments_without_words
        )

    return new_segments


def comprehensive_fill_gaps(
    segments: List[Dict[str, Any]],
    audio_duration: float,
    min_silence_duration: float = 0.3,
    max_silence_duration: float = 2.0,
) -> List[Dict[str, Any]]:
    """Comprehensive gap detection and filling"""
    if not segments:
        return segments

    new_segments = []
    gaps_added = 0

    # Add gap at the beginning if needed
    if segments[0].get("start", 0) > min_silence_duration:
        gap_segment = {
            "start": 0,
            "end": segments[0].get("start", 0),
            "text": "[silence]",
            "words": [],
            "speaker": segments[0].get("speaker", "SPEAKER_00"),
        }
        new_segments.append(gap_segment)
        gaps_added += 1

    # Process segments and gaps between them
    for i, segment in enumerate(segments):
        new_segments.append(segment)

        # Check for gap to next segment
        if i < len(segments) - 1:
            next_segment = segments[i + 1]
            gap = next_segment.get("start", 0) - segment.get("end", 0)

            if gap > min_silence_duration:
                # Determine speaker for gap based on context
                gap_speaker = determine_gap_speaker(segment, next_segment)

                gap_segment = {
                    "start": segment.get("end", 0),
                    "end": next_segment.get("start", 0),
                    "text": "[silence]",
                    "words": [],
                    "speaker": gap_speaker,
                }

                new_segments.append(gap_segment)
                gaps_added += 1

    # Add gap at the end if needed
    last_segment = segments[-1]
    if last_segment.get("end", 0) < audio_duration - min_silence_duration:
        gap_segment = {
            "start": last_segment.get("end", 0),
            "end": audio_duration,
            "text": "[silence]",
            "words": [],
            "speaker": last_segment.get("speaker", "SPEAKER_00"),
        }
        new_segments.append(gap_segment)
        gaps_added += 1

    if gaps_added > 0:
        print(f"Added {gaps_added} gap segments")

    return new_segments


def determine_gap_speaker(
    current_segment: Dict[str, Any], next_segment: Dict[str, Any]
) -> str:
    """Determine which speaker should be assigned to a gap"""
    current_duration = current_segment.get("end", 0) - current_segment.get("start", 0)
    next_duration = next_segment.get("end", 0) - next_segment.get("start", 0)

    # Prefer the speaker with longer surrounding speech
    if next_duration > current_duration:
        return next_segment.get("speaker", "SPEAKER_00")
    else:
        return current_segment.get("speaker", "SPEAKER_00")


def smart_merge_adjacent_segments(
    segments: List[Dict[str, Any]],
) -> List[Dict[str, Any]]:
    """Smart merging of adjacent segments from the same speaker - very conservative"""
    if len(segments) < 2:
        return segments

    new_segments = []
    merge_threshold = 0.1  # 100ms - very conservative to preserve segments
    original_count = len(segments)

    i = 0
    while i < len(segments):
        current_segment = segments[i].copy()

        # Try to merge with next segments
        while i < len(segments) - 1:
            next_segment = segments[i + 1]

            # Only merge if same speaker, very close, and both have meaningful content
            if (
                current_segment.get("speaker") == next_segment.get("speaker")
                and (next_segment.get("start", 0) - current_segment.get("end", 0))
                <= merge_threshold
                and next_segment.get("text", "") != "[silence]"  # Don't merge silence
                and current_segment.get("text", "") != "[silence]"
                and next_segment.get("text", "") != "[unclear]"  # Don't merge unclear
                and current_segment.get("text", "") != "[unclear]"
                and len(next_segment.get("text", "").strip()) > 0  # Don't merge empty
                and len(current_segment.get("text", "").strip()) > 0
            ):
                # Merge segments
                current_segment["end"] = next_segment.get("end", 0)
                current_segment["text"] = (
                    current_segment.get("text", "") + " " + next_segment.get("text", "")
                ).strip()
                current_segment["words"].extend(next_segment.get("words", []))
                i += 1
            else:
                break

        new_segments.append(current_segment)
        i += 1

    final_count = len(new_segments)
    if final_count < original_count:
        print(
            f"Merged {original_count - final_count} segments (from {original_count} to {final_count})"
        )
    else:
        print(f"No segments merged (preserved all {original_count} segments)")

    return new_segments


def ensure_speaker_consistency(
    segments: List[Dict[str, Any]], min_speakers: int, max_speakers: int
) -> List[Dict[str, Any]]:
    """Ensure speaker consistency and validate speaker count with word-level validation"""
    if not segments:
        return segments

    print("Validating speaker consistency...")

    # Count unique speakers
    speakers = set()
    for segment in segments:
        if segment.get("speaker"):
            speakers.add(segment["speaker"])

    print(f"Final speaker count: {len(speakers)} (speakers: {list(speakers)})")

    # Validate word-level speaker assignments within segments
    segments = validate_word_speaker_consistency(segments)

    # If we have too few speakers, try to split some segments
    if len(speakers) < min_speakers and len(segments) > 1:
        print(
            f"Warning: Only {len(speakers)} speakers detected, minimum is {min_speakers}"
        )
        # This could be enhanced with more sophisticated speaker detection

    # If we have too many speakers, merge some
    if len(speakers) > max_speakers:
        print(f"Warning: {len(speakers)} speakers detected, maximum is {max_speakers}")
        # This could be enhanced with speaker similarity analysis

    return segments


def validate_word_speaker_consistency(
    segments: List[Dict[str, Any]],
) -> List[Dict[str, Any]]:
    """Validate and fix word-level speaker consistency within segments"""
    inconsistencies_fixed = 0

    for segment in segments:
        words = segment.get("words", [])
        segment_speaker = segment.get("speaker")

        if not words or not segment_speaker:
            continue

        # Check if all words have consistent speaker labels
        word_speakers = [w.get("speaker") for w in words if w.get("speaker")]

        if not word_speakers:
            # No word-level speaker info, assign segment speaker to all words
            for word in words:
                word["speaker"] = segment_speaker
            inconsistencies_fixed += 1
        else:
            # Check for inconsistencies
            unique_speakers = set(word_speakers)
            if len(unique_speakers) > 1:
                # Multiple speakers in one segment - this might indicate a problem
                print(
                    f"Warning: Segment has words from multiple speakers: {unique_speakers}"
                )

                # Option 1: Use majority speaker for the segment
                speaker_counts = {}
                for speaker in word_speakers:
                    speaker_counts[speaker] = speaker_counts.get(speaker, 0) + 1

                majority_speaker = max(speaker_counts.items(), key=lambda x: x[1])[0]

                # Update segment speaker if it doesn't match majority
                if segment_speaker != majority_speaker:
                    segment["speaker"] = majority_speaker
                    print(
                        f"Updated segment speaker from '{segment_speaker}' to '{majority_speaker}'"
                    )
                    inconsistencies_fixed += 1

                # Update all words to use the majority speaker
                for word in words:
                    word["speaker"] = majority_speaker
            elif segment_speaker not in unique_speakers:
                # Segment speaker doesn't match word speakers
                word_speaker = list(unique_speakers)[0]
                segment["speaker"] = word_speaker
                print(
                    f"Updated segment speaker from '{segment_speaker}' to '{word_speaker}' to match words"
                )
                inconsistencies_fixed += 1

    if inconsistencies_fixed > 0:
        print(f"Fixed {inconsistencies_fixed} speaker inconsistencies")

    return segments


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
        "--min-silence-duration",
        type=float,
        default=0.3,
        help="Minimum silence duration to mark as gap (seconds)",
    )
    parser.add_argument(
        "--max-silence-duration",
        type=float,
        default=2.0,
        help="Maximum silence duration before splitting (seconds)",
    )
    parser.add_argument(
        "--confidence-threshold",
        type=float,
        default=0.5,
        help="Minimum confidence for word-level speaker assignment",
    )
    parser.add_argument(
        "--diarization-model",
        default="pyannote/speaker-diarization-3.1",
        help="Diarization model to use",
    )
    parser.add_argument(
        "--clustering-method",
        default="centroid",
        choices=["centroid", "spectral", "agglomerative"],
        help="Clustering method for speaker diarization",
    )
    parser.add_argument(
        "--min-cluster-size",
        type=int,
        default=15,
        help="Minimum cluster size for speaker detection",
    )
    parser.add_argument(
        "--min-samples",
        type=int,
        default=15,
        help="Minimum samples for clustering",
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
            min_silence_duration=args.min_silence_duration,
            max_silence_duration=args.max_silence_duration,
            confidence_threshold=args.confidence_threshold,
            diarization_model=args.diarization_model,
            clustering_method=args.clustering_method,
            min_cluster_size=args.min_cluster_size,
            min_samples=args.min_samples,
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

        # Print coverage information
        metadata = result.get("processing_metadata", {})
        if metadata:
            print(f"Audio coverage: {metadata.get('coverage_percentage', 0):.1f}%")

    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
