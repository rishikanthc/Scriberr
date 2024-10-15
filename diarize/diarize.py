import argparse
from pyannote.audio import Pipeline
from pyannote.audio.pipelines.utils.hook import ProgressHook

# Set up the argument parser
parser = argparse.ArgumentParser(
    description="Run speaker diarization on an audio file and save the output in RTTM format."
)

# Add arguments for input and output file paths
parser.add_argument(
    "input_audio", type=str, help="Path to the input audio file (e.g., jfk.wav)"
)
parser.add_argument(
    "output_rttm", type=str, help="Path to the output RTTM file (e.g., audio.rttm)"
)

# Parse the command-line arguments
args = parser.parse_args()

# Load the PyAnnote pipeline
pipeline = Pipeline.from_pretrained(
    "pyannote/speaker-diarization-3.1",
    use_auth_token="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",
)

# Run the pipeline on the input audio file
with ProgressHook() as hook:
    diarization = pipeline(args.input_audio, hook=hook)

# Write the diarization output to the specified RTTM file
with open(args.output_rttm, "w") as rttm:
    diarization.write_rttm(rttm)

print(f"Speaker diarization completed. RTTM file saved at: {args.output_rttm}")
