import argparse
import json
import os
import whisperx
from diarize import diarize_transcript

# Configure environment variables for HuggingFace
os.environ["HF_HUB_DISABLE_TELEMETRY"] = "1"
os.environ["TRUST_REMOTE_CODE"] = "1"

# Use HuggingFace token from environment variable if available
hf_token = os.environ.get("HF_API_KEY", "")

def main():
    parser = argparse.ArgumentParser(
        description="WhisperX Transcription Script with optional diarization/alignment."
    )
    parser.add_argument(
        "--audio-file",
        type=str,
        required=True,
        help="Path to the audio file to transcribe.",
    )
    parser.add_argument(
        "--model-size",
        type=str,
        default="small",
        help="Whisper model size (e.g., tiny, base, small, medium, large, large-v2).",
    )
    parser.add_argument(
        "--language",
        type=str,
        default=None,
        help="Optional language code to force Whisper to use (e.g., 'en').",
    )
    parser.add_argument(
        "--diarize",
        action="store_true",
        default=False,
        help="Enable speaker diarization.",
    )
    parser.add_argument(
        "--align",
        action="store_true",
        default=False,
        help="Perform alignment on the transcribed segments.",
    )
    parser.add_argument(
        "--device",
        type=str,
        default="cpu",
        help="Device to run WhisperX on (e.g., 'cpu' or 'cuda').",
    )
    parser.add_argument(
        "--compute-type",
        type=str,
        default="int8",
        help="Compute type for WhisperX inference (e.g., 'float16', 'int8').",
    )
    parser.add_argument(
        "--output-file",
        type=str,
        default="transcript.json",
        help="Path to the output JSON file.",
    )
    parser.add_argument(
        "--threads",
        type=int,
        default=1,
        help="number of threads used by torch for CPU inference",
    )
    parser.add_argument(
        "--models-dir",
        type=str,
        default="/scriberr/models",
        help="Directory where models are stored and cached",
    )
    parser.add_argument(
        "--diarization-model",
        type=str,
        default="pyannote/speaker-diarization-3.1",
        help="Speaker diarization model to use",
    )
    parser.add_argument(
        "--batch_size",
        type=int,
        default=16,
        help="Batch size for inference",
    )
    args = parser.parse_args()

    # 1. Load the WhisperX model
    model = whisperx.load_model(
        args.model_size,
        device=args.device,
        compute_type=args.compute_type,
        language=args.language,  # if None, Whisper will attempt language detection
        download_root=args.models_dir  # Specify the download directory
    )

    # 2. Load audio
    audio = whisperx.load_audio(args.audio_file)

    # 3. Transcribe
    result = model.transcribe(audio, batch_size=args.batch_size, print_progress=True)
    # result is a dictionary with keys like "segments", "language", etc.

    # 4. Optionally align the segments
    if args.align:
        # load alignment model
        model_a, metadata = whisperx.load_align_model(
            language_code=result["language"], 
            device=args.device,
            model_dir=args.models_dir  # Specify the model directory
        )
        aligned_result = whisperx.align(
            result["segments"],
            model_a,
            metadata,
            audio,
            args.device,
            return_char_alignments=False,
        )
        # Overwrite the old segments with the aligned segments
        result["segments"] = aligned_result["segments"]

    # 5. Optionally perform diarization
    if args.diarize:
        try:
            diarized_result = diarize_transcript(args.audio_file, result, args.device, args.diarization_model)
            result["segments"] = diarized_result["segments"]
        except Exception as e:
            print(f"Diarization failed: {e}")
            # If diarization fails, we can still save the transcript without speaker labels
            for segment in result["segments"]:
                segment["speaker"] = ""
    else:
        for segment in result["segments"]:
            segment["speaker"] = ""

    with open(args.output_file, "w", encoding="utf-8") as f:
        json.dump(result, f, indent=2, ensure_ascii=False)
        print(f"Transcript saved to {args.output_file}")

if __name__ == "__main__":
    main()