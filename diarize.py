import argparse
import json
import os
import whisperx

# Configure environment variables for HuggingFace
os.environ["HF_HUB_DISABLE_TELEMETRY"] = "1"
os.environ["TRUST_REMOTE_CODE"] = "1"

# Use HuggingFace token from environment variable if available
hf_token = os.environ.get("HUGGINGFACE_TOKEN", "")

def diarize_transcript(audio_file, transcript, device="cpu", model_name="pyannote/speaker-diarization-3.1"):
    """
    Performs speaker diarization on a transcript
    
    Args:
        audio_file (str): Path to the audio file
        transcript (dict): WhisperX transcript result
        device (str): Device to run diarization on (e.g., 'cpu' or 'cuda')
        model_name (str): Diarization model to use
        
    Returns:
        dict: Transcript with speaker labels
    """
    try:
        print(f"Loading diarization model: {model_name}")
        # Load diarization pipeline with HF auth token
        diarize_model = whisperx.DiarizationPipeline(
            model_name=model_name,
            device=device,
            use_auth_token=hf_token if hf_token else None
        )
        print("Diarization model loaded successfully")
        
        # Run diarization on audio file
        print(f"Running diarization on {audio_file}")
        diarize_segments = diarize_model(audio_file)
        print("Diarization completed")
        
        # Assign speaker labels to transcript
        print("Assigning speaker labels to transcript")
        diarized_result = whisperx.assign_word_speakers(diarize_segments, transcript)
        print("Speaker labels assigned")
        
        # Post-process segments to improve speaker boundaries
        if "segments" in diarized_result:
            # First pass: establish dominant speaker for each segment
            print("Post-processing segment speakers")
            for segment in diarized_result["segments"]:
                if "words" in segment and len(segment["words"]) > 0:
                    # Count speakers in words
                    speaker_counts = {}
                    for word in segment["words"]:
                        if "speaker" in word:
                            speaker = word.get("speaker", "UNKNOWN")
                            # Clean up speaker labels
                            if not speaker or speaker == "UNKNOWN" or speaker == "unknown":
                                speaker = "SPEAKER_00"
                            if speaker not in speaker_counts:
                                speaker_counts[speaker] = 0
                            speaker_counts[speaker] += 1
                    
                    # Find the dominant speaker
                    if speaker_counts:
                        dominant_speaker = max(speaker_counts.items(), key=lambda x: x[1])[0]
                        segment["speaker"] = dominant_speaker
                    else:
                        segment["speaker"] = "SPEAKER_00"  # Default if no speaker found
            
            # Second pass: group consecutive segments with the same speaker
            print("Grouping segments from the same speaker")
            processed_segments = []
            current_segment = None
            
            # Minimum time gap (in seconds) to consider segments from the same speaker as separate
            min_speaker_gap = 2.0  
            
            for segment in diarized_result["segments"]:
                if current_segment is None:
                    current_segment = segment.copy()
                elif (segment.get("speaker") == current_segment.get("speaker") and 
                        (segment.get("start", 0) - current_segment.get("end", 0)) < min_speaker_gap):
                    # Merge with previous segment if same speaker and close enough
                    current_segment["end"] = segment["end"]
                    current_segment["text"] += " " + segment["text"]
                    if "words" in segment and "words" in current_segment:
                        current_segment["words"].extend(segment["words"])
                else:
                    # Different speaker or too far apart, add the current segment and start a new one
                    processed_segments.append(current_segment)
                    current_segment = segment.copy()
            
            # Add the last segment
            if current_segment is not None:
                processed_segments.append(current_segment)
            
            # Final pass: ensure reasonable segment durations
            print("Refining segment durations")
            final_segments = []
            max_segment_duration = 15.0  # Maximum segment duration in seconds
            
            for segment in processed_segments:
                duration = segment.get("end", 0) - segment.get("start", 0)
                if duration > max_segment_duration and "words" in segment and len(segment["words"]) > 3:
                    # Split long segments at natural sentence breaks
                    split_points = []
                    words = segment["words"]
                    
                    # Find potential split points at punctuation
                    for i, word in enumerate(words):
                        if i > 0 and any(punct in word.get("text", "") for punct in ['.', '!', '?']):
                            split_points.append(i)
                    
                    if split_points:
                        # Create new segments at split points
                        last_idx = 0
                        for idx in split_points:
                            if idx - last_idx >= 3:  # Ensure at least 3 words per segment
                                sub_segment = segment.copy()
                                sub_segment["words"] = words[last_idx:idx+1]
                                sub_segment["start"] = words[last_idx].get("start", segment["start"])
                                sub_segment["end"] = words[idx].get("end", sub_segment["start"] + 1)
                                sub_segment["text"] = " ".join(w.get("text", "") for w in sub_segment["words"])
                                final_segments.append(sub_segment)
                                last_idx = idx + 1
                        
                        # Add remaining words if any
                        if last_idx < len(words):
                            sub_segment = segment.copy()
                            sub_segment["words"] = words[last_idx:]
                            sub_segment["start"] = words[last_idx].get("start", segment["start"])
                            sub_segment["end"] = segment["end"]
                            sub_segment["text"] = " ".join(w.get("text", "") for w in sub_segment["words"])
                            final_segments.append(sub_segment)
                    else:
                        final_segments.append(segment)
                else:
                    final_segments.append(segment)
            
            print(f"Diarization complete: found {len(final_segments)} segments with speakers")
            return {"segments": final_segments}
        else:
            print("No segments in diarized result, returning original")
            # Fallback to original segments
            return diarized_result
    except Exception as e:
        print(f"Diarization failed: {str(e)}")
        # Set speaker labels to 'unknown' on failure
        for segment in transcript["segments"]:
            segment["speaker"] = "unknown"
        return transcript

def main():
    parser = argparse.ArgumentParser(
        description="Diarization script for WhisperX transcripts"
    )
    parser.add_argument(
        "--audio-file",
        type=str,
        required=True,
        help="Path to the audio file to diarize",
    )
    parser.add_argument(
        "--transcript-file",
        type=str,
        required=True,
        help="Path to the WhisperX transcript JSON file",
    )
    parser.add_argument(
        "--output-file",
        type=str,
        default="diarized-transcript.json",
        help="Path to the output JSON file",
    )
    parser.add_argument(
        "--device",
        type=str,
        default="cpu",
        help="Device to run diarization on (e.g., 'cpu' or 'cuda')",
    )
    parser.add_argument(
        "--diarization-model",
        type=str,
        default="pyannote/speaker-diarization-3.1",
        help="Speaker diarization model to use",
    )
    
    args = parser.parse_args()
    
    # Load transcript
    with open(args.transcript_file, "r", encoding="utf-8") as f:
        transcript = json.load(f)
    
    # Process diarization
    diarized_transcript = diarize_transcript(
        args.audio_file, 
        transcript,
        args.device,
        args.diarization_model
    )
    
    # Write the result to the output file
    with open(args.output_file, "w", encoding="utf-8") as f:
        json.dump(diarized_transcript, f, indent=2, ensure_ascii=False)
    
    print(f"Diarized transcript saved to {args.output_file}")

if __name__ == "__main__":
    main()