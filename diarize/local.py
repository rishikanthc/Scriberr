from pathlib import Path
from pyannote.audio import Pipeline
from pyannote.audio.pipelines.utils.hook import ProgressHook
import os
import argparse

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

def load_pipeline_from_pretrained(path_to_config: str | Path) -> Pipeline:
    path_to_config = Path(path_to_config)

    print(f"Loading pyannote pipeline from {path_to_config}...")
    # the paths in the config are relative to the current working directory
    # so we need to change the working directory to the model path
    # and then change it back

    cwd = Path.cwd().resolve()  # store current working directory

    # first .parent is the folder of the config, second .parent is the folder containing the 'models' folder
    cd_to = path_to_config.parent.parent.resolve()

    print(f"Changing working directory to {cd_to}")
    os.chdir(cd_to)

    pipeline = Pipeline.from_pretrained(checkpoint_path=path_to_config, hparams_file=path_to_config)

    print(f"Changing working directory back to {cwd}")
    os.chdir(cwd)

    return pipeline

# PATH_TO_CONFIG = "/Users/richandrasekaran/Code/Scriberr/diarize/models/pyannote_diarization_config.yaml"
PATH_TO_CONFIG = "/app/diarize/models/pyannote_diarization_config.yaml"
pipeline = load_pipeline_from_pretrained(PATH_TO_CONFIG)


# diarization = pipeline(args.input_audio)

with ProgressHook() as hook:
    diarization = pipeline(args.input_audio, hook=hook)

# # Write the diarization output to the specified RTTM file
with open(args.output_rttm, "w") as rttm:
    diarization.write_rttm(rttm)

print(f"Speaker diarization completed. RTTM file saved at: {args.output_rttm}")
