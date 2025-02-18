import argparse
import json
import whisperx

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
        "--HF_TOKEN",
        type=str,
        required=False,
        default=None,
        help="HuggingFace token, necessary for diarization",
    )
    parser.add_argument(
        "--diarization-model",
        type=str,
        default="pyannote/speaker-diarization",
        help="Speaker diarization model to use",
    )
    args = parser.parse_args()

    # 1. Load the WhisperX model
    model = whisperx.load_model(
        args.model_size,
        device=args.device,
        compute_type=args.compute_type,
        language=args.language,  # if None, Whisper will attempt language detection
    )

    # 2. Load audio
    audio = whisperx.load_audio(args.audio_file)

    # 3. Transcribe
    result = model.transcribe(audio, batch_size=16, print_progress=True)
    # result is a dictionary with keys like "segments", "language", etc.

    # 4. Optionally align the segments
    if args.align:
        # load alignment model
        model_a, metadata = whisperx.load_align_model(
            language_code=result["language"], device=args.device
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
        if args.HF_TOKEN:
            # load diarization pipeline
            diarize_model = whisperx.DiarizationPipeline(
                model_name=args.diarization_model,
                use_auth_token=args.HF_TOKEN,
                device=args.device,
            )
            # run diarization
            diarize_segments = diarize_model(audio)
            # assign speaker labels
            diarized_result = whisperx.assign_word_speakers(diarize_segments, result)
            result["segments"] = diarized_result["segments"]
        else:
            print("Hugging Face token not provided. Diarization will not be performed.")
            # Optionally set speaker labels to 'unknown'
            #for segment in result["segments"]:
                #segment["speaker"] = "unknown"
    else:
        # If diarization not requested, set speakers to blank
        for segment in result["segments"]:
            segment["speaker"] = ""


    # 6. Write the result to the output file
    with open(args.output_file, "w", encoding="utf-8") as f:
        json.dump(result, f, indent=2, ensure_ascii=False)

if __name__ == "__main__":
    main()