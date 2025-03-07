import argparse
import json
import os
import whisperx

# Configure environment variables for HuggingFace
os.environ["HF_HUB_DISABLE_TELEMETRY"] = "1"
os.environ["TRUST_REMOTE_CODE"] = "1"

# Use HuggingFace token from environment variable if available
hf_token = os.environ.get("HF_API_KEY", "")

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
        diarize_model = whisperx.DiarizationPipeline(
            model_name=model_name,
            device=device,
            use_auth_token=hf_token if hf_token else None
        )
        print("Diarization model loaded")

        print(f"Diarizing {audio_file}")
        diarize_segments = diarize_model(audio_file)
        print(f"Diarization produced {len(diarize_segments)} segments")

        print("Assigning speaker labels")
        diarized_result = whisperx.assign_word_speakers(diarize_segments, transcript)

        # Fallback: if the first segment’s speaker is missing or "unknown", assign sequential labels
        segments = diarized_result.get("segments", [])
        if segments:
            first_speaker = segments[0].get("speaker", "").strip().lower()
            if not first_speaker or first_speaker == "unknown":
                print("No proper speaker labels found in diarized result; assigning sequential speaker labels.")
                for i, segment in enumerate(segments):
                    segment["speaker"] = f"Speaker_{i+1:02d}"
                # No further post-processing is needed now.
                return { "segments": segments }

        # Post-process segments to improve speaker boundaries
        if "segments" in diarized_result and len(diarized_result["segments"]) > 0:
            # Only perform detailed post-processing if at least one segment already has a non-fallback speaker label.
            first_speaker = diarized_result["segments"][0].get("speaker", "")
            if first_speaker.lower() not in ["", "unknown"]:
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

                # Bypass grouping to preserve all detected speaker turns without merging
                print("Skipping grouping; using all diarization segments as-is")
                return {"segments": diarized_result["segments"]}
    except Exception as e:
        print(f"Diarization failed: {str(e)}")
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
