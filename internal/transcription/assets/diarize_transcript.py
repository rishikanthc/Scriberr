#!/usr/bin/env python3
"""
Minimal diarization wrapper around whisperx.

Usage:
  python diarize_transcript.py <audio_path> <output_json> \
    --model <whisper_model> --device <cpu|cuda> --compute_type <float16|int8|...> \
    --batch_size <int> [--language <code>] --diarize \
    --diarize_model <hf_pipeline> [--min_speakers N] [--max_speakers N] [--hf_token TOKEN]

Writes a JSON with keys: segments, word_segments (if available), language.
"""
import argparse
import json
import os
import sys


def main():
    try:
        import torch  # noqa: F401
        import whisperx
    except Exception as e:
        print(f"Failed to import dependencies: {e}", file=sys.stderr)
        sys.exit(1)

    p = argparse.ArgumentParser()
    p.add_argument("audio_path")
    p.add_argument("output_json")
    
    # Core model parameters
    p.add_argument("--model", default="small")
    p.add_argument("--device", default="cpu")
    p.add_argument("--compute_type", default="float32")
    p.add_argument("--batch_size", type=int, default=8)
    p.add_argument("--threads", type=int, default=0)
    
    # Language
    p.add_argument("--language", default=None)
    
    # Alignment parameters
    p.add_argument("--align_model", default=None)
    p.add_argument("--interpolate_method", default="nearest")
    p.add_argument("--no_align", action="store_true")
    p.add_argument("--return_char_alignments", action="store_true")
    
    # Diarization parameters
    p.add_argument("--diarize", action="store_true")
    p.add_argument("--diarize_model", default="pyannote/speaker-diarization-3.1")
    p.add_argument("--min_speakers", type=int, default=None)
    p.add_argument("--max_speakers", type=int, default=None)
    p.add_argument("--speaker_embeddings", action="store_true")
    p.add_argument("--hf_token", default=None)
    args = p.parse_args()

    audio_path = args.audio_path
    out_path = args.output_json
    os.makedirs(os.path.dirname(out_path), exist_ok=True)

    device = args.device
    
    # Load model with threading support
    load_kwargs = {"compute_type": args.compute_type}
    if args.threads > 0:
        load_kwargs["threads"] = args.threads
    model = whisperx.load_model(args.model, device, **load_kwargs)
    
    audio = whisperx.load_audio(audio_path)
    
    # Build transcribe parameters (only parameters that transcribe() actually accepts)
    transcribe_kwargs = {
        "batch_size": int(args.batch_size)
    }
    
    # Language is the only additional parameter that transcribe() accepts
    if args.language:
        transcribe_kwargs["language"] = args.language
        
    result = model.transcribe(audio, **transcribe_kwargs)

    # Alignment step for better word timings (if not disabled)
    if not args.no_align:
        try:
            # Use custom align model if specified
            language_code = result.get("language")
            if args.align_model:
                model_a, metadata = whisperx.load_align_model(model_name=args.align_model, language_code=language_code, device=device)
            else:
                model_a, metadata = whisperx.load_align_model(language_code=language_code, device=device)
            
            # Build alignment parameters
            align_kwargs = {
                "interpolate_method": args.interpolate_method,
                "return_char_alignments": args.return_char_alignments
            }
            
            result_aligned = whisperx.align(result["segments"], model_a, metadata, audio, device, **align_kwargs)
            result["segments"] = result_aligned["segments"]
            if "word_segments" in result_aligned:
                result["word_segments"] = result_aligned["word_segments"]
        except Exception as e:
            print(f"Alignment failed: {e}", file=sys.stderr)

    # Optional diarization
    if args.diarize:
        try:
            # Use the correct WhisperX diarization API
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
        except Exception as e:
            print(f"Diarization failed: {e}", file=sys.stderr)

    out = {
        "segments": result.get("segments", []),
        "word_segments": result.get("word_segments", []),
        "language": result.get("language", "")
    }
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump(out, f, ensure_ascii=False)

    print(f"Wrote {out_path}")


if __name__ == "__main__":
    main()

