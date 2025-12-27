#!/usr/bin/env python3
"""
NVIDIA Parakeet buffered inference for long audio files.
Splits audio into chunks to avoid GPU memory issues.
"""

import argparse
import json
import sys
import os
import librosa
import soundfile as sf
import numpy as np
from pathlib import Path
import nemo.collections.asr as nemo_asr


def split_audio_file(audio_path, chunk_duration_secs=300):
    """Split audio file into chunks of specified duration."""
    audio, sr = librosa.load(audio_path, sr=None, mono=True)
    total_duration = len(audio) / sr
    chunk_samples = int(chunk_duration_secs * sr)

    chunks = []
    for start_sample in range(0, len(audio), chunk_samples):
        end_sample = min(start_sample + chunk_samples, len(audio))
        chunk_audio = audio[start_sample:end_sample]
        start_time = start_sample / sr
        chunks.append({
            'audio': chunk_audio,
            'start_time': start_time,
            'duration': len(chunk_audio) / sr
        })

    return chunks, sr


def transcribe_buffered(
    audio_path: str,
    output_file: str = None,
    chunk_duration_secs: float = 300,  # 5 minutes default
):
    """
    Transcribe long audio by splitting into chunks and merging results.
    """
    # Determine model path
    model_filename = "parakeet-tdt-0.6b-v3.nemo"
    model_path = None

    # Locate project root: derived from VIRTUAL_ENV, which is set by `uv run` to path/.venv
    virtual_env = os.environ.get("VIRTUAL_ENV")
    if not virtual_env:
        print("Error: VIRTUAL_ENV environment variable not set. Script must be run with 'uv run'.")
        sys.exit(1)

    project_root = os.path.dirname(virtual_env)
    model_path = os.path.join(project_root, model_filename)

    if not os.path.exists(model_path):
        print(f"Error during transcription: Can't find {model_filename} in project root: {project_root}")
        sys.exit(1)

    print(f"Loading NVIDIA Parakeet model from: {model_path}")

    asr_model = nemo_asr.models.ASRModel.restore_from(model_path)

    # Disable CUDA graphs to fix Error 35 on RTX 2000e Ada GPU
    # Uses change_decoding_strategy() to properly reconfigure the TDT decoder
    from omegaconf import OmegaConf, open_dict

    print("Disabling CUDA graphs in TDT decoder...")
    dec_cfg = asr_model.cfg.decoding

    # Add use_cuda_graph_decoder parameter to greedy config
    with open_dict(dec_cfg.greedy):
        dec_cfg.greedy['use_cuda_graph_decoder'] = False

    # Apply the new decoding strategy (this rebuilds the decoder with our config)
    asr_model.change_decoding_strategy(dec_cfg)
    print("✓ CUDA graphs disabled successfully")

    print(f"Splitting audio into {chunk_duration_secs}s chunks...")
    chunks, sr = split_audio_file(audio_path, chunk_duration_secs)
    print(f"Created {len(chunks)} chunks")

    all_words = []
    all_segments = []
    full_text = []

    for i, chunk_info in enumerate(chunks):
        print(f"Transcribing chunk {i+1}/{len(chunks)} (duration: {chunk_info['duration']:.1f}s)...")

        # Save chunk to temporary file
        chunk_path = f"/tmp/chunk_{i}.wav"
        sf.write(chunk_path, chunk_info['audio'], sr)

        try:
            # Transcribe chunk
            output = asr_model.transcribe(
                [chunk_path],
                batch_size=1,
                timestamps=True,
            )

            result_data = output[0]
            chunk_text = result_data.text
            full_text.append(chunk_text)

            # Extract and adjust timestamps
            if hasattr(result_data, 'timestamp') and result_data.timestamp:
                chunk_words = result_data.timestamp.get("word", [])
                chunk_segments = result_data.timestamp.get("segment", [])

                # Adjust timestamps by chunk start time
                for word in chunk_words:
                    word_copy = dict(word)
                    word_copy['start'] += chunk_info['start_time']
                    word_copy['end'] += chunk_info['start_time']
                    all_words.append(word_copy)

                for segment in chunk_segments:
                    seg_copy = dict(segment)
                    seg_copy['start'] += chunk_info['start_time']
                    seg_copy['end'] += chunk_info['start_time']
                    all_segments.append(seg_copy)

            print(f"Chunk {i+1} complete: {len(chunk_text)} characters")

        finally:
            # Clean up temp file
            if os.path.exists(chunk_path):
                os.remove(chunk_path)

    final_text = " ".join(full_text)
    print(f"Transcription complete: {len(final_text)} characters total")

    output_data = {
        "transcription": final_text,
        "language": "en",
        "word_timestamps": all_words,
        "segment_timestamps": all_segments,
        "audio_file": audio_path,
        "model": "parakeet-tdt-0.6b-v3",
        "buffered": True,
        "chunk_duration_secs": chunk_duration_secs,
        "num_chunks": len(chunks),
    }

    if output_file:
        with open(output_file, 'w', encoding='utf-8') as f:
            json.dump(output_data, f, indent=2, ensure_ascii=False)
        print(f"Results saved to: {output_file}")
    else:
        print(json.dumps(output_data, indent=2, ensure_ascii=False))


def main():
    parser = argparse.ArgumentParser(
        description="Transcribe long audio using NVIDIA Parakeet with chunking"
    )
    parser.add_argument("audio_file", help="Path to audio file")
    parser.add_argument("--output", "-o", help="Output file path", required=True)
    parser.add_argument(
        "--chunk-len", type=float, default=300,
        help="Chunk duration in seconds (default: 300 = 5 minutes)"
    )

    args = parser.parse_args()

    if not os.path.exists(args.audio_file):
        print(f"Error: Audio file not found: {args.audio_file}")
        sys.exit(1)

    transcribe_buffered(
        audio_path=args.audio_file,
        output_file=args.output,
        chunk_duration_secs=args.chunk_len,
    )


if __name__ == "__main__":
    main()
