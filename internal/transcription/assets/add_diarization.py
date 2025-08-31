#!/usr/bin/env python3
"""
Add diarization to existing WhisperX transcript.

Usage:
  python add_diarization.py <audio_path> <transcript_json> <output_json> \
    --device <cpu|cuda> [--diarize_model <model>] [--min_speakers N] [--max_speakers N] \
    [--speaker_embeddings] [--hf_token TOKEN]

Takes an existing WhisperX transcript and adds speaker diarization to it.
"""
import argparse
import json
import os
import sys


def main():
    try:
        import whisperx
        import pandas as pd
    except Exception as e:
        print(f"Failed to import dependencies: {e}", file=sys.stderr)
        sys.exit(1)

    p = argparse.ArgumentParser()
    p.add_argument("audio_path", help="Path to audio file")
    p.add_argument("transcript_json", help="Path to existing WhisperX transcript JSON")
    p.add_argument("output_json", help="Path to output JSON with diarization")
    
    # Diarization parameters
    p.add_argument("--device", default="cpu", help="Device to use (cpu/cuda)")
    p.add_argument("--diarize_model", default="pyannote/speaker-diarization-3.1", help="Diarization model")
    p.add_argument("--min_speakers", type=int, default=None, help="Minimum speakers")
    p.add_argument("--max_speakers", type=int, default=None, help="Maximum speakers")
    p.add_argument("--speaker_embeddings", action="store_true", help="Include speaker embeddings")
    p.add_argument("--hf_token", default=None, help="HuggingFace token")
    
    args = p.parse_args()

    # Load existing transcript
    if not os.path.exists(args.transcript_json):
        print(f"Transcript file not found: {args.transcript_json}", file=sys.stderr)
        sys.exit(1)
        
    with open(args.transcript_json, 'r', encoding='utf-8') as f:
        result = json.load(f)
    
    print(f"Loaded transcript with {len(result.get('segments', []))} segments")

    # Load audio for diarization
    audio = whisperx.load_audio(args.audio_path)
    device = args.device

    # Perform diarization
    try:
        print("Starting diarization...")
        diarize_model = whisperx.diarize.DiarizationPipeline(
            model_name=args.diarize_model,
            use_auth_token=args.hf_token,
            device=device
        )
        
        # Run diarization with speaker constraints and embeddings
        if args.speaker_embeddings:
            diarize_segments, speaker_embeddings = diarize_model(
                audio, 
                min_speakers=args.min_speakers, 
                max_speakers=args.max_speakers,
                return_embeddings=True
            )
            # Assign speakers with embeddings
            result = whisperx.assign_word_speakers(diarize_segments, result, speaker_embeddings)
        else:
            diarize_segments = diarize_model(
                audio, 
                min_speakers=args.min_speakers, 
                max_speakers=args.max_speakers
            )
            # Assign speakers to transcript segments and words
            result = whisperx.assign_word_speakers(diarize_segments, result)
            
        print("Diarization completed successfully")
        
        # Count speakers found
        speakers = set()
        for segment in result.get('segments', []):
            if 'speaker' in segment and segment['speaker']:
                speakers.add(segment['speaker'])
        print(f"Found {len(speakers)} speakers: {sorted(speakers)}")
        
    except Exception as e:
        print(f"Diarization failed: {e}", file=sys.stderr)
        sys.exit(1)

    # Save result
    os.makedirs(os.path.dirname(args.output_json), exist_ok=True)
    with open(args.output_json, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)

    print(f"Wrote diarized transcript to {args.output_json}")


if __name__ == "__main__":
    main()