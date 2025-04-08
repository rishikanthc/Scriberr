import os
import whisperx

# Configure environment variables for HuggingFace
os.environ["HF_HUB_DISABLE_TELEMETRY"] = "1"
os.environ["TRUST_REMOTE_CODE"] = "1"

# Use HuggingFace token from environment variable if available
hf_token = os.environ.get("HF_API_KEY", "")

def diarize_transcript(audio_file, transcript, device="cpu", model_name="pyannote/speaker-diarization-3.1"):
    """
    Performs speaker diarization on a transcript, splitting segments by speaker changes.

    Args:
        audio_file (str): Path to the audio file.
        transcript (dict): WhisperX transcript result with segments and words.
        device (str): Device to run diarization on (e.g., 'cpu' or 'cuda').
        model_name (str): Diarization model to use.

    Returns:
        dict: Transcript with segments split by individual speakers.
    """
    try:
        # Clear CUDA cache before loading diarization model
        if device == "cuda":
            try:
                import torch
                torch.cuda.empty_cache()
                print("Cleared CUDA cache before diarization")
                
                # Set memory management for PyTorch
                if os.environ.get("PYTORCH_CUDA_ALLOC_CONF") is None:
                    os.environ["PYTORCH_CUDA_ALLOC_CONF"] = "expandable_segments:True"
                    print("Set PYTORCH_CUDA_ALLOC_CONF to expandable_segments:True")
            except ImportError:
                print("Could not import torch for CUDA memory management")
        
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

        # Step 1: Collect all words from segments
        all_words = []
        for segment in diarized_result.get("segments", []):
            for word in segment.get("words", []):
                # Ensure word has required keys
                if "start" in word and "end" in word and "word" in word:
                    all_words.append(word)
                else:
                    print(f"Warning: Skipping malformed word: {word}")

        # Step 2: Sort words by start time
        all_words.sort(key=lambda x: x["start"])

        # Step 3: Group words by consecutive speakers
        new_segments = []
        current_segment = None

        for word in all_words:
            speaker = word.get("speaker", "unknown")  # Default to "unknown" if speaker is missing
            if not speaker or speaker.lower() == "unknown":
                speaker = "unknown"  # Standardize unknown labels

            if current_segment is None:
                # Start the first segment
                current_segment = {
                    "start": word["start"],
                    "end": word["end"],
                    "text": word["word"],
                    "speaker": speaker,
                    "words": [word]
                }
            elif speaker == current_segment["speaker"]:
                # Same speaker, extend the current segment
                current_segment["end"] = word["end"]
                current_segment["text"] += " " + word["word"]
                current_segment["words"].append(word)
            else:
                # Different speaker, save current segment and start a new one
                new_segments.append(current_segment)
                current_segment = {
                    "start": word["start"],
                    "end": word["end"],
                    "text": word["word"],
                    "speaker": speaker,
                    "words": [word]
                }

        # Step 4: Add the last segment if it exists
        if current_segment:
            new_segments.append(current_segment)

        # Step 5: Return the new segments
        print(f"Created {len(new_segments)} speaker-specific segments")
        
        # Clear memory after processing
        if device == "cuda":
            try:
                import torch
                # Delete model to free memory
                del diarize_model
                # Explicit garbage collection
                import gc
                gc.collect()
                # Clear CUDA cache again
                torch.cuda.empty_cache()
                print("Cleared memory after diarization")
            except ImportError:
                pass
                
        return {"segments": new_segments}

    except Exception as e:
        print(f"Diarization failed: {str(e)}")
        # Fallback: Assign "unknown" to original segments
        for segment in transcript["segments"]:
            segment["speaker"] = "unknown"
        return transcript