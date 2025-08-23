#!/usr/bin/env python3
"""
WhisperX Diarization Script
Implements speaker diarization using the WhisperX pipeline with pyannote.audio
"""
import argparse
import json
import sys
import os
import ssl
import whisperx
import gc
import torch

# Disable SSL verification for corporate firewalls
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
ssl._create_default_https_context = ssl._create_unverified_context

# Set environment variables to disable SSL verification
os.environ['CURL_CA_BUNDLE'] = ''
os.environ['REQUESTS_CA_BUNDLE'] = ''

def main():
    parser = argparse.ArgumentParser(description="Transcribe and diarize audio using WhisperX")
    
    # Required arguments
    parser.add_argument("audio_file", help="Path to the input audio file")
    parser.add_argument("output_file", help="Path to the output JSON file")
    
    # Model parameters
    parser.add_argument("--model", default="base", help="Whisper model size")
    parser.add_argument("--device", default="cpu", help="Device to use (cpu/cuda)")
    parser.add_argument("--compute_type", default="float32", help="Compute type (float16/float32/int8)")
    parser.add_argument("--batch_size", type=int, default=16, help="Batch size for processing")
    
    # Diarization parameters
    parser.add_argument("--diarize", action="store_true", help="Enable speaker diarization")
    parser.add_argument("--min_speakers", type=int, help="Minimum number of speakers")
    parser.add_argument("--max_speakers", type=int, help="Maximum number of speakers")
    parser.add_argument("--hf_token", help="Hugging Face token for diarization models")
    
    # Other parameters
    parser.add_argument("--language", help="Force language (leave empty for auto-detect)")
    
    args = parser.parse_args()
    
    # Check if CUDA is available
    if args.device == "cuda" and not torch.cuda.is_available():
        print("CUDA requested but not available, falling back to CPU")
        args.device = "cpu"
        args.compute_type = "float32"
    
    try:
        # 1. Transcribe with original whisper (batched)
        print(f"Loading Whisper model: {args.model}")
        model = whisperx.load_model(args.model, args.device, compute_type=args.compute_type)
        
        print(f"Loading audio: {args.audio_file}")
        audio = whisperx.load_audio(args.audio_file)
        
        print("Transcribing audio...")
        result = model.transcribe(audio, batch_size=args.batch_size, language=args.language)
        
        print(f"Detected language: {result.get('language', 'unknown')}")
        
        # Delete model to free up GPU memory if low on resources
        del model
        gc.collect()
        if args.device == "cuda":
            torch.cuda.empty_cache()
        
        # 2. Align whisper output
        print("Loading alignment model...")
        model_a, metadata = whisperx.load_align_model(
            language_code=result["language"], 
            device=args.device
        )
        result = whisperx.align(
            result["segments"], 
            model_a, 
            metadata, 
            audio, 
            args.device, 
            return_char_alignments=False
        )
        
        # Delete alignment model to free up memory
        del model_a
        gc.collect()
        if args.device == "cuda":
            torch.cuda.empty_cache()
        
        # 3. Perform speaker diarization if requested
        if args.diarize:
            print("Performing speaker diarization...")
            
            if not args.hf_token:
                print("Warning: No HuggingFace token provided. Diarization may fail for some models.")
            
            # Initialize diarization pipeline
            diarize_model = whisperx.diarize.DiarizationPipeline(
                use_auth_token=args.hf_token, 
                device=args.device
            )
            
            # Perform diarization
            diarize_kwargs = {}
            if args.min_speakers:
                diarize_kwargs['min_speakers'] = args.min_speakers
            if args.max_speakers:
                diarize_kwargs['max_speakers'] = args.max_speakers
            
            diarize_segments = diarize_model(audio, **diarize_kwargs)
            
            # Assign speaker labels to words
            result = whisperx.assign_word_speakers(diarize_segments, result)
            
            print(f"Found {len(set(seg.get('speaker') for seg in result['segments'] if seg.get('speaker')))} unique speakers")
        
        # 4. Save results
        print(f"Saving results to: {args.output_file}")
        
        # Ensure output directory exists
        os.makedirs(os.path.dirname(args.output_file), exist_ok=True)
        
        # Save the result as JSON
        with open(args.output_file, 'w', encoding='utf-8') as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
        
        print("Transcription and diarization completed successfully!")
        
        # Print summary
        total_segments = len(result.get('segments', []))
        speakers = set(seg.get('speaker') for seg in result.get('segments', []) if seg.get('speaker'))
        
        print(f"Summary:")
        print(f"  - Total segments: {total_segments}")
        if args.diarize:
            print(f"  - Unique speakers: {len(speakers)}")
            print(f"  - Speaker labels: {sorted(speakers) if speakers else 'None'}")
        print(f"  - Language: {result.get('language', 'unknown')}")
        
    except Exception as e:
        print(f"Error during transcription/diarization: {str(e)}", file=sys.stderr)
        return 1
    
    return 0

if __name__ == "__main__":
    sys.exit(main())